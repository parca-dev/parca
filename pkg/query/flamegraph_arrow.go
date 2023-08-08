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
	"strings"
	"unsafe"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/ipc"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"

	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
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

	FlamegraphFieldLabels     = "labels"
	FlamegraphFieldChildren   = "children"
	FlamegraphFieldCumulative = "cumulative"
	FlamegraphFieldDiff       = "diff"
)

func GenerateFlamegraphArrow(ctx context.Context, mem memory.Allocator, tracer trace.Tracer, p profile.Profile, aggregate []string, trimFraction float32) (*queryv1alpha1.FlamegraphArrow, int64, error) {
	record, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, aggregate, trimFraction)
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

	return &queryv1alpha1.FlamegraphArrow{
		Record:  buf.Bytes(),
		Unit:    p.Meta.SampleType.Unit,
		Height:  height, // add one for the root
		Trimmed: trimmed,
	}, cumulative, nil
}

func generateFlamegraphArrowRecord(ctx context.Context, mem memory.Allocator, tracer trace.Tracer, p profile.Profile, aggregate []string, trimFraction float32) (arrow.Record, int64, int32, int64, error) {
	aggregateFields := make(map[string]struct{}, len(aggregate))
	for _, f := range aggregate {
		aggregateFields[f] = struct{}{}
	}
	// this is a helper as it's frequently accessed below
	aggregateLabels := false
	if _, found := aggregateFields[FlamegraphFieldLabels]; found {
		aggregateLabels = true
	}

	totalRows := int64(0)
	for _, r := range p.Samples {
		totalRows += r.NumRows()
	}

	fb := newFlamegraphBuilder(mem, totalRows)
	defer fb.Release()

	// This keeps track of the max depth of our flame graph.
	maxHeight := int32(0)

	// these change with every iteration below
	row := fb.builderCumulative.Len()
	// compareRows are the rows that we compare to the current location against and potentially merge.
	compareRows := []int{}

	profileReader := profile.NewReader(p)
	for _, r := range profileReader.RecordReaders {
		// This field compares the current sample with the already added values in the builders.
		equalField := func(
			fieldName string,
			pprofLabels map[string]string,
			sampleRow,
			locationRow,
			lineRow,
			flamegraphRow int,
			height int,
		) bool {
			switch fieldName {
			case FlamegraphFieldMappingFile:
				if !r.Mapping.IsValid(locationRow) {
					return true
				}
				rowMappingFile := fb.builderMappingFile.Value(fb.builderMappingFile.GetValueIndex(flamegraphRow))
				// rather than comparing the strings, we compare bytes to avoid allocations.
				return bytes.Equal(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(locationRow)), rowMappingFile)
			case FlamegraphFieldLocationAddress:
				// TODO: do we need to check for null?
				rowLocationAddress := fb.builderLocationAddress.Value(flamegraphRow)
				return r.Address.Value(locationRow) == rowLocationAddress
			case FlamegraphFieldFunctionName:
				isNull := fb.builderFunctionName.IsNull(flamegraphRow)
				if !isNull {
					rowFunctionName := fb.builderFunctionName.Value(fb.builderFunctionName.GetValueIndex(flamegraphRow))
					// rather than comparing the strings, we compare bytes to avoid allocations.
					return bytes.Equal(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow)), rowFunctionName)
				}
				// isNull
				if !r.LineFunction.IsValid(lineRow) || len(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow))) == 0 {
					return true
				}
				return false
			case FlamegraphFieldLabels:
				// We only compare the labels of roots of stacktraces.
				if height > 0 {
					return true
				}
				if len(pprofLabels) == 0 && fb.labels[flamegraphRow] == nil {
					return true
				}
				if len(pprofLabels) > 0 && fb.labels[flamegraphRow] == nil {
					return false
				}
				if len(pprofLabels) == 0 && fb.labels[flamegraphRow] != nil {
					return false
				}
				if len(pprofLabels) != len(fb.labels[flamegraphRow][0]) {
					return false
				}
				return maps.Equal(pprofLabels, fb.labels[flamegraphRow][0])
			default:
				return false
			}
		}

		lsbytes := make([]byte, 0, 512)
		for i := 0; i < int(r.Record.NumRows()); i++ {
			beg, end := r.Locations.ValueOffsets(i)

			// TODO: This height is only an estimation, inlined functions are not taken into account.
			numLocations := int32(end - beg)
			if numLocations > maxHeight {
				maxHeight = numLocations
			}

			var sampleLabels map[string]string
			for j, labelColumn := range r.LabelColumns {
				if labelColumn.Col.IsValid(i) {
					if sampleLabels == nil {
						sampleLabels = map[string]string{}
					}

					labelName := strings.TrimPrefix(r.LabelFields[j].Name, profile.ColumnPprofLabelsPrefix)
					sampleLabels[labelName] = string(labelColumn.Dict.Value(labelColumn.Col.GetValueIndex(i)))
				}
			}

			if aggregateLabels && len(sampleLabels) > 0 {
				lsbytes = lsbytes[:0]
				lsbytes = MarshalStringMap(lsbytes, sampleLabels)

				sampleLabelRow := row
				if _, ok := fb.rootsRow[unsafeString(lsbytes)]; ok {
					sampleLabelRow = fb.rootsRow[unsafeString(lsbytes)][0]
					compareRows = compareRows[:0] //  reset the compare rows
					// We want to compare against this found label root's children.
					rootRow := fb.rootsRow[unsafeString(lsbytes)][0]
					compareRows = append(compareRows, fb.children[rootRow]...)
					fb.addRowValues(r, sampleLabelRow, i) // adds the cumulative and diff values to the existing row
				} else {
					err := fb.AppendLabelRow(r, sampleLabelRow, unsafeString(lsbytes), sampleLabels, i)
					if err != nil {
						return nil, 0, 0, 0, fmt.Errorf("failed to inject label row: %w", err)
					}
					fb.rootsRow[unsafeString(lsbytes)] = []int{sampleLabelRow}
				}
				fb.parent.Set(sampleLabelRow)
				row = fb.builderCumulative.Len()
			}

			// every new sample resets the childRow to -1 indicating that we start with a leaf again.
			// pprof stores locations in reverse order, thus we loop over locations in reverse order.
		locations:
			for j := int(end - 1); j >= int(beg); j-- {
				// If the location has no lines, it's not symbolized.
				// We work with the location address instead.

				// This returns whether this location is a root of a stacktrace.
				isLocationRoot := isLocationRoot(int(end), j)
				// Depending on whether we aggregate the labels (and thus inject node labels), we either compare the rows or not.
				isRoot := isLocationRoot && !(aggregateLabels && len(sampleLabels) > 0)

				if isLocationLeaf(int(beg), j) {
					fb.cumulative += r.Value.Value(i)
				}

				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)
				if !r.Lines.IsValid(j) || llOffsetEnd-llOffsetStart <= 0 {
					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						compareRows = compareRows[:0] //  reset the compare rows
						compareRows = append(compareRows, fb.rootsRow[unsafeString(lsbytes)]...)
						// append this row afterward to not compare to itself
						fb.parent.Reset()
					}

					// We compare the location address to the existing rows.
					// If we find a matching address, we merge the values.
				compareRowsAddr:
					for _, cr := range compareRows {
						if !equalField(FlamegraphFieldLocationAddress, sampleLabels, i, j, 0, cr, int(end)-1-j) {
							continue compareRowsAddr
						}

						// If we don't group by the labels, we add all labels to the row and later on intersect the values before adding them to the flame graph.
						if _, groupBy := aggregateFields[FlamegraphFieldLabels]; !groupBy {
							fb.labels[cr] = append(fb.labels[cr], sampleLabels)
						}

						fb.builderCumulative.Add(cr, r.Value.Value(i))
						fb.parent.Set(cr)
						compareRows = copyChildren(fb.children[cr])
						continue locations
					}
					// reset the compare rows
					// if there are no matching rows here, we don't want to merge their children either.
					compareRows = compareRows[:0]

					if isRoot {
						// We aren't merging this root, so we'll keep track of it as a new one.
						fb.rootsRow[unsafeString(lsbytes)] = append(fb.rootsRow[unsafeString(lsbytes)], row)
					}

					err := fb.appendRow(r, sampleLabels, i, j, -1, row)
					if err != nil {
						return nil, 0, 0, 0, err
					}

					fb.parent.Set(row)
					row = fb.builderCumulative.Len()
				}

				llOffsetStart, llOffsetEnd = r.Lines.ValueOffsets(j)
			stacktraces:
				// just like locations, pprof stores lines in reverse order.
				for k := int(llOffsetEnd - 1); k >= int(llOffsetStart); k-- {
					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						compareRows = compareRows[:0] //  reset the compare rows
						compareRows = append(compareRows, fb.rootsRow[unsafeString(lsbytes)]...)
						// append this row afterward to not compare to itself
						fb.parent.Reset()
					}

					// If there are no fields we should aggregate we can skip the comparison
					if len(aggregateFields) > 0 {
					compareRows:
						for _, cr := range compareRows {
							for f := range aggregateFields {
								if !equalField(f, sampleLabels, i, j, k, cr, int(end)-1-j) {
									// If a field doesn't match, we can't aggregate this row with the existing one.
									continue compareRows
								}
							}

							// If we don't group by the labels, we add all labels to the row and later on intersect the values before adding them to the flame graph.
							if _, groupBy := aggregateFields[FlamegraphFieldLabels]; !groupBy {
								fb.labels[cr] = append(fb.labels[cr], sampleLabels)
							}

							// All fields match, so we can aggregate this new row with the existing one.
							fb.addRowValues(r, cr, i)
							// Continue with this row as the parent for the next iteration and compare to its children.
							fb.parent.Set(cr)
							compareRows = copyChildren(fb.children[cr])
							continue stacktraces
						}
						// reset the compare rows
						// if there are no matching rows here, we don't want to merge their children either.
						compareRows = compareRows[:0]
					}

					if isRoot {
						// We aren't merging this root, so we'll keep track of it as a new one.
						fb.rootsRow[unsafeString(lsbytes)] = append(fb.rootsRow[unsafeString(lsbytes)], row)
					}

					err := fb.appendRow(r, sampleLabels, i, j, k, row)
					if err != nil {
						return nil, 0, 0, 0, err
					}

					fb.parent.Set(row)
					row = fb.builderCumulative.Len()
				}
			}
		}
	}

	record, err := fb.NewRecord()
	if err != nil {
		return nil, 0, 0, 0, err
	}

	return record, fb.cumulative, maxHeight + 1, 0, nil
}

