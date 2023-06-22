// Copyright 2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package query

import (
	"context"
	"fmt"

	"github.com/apache/arrow/go/v10/arrow"
	"github.com/apache/arrow/go/v10/arrow/array"
	"github.com/apache/arrow/go/v10/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/trace"

	"github.com/parca-dev/parca/pkg/profile"
)

const (
	flamegraphFieldMappingStart   = "mapping_start"
	flamegraphFieldMappingLimit   = "mapping_limit"
	flamegraphFieldMappingOffset  = "mapping_offset"
	flamegraphFieldMappingFile    = "mapping_file"
	flamegraphFieldMappingBuildID = "mapping_build_id"

	flamegraphFieldLocationAddress = "location_address"
	flamegraphFieldLocationFolded  = "location_folded"
	flamegraphFieldLocationLine    = "location_line"

	flamegraphFieldFunctionStartLine  = "function_startline"
	flamegraphFieldFunctionName       = "function_name"
	flamegraphFieldFunctionSystemName = "function_system_name"
	flamegraphFieldFunctionFileName   = "function_file_name"

	flamegraphFieldChildren   = "children"
	flamegraphFieldCumulative = "cumulative"
	flamegraphFieldDiff       = "diff"
)

func GenerateFlamegraphArrow(ctx context.Context, tracer trace.Tracer, p *profile.Profile, trimFraction float32) (arrow.Record, error) {
	return convertSymbolizedProfile(p)
}

