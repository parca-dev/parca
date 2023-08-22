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
	"bytes"
	"context"
	"fmt"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/ipc"
	"github.com/apache/arrow/go/v13/arrow/math"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/trace"

	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	TableFieldMappingStart   = "mapping_start"
	TableFieldMappingLimit   = "mapping_limit"
	TableFieldMappingOffset  = "mapping_offset"
	TableFieldMappingFile    = "mapping_file"
	TableFieldMappingBuildID = "mapping_build_id"

	TableFieldLocationAddress = "location_address"
	TableFieldLocationFolded  = "location_folded"
	TableFieldLocationLine    = "location_line"

	TableFieldFunctionStartLine  = "function_startline"
	TableFieldFunctionName       = "function_name"
	TableFieldFunctionSystemName = "function_system_name"
	TableFieldFunctionFileName   = "function_file_name"

	TableFieldCumulative     = "cumulative"
	TableFieldCumulativeDiff = "cumulative_diff"
	TableFieldFlat           = "flat"
	TableFieldFlatDiff       = "flat_diff"
)

func GenerateTable(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
) (*queryv1alpha1.TableArrow, int64, error) {
	ctx, span := tracer.Start(ctx, "GenerateTable")
	defer span.End()

	record, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, p)
	if err != nil {
		return nil, 0, err
	}
	defer record.Release()

	// TODO: Reuse buffer and potentially writers
	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(mem),
	)
	defer w.Close()

	if err = w.Write(record); err != nil {
		return nil, 0, err
	}

	return &queryv1alpha1.TableArrow{
		Record: buf.Bytes(),
		Unit:   p.Meta.SampleType.Unit,
	}, cumulative, nil
}

func generateTableArrowRecord(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
) (arrow.Record, int64, error) {
	_, span := tracer.Start(ctx, "generateTableArrowRecord")
	defer span.End()

	tb := newTableBuilder(mem)
	defer tb.Release()

	row := 0

	profileReader := profile.NewReader(p)
	for _, r := range profileReader.RecordReaders {
		tb.cumulative += math.Int64.Sum(r.Value)

		for sampleRow := 0; sampleRow < int(r.Record.NumRows()); sampleRow++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(sampleRow)
			for locationRow := int(lOffsetStart); locationRow < int(lOffsetEnd); locationRow++ {
				isLeaf := locationRow == int(lOffsetEnd)-1

				if r.Lines.IsNull(locationRow) {
					var buildID []byte
					if r.MappingBuildIDDict.IsValid(locationRow) {
						buildID = r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(locationRow))
					}
					addr := r.Address.Value(locationRow)

					// Check if we've seen the address for the mapping before.
					// If not, we add it as a new row and add the address to the mapping to keep track of it.
					// If we have seen the address before, we merge the address with the existing row by summing the values.
					if cr, ok := tb.addresses[unsafeString(buildID)][addr]; !ok {
						if err := tb.appendRow(r, sampleRow, locationRow, -1, isLeaf); err != nil {
							return nil, 0, err
						}

						if _, ok := tb.addresses[unsafeString(buildID)]; !ok {
							tb.addresses[string(buildID)] = map[uint64]int{addr: row}
						} else {
							tb.addresses[string(buildID)][addr] = row
						}
						row++
					} else {
						tb.mergeRow(r, cr, sampleRow)
					}
				}

				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(locationRow)
				for lineRow := int(llOffsetStart); lineRow < int(llOffsetEnd); lineRow++ {
					if r.Line.IsValid(lineRow) && r.LineFunction.IsValid(lineRow) {
						fn := r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow))
						if cr, ok := tb.functions[unsafeString(fn)]; !ok {
							if err := tb.appendRow(r, sampleRow, locationRow, lineRow, isLeaf); err != nil {
								return nil, 0, err
							}
							tb.functions[string(fn)] = row
							row++
						} else {
							tb.mergeRow(r, cr, sampleRow)
						}
					}
				}
			}
		}
	}

	rec, err := tb.NewRecord()
	return rec, tb.cumulative, err
}

type tableBuilder struct {
	mem        memory.Allocator
	cumulative int64
	addresses  map[string]map[uint64]int
	functions  map[string]int

	rb     *builder.RecordBuilder
	schema *arrow.Schema

	builderMappingStart       *array.Uint64Builder
	builderMappingLimit       *array.Uint64Builder
	builderMappingOffset      *array.Uint64Builder
	builderMappingFile        *array.BinaryDictionaryBuilder
	builderMappingBuildID     *array.BinaryDictionaryBuilder
	builderLocationAddress    *array.Uint64Builder
	builderLocationFolded     *builder.OptBooleanBuilder
	builderLocationLine       *builder.OptInt64Builder
	builderFunctionStartLine  *builder.OptInt64Builder
	builderFunctionName       *array.BinaryDictionaryBuilder
	builderFunctionSystemName *array.BinaryDictionaryBuilder
	builderFunctionFileName   *array.BinaryDictionaryBuilder
	builderCumulative         *builder.OptInt64Builder
	builderCumulativeDiff     *builder.OptInt64Builder
	builderFlat               *builder.OptInt64Builder
	builderFlatDiff           *builder.OptInt64Builder
}