func copyChildren(children []int) []int {
	newChildren := make([]int, len(children))
	copy(newChildren, children)
	return newChildren
}

type flamegraphBuilder struct {
	rb     *builder.RecordBuilder
	schema *arrow.Schema
	// This keeps track of the total cumulative value so that we can set the first row's cumulative value at the end.
	cumulative int64
	parent     parent
	children   [][]int
	labels     map[int][]map[string]string

	// This keeps track of the root rows indexed by the labels string.
	// If the stack trace has no labels, we use the empty string as the key.
	// This will be the root row's children, which is always our row 0 in flame graphs.
	rootsRow map[string][]int

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
	builderLabels             *array.BinaryDictionaryBuilder
	builderChildren           *builder.ListBuilder
	builderChildrenValues     *array.Uint32Builder
	builderCumulative         *builder.OptInt64Builder
	builderDiff               *builder.OptInt64Builder
}

func newFlamegraphBuilder(mem memory.Allocator, rows int64) *flamegraphBuilder {
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
		{Name: FlamegraphFieldLabels, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		{Name: FlamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: FlamegraphFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	}, nil)

	rb := builder.NewRecordBuilder(mem, schema)

	builderChildren := rb.Field(schema.FieldIndices(FlamegraphFieldChildren)[0]).(*builder.ListBuilder)
	fb := &flamegraphBuilder{
		rb:     rb,
		schema: schema,
		// parent keeps track of the parent of a row. This is used to build the children array.
		parent: parent(-1),
		// This keeps track of a row's children and will be converted to an arrow array of lists at the end.
		// Allocating for an average of 8 children per stacktrace upfront.
		children: make([][]int, rows),
		labels:   make(map[int][]map[string]string),
		rootsRow: make(map[string][]int),

		builderMappingStart:   rb.Field(schema.FieldIndices(FlamegraphFieldMappingStart)[0]).(*array.Uint64Builder),
		builderMappingLimit:   rb.Field(schema.FieldIndices(FlamegraphFieldMappingLimit)[0]).(*array.Uint64Builder),
		builderMappingOffset:  rb.Field(schema.FieldIndices(FlamegraphFieldMappingOffset)[0]).(*array.Uint64Builder),
		builderMappingFile:    rb.Field(schema.FieldIndices(FlamegraphFieldMappingFile)[0]).(*array.BinaryDictionaryBuilder),
		builderMappingBuildID: rb.Field(schema.FieldIndices(FlamegraphFieldMappingBuildID)[0]).(*array.BinaryDictionaryBuilder),

		builderLocationAddress: rb.Field(schema.FieldIndices(FlamegraphFieldLocationAddress)[0]).(*array.Uint64Builder),
		builderLocationFolded:  rb.Field(schema.FieldIndices(FlamegraphFieldLocationFolded)[0]).(*builder.OptBooleanBuilder),
		builderLocationLine:    rb.Field(schema.FieldIndices(FlamegraphFieldLocationLine)[0]).(*builder.OptInt64Builder),

		builderFunctionStartLine:  rb.Field(schema.FieldIndices(FlamegraphFieldFunctionStartLine)[0]).(*builder.OptInt64Builder),
		builderFunctionName:       rb.Field(schema.FieldIndices(FlamegraphFieldFunctionName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionSystemName: rb.Field(schema.FieldIndices(FlamegraphFieldFunctionSystemName)[0]).(*array.BinaryDictionaryBuilder),
		builderFunctionFileName:   rb.Field(schema.FieldIndices(FlamegraphFieldFunctionFileName)[0]).(*array.BinaryDictionaryBuilder),

		builderLabels:         rb.Field(schema.FieldIndices(FlamegraphFieldLabels)[0]).(*array.BinaryDictionaryBuilder),
		builderChildren:       builderChildren,
		builderChildrenValues: builderChildren.ValueBuilder().(*array.Uint32Builder),
		builderCumulative:     rb.Field(schema.FieldIndices(FlamegraphFieldCumulative)[0]).(*builder.OptInt64Builder),
		builderDiff:           rb.Field(schema.FieldIndices(FlamegraphFieldDiff)[0]).(*builder.OptInt64Builder),
	}

	// The very first row is the root row. It doesn't contain any metadata.
	// It only contains the root cumulative value and list of children (which are actual roots).
	fb.builderMappingStart.AppendNull()
	fb.builderMappingLimit.AppendNull()
	fb.builderMappingOffset.AppendNull()
	fb.builderMappingFile.AppendNull()
	fb.builderMappingBuildID.AppendNull()

	fb.builderLocationAddress.AppendNull()
	fb.builderLocationFolded.AppendNull()
	fb.builderLocationLine.AppendNull()

	fb.builderFunctionStartLine.AppendNull()
	fb.builderFunctionName.AppendNull()
	fb.builderFunctionSystemName.AppendNull()
	fb.builderFunctionFileName.AppendNull()

	// The cumulative values is calculated and at the end set to the correct value.
	fb.builderCumulative.Append(0)
	fb.builderDiff.AppendNull()

	return fb
}

// NewRecord returns a new record from the builders.
// It adds the children to the children column and the labels intersection to the labels column.
// Finally, it assembles all columns from the builders into an arrow record.
func (fb *flamegraphBuilder) NewRecord() (arrow.Record, error) {
	// We have manually tracked the total cumulative value.
	// Now we set/overwrite the cumulative value for the root row (which is always the 0 row in our flame graphs).
	fb.builderCumulative.Set(0, fb.cumulative)

	// We have manually tracked each row's children.
	// So now we need to iterate over all rows in the record and append their children.
	// We cannot do this while building the rows as we need to append the children while iterating over the rows.
	for i := 0; i < fb.builderCumulative.Len(); i++ {
		if i == 0 {
			fb.builderChildren.Append(true)
			for _, sampleLabelChildren := range fb.rootsRow {
				for _, child := range sampleLabelChildren {
					fb.builderChildrenValues.Append(uint32(child))
				}
			}
			continue
		}
		if len(fb.children[i]) == 0 {
			fb.builderChildren.AppendNull() // leaf
		} else {
			fb.builderChildren.Append(true)
			for _, child := range fb.children[i] {
				fb.builderChildrenValues.Append(uint32(child))
			}
		}
	}
	lsbytes := make([]byte, 0, 512)
	for i := 0; i < fb.builderCumulative.Len(); i++ {
		if lsets, hasLabels := fb.labels[i]; hasLabels {
			inter := mapsIntersection(lsets)
			if len(inter) == 0 {
				fb.builderLabels.AppendNull()
				continue
			}

			lsbytes = lsbytes[:0]
			lsbytes = MarshalStringMap(lsbytes, inter)
			if err := fb.builderLabels.Append(lsbytes); err != nil {
				return nil, err
			}
		} else {
			fb.builderLabels.AppendNull()
		}
	}

	return fb.rb.NewRecord(), nil
}

func (fb *flamegraphBuilder) Release() {
	fb.rb.Release()
}

func (fb *flamegraphBuilder) appendRow(
	r profile.RecordReader,
	labels map[string]string,
	sampleRow, locationRow, lineRow int,
	row int,
) error {
	for j := range fb.rb.Fields() {
		switch fb.schema.Field(j).Name {
		// Mapping
		case FlamegraphFieldMappingStart:
			if r.Mapping.IsValid(locationRow) && r.MappingStart.Value(locationRow) > 0 {
				fb.builderMappingStart.Append(r.MappingStart.Value(locationRow))
			} else {
				fb.builderMappingStart.AppendNull()
			}
		case FlamegraphFieldMappingLimit:
			if r.Mapping.IsValid(locationRow) && r.MappingLimit.Value(locationRow) > 0 {
				fb.builderMappingLimit.Append(r.MappingLimit.Value(locationRow))
			} else {
				fb.builderMappingLimit.AppendNull()
			}
		case FlamegraphFieldMappingOffset:
			if r.Mapping.IsValid(locationRow) && r.MappingOffset.Value(locationRow) > 0 {
				fb.builderMappingOffset.Append(r.MappingOffset.Value(locationRow))
			} else {
				fb.builderMappingOffset.AppendNull()
			}
		case FlamegraphFieldMappingFile:
			if r.MappingFileDict.Len() == 0 {
				fb.builderMappingFile.AppendNull()
			} else {
				if r.Mapping.IsValid(locationRow) && len(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(locationRow))) > 0 {
					_ = fb.builderMappingFile.Append(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(locationRow)))
				} else {
					fb.builderMappingFile.AppendNull()
				}
			}
		case FlamegraphFieldMappingBuildID:
			if r.MappingBuildIDDict.Len() == 0 {
				fb.builderMappingBuildID.AppendNull()
			} else {
				if r.Mapping.IsValid(locationRow) && len(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(locationRow))) > 0 {
					_ = fb.builderMappingBuildID.Append(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(locationRow)))
				} else {
					fb.builderMappingBuildID.AppendNull()
				}
			}
		// Location
		case FlamegraphFieldLocationAddress:
			fb.builderLocationAddress.Append(r.Address.Value(locationRow))

		// TODO: Location isFolded we should remove this until we actually support folded functions.
		case FlamegraphFieldLocationFolded:
			fb.builderLocationFolded.AppendSingle(false)
		case FlamegraphFieldLocationLine:
			if lineRow >= 0 && r.Line.IsValid(lineRow) {
				fb.builderLocationLine.Append(r.LineNumber.Value(lineRow))
			} else {
				fb.builderLocationLine.AppendNull()
			}
		// Function
		case FlamegraphFieldFunctionStartLine:
			if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && r.LineFunctionStartLine.Value(lineRow) > 0 {
				fb.builderFunctionStartLine.Append(r.LineFunctionStartLine.Value(lineRow))
			} else {
				fb.builderFunctionStartLine.AppendNull()
			}
		case FlamegraphFieldFunctionName:
			if r.LineFunctionNameDict.Len() == 0 {
				fb.builderFunctionName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow))) > 0 {
					_ = fb.builderFunctionName.Append(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(lineRow)))
				} else {
					fb.builderFunctionName.AppendNull()
				}
			}
		case FlamegraphFieldFunctionSystemName:
			if r.LineFunctionSystemNameDict.Len() == 0 {
				fb.builderFunctionSystemName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionSystemNameDict.Value(r.LineFunctionSystemName.GetValueIndex(lineRow))) > 0 {
					_ = fb.builderFunctionSystemName.Append(r.LineFunctionSystemNameDict.Value(r.LineFunctionSystemName.GetValueIndex(lineRow)))
				} else {
					fb.builderFunctionSystemName.AppendNull()
				}
			}
		case FlamegraphFieldFunctionFileName:
			if r.LineFunctionFilenameDict.Len() == 0 {
				fb.builderFunctionFileName.AppendNull()
			} else {
				if lineRow >= 0 && r.LineFunction.IsValid(lineRow) && len(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(lineRow))) > 0 {
					_ = fb.builderFunctionFileName.Append(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(lineRow)))
				} else {
					fb.builderFunctionFileName.AppendNull()
				}
			}
		// Values
		case FlamegraphFieldLabels:
			if len(labels) > 0 {
				// We add the labels to the potential labels for this row.
				fb.labels[row] = append(fb.labels[row], labels)
			}
		case FlamegraphFieldChildren:
			if len(fb.children) == row {
				// We need to grow the children slice, so we'll do that here.
				// We'll double the capacity of the slice.
				newChildren := make([][]int, len(fb.children)*2)
				copy(newChildren, fb.children)
				fb.children = newChildren
			}
			// If there is a parent for this stack the parent is not -1 but the parent's row number.
			if fb.parent.Has() {
				// this is the first time we see this parent have a child, so we need to initialize the slice
				if len(fb.children[fb.parent.Get()]) == 0 {
					fb.children[fb.parent.Get()] = []int{row}
				} else {
					// otherwise we can just append this row's number to the parent's slice
					fb.children[fb.parent.Get()] = append(fb.children[fb.parent.Get()], row)
				}
			}
		case FlamegraphFieldCumulative:
			fb.builderCumulative.Append(r.Value.Value(sampleRow))
		case FlamegraphFieldDiff:
			if r.Diff.Value(sampleRow) > 0 {
				fb.builderDiff.Append(r.Diff.Value(sampleRow))
			} else {
				fb.builderDiff.AppendNull()
			}
		default:
			panic(fmt.Sprintf("unknown field %s", fb.schema.Field(j).Name))
		}
	}
	return nil
}