func convertSymbolizedProfile(p *profile.Profile) (arrow.Record, error) {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: flamegraphFieldMappingStart, Type: arrow.PrimitiveTypes.Uint64},
		{Name: flamegraphFieldMappingLimit, Type: arrow.PrimitiveTypes.Uint64},
		{Name: flamegraphFieldMappingOffset, Type: arrow.PrimitiveTypes.Uint64},
		{Name: flamegraphFieldMappingFile, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: flamegraphFieldMappingBuildID, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		// Location
		{Name: flamegraphFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
		{Name: flamegraphFieldLocationFolded, Type: &arrow.BooleanType{}},
		{Name: flamegraphFieldLocationLine, Type: arrow.PrimitiveTypes.Int64},
		// Function
		{Name: flamegraphFieldFunctionStartLine, Type: arrow.PrimitiveTypes.Int64},
		{Name: flamegraphFieldFunctionName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		{Name: flamegraphFieldFunctionSystemName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: flamegraphFieldFunctionFileName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		// Values
		{Name: flamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: flamegraphFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: flamegraphFieldDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	}, nil)

	mem := memory.NewGoAllocator()
	rb := builder.NewRecordBuilder(mem, schema)

	// TODO: Potentially good to .Reserve() the number of samples to avoid reallocations
	builderMappingStart := rb.Field(schema.FieldIndices(flamegraphFieldMappingStart)[0]).(*array.Uint64Builder)
	builderMappingLimit := rb.Field(schema.FieldIndices(flamegraphFieldMappingLimit)[0]).(*array.Uint64Builder)
	builderMappingOffset := rb.Field(schema.FieldIndices(flamegraphFieldMappingOffset)[0]).(*array.Uint64Builder)
	builderMappingFile := rb.Field(schema.FieldIndices(flamegraphFieldMappingFile)[0]).(*array.BinaryDictionaryBuilder)
	builderMappingBuildID := rb.Field(schema.FieldIndices(flamegraphFieldMappingBuildID)[0]).(*array.BinaryDictionaryBuilder)

	builderLocationAddress := rb.Field(schema.FieldIndices(flamegraphFieldLocationAddress)[0]).(*array.Uint64Builder)
	builderLocationFolded := rb.Field(schema.FieldIndices(flamegraphFieldLocationFolded)[0]).(*builder.OptBooleanBuilder)
	builderLocationLine := rb.Field(schema.FieldIndices(flamegraphFieldLocationLine)[0]).(*builder.OptInt64Builder)

	builderFunctionStartLine := rb.Field(schema.FieldIndices(flamegraphFieldFunctionStartLine)[0]).(*builder.OptInt64Builder)
	builderFunctionName := rb.Field(schema.FieldIndices(flamegraphFieldFunctionName)[0]).(*array.BinaryDictionaryBuilder)
	builderFunctionSystemName := rb.Field(schema.FieldIndices(flamegraphFieldFunctionSystemName)[0]).(*array.BinaryDictionaryBuilder)
	builderFunctionFileName := rb.Field(schema.FieldIndices(flamegraphFieldFunctionFileName)[0]).(*array.BinaryDictionaryBuilder)

	builderChildren := rb.Field(schema.FieldIndices(flamegraphFieldChildren)[0]).(*builder.ListBuilder)
	builderChildrenValues := builderChildren.ValueBuilder().(*array.Uint32Builder)
	builderCumulative := rb.Field(schema.FieldIndices(flamegraphFieldCumulative)[0]).(*builder.OptInt64Builder)
	builderDiff := rb.Field(schema.FieldIndices(flamegraphFieldDiff)[0]).(*builder.OptInt64Builder)

	// start with -1 so the first row++ will be 0
	row := -1
	for _, s := range p.Samples {
		// every new sample resets the childRow to -1 indicating that we start with a leaf again.
		childRow := -1
		for i := len(s.Locations) - 1; i >= 0; i-- {
			location := s.Locations[i]
			for _, line := range location.Lines {
				row++
				for j := range rb.Fields() {
					switch schema.Field(j).Name {
					// Mapping
					case flamegraphFieldMappingStart:
						if location.Mapping.Start > 0 {
							builderMappingStart.Append(location.Mapping.Start)
						} else {
							builderMappingStart.AppendNull()
						}
					case flamegraphFieldMappingLimit:
						builderMappingLimit.Append(location.Mapping.Limit)
					case flamegraphFieldMappingOffset:
						builderMappingOffset.Append(location.Mapping.Offset)
					case flamegraphFieldMappingFile:
						if location.Mapping.File != "" {
							_ = builderMappingFile.AppendString(location.Mapping.File)
						} else {
							builderMappingFile.AppendNull()
						}
					case flamegraphFieldMappingBuildID:
						if location.Mapping.BuildId != "" {
							_ = builderMappingBuildID.AppendString(location.Mapping.BuildId)
						} else {
							builderMappingBuildID.AppendNull()
						}
					// Location
					case flamegraphFieldLocationAddress:
						builderLocationAddress.Append(location.Address)
					case flamegraphFieldLocationFolded:
						builderLocationFolded.AppendSingle(location.IsFolded)
					case flamegraphFieldLocationLine:
						builderLocationLine.Append(line.Line)
					// Function
					case flamegraphFieldFunctionStartLine:
						builderFunctionStartLine.Append(line.Function.StartLine)
					case flamegraphFieldFunctionName:
						_ = builderFunctionName.AppendString(line.Function.Name)
					case flamegraphFieldFunctionSystemName:
						_ = builderFunctionSystemName.AppendString(line.Function.SystemName)
					case flamegraphFieldFunctionFileName:
						_ = builderFunctionFileName.AppendString(line.Function.Filename)
					// Values
					case flamegraphFieldChildren:
						if childRow >= 0 {
							builderChildren.Append(true)
							builderChildrenValues.Append(uint32(childRow))
						} else {
							builderChildren.AppendNull() // leaf
						}
					case flamegraphFieldCumulative:
						builderCumulative.Append(s.Value)
					case flamegraphFieldDiff:
						if s.DiffValue > 0 {
							builderDiff.Append(s.DiffValue)
						} else {
							builderDiff.AppendNull()
						}
					default:
						panic(fmt.Sprintf("unknown field %s", schema.Field(j).Name))
					}
				}
				childRow = row
			}
		}
	}

	return rb.NewRecord(), nil
}
