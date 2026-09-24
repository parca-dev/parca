// Copyright 2023-2026 The Parca Authors
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

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"

	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	TableFieldMappingFile    = "mapping_file"
	TableFieldMappingBuildID = "mapping_build_id"

	TableFieldLocationAddress = "location_address"
	TableFieldLocationLine    = "location_line"

	TableFieldFunctionStartLine  = "function_startline"
	TableFieldFunctionName       = "function_name"
	TableFieldFunctionSystemName = "function_system_name"
	TableFieldFunctionFileName   = "function_file_name"

	TableFieldCumulative     = "cumulative"
	TableFieldCumulativeDiff = "cumulative_diff"
	TableFieldFlat           = "flat"
	TableFieldFlatDiff       = "flat_diff"

	TableFieldCallers = "callers"
	TableFieldCallees = "callees"
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
		ipc.WithLZ4(),
	)
	defer w.Close()

	if err = w.Write(record); err != nil {
		return nil, 0, err
	}

	span.SetAttributes(attribute.Int("record_size", buf.Len()))
	if buf.Len() > 1<<22 { // 4MiB
		span.SetAttributes(attribute.String("record_stats", recordStats(record)))
	}

	return &queryv1alpha1.TableArrow{
		Record: buf.Bytes(),
		Unit:   p.Meta.SampleType.Unit,
	}, cumulative, nil
}

// isFirstNonNil returns true if the row is the first non-nil value in the list found at the given row.
func isFirstNonNil(row, listRow int, list *array.List) bool {
	start, end := list.ValueOffsets(row)
	for i := int(start); i < int(end); i++ {
		if !list.ListValues().IsNull(i) {
			return i == listRow
		}
	}
	return false
}

func estimateTableRows(r profile.Reader) int {
	if len(r.RecordReaders) == 0 {
		return 0
	}

	// The number of unique function names is a good baseline for the number of rows, so going with that.
	return r.RecordReaders[0].LineFunctionNameDict.Len()
}