func (fb *flamegraphBuilder) AppendLabelRow(r profile.RecordReader, row int, labelKey string, labels map[string]string, sampleRow int) error {
	fb.labels[row] = []map[string]string{labels}

	if len(fb.children) == row {
		// We need to grow the children slice, so we'll do that here.
		// We'll double the capacity of the slice.
		newChildren := make([][]int, len(fb.children)*2)
		copy(newChildren, fb.children)
		fb.children = newChildren
	}
	// Add this label row to the root row's children.
	fb.children[0] = append(fb.children[0], row)
	//// Add the next row as child of this label row.
	//fb.children[row] = append(fb.children[row], row+1)

	fb.builderMappingStart.AppendNull()
	fb.builderMappingLimit.AppendNull()
	fb.builderMappingOffset.AppendNull()
	fb.builderMappingFile.AppendNull()
	fb.builderMappingBuildID.AppendNull()
	fb.builderLocationAddress.AppendNull()
	fb.builderLocationFolded.AppendNull()
	fb.builderLocationLine.AppendNull()
	fb.builderFunctionStartLine.AppendNull()

	err := fb.builderFunctionName.AppendString(labelKey)
	if err != nil {
		return err
	}

	fb.builderFunctionSystemName.AppendNull()
	fb.builderFunctionFileName.AppendNull()

	// Append both cumulative and diff values and overwrite them below.
	fb.builderCumulative.Append(0)
	fb.builderDiff.AppendNull()
	fb.addRowValues(r, row, sampleRow)

	return nil
}

