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
	"encoding/json"
	"fmt"
	"unsafe"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/ipc"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/exp/maps"
	"golang.org/x/exp/slices"

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

func GenerateFlamegraphArrow(ctx context.Context, tracer trace.Tracer, p *profile.Profile, aggregate []string, trimFraction float32) (*queryv1alpha1.FlamegraphArrow, int64, error) {
	mem := memory.NewGoAllocator()
	record, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, aggregate, trimFraction)
	if err != nil {
		return nil, 0, err
	}

	// TODO: Reuse buffer and potentially writers
	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(mem),
	)

	if err = w.Write(record); err != nil {
		return nil, 0, err
	}
	if err := w.Close(); err != nil {
		return nil, 0, err
	}

	return &queryv1alpha1.FlamegraphArrow{
		Record:  buf.Bytes(),
		Unit:    p.Meta.SampleType.Unit,
		Height:  height, // add one for the root
		Trimmed: trimmed,
	}, cumulative, nil
}

func generateFlamegraphArrowRecord(ctx context.Context, mem memory.Allocator, tracer trace.Tracer, p *profile.Profile, aggregate []string, trimFraction float32) (arrow.Record, int64, int32, int64, error) {
	aggregateFields := make(map[string]struct{}, len(aggregate))
	for _, f := range aggregate {
		aggregateFields[f] = struct{}{}
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
		{Name: FlamegraphFieldLabels, Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}},
		{Name: FlamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: FlamegraphFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	}, nil)

	rb := builder.NewRecordBuilder(mem, schema)
	builderChildren := rb.Field(schema.FieldIndices(FlamegraphFieldChildren)[0]).(*builder.ListBuilder)
	fb := flamegraphBuilder{
		rb:     rb,
		schema: schema,
		// parent keeps track of the parent of a row. This is used to build the children array.
		parent: parent(-1),
		// This keeps track of a row's children and will be converted to an arrow array of lists at the end.
		// Allocating for an average of 8 children per stacktrace upfront.
		children: make([][]int, len(p.Samples)*8),

		// TODO: Potentially good to .Reserve() the number of samples to avoid re-allocations
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

	// This field compares the current sample with the already added values in the builders.
	equalField := func(fieldName string, location *profile.Location, line profile.LocationLine, pprofLabels map[string]string, row, height int) bool {
		switch fieldName {
		case FlamegraphFieldMappingFile:
			if location.Mapping == nil {
				return true
			}
			rowMappingFile := fb.builderMappingFile.Value(fb.builderMappingFile.GetValueIndex(row))
			// rather than comparing the strings, we compare bytes to avoid allocations.
			return bytes.Equal(stringToBytes(location.Mapping.File), rowMappingFile)
		case FlamegraphFieldLocationAddress:
			// TODO: do we need to check for null?
			rowLocationAddress := fb.builderLocationAddress.Value(row)
			return location.Address == rowLocationAddress
		case FlamegraphFieldFunctionName:
			rowFunctionName := fb.builderFunctionName.Value(fb.builderFunctionName.GetValueIndex(row))
			// rather than comparing the strings, we compare bytes to avoid allocations.
			return bytes.Equal(stringToBytes(line.Function.Name), rowFunctionName)
		case FlamegraphFieldLabels:
			// We only compare the labels of roots of stacktraces.
			if height > 0 {
				return true
			}

			isNull := fb.builderLabels.IsNull(row)
			if len(pprofLabels) == 0 && isNull {
				return true
			}
			if len(pprofLabels) > 0 && isNull {
				return false
			}
			if len(pprofLabels) == 0 && !isNull {
				return false
			}
			// Both sides have values, let's compare them properly.
			value := fb.builderLabels.Value(fb.builderLabels.GetValueIndex(row))
			compareLabels := map[string]string{}
			err := json.Unmarshal(value, &compareLabels)
			if err != nil {
				return false
			}

			return maps.Equal(pprofLabels, compareLabels)
		default:
			return false
		}
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
	fb.builderLabels.AppendNull()
	// The cumulative values is calculated and at the end set to the correct value.
	fb.builderCumulative.Append(0)
	fb.builderDiff.AppendNull()

	// This keeps track of the total cumulative value so that we can set the first row's cumulative value at the end.
	cumulative := int64(0)
	// This keeps track of the max depth of our flame graph.
	height := int32(0)
	// This keeps track of the root rows.
	// This will be the root row's children, which is always our row 0 in flame graphs.
	rootsRow := []int{}

	// these change with every iteration below
	row := fb.builderCumulative.Len()
	// compareRows are the rows that we compare to the current location against and potentially merge.
	compareRows := []int{}

	for _, s := range p.Samples {
		if int32(len(s.Locations)) > height {
			height = int32(len(s.Locations))
		}

		// every new sample resets the childRow to -1 indicating that we start with a leaf again.
		// pprof stores locations in reverse order, thus we loop over locations in reverse order.
	locations:
		for i := len(s.Locations) - 1; i >= 0; i-- {
			location := s.Locations[i]
			// If the location has no lines, it's not symbolized.
			// We work with the location address instead.
			isRoot := isRoot(s.Locations, i)

			if len(location.Lines) == 0 {
				if isRoot {
					compareRows = compareRows[:0] //  reset the compare rows
					compareRows = append(compareRows, rootsRow...)
					// append this row afterward to not compare to itself
					fb.parent.Reset()
				}
				if isLeaf(i) {
					cumulative += s.Value
				}

				// We compare the location address to the existing rows.
				// If we find a matching address, we merge the values.
			compareRowsAddr:
				for _, cr := range compareRows {
					// TODO: Add support to group by pprof labels with just address available
					if !equalField(FlamegraphFieldLocationAddress, location, profile.LocationLine{}, s.Label, cr, len(s.Locations)-1-i) {
						continue compareRowsAddr
					}

					fb.builderCumulative.Add(cr, s.Value)
					fb.parent.Set(cr)
					compareRows = slices.Clone(fb.children[cr])
					continue locations
				}
				// reset the compare rows
				// if there are no matching rows here, we don't want to merge their children either.
				compareRows = compareRows[:0]

				if isRoot {
					// We aren't merging this root, so we'll keep track of it as a new one.
					rootsRow = append(rootsRow, row)
				}

				err := fb.appendRow(location, profile.LocationLine{}, s, row, isRoot)
				if err != nil {
					return nil, 0, 0, 0, err
				}

				fb.parent.Set(row)
				row = fb.builderCumulative.Len()
			}

		stacktraces:
			// just like locations, pprof stores lines in reverse order.
			for k := len(location.Lines) - 1; k >= 0; k-- {
				line := location.Lines[k]
				if isRoot {
					compareRows = compareRows[:0] //  reset the compare rows
					compareRows = append(compareRows, rootsRow...)
					// append this row afterward to not compare to itself
					fb.parent.Reset()
				}
				if isLeaf(i) {
					cumulative += s.Value
				}

				// If there are no fields we should aggregate we can skip the comparison
				if len(aggregateFields) > 0 {
				compareRows:
					for _, cr := range compareRows {
						for f := range aggregateFields {
							if !equalField(f, location, line, s.Label, cr, len(s.Locations)-1-i) {
								// If a field doesn't match, we can't aggregate this row with the existing one.
								continue compareRows
							}
						}

						// All fields match, so we can aggregate this new row with the existing one.
						fb.builderCumulative.Add(cr, s.Value)
						// Continue with this row as the parent for the next iteration and compare to its children.
						fb.parent.Set(cr)
						compareRows = slices.Clone(fb.children[cr])
						continue stacktraces
					}
					// reset the compare rows
					// if there are no matching rows here, we don't want to merge their children either.
					compareRows = compareRows[:0]
				}

				if isRoot {
					// We aren't merging this root, so we'll keep track of it as a new one.
					rootsRow = append(rootsRow, row)
				}

				err := fb.appendRow(location, line, s, row, isRoot)
				if err != nil {
					return nil, 0, 0, 0, err
				}

				fb.parent.Set(row)
				row = fb.builderCumulative.Len()
			}
		}
	}

	// We have manually tracked the total cumulative value.
	// Now we set/overwrite the cumulative value for the root row (which is always the 0 row in our flame graphs).
	fb.builderCumulative.Set(0, cumulative)

	// We have manually tracked each row's children.
	// So now we need to iterate over all rows in the record and append their children.
	// We cannot do this while building the rows as we need to append the children while iterating over the rows.
	for i := 0; i < fb.builderCumulative.Len(); i++ {
		if i == 0 {
			builderChildren.Append(true)
			for _, child := range rootsRow {
				fb.builderChildrenValues.Append(uint32(child))
			}
			continue
		}
		if len(fb.children[i]) == 0 {
			builderChildren.AppendNull() // leaf
		} else {
			builderChildren.Append(true)
			for _, child := range fb.children[i] {
				fb.builderChildrenValues.Append(uint32(child))
			}
		}
	}

	return rb.NewRecord(), cumulative, height + 1, 0, nil
}

type flamegraphBuilder struct {
	rb       *builder.RecordBuilder
	schema   *arrow.Schema
	parent   parent
	children [][]int

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

func (fb *flamegraphBuilder) appendRow(
	location *profile.Location,
	line profile.LocationLine,
	s *profile.SymbolizedSample,
	row int,
	isRoot bool,
) error {
	for j := range fb.rb.Fields() {
		switch fb.schema.Field(j).Name {
		// Mapping
		case FlamegraphFieldMappingStart:
			if location.Mapping != nil && location.Mapping.Start > 0 {
				fb.builderMappingStart.Append(location.Mapping.Start)
			} else {
				fb.builderMappingStart.AppendNull()
			}
		case FlamegraphFieldMappingLimit:
			if location.Mapping != nil && location.Mapping.Limit > 0 {
				fb.builderMappingLimit.Append(location.Mapping.Limit)
			} else {
				fb.builderMappingLimit.AppendNull()
			}
		case FlamegraphFieldMappingOffset:
			if location.Mapping != nil && location.Mapping.Offset > 0 {
				fb.builderMappingOffset.Append(location.Mapping.Offset)
			} else {
				fb.builderMappingOffset.AppendNull()
			}
		case FlamegraphFieldMappingFile:
			if location.Mapping != nil && location.Mapping.File != "" {
				_ = fb.builderMappingFile.AppendString(location.Mapping.File)
			} else {
				fb.builderMappingFile.AppendNull()
			}
		case FlamegraphFieldMappingBuildID:
			if location.Mapping != nil && location.Mapping.BuildId != "" {
				_ = fb.builderMappingBuildID.AppendString(location.Mapping.BuildId)
			} else {
				fb.builderMappingBuildID.AppendNull()
			}
		// Location
		case FlamegraphFieldLocationAddress:
			fb.builderLocationAddress.Append(location.Address)
		case FlamegraphFieldLocationFolded:
			fb.builderLocationFolded.AppendSingle(location.IsFolded)
		case FlamegraphFieldLocationLine:
			fb.builderLocationLine.Append(line.Line)
		// Function
		case FlamegraphFieldFunctionStartLine:
			if line.Function != nil {
				fb.builderFunctionStartLine.Append(line.Function.StartLine)
			} else {
				fb.builderFunctionStartLine.AppendNull()
			}
		case FlamegraphFieldFunctionName:
			if line.Function != nil {
				_ = fb.builderFunctionName.AppendString(line.Function.Name)
			} else {
				fb.builderFunctionName.AppendNull()
			}
		case FlamegraphFieldFunctionSystemName:
			if line.Function != nil {
				_ = fb.builderFunctionSystemName.AppendString(line.Function.SystemName)
			} else {
				fb.builderFunctionSystemName.AppendNull()
			}
		case FlamegraphFieldFunctionFileName:
			if line.Function != nil {
				_ = fb.builderFunctionFileName.AppendString(line.Function.Filename)
			} else {
				fb.builderFunctionFileName.AppendNull()
			}
		// Values
		case FlamegraphFieldLabels:
			// Only append labels if there are any and only on the root of the stack.
			// Otherwise, append null.
			if len(s.Label) > 0 && isRoot {
				lset, err := json.Marshal(s.Label)
				if err != nil {
					return err
				}
				_ = fb.builderLabels.Append(lset)
			} else {
				fb.builderLabels.AppendNull()
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
			fb.builderCumulative.Append(s.Value)
		case FlamegraphFieldDiff:
			if s.DiffValue > 0 {
				fb.builderDiff.Append(s.DiffValue)
			} else {
				fb.builderDiff.AppendNull()
			}
		default:
			panic(fmt.Sprintf("unknown field %s", fb.schema.Field(j).Name))
		}
	}
	return nil
}

func isRoot(ls []*profile.Location, i int) bool {
	return len(ls)-1 == i
}

func isLeaf(i int) bool {
	return i == 0
}

// parent stores the parent's row number of a stack.
type parent int

func (p *parent) Set(i int) { *p = parent(i) }

func (p *parent) Reset() { *p = -1 }

func (p *parent) Get() int { return int(*p) }

func (p *parent) Has() bool { return *p > -1 }

func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}