func generateTableArrowRecord(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
) (arrow.RecordBatch, int64, error) {
	_, span := tracer.Start(ctx, "generateTableArrowRecord")
	defer span.End()

	profileReader, err := profile.NewReader(p)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to create profile reader: %w", err)
	}

	tb := newTableBuilder(mem, estimateTableRows(profileReader))
	defer tb.Release()

	tableRow := 0

	for _, r := range profileReader.RecordReaders {
		tb.cumulative += math.Int64.Sum(r.Value)

		for sampleRow := 0; sampleRow < int(r.Record.NumRows()); sampleRow++ {
			previousTableRow := -1
			// Track which table rows have been counted for this sample to avoid
			// double-counting cumulative values for recursive functions.
			seenInSample := make(map[int]struct{})

			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(sampleRow)
			for locationRow := int(lOffsetStart); locationRow < int(lOffsetEnd); locationRow++ {
				if r.Locations.ListValues().IsNull(locationRow) {
					continue // Skip null locations; they have been filtered out.
				}
				if r.Lines.IsNull(locationRow) {
					// The location has no lines, we therefore compare its address.

					isLeaf := isFirstNonNil(sampleRow, locationRow, r.Locations)
					var buildID []byte
					if r.MappingBuildIDIndices.IsValid(locationRow) {
						buildID = r.MappingBuildIDDict.Value(int(r.MappingBuildIDIndices.Value(locationRow)))
					}
					addr := r.Address.Value(locationRow)

					// Check if we've seen the address for the mapping before.
					// If not, we add it as a new row and add the address to the mapping to keep track of it.
					// If we have seen the address before, we merge the address with the existing row by summing the values.
					// Note for Go developers: This won't panic. Tests have shown that if the first check fails, the second check won't be run.
					if cr, ok := tb.addresses[unsafeString(buildID)][addr]; !ok {
						if err := tb.appendRow(r, sampleRow, locationRow, -1, tableRow, previousTableRow, isLeaf); err != nil {
							return nil, 0, err
						}

						if _, ok := tb.addresses[unsafeString(buildID)]; !ok {
							tb.addresses[string(buildID)] = map[uint64]int{addr: tableRow}
						} else {
							tb.addresses[string(buildID)][addr] = tableRow
						}
						seenInSample[tableRow] = struct{}{}
						previousTableRow = tableRow
						tableRow++
					} else {
						// Only add to cumulative if this is the first occurrence in this sample
						_, alreadySeen := seenInSample[cr]
						addCumulative := !alreadySeen
						if addCumulative {
							seenInSample[cr] = struct{}{}
						}
						tb.mergeRow(r, cr, sampleRow, locationRow, -1, tb.addresses[unsafeString(buildID)][addr], previousTableRow, isLeaf, addCumulative)
						previousTableRow = tb.addresses[unsafeString(buildID)][addr]
					}
				} else {
					// The location has lines, we therefore compare its function names.

					llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(locationRow)
					for lineRow := int(llOffsetStart); lineRow < int(llOffsetEnd); lineRow++ {
						isLeaf := isFirstNonNil(sampleRow, locationRow, r.Locations) && isFirstNonNil(locationRow, lineRow, r.Lines)

						if r.Line.IsValid(lineRow) && r.LineFunctionNameIndices.IsValid(lineRow) {
							fn := r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(lineRow)))
							if cr, ok := tb.functions[unsafeString(fn)]; !ok {
								if err := tb.appendRow(r, sampleRow, locationRow, lineRow, tableRow, previousTableRow, isLeaf); err != nil {
									return nil, 0, err
								}
								tb.functions[string(fn)] = tableRow
								seenInSample[tableRow] = struct{}{}
								previousTableRow = tableRow
								tableRow++
							} else {
								// Only add to cumulative if this is the first occurrence in this sample
								_, alreadySeen := seenInSample[cr]
								addCumulative := !alreadySeen
								if addCumulative {
									seenInSample[cr] = struct{}{}
								}
								tb.mergeRow(r, cr, sampleRow, locationRow, lineRow, tb.functions[unsafeString(fn)], previousTableRow, isLeaf, addCumulative)
								previousTableRow = tb.functions[unsafeString(fn)]
							}
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
	callers    []map[int64]struct{}
	callees    []map[int64]struct{}

	rb     *builder.RecordBuilder
	schema *arrow.Schema

	builderMappingFile        *array.BinaryDictionaryBuilder
	builderMappingBuildID     *array.BinaryDictionaryBuilder
	builderLocationAddress    *array.Uint64Builder
	builderLocationLine       *builder.OptInt64Builder
	builderFunctionStartLine  *builder.OptInt64Builder
	builderFunctionName       *array.BinaryDictionaryBuilder
	builderFunctionSystemName *array.BinaryDictionaryBuilder
	builderFunctionFileName   *array.BinaryDictionaryBuilder
	builderCumulative         *builder.OptInt64Builder
	builderCumulativeDiff     *builder.OptInt64Builder
	builderFlat               *builder.OptInt64Builder
	builderFlatDiff           *builder.OptInt64Builder
	builderCallers            *builder.ListBuilder
	builderCallees            *builder.ListBuilder
}

func newTableBuilder(mem memory.Allocator, rowCountEstimate int) *tableBuilder {
	schema := arrow.NewSchema([]arrow.Field{
		{Name: TableFieldMappingFile, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		{Name: TableFieldMappingBuildID, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.String}},
		// Location
		{Name: TableFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
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

		// Call View
		{Name: TableFieldCallers, Type: arrow.ListOf(arrow.PrimitiveTypes.Int64)},
		{Name: TableFieldCallees, Type: arrow.ListOf(arrow.PrimitiveTypes.Int64)},
	}, nil)

	rb := builder.NewRecordBuilder(mem, schema)

	tb := &tableBuilder{
		mem:       mem,
		addresses: map[string]map[uint64]int{},
		functions: map[string]int{},
		callers:   make([]map[int64]struct{}, rowCountEstimate),
		callees:   make([]map[int64]struct{}, rowCountEstimate),

		rb:                        rb,
		schema:                    schema,
		builderMappingFile:        rb.Field(schema.FieldIndices(TableFieldMappingFile)[0]).(*array.BinaryDictionaryBuilder),
		builderMappingBuildID:     rb.Field(schema.FieldIndices(TableFieldMappingBuildID)[0]).(*array.BinaryDictionaryBuilder),
		builderLocationAddress:    rb.Field(schema.FieldIndices(TableFieldLocationAddress)[0]).(*array.Uint64Builder),
		builderLocationLine:       rb.Field(schema.FieldIndices(TableFieldLocationLine)[0]).(*builder.OptInt64Builder),
		builderFunctionStartLine:  rb.Field(schema.FieldIndices(TableFieldFunctionStartLine)[0]).(*builder.OptInt64Builder),
		builderFunctionName:       rb.Field(schema.FieldIndices(TableFieldFunctionName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionSystemName: rb.Field(schema.FieldIndices(TableFieldFunctionSystemName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionFileName:   rb.Field(schema.FieldIndices(TableFieldFunctionFileName)[0]).(*array.BinaryDictionaryBuilder),
		builderCumulative:         rb.Field(schema.FieldIndices(TableFieldCumulative)[0]).(*builder.OptInt64Builder),
		builderCumulativeDiff:     rb.Field(schema.FieldIndices(TableFieldCumulativeDiff)[0]).(*builder.OptInt64Builder),
		builderFlat:               rb.Field(schema.FieldIndices(TableFieldFlat)[0]).(*builder.OptInt64Builder),
		builderFlatDiff:           rb.Field(schema.FieldIndices(TableFieldFlatDiff)[0]).(*builder.OptInt64Builder),
		builderCallers:            rb.Field(schema.FieldIndices(TableFieldCallers)[0]).(*builder.ListBuilder),
		builderCallees:            rb.Field(schema.FieldIndices(TableFieldCallees)[0]).(*builder.ListBuilder),
	}

	return tb
}

func (tb *tableBuilder) populateCallerAndCalleeData() {
	for i := range tb.builderFunctionName.Len() {
		// We need to check if the caller list exists for this index as the length of the callers maybe less than the length of the
		// function names table due to the fact that some functions may not have any callers.
		if len(tb.callers) > i {
			callers := maps.Keys(tb.callers[i])
			if len(callers) == 0 {
				tb.builderCallers.AppendNull()
			} else {
				tb.builderCallers.Append(true)
				tb.builderCallers.ValueBuilder().(*builder.OptInt64Builder).AppendData(callers)
			}
		} else {
			tb.builderCallers.AppendNull()
		}

		// Same as above, we need to check if the callee list exists for this index.
		if len(tb.callees) > i {
			callees := maps.Keys(tb.callees[i])
			if len(callees) == 0 {
				tb.builderCallees.AppendNull()
			} else {
				tb.builderCallees.Append(true)
				tb.builderCallees.ValueBuilder().(*builder.OptInt64Builder).AppendData(callees)
			}
		} else {
			tb.builderCallees.AppendNull()
		}
	}
}

// NewRecord returns a new record from the builders.
func (tb *tableBuilder) NewRecord() (arrow.RecordBatch, error) {
	tb.populateCallerAndCalleeData()
	return tb.rb.NewRecord(), nil
}

func (tb *tableBuilder) Release() {
	tb.rb.Release()
}

func (tb *tableBuilder) appendRow(
	r *profile.RecordReader,
	sampleRow, locationRow, lineRow, currentTableRow, previousTableRow int,
	leaf bool,
) error {
	for j := range tb.rb.Fields() {
		switch tb.schema.Field(j).Name {
		// Mapping
		case TableFieldMappingFile:
			if r.MappingFileDict.Len() == 0 {
				tb.builderMappingFile.AppendNull()
			} else {
				if r.MappingFileIndices.IsValid(locationRow) {
					_ = tb.builderMappingFile.Append(r.MappingFileDict.Value(int(r.MappingFileIndices.Value(locationRow))))
				} else {
					tb.builderMappingFile.AppendNull()
				}
			}
		case TableFieldMappingBuildID:
			if r.MappingBuildIDDict.Len() == 0 {
				tb.builderMappingBuildID.AppendNull()
			} else {
				if r.MappingBuildIDIndices.IsValid(locationRow) {
					_ = tb.builderMappingBuildID.Append(r.MappingBuildIDDict.Value(int(r.MappingBuildIDIndices.Value(locationRow))))
				} else {
					tb.builderMappingBuildID.AppendNull()
				}
			}
		// Location
		case TableFieldLocationAddress:
			tb.builderLocationAddress.Append(r.Address.Value(locationRow))

		// TODO: Location isFolded we should remove this until we actually support folded functions.
		case TableFieldLocationLine:
			if lineRow >= 0 && r.Line.IsValid(lineRow) {
				tb.builderLocationLine.Append(r.LineNumber.Value(lineRow))
			} else {
				tb.builderLocationLine.AppendNull()
			}
		// Function
		case TableFieldFunctionStartLine:
			if lineRow >= 0 && r.LineFunctionStartLine.Value(lineRow) > 0 {
				tb.builderFunctionStartLine.Append(r.LineFunctionStartLine.Value(lineRow))
			} else {
				tb.builderFunctionStartLine.AppendNull()
			}
		case TableFieldFunctionName:
			if r.LineFunctionNameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunctionNameIndices.IsValid(lineRow) {
					_ = tb.builderFunctionName.Append(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(lineRow))))
				} else {
					tb.builderFunctionName.AppendNull()
				}
			}
		case TableFieldFunctionSystemName:
			if r.LineFunctionSystemNameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionSystemName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunctionSystemNameIndices.IsValid(lineRow) {
					_ = tb.builderFunctionSystemName.Append(r.LineFunctionSystemNameDict.Value(int(r.LineFunctionSystemNameIndices.Value(lineRow))))
				} else {
					tb.builderFunctionSystemName.AppendNull()
				}
			}
		case TableFieldFunctionFileName:
			if r.LineFunctionFilenameDict.Len() == 0 || lineRow == -1 {
				tb.builderFunctionFileName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunctionFilenameIndices.IsValid(lineRow) {
					_ = tb.builderFunctionFileName.Append(r.LineFunctionFilenameDict.Value(int(r.LineFunctionFilenameIndices.Value(lineRow))))
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
		case TableFieldCallers:
			tb.addCaller(previousTableRow, int64(currentTableRow))
		case TableFieldCallees:
			tb.addCallee(currentTableRow, int64(previousTableRow))
		default:
			panic(fmt.Sprintf("unknown field %s", tb.schema.Field(j).Name))
		}
	}
	return nil
}

// mergeRow merges sample data into an existing table row.
// If addCumulative is false, only caller/callee relationships and flat values (if leaf) are updated.
// This is used to avoid double-counting cumulative values for recursive functions.
func (tb *tableBuilder) mergeRow(r *profile.RecordReader, mergeRow, sampleRow, _, lineRow, currentTableRow, previousTableRow int, isLeaf, addCumulative bool) {
	if addCumulative {
		tb.builderCumulative.Add(mergeRow, r.Value.Value(sampleRow))
		if r.Diff.Value(sampleRow) != 0 {
			tb.builderCumulativeDiff.Add(mergeRow, r.Diff.Value(sampleRow))
		}
	}

	if isLeaf {
		tb.builderFlat.Add(mergeRow, r.Value.Value(sampleRow))
		if r.Diff.Value(sampleRow) != 0 {
			tb.builderFlatDiff.Add(mergeRow, r.Diff.Value(sampleRow))
		}
	}

	tb.addCaller(previousTableRow, int64(currentTableRow))
	tb.addCallee(currentTableRow, int64(previousTableRow))
}

func (tb *tableBuilder) addCaller(idx int, caller int64) {
	if caller == -1 || idx == -1 {
		return
	}
	for len(tb.callers) <= idx+1 {
		tb.callers = append(tb.callers, map[int64]struct{}{})
	}
	if tb.callers[idx] == nil {
		tb.callers[idx] = map[int64]struct{}{}
	}
	tb.callers[idx][caller] = struct{}{}
}

func (tb *tableBuilder) addCallee(idx int, callee int64) {
	if callee == -1 || idx == -1 {
		return
	}
	for len(tb.callees) <= idx+1 {
		tb.callees = append(tb.callees, map[int64]struct{}{})
	}
	if tb.callees[idx] == nil {
		tb.callees[idx] = map[int64]struct{}{}
	}
	tb.callees[idx][callee] = struct{}{}
}