// addRowValues updates the existing row's values and potentially adding existing values on top.
func (fb *flamegraphBuilder) addRowValues(r profile.RecordReader, row, sampleRow int) {
	fb.builderCumulative.Add(row, r.Value.Value(sampleRow))
	if r.Diff.Value(sampleRow) != 0 {
		fb.builderDiff.Add(row, r.Diff.Value(sampleRow))
	}
}

func isLocationRoot(end, i int) bool {
	return i == end-1
}

func isLocationLeaf(beg, i int) bool {
	return i == beg
}

// parent stores the parent's row number of a stack.
type parent int

func (p *parent) Set(i int) { *p = parent(i) }

func (p *parent) Reset() { *p = -1 }

func (p *parent) Get() int { return int(*p) }

func (p *parent) Has() bool { return *p > -1 }

func mapsIntersection(maps []map[string]string) map[string]string {
	if len(maps) == 0 {
		return map[string]string{}
	}
	if len(maps) == 1 {
		return maps[0]
	}

	// this compares the first maps keys to all other maps keys
	// only if a key exists in all maps, and it has the SAME VALUE it will be added to the intersection
	intersection := map[string]string{}
keys:
	for k, v := range maps[0] {
		for i, m := range maps {
			if i == 0 { // don't compare to self
				continue
			}
			if m[k] != v {
				continue keys
			}
		}
		// all maps have the same value for this key
		intersection[k] = v
	}

	return intersection
}

func unsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}
