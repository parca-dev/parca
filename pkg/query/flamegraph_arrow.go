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

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/slices"

	"github.com/parca-dev/parca/pkg/profile"
)

const (
	FlamegraphFieldMappingStart   = "mapping_start"
	FlamegraphFieldMappingLimit   = "mapping_limit"
	FlamegraphFieldMappingOffset  = "mapping_offset"
	FlamegraphFieldMappingFile    = "mapping_file"
	FlamegraphFieldMappingBuildID = "mapping_build_id"

	FlamegraphFieldLocationAddress = "location_address"
	FlamegraphFieldLocationFolded  = "location_folded"
	FlamegraphFieldLocationLine    = "location_line"

	FlamegraphFieldFunctionStartLine  = "function_startline"
	FlamegraphFieldFunctionName       = "function_name"
	FlamegraphFieldFunctionSystemName = "function_system_name"
	FlamegraphFieldFunctionFileName   = "function_file_name"

	FlamegraphFieldChildren   = "children"
	FlamegraphFieldCumulative = "cumulative"
	FlamegraphFieldDiff       = "diff"
)

func GenerateFlamegraphArrow(ctx context.Context, tracer trace.Tracer, p *profile.Profile, groupBy []string, trimFraction float32) (arrow.Record, error) {
	mem := memory.NewGoAllocator()

	aggregateFields := map[string]struct{}{
		// TODO: Add pprof labels by default
		FlamegraphFieldMappingFile:  {},
		FlamegraphFieldFunctionName: {},
	}
	for _, f := range groupBy {
		// don't aggregate by fields that we should group by.
		delete(aggregateFields, f)
	}

	schema := arrow.NewSchema([]arrow.Field{
		{Name: FlamegraphFieldMappingStart, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingLimit, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingOffset, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingFile, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: FlamegraphFieldMappingBuildID, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		// Location
		{Name: FlamegraphFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldLocationFolded, Type: &arrow.BooleanType{}},
		{Name: FlamegraphFieldLocationLine, Type: arrow.PrimitiveTypes.Int64},
		// Function
		{Name: FlamegraphFieldFunctionStartLine, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldFunctionName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		{Name: FlamegraphFieldFunctionSystemName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: FlamegraphFieldFunctionFileName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		// Values
		{Name: FlamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: FlamegraphFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	}, nil)

	rb := builder.NewRecordBuilder(mem, schema)

	// TODO: Potentially good to .Reserve() the number of samples to avoid re-allocations
	builderMappingStart := rb.Field(schema.FieldIndices(FlamegraphFieldMappingStart)[0]).(*array.Uint64Builder)
	builderMappingLimit := rb.Field(schema.FieldIndices(FlamegraphFieldMappingLimit)[0]).(*array.Uint64Builder)
	builderMappingOffset := rb.Field(schema.FieldIndices(FlamegraphFieldMappingOffset)[0]).(*array.Uint64Builder)
	builderMappingFile := rb.Field(schema.FieldIndices(FlamegraphFieldMappingFile)[0]).(*array.BinaryDictionaryBuilder)
	builderMappingBuildID := rb.Field(schema.FieldIndices(FlamegraphFieldMappingBuildID)[0]).(*array.BinaryDictionaryBuilder)

	builderLocationAddress := rb.Field(schema.FieldIndices(FlamegraphFieldLocationAddress)[0]).(*array.Uint64Builder)
	builderLocationFolded := rb.Field(schema.FieldIndices(FlamegraphFieldLocationFolded)[0]).(*builder.OptBooleanBuilder)
	builderLocationLine := rb.Field(schema.FieldIndices(FlamegraphFieldLocationLine)[0]).(*builder.OptInt64Builder)

	builderFunctionStartLine := rb.Field(schema.FieldIndices(FlamegraphFieldFunctionStartLine)[0]).(*builder.OptInt64Builder)
	builderFunctionName := rb.Field(schema.FieldIndices(FlamegraphFieldFunctionName)[0]).(*array.BinaryDictionaryBuilder)
	builderFunctionSystemName := rb.Field(schema.FieldIndices(FlamegraphFieldFunctionSystemName)[0]).(*array.BinaryDictionaryBuilder)
	builderFunctionFileName := rb.Field(schema.FieldIndices(FlamegraphFieldFunctionFileName)[0]).(*array.BinaryDictionaryBuilder)

	builderChildren := rb.Field(schema.FieldIndices(FlamegraphFieldChildren)[0]).(*builder.ListBuilder)
	builderChildrenValues := builderChildren.ValueBuilder().(*array.Uint32Builder)
	builderCumulative := rb.Field(schema.FieldIndices(FlamegraphFieldCumulative)[0]).(*builder.OptInt64Builder)
	builderDiff := rb.Field(schema.FieldIndices(FlamegraphFieldDiff)[0]).(*builder.OptInt64Builder)

	// This field compares the current sample with the already added values in the builders.
	equalField := func(fieldName string, location *profile.Location, line profile.LocationLine, row uint32) bool {
		switch fieldName {
		case FlamegraphFieldMappingStart:
			rowMappingFile := builderMappingFile.ValueStr(builderMappingFile.GetValueIndex(int(row)))
			return location.Mapping.File == rowMappingFile
		case FlamegraphFieldFunctionName:
			rowFunctionName := builderFunctionName.ValueStr(builderFunctionName.GetValueIndex(int(row)))
			return line.Function.Name == rowFunctionName
		default:
			return false
		}
	}

	// The very first row is the root row. It doesn't contain any metadata.
	// It only contains the root cumulative value and list of children (which are actual roots).
	builderMappingStart.AppendNull()
	builderMappingLimit.AppendNull()
	builderMappingOffset.AppendNull()
	builderMappingFile.AppendNull()
	builderMappingBuildID.AppendNull()
	builderLocationAddress.AppendNull()
	builderLocationFolded.AppendNull()
	builderLocationLine.AppendNull()
	builderFunctionStartLine.AppendNull()
	builderFunctionName.AppendNull()
	builderFunctionSystemName.AppendNull()
	builderFunctionFileName.AppendNull()
	builderCumulative.AppendNull()
	builderDiff.AppendNull()

	cumulative := int64(0)
	rootsRow := []uint32{}
	children := make([][]uint32, len(p.Samples))

	// these change with every iteration below
	row := uint32(builderCumulative.Len())
	parent := -1
	compareRows := []uint32{}

	for _, s := range p.Samples {
		// every new sample resets the childRow to -1 indicating that we start with a leaf again.
		for i := len(s.Locations) - 1; i >= 0; i-- {
			location := s.Locations[i]
		stacktraces:
			for _, line := range location.Lines {
				if i == len(s.Locations)-1 { // root of the stacktrace
					compareRows = compareRows[:0] //  reset the compare rows
					compareRows = append(compareRows, rootsRow...)
					// append this row afterward to not compare to itself
					parent = -1
				}
				if i == 0 { // leaf of the stacktrace
					cumulative += s.Value
				}

				// If there are no fields we should aggregate we can skip the comparison
				if len(aggregateFields) > 0 {
				compareRows:
					for _, cr := range compareRows {
						for f := range aggregateFields {
							if !equalField(f, location, line, cr) {
								// If any field doesn't match, we can't aggregate this row with the existing one.
								continue compareRows
							}
						}

						// All fields match, so we can aggregate this new row with the existing one.
						builderCumulative.Add(int(cr), s.Value)
						// Continue with this row as the parent for the next iteration and compare to its children.
						parent = int(cr)
						compareRows = children[cr]
						continue stacktraces
					}
				}

				if i == len(s.Locations)-1 { // root of the stacktrace
					// We aren't merging this root, so we'll keep track of it as a new one.
					rootsRow = append(rootsRow, row)
				}

				for j := range rb.Fields() {
					switch schema.Field(j).Name {
					// Mapping
					case FlamegraphFieldMappingStart:
						if location.Mapping != nil && location.Mapping.Start > 0 {
							builderMappingStart.Append(location.Mapping.Start)
						} else {
							builderMappingStart.AppendNull()
						}
					case FlamegraphFieldMappingLimit:
						if location.Mapping != nil && location.Mapping.Limit > 0 {
							builderMappingLimit.Append(location.Mapping.Limit)
						} else {
							builderMappingLimit.AppendNull()
						}
					case FlamegraphFieldMappingOffset:
						if location.Mapping != nil && location.Mapping.Offset > 0 {
							builderMappingOffset.Append(location.Mapping.Offset)
						} else {
							builderMappingOffset.AppendNull()
						}
					case FlamegraphFieldMappingFile:
						if location.Mapping != nil && location.Mapping.File != "" {
							_ = builderMappingFile.AppendString(location.Mapping.File)
						} else {
							builderMappingFile.AppendNull()
						}
					case FlamegraphFieldMappingBuildID:
						if location.Mapping != nil && location.Mapping.BuildId != "" {
							_ = builderMappingBuildID.AppendString(location.Mapping.BuildId)
						} else {
							builderMappingBuildID.AppendNull()
						}
					// Location
					case FlamegraphFieldLocationAddress:
						builderLocationAddress.Append(location.Address)
					case FlamegraphFieldLocationFolded:
						builderLocationFolded.AppendSingle(location.IsFolded)
					case FlamegraphFieldLocationLine:
						builderLocationLine.Append(line.Line)
					// Function
					case FlamegraphFieldFunctionStartLine:
						builderFunctionStartLine.Append(line.Function.StartLine)
					case FlamegraphFieldFunctionName:
						_ = builderFunctionName.AppendString(line.Function.Name)
					case FlamegraphFieldFunctionSystemName:
						_ = builderFunctionSystemName.AppendString(line.Function.SystemName)
					case FlamegraphFieldFunctionFileName:
						_ = builderFunctionFileName.AppendString(line.Function.Filename)
					// Values
					case FlamegraphFieldChildren:
						if uint32(len(children)) == row {
							children = slices.Grow(children, len(children))
							children = children[:cap(children)]
						}
						if parent > -1 {
							if len(children[parent]) == 0 {
								children[parent] = []uint32{row}
							} else {
								children[parent] = append(children[parent], row)
							}
						}
					case FlamegraphFieldCumulative:
						builderCumulative.Append(s.Value)
					case FlamegraphFieldDiff:
						if s.DiffValue > 0 {
							builderDiff.Append(s.DiffValue)
						} else {
							builderDiff.AppendNull()
						}
					default:
						panic(fmt.Sprintf("unknown field %s", schema.Field(j).Name))
					}
				}
				parent = int(row)
				row = uint32(builderCumulative.Len())
			}
		}
	}

	builderCumulative.Set(0, cumulative)

	for i := 0; i < builderCumulative.Len(); i++ {
		if i == 0 {
			builderChildren.Append(true)
			for _, child := range rootsRow {
				builderChildrenValues.Append(child)
			}
			continue
		}
		if len(children[i]) == 0 {
			builderChildren.AppendNull() // leaf
		} else {
			builderChildren.Append(true)
			for _, child := range children[i] {
				builderChildrenValues.Append(child)
			}
		}
	}

	return rb.NewRecord(), nil
}