func newTableBuilder(mem memory.Allocator) *tableBuilder {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: TableFieldMappingStart, Type: arrow.PrimitiveTypes.Uint64},
		{Name: TableFieldMappingLimit, Type: arrow.PrimitiveTypes.Uint64},
		{Name: TableFieldMappingOffset, Type: arrow.PrimitiveTypes.Uint64},
		{Name: TableFieldMappingFile, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: TableFieldMappingBuildID, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		// Location
		{Name: TableFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
		{Name: TableFieldLocationFolded, Type: &arrow.BooleanType{}},
		{Name: TableFieldLocationLine, Type: arrow.PrimitiveTypes.Int64},
		// Function
		{Name: TableFieldFunctionStartLine, Type: arrow.PrimitiveTypes.Int64},
		{Name: TableFieldFunctionName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		{Name: TableFieldFunctionSystemName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: TableFieldFunctionFileName, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		// Values
		{Name: TableFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: TableFieldCumulativeDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
		{Name: TableFieldFlat, Type: arrow.PrimitiveTypes.Int64},
		{Name: TableFieldFlatDiff, Type: arrow.PrimitiveTypes.Int64},
	}, nil)

	rb := builder.NewRecordBuilder(mem, schema)

	tb := &tableBuilder{
		mem:       mem,
		addresses: map[string]map[uint64]int{},
		functions: map[string]int{},

		rb:                        rb,
		schema:                    schema,
		builderMappingStart:       rb.Field(schema.FieldIndices(TableFieldMappingStart)[0]).(*array.Uint64Builder),
		builderMappingLimit:       rb.Field(schema.FieldIndices(TableFieldMappingLimit)[0]).(*array.Uint64Builder),
		builderMappingOffset:      rb.Field(schema.FieldIndices(TableFieldMappingOffset)[0]).(*array.Uint64Builder),
		builderMappingFile:        rb.Field(schema.FieldIndices(TableFieldMappingFile)[0]).(*array.BinaryDictionaryBuilder),
		builderMappingBuildID:     rb.Field(schema.FieldIndices(TableFieldMappingBuildID)[0]).(*array.BinaryDictionaryBuilder),
		builderLocationAddress:    rb.Field(schema.FieldIndices(TableFieldLocationAddress)[0]).(*array.Uint64Builder),
		builderLocationFolded:     rb.Field(schema.FieldIndices(TableFieldLocationFolded)[0]).(*builder.OptBooleanBuilder),
		builderLocationLine:       rb.Field(schema.FieldIndices(TableFieldLocationLine)[0]).(*builder.OptInt64Builder),
		builderFunctionStartLine:  rb.Field(schema.FieldIndices(TableFieldFunctionStartLine)[0]).(*builder.OptInt64Builder),
		builderFunctionName:       rb.Field(schema.FieldIndices(TableFieldFunctionName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionSystemName: rb.Field(schema.FieldIndices(TableFieldFunctionSystemName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionFileName:   rb.Field(schema.FieldIndices(TableFieldFunctionFileName)[0]).(*array.BinaryDictionaryBuilder),
		builderCumulative:         rb.Field(schema.FieldIndices(TableFieldCumulative)[0]).(*builder.OptInt64Builder),
		builderCumulativeDiff:     rb.Field(schema.FieldIndices(TableFieldCumulativeDiff)[0]).(*builder.OptInt64Builder),
		builderFlat:               rb.Field(schema.FieldIndices(TableFieldFlat)[0]).(*builder.OptInt64Builder),
		builderFlatDiff:           rb.Field(schema.FieldIndices(TableFieldFlatDiff)[0]).(*builder.OptInt64Builder),
	}

	return tb
}

// NewRecord returns a new record from the builders.
// It adds the children to the children column and the labels intersection to the labels column.
// Finally, it assembles all columns from the builders into an arrow record.
func (tb *tableBuilder) NewRecord() (arrow.Record, error) {
	// TODO: Is this how we want to handle empty data?
	if tb.builderCumulative.Len() == 0 {
		return tb.rb.NewRecord(), nil
	}

	// We have manually tracked the total cumulative value.
	// Now we set/overwrite the cumulative value for the root row (which is always the 0 row in our flame graphs).
	tb.builderCumulative.Set(0, tb.cumulative)

	return tb.rb.NewRecord(), nil
}

func (tb *tableBuilder) Release() {
	tb.rb.Release()
}

func (tb *tableBuilder) appendRow(
	r profile.RecordReader,
	sampleRow, locationRow, lineRow int,
	leaf bool,
) error {
	for j := range tb.rb.Fields() {
		switch tb.schema.Field(j).Name {
		// Mapping
		case TableFieldMappingStart:
			if r.Mapping.IsValid(locationRow) && r.MappingStart.Value(locationRow) > 0 {
				tb.builderMappingStart.Append(r.MappingStart.Value(locationRow))
			} else {
				tb.builderMappingStart.AppendNull()
			}
		case TableFieldMappingLimit:
			if r.Mapping.IsValid(locationRow) && r.MappingLimit.Value(locationRow) > 0 {
				tb.builderMappingLimit.Append(r.MappingLimit.Value(locationRow))
			} else {
				tb.builderMappingLimit.AppendNull()
			}
		case TableFieldMappingOffset:
			if r.Mapping.IsValid(locationRow) && r.MappingOffset.Value(locationRow) > 0 {
				tb.builderMappingOffset.Append(r.MappingOffset.Value(locationRow))
			} else {
				tb.builderMappingOffset.AppendNull()
			}
		case TableFieldMappingFile:
			if r.MappingFileDict.Len() == 0 {
				tb.builderMappingFile.AppendNull()
			} else {
				if r.Mapping.IsValid(locationRow) && len(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(locationRow))) > 0 {
					_ = tb.builderMappingFile.Append(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(locationRow)))
				} else {
					tb.builderMappingFile.AppendNull()
				}
			}
		case TableFieldMappingBuildID:
			if r.MappingBuildIDDict.Len() == 0 {
				tb.builderMappingBuildID.AppendNull()
			} else {
				if r.Mapping.IsValid(locationRow) && len(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(locationRow))) > 0 {
					_ = tb.builderMappingBuildID.Append(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(locationRow)))
				} else {
					tb.builderMappingBuildID.AppendNull()
				}
			}
		// Location
		case TableFieldLocationAddress:
			tb.builderLocationAddress.Append(r.Address.Value(locationRow))

		// TODO: Location isFolded we should remove this until we actually support folded functions.
		case TableFieldLocationFolded:
			tb.builderLocationFolded.AppendSingle(false)
		case TableFieldLocationLine:
			if lineRow >= 0 && r.Line.IsValid(lineRow) {
				tb.builderLocationLine.Append(r.LineNumber.Value(lineRow))
			} else {
				tb.builderLocationLine.AppendNull()
			}
		// Function
		case TableFieldFunctionStartLine:
			if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && r.LineFunctionStartLine.Value(lineRow) > 0 {
				tb.builderFunctionStartLine.Append(r.LineFunctionStartLine.Value(lineRow))
			} else {
				tb.builderFunctionStartLine.AppendNull()
			}
		case TableFieldFunctionName:
			if r.LineFunctionNameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow))) > 0 {
					_ = tb.builderFunctionName.Append(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow)))
				} else {
					tb.builderFunctionName.AppendNull()
				}
			}
		case TableFieldFunctionSystemName:
			if r.LineFunctionSystemNameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionSystemName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionSystemNameDict.Value(r.LineFunctionSystemName.GetValueIndex(lineRow))) > 0 {
					_ = tb.builderFunctionSystemName.Append(r.LineFunctionSystemNameDict.Value(r.LineFunctionSystemName.GetValueIndex(lineRow)))
				} else {
					tb.builderFunctionSystemName.AppendNull()
				}
			}
		case TableFieldFunctionFileName:
			if r.LineFunctionFilenameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionFileName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(lineRow))) > 0 {
					_ = tb.builderFunctionFileName.Append(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(lineRow)))
				} else {
					tb.builderFunctionFileName.AppendNull()
				}
			}
		// Values
		case TableFieldCumulative:
			tb.builderCumulative.Append(r.Value.Value(sampleRow))
		case TableFieldCumulativeDiff:
			if r.Diff.Value(sampleRow) > 0 {
				tb.builderCumulativeDiff.Append(r.Diff.Value(sampleRow))
			} else {
				tb.builderCumulativeDiff.AppendNull()
			}
		case TableFieldFlat:
			if leaf {
				tb.builderFlat.Append(r.Value.Value(sampleRow))
			} else {
				// don't set null as it might also just be merged into a bigger number.
				tb.builderFlat.Append(0)
			}
		case TableFieldFlatDiff:
			if leaf {
				tb.builderFlatDiff.Append(r.Diff.Value(sampleRow))
			} else {
				// don't set null as it might also just be merged into a bigger number.
				tb.builderFlatDiff.Append(0)
			}
		default:
			panic(fmt.Sprintf("unknown field %s", tb.schema.Field(j).Name))
		}
	}
	return nil
}

func (tb *tableBuilder) mergeRow(r profile.RecordReader, mergeRow, sampleRow int) {
	tb.builderCumulative.Add(mergeRow, r.Value.Value(sampleRow))
	if r.Diff.Value(sampleRow) != 0 {
		tb.builderCumulativeDiff.Add(mergeRow, r.Diff.Value(sampleRow))
	}
}
