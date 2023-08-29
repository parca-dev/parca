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
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	FlamegraphFieldLabelsOnly = "labels_only"

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

func GenerateFlamegraphArrow(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
	aggregate []string,
	trimFraction float32,
) (*queryv1alpha1.FlamegraphArrow, int64, error) {
	ctx, span := tracer.Start(ctx, "GenerateFlamegraphArrow")
	defer span.End()

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
	ctx, span := tracer.Start(ctx, "generateFlamegraphArrowRecord")
	defer span.End()

	aggregateLabels := false
	for _, f := range aggregate {
		if f == FlamegraphFieldLabels {
			aggregateLabels = true
		}
	}

	totalRows := int64(0)
	for _, r := range p.Samples {
		totalRows += r.NumRows()
	}

	fb, err := newFlamegraphBuilder(mem, totalRows, aggregate, aggregateLabels)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("create flamegraph builder: %w", err)
	}
	defer fb.Release()

	// these change with every iteration below
	row := fb.builderCumulative.Len()

	profileReader := profile.NewReader(p)
	for _, r := range profileReader.RecordReaders {
		if err := fb.ensureLabelColumns(r.LabelFields); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("ensure label columns: %w", err)
		}
		recordLabelIndex, builderToRecordIndexMapping := fb.labelIndexMappings(r.LabelFields)

		t, err := fb.newTranspositions(r)
		if err != nil {
			return nil, 0, 0, 0, fmt.Errorf("create transpositions: %w", err)
		}
		defer t.Release()

		// This field compares the current sample with the already added values in the builders.
		lsbytes := make([]byte, 0, 512)
		for i := 0; i < int(r.Record.NumRows()); i++ {
			beg, end := r.Locations.ValueOffsets(i)

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
				lsbytes = MarshalStringMapSorted(lsbytes, sampleLabels)

				sampleLabelRow := row
				if _, ok := fb.rootsRow[unsafeString(lsbytes)]; ok {
					sampleLabelRow = fb.rootsRow[unsafeString(lsbytes)][0]
					// We want to compare against this found label root's children.
					rootRow := fb.rootsRow[unsafeString(lsbytes)][0]
					fb.copyChildren(fb.children[rootRow])
					fb.addRowValues(r, sampleLabelRow, i) // adds the cumulative and diff values to the existing row
				} else {
					lsstring := string(lsbytes) // we want to cast the bytes to a string and thus copy them.
					err := fb.AppendLabelRow(r, t, recordLabelIndex, sampleLabelRow, i)
					if err != nil {
						return nil, 0, 0, 0, fmt.Errorf("failed to inject label row: %w", err)
					}
					fb.rootsRow[lsstring] = []int{sampleLabelRow}
				}
				fb.maxHeight = max(fb.maxHeight, fb.height)
				fb.height = 1

				fb.parent.Set(sampleLabelRow)
				row = fb.builderCumulative.Len()
			}

			// every new sample resets the childRow to -1 indicating that we start with a leaf again.
			// pprof stores locations in reverse order, thus we loop over locations in reverse order.
			for j := int(end - 1); j >= int(beg); j-- {
				// If the location has no lines, it's not symbolized.
				// We work with the location address instead.

				// This returns whether this location is a root of a stacktrace.
				isLocationRoot := isLocationRoot(int(end), j)
				// Depending on whether we aggregate the labels (and thus inject node labels), we either compare the rows or not.
				isRoot := isLocationRoot && !(aggregateLabels && len(sampleLabels) > 0)

				if isLocationLeaf(int(beg), j) {
					fb.cumulative += r.Value.Value(i)
					fb.diff += r.Diff.Value(i)
				}

				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)
				if !r.Lines.IsValid(j) || llOffsetEnd-llOffsetStart <= 0 {
					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						fb.copyChildren(fb.rootsRow[unsafeString(lsbytes)])
						// append this row afterward to not compare to itself
						fb.parent.Reset()
						fb.maxHeight = max(fb.maxHeight, fb.height)
						fb.height = 0
					}

					merged, err := fb.mergeUnsymbolizedRows(
						r,
						t,
						aggregateLabels,
						recordLabelIndex,
						i, j, int(end),
					)
					if err != nil {
						return nil, 0, 0, 0, err
					}
					if merged {
						fb.height++
						continue
					}

					if isRoot {
						// We aren't merging this root, so we'll keep track of it as a new one.
						lsstring := string(lsbytes) // we want to cast the bytes to a string and thus copy them.
						fb.rootsRow[lsstring] = append(fb.rootsRow[lsstring], row)
					}

					err = fb.appendRow(r, t, builderToRecordIndexMapping, i, j, -1, row)
					if err != nil {
						return nil, 0, 0, 0, err
					}

					fb.parent.Set(row)
					row = fb.builderCumulative.Len()
					continue
				}

				// just like locations, pprof stores lines in reverse order.
				for k := int(llOffsetEnd - 1); k >= int(llOffsetStart); k-- {
					isRoot = isLocationRoot && !(aggregateLabels && len(sampleLabels) > 0) && k == int(llOffsetEnd-1)

					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						fb.copyChildren(fb.rootsRow[unsafeString(lsbytes)])
						// append this row afterward to not compare to itself
						fb.parent.Reset()
						fb.maxHeight = max(fb.maxHeight, fb.height)
						fb.height = 0
					}

					merged, err := fb.mergeSymbolizedRows(r, t, recordLabelIndex, i, j, k, int(end))
					if err != nil {
						return nil, 0, 0, 0, err
					}
					if merged {
						fb.height++
						continue
					}

					if isRoot {
						// We aren't merging this root, so we'll keep track of it as a new one.
						lsstring := string(lsbytes) // we want to cast the bytes to a string and thus copy them.
						fb.rootsRow[lsstring] = append(fb.rootsRow[lsstring], row)
					}

					err = fb.appendRow(r, t, recordLabelIndex, i, j, k, row)
					if err != nil {
						return nil, 0, 0, 0, err
					}

					fb.parent.Set(row)
					row = fb.builderCumulative.Len()
				}
			}
		}
	}
	// the last row can also have the most height.
	fb.maxHeight = max(fb.maxHeight, fb.height)

	_, spanNewRecord := tracer.Start(ctx, "NewRecord")
	defer spanNewRecord.End()

	record, err := fb.NewRecord()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	spanNewRecord.SetAttributes(attribute.Int64("rows", record.NumRows()))

	return record, fb.cumulative, fb.maxHeight + 1, 0, nil
}

func (fb *flamegraphBuilder) labelsEqual(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	sampleIndex int,
	flamegraphRow int,
) bool {
	for i := range fb.builderLabelFields {
		if !fb.labelEqual(r, t, recordLabelIndex, sampleIndex, flamegraphRow, i) {
			return false
		}
	}

	return true
}

func (fb *flamegraphBuilder) labelEqual(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	sampleIndex int,
	flamegraphRow int,
	labelFieldIndex int,
) bool {
	labelColumn := fb.builderLabels[labelFieldIndex]
	fieldIndex := recordLabelIndex[labelFieldIndex]
	recordLabelColumn := r.LabelColumns[fieldIndex]
	dict := r.LabelColumns[fieldIndex].Dict

	sampleHasNonEmptyLabel := false
	sampleLabelValueValueIndex := -1
	var sampleLabelValue []byte
	if recordLabelColumn.Col.IsValid(sampleIndex) {
		sampleLabelValueValueIndex = recordLabelColumn.Col.GetValueIndex(sampleIndex)
		sampleLabelValue = dict.Value(sampleLabelValueValueIndex)
		sampleHasNonEmptyLabel = len(sampleLabelValue) > 0
	}
	flamegraphRowHasNonEmptyLabel := labelColumn.IsValid(flamegraphRow)

	if !sampleHasNonEmptyLabel && !flamegraphRowHasNonEmptyLabel {
		return true
	}
	if flamegraphRowHasNonEmptyLabel && !sampleHasNonEmptyLabel {
		return false
	}
	if !flamegraphRowHasNonEmptyLabel && sampleHasNonEmptyLabel {
		return false
	}

	transposedIndex := t.labels[fieldIndex].indices.Value(sampleLabelValueValueIndex)
	return labelColumn.Value(flamegraphRow) == transposedIndex
}

type transposition struct {
	data    *array.Data
	indices *array.Int32
}

func (t transposition) Release() {
	t.data.Release()
	t.indices.Release()
}

type transpositions struct {
	mappingBuildID transposition
	mappingFile    transposition

	functionName       transposition
	functionSystemName transposition
	functionFilename   transposition

	labels []transposition
}

func (t *transpositions) Release() {
	t.mappingBuildID.Release()
	t.mappingFile.Release()

	t.functionName.Release()
	t.functionSystemName.Release()
	t.functionFilename.Release()

	for i := range t.labels {
		t.labels[i].Release()
	}
}

func (fb *flamegraphBuilder) labelIndexMappings(fields []arrow.Field) ([]int, []int) {
	builderToRecord := make([]int, len(fb.builderLabelFields))
	for i := range fb.builderLabelFields {
		builderToRecord[i] = -1
	}

	recordToBuilder := make([]int, len(fields))
	for i := range fields {
		idx := fb.labelNameIndex[fields[i].Name]
		recordToBuilder[i] = idx
		builderToRecord[idx] = i
	}

	return recordToBuilder, builderToRecord
}

func (fb *flamegraphBuilder) newTranspositions(r *profile.RecordReader) (*transpositions, error) {
	mappingIDIndicesData, mappingIDIndices, err := transpositionFromDict(fb.builderMappingBuildIDDictUnifier, r.MappingBuildIDDict)
	if err != nil {
		return nil, fmt.Errorf("unify and transpose mapping build id dict: %w", err)
	}

	mappingFileIndicesData, mappingFileIndices, err := transpositionFromDict(fb.builderMappingFileDictUnifier, r.MappingFileDict)
	if err != nil {
		return nil, fmt.Errorf("unify and transpose mapping build id dict: %w", err)
	}

	functionNameIndicesData, functionNameIndices, err := transpositionFromDict(fb.builderFunctionNameDictUnifier, r.LineFunctionNameDict)
	if err != nil {
		return nil, fmt.Errorf("unify and transpose function name dict: %w", err)
	}

	functionSystemNameIndicesData, functionSystemNameIndices, err := transpositionFromDict(fb.builderFunctionSystemNameDictUnifier, r.LineFunctionSystemNameDict)
	if err != nil {
		return nil, fmt.Errorf("unify and transpose function system name dict: %w", err)
	}

	functionFilenameIndicesData, functionFilenameIndices, err := transpositionFromDict(fb.builderFunctionFilenameDictUnifier, r.LineFunctionFilenameDict)
	if err != nil {
		return nil, fmt.Errorf("unify and transpose function filename dict: %w", err)
	}

	labels := make([]transposition, len(fb.builderLabelFields))
	for i, labelField := range r.LabelFields {
		builderIndex := fb.labelNameIndex[labelField.Name]
		labelColumn := r.LabelColumns[i]
		labelTranspositionData, labelTransposition, err := transpositionFromDict(fb.builderLabelsDictUnifiers[builderIndex], labelColumn.Dict)
		if err != nil {
			return nil, fmt.Errorf("unify and transpose label dict %q: %w", labelField.Name, err)
		}
		labels[i] = transposition{
			data:    labelTranspositionData,
			indices: labelTransposition,
		}
	}

	return &transpositions{
		mappingBuildID: transposition{
			data:    mappingIDIndicesData,
			indices: mappingIDIndices,
		},
		mappingFile: transposition{
			data:    mappingFileIndicesData,
			indices: mappingFileIndices,
		},
		functionName: transposition{
			data:    functionNameIndicesData,
			indices: functionNameIndices,
		},
		functionSystemName: transposition{
			data:    functionSystemNameIndicesData,
			indices: functionSystemNameIndices,
		},
		functionFilename: transposition{
			data:    functionFilenameIndicesData,
			indices: functionFilenameIndices,
		},
		labels: labels,
	}, nil
}

func transpositionFromDict(unifier array.DictionaryUnifier, dict *array.Binary) (*array.Data, *array.Int32, error) {
	buffer, err := unifier.UnifyAndTranspose(dict)
	if err != nil {
		return nil, nil, err
	}
	defer buffer.Release()
	data := array.NewData(
		arrow.PrimitiveTypes.Int32,
		dict.Len(),
		[]*memory.Buffer{nil, buffer}, // what a quirky API ...
		nil,
		0,
		0,
	)
	defer data.Release()
	indices := array.NewInt32Data(data)

	return data, indices, nil
}

func (fb *flamegraphBuilder) ensureLabelColumns(fields []arrow.Field) error {
	for _, field := range fields {
		if fb.labelExists(field.Name) {
			continue
		}

		fb.addLabelColumn(field)
	}

	fb.ensureLabelColumnsComplete()
	return nil
}

func (fb *flamegraphBuilder) ensureLabelColumnsComplete() {
	numRows := fb.builderCumulative.Len()
	for _, column := range fb.builderLabels {
		if column.Len() < numRows {
			column.AppendNulls(numRows - column.Len())
		}
	}
}

func (fb *flamegraphBuilder) addLabelColumn(field arrow.Field) {
	fb.builderLabelsDictUnifiers = append(fb.builderLabelsDictUnifiers, array.NewBinaryDictionaryUnifier(fb.pool))
	fb.builderLabels = append(fb.builderLabels, builder.NewOptInt32Builder(arrow.PrimitiveTypes.Int32))
	fb.builderLabelFields = append(fb.builderLabelFields, field)
	fb.labelNameIndex[field.Name] = len(fb.builderLabels) - 1
}

func (fb *flamegraphBuilder) labelExists(labelFieldName string) bool {
	_, ok := fb.labelNameIndex[labelFieldName]
	return ok
}

// mergeSymbolizedRows compares the symbolized fields by function name and labels and merges them if they equal.
func (fb *flamegraphBuilder) mergeSymbolizedRows(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	sampleIndex, locationIndex, lineIndex, end int,
) (bool, error) {
	if len(fb.aggregateFields) > 0 {
	compareRows:
		for _, cr := range fb.compareRows {
			for _, f := range fb.aggregateFields {
				if !fb.equalField(r, t, recordLabelIndex, f, sampleIndex, locationIndex, lineIndex, cr, end-1-locationIndex) {
					// If a field doesn't match, we can't aggregate this row with the existing one.
					continue compareRows
				}
			}

			// If we don't group by the labels, we intersect the values before adding them to the flame graph.
			if !fb.aggregateLabels {
				fb.intersectLabels(r, t, recordLabelIndex, sampleIndex, cr)
			}

			// All fields match, so we can aggregate this new row with the existing one.
			fb.addRowValues(r, cr, sampleIndex)
			// Continue with this row as the parent for the next iteration and compare to its children.
			fb.parent.Set(cr)
			fb.copyChildren(fb.children[cr])
			return true, nil
		}
		// reset the compare rows
		// if there are no matching rows here, we don't want to merge their children either.
		fb.compareRows = fb.compareRows[:0]
	}
	return false, nil
}

// mergeUnsymbolizedRows compares the addresses only and ignores potential function names as they are not available.
func (fb *flamegraphBuilder) mergeUnsymbolizedRows(
	r *profile.RecordReader,
	t *transpositions,
	aggregateLabels bool,
	recordLabelIndex []int,
	sampleIndex, locationIndex, end int,
) (bool, error) {
	for _, cr := range fb.compareRows {
		if !fb.equalField(r, t, recordLabelIndex, FlamegraphFieldLocationAddress, sampleIndex, locationIndex, 0, cr, int(end)-1-locationIndex) {
			continue
		}

		// If we don't group by the labels, we only keep those labels that are identical.
		if !aggregateLabels {
			fb.intersectLabels(r, t, recordLabelIndex, sampleIndex, cr)
		}

		fb.builderCumulative.Add(cr, r.Value.Value(sampleIndex))
		fb.parent.Set(cr)
		fb.copyChildren(fb.children[cr])
		return true, nil
	}
	// reset the compare rows
	// if there are no matching rows here, we don't want to merge their children either.
	fb.compareRows = fb.compareRows[:0]
	return false, nil
}

func (fb *flamegraphBuilder) intersectLabels(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	sampleIndex int,
	flamegraphRow int,
) {
	if !fb.builderLabelsExist.Value(flamegraphRow) {
		// No need to intersect if there are no labels.
		return
	}

	labelsExists := false
	for i, labelColumn := range fb.builderLabels {
		if !labelColumn.IsValid(flamegraphRow) {
			// Intersecting with a null value is a no-op.
			continue
		}

		fieldIndex := recordLabelIndex[i]
		recordLabelColumn := r.LabelColumns[fieldIndex]

		if !recordLabelColumn.Col.IsValid(sampleIndex) {
			// At this point we know that the flamegraph row is valid, so
			// intersecting with a null value results in null, so we need to
			// reset it.
			labelColumn.SetNull(flamegraphRow)

			continue
		}

		// if the labels are equal we don't do anything, only when they are
		// different do we have to remove it
		transposedLabelIndex := t.labels[fieldIndex].indices.Value(recordLabelColumn.Col.GetValueIndex(sampleIndex))
		if transposedLabelIndex != labelColumn.Value(flamegraphRow) {
			labelColumn.SetNull(flamegraphRow)
			continue
		}

		// If we get here the labels are equal meaning we have to keep it.
		labelsExists = true
	}
	if !labelsExists {
		// Only need to record change.
		fb.builderLabelsExist.Set(flamegraphRow, false)
	}
}

func (fb *flamegraphBuilder) equalField(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	fieldName string,
	sampleIndex,
	locationRow,
	lineRow,
	flamegraphRow int,
	height int,
) bool {
	switch fieldName {
	case FlamegraphFieldMappingFile:
		return fb.equalMappingFile(r, t, locationRow, flamegraphRow)
	case FlamegraphFieldLocationAddress:
		return fb.equalLocationAddress(r, locationRow, flamegraphRow)
	case FlamegraphFieldFunctionName:
		return fb.equalFunctionName(r, t, lineRow, flamegraphRow)
	case FlamegraphFieldLabels:
		// We only compare the labels of roots of stacktraces.
		if height > 0 {
			return true
		}

		return fb.labelsEqual(r, t, recordLabelIndex, sampleIndex, flamegraphRow)
	default:
		return false
	}
}

func (fb *flamegraphBuilder) equalMappingFile(
	r *profile.RecordReader,
	t *transpositions,
	locationRow,
	flamegraphRow int,
) bool {
	if !r.Mapping.IsValid(locationRow) {
		return true
	}
	rowMappingFileIndex := fb.builderMappingFileIndices.Value(flamegraphRow)
	translatedMappingFileIndex := t.mappingFile.indices.Value(r.MappingFile.GetValueIndex(locationRow))

	return rowMappingFileIndex == translatedMappingFileIndex
}

func (fb *flamegraphBuilder) equalLocationAddress(
	r *profile.RecordReader,
	locationRow,
	flamegraphRow int,
) bool {
	return r.Address.Value(locationRow) == fb.builderLocationAddress.Value(flamegraphRow)
}

func (fb *flamegraphBuilder) equalFunctionName(
	r *profile.RecordReader,
	t *transpositions,
	lineRow,
	flamegraphRow int,
) bool {
	fgRowFunctionIsNull := fb.builderFunctionNameIndices.IsNull(flamegraphRow)
	lineRowFunctionIsNull := r.LineFunctionName.IsNull(lineRow)

	if fgRowFunctionIsNull != lineRowFunctionIsNull {
		return false
	}
	if fgRowFunctionIsNull && lineRowFunctionIsNull {
		return true
	}

	rowFunctionNameIndex := fb.builderFunctionNameIndices.Value(flamegraphRow)
	translatedFunctionNameIndex := t.functionName.indices.Value(r.LineFunctionName.GetValueIndex(lineRow))
	return rowFunctionNameIndex == translatedFunctionNameIndex
}

func (fb *flamegraphBuilder) copyChildren(children []int) {
	if cap(fb.compareRows) < len(children) {
		fb.compareRows = make([]int, len(children))
	}

	fb.compareRows = fb.compareRows[:len(children)]
	copy(fb.compareRows, children)
}

type flamegraphBuilder struct {
	pool memory.Allocator

	aggregateFields []string
	aggregateLabels bool

	// This keeps track of the total cumulative value so that we can set the first row's cumulative value at the end.
	cumulative int64
	// This keeps track of the total diff values so that we can set the irst row's diff value at the end.
	diff int64
	// This keeps track of the max height of the flame graph.
	maxHeight int32
	// parent keeps track of the parent of a row. This is used to build the children array.
	parent parent
	// This keeps track of a row's children and will be converted to an arrow array of lists at the end.
	// Allocating for an average of 8 children per stacktrace upfront.
	children [][]int

	// This keeps track of the root rows indexed by the labels string.
	// If the stack trace has no labels, we use the empty string as the key.
	// This will be the root row's children, which is always our row 0 in flame graphs.
	rootsRow map[string][]int
	// compareRows are the rows that we compare to the current location against and potentially merge.
	compareRows []int
	// height keeps track of the current stack trace's height of the flame graph.
	height int32

	builderLabelsOnly                    *array.BooleanBuilder
	builderMappingStart                  *array.Uint64Builder
	builderMappingLimit                  *array.Uint64Builder
	builderMappingOffset                 *array.Uint64Builder
	builderMappingFileIndices            *array.Int32Builder
	builderMappingFileDictUnifier        array.DictionaryUnifier
	builderMappingBuildIDIndices         *array.Int32Builder
	builderMappingBuildIDDictUnifier     array.DictionaryUnifier
	builderLocationAddress               *array.Uint64Builder
	builderLocationFolded                *builder.OptBooleanBuilder
	builderLocationLine                  *builder.OptInt64Builder
	builderFunctionStartLine             *builder.OptInt64Builder
	builderFunctionNameIndices           *array.Int32Builder
	builderFunctionNameDictUnifier       array.DictionaryUnifier
	builderFunctionSystemNameIndices     *array.Int32Builder
	builderFunctionSystemNameDictUnifier array.DictionaryUnifier
	builderFunctionFilenameIndices       *array.Int32Builder
	builderFunctionFilenameDictUnifier   array.DictionaryUnifier
	builderLabelFields                   []arrow.Field
	builderLabelsExist                   *builder.OptBooleanBuilder
	builderLabels                        []*builder.OptInt32Builder
	builderLabelsDictUnifiers            []array.DictionaryUnifier
	builderChildren                      *builder.ListBuilder
	builderChildrenValues                *array.Uint32Builder
	builderCumulative                    *builder.OptInt64Builder
	builderDiff                          *builder.OptInt64Builder

	labelNameIndex map[string]int
}

func newFlamegraphBuilder(
	pool memory.Allocator,
	rows int64,
	aggregateFields []string,
	aggregateLabels bool,
) (*flamegraphBuilder, error) {
	builderChildren := builder.NewListBuilder(pool, arrow.PrimitiveTypes.Uint32)
	fb := &flamegraphBuilder{
		pool: pool,

		aggregateFields: aggregateFields,
		aggregateLabels: aggregateLabels,

		parent:         parent(-1),
		children:       make([][]int, rows),
		rootsRow:       map[string][]int{},
		labelNameIndex: map[string]int{},
		compareRows:    make([]int, 0, 32),

		builderLabelsOnly:  array.NewBooleanBuilder(pool),
		builderLabelsExist: builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean),

		builderMappingStart:              array.NewUint64Builder(pool),
		builderMappingLimit:              array.NewUint64Builder(pool),
		builderMappingOffset:             array.NewUint64Builder(pool),
		builderMappingFileIndices:        array.NewInt32Builder(pool),
		builderMappingFileDictUnifier:    array.NewBinaryDictionaryUnifier(pool),
		builderMappingBuildIDIndices:     array.NewInt32Builder(pool),
		builderMappingBuildIDDictUnifier: array.NewBinaryDictionaryUnifier(pool),

		builderLocationAddress: array.NewUint64Builder(pool),
		builderLocationFolded:  builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean),
		builderLocationLine:    builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),

		builderFunctionStartLine:             builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderFunctionNameIndices:           array.NewInt32Builder(pool),
		builderFunctionNameDictUnifier:       array.NewBinaryDictionaryUnifier(pool),
		builderFunctionSystemNameIndices:     array.NewInt32Builder(pool),
		builderFunctionSystemNameDictUnifier: array.NewBinaryDictionaryUnifier(pool),
		builderFunctionFilenameIndices:       array.NewInt32Builder(pool),
		builderFunctionFilenameDictUnifier:   array.NewBinaryDictionaryUnifier(pool),

		builderChildren:       builderChildren,
		builderChildrenValues: builderChildren.ValueBuilder().(*array.Uint32Builder),
		builderCumulative:     builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderDiff:           builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
	}

	// The very first row is the root row. It doesn't contain any metadata.
	// It only contains the root cumulative value and list of children (which are actual roots).
	fb.builderLabelsExist.AppendSingle(false)
	fb.builderLabelsOnly.AppendNull()
	fb.builderMappingStart.AppendNull()
	fb.builderMappingLimit.AppendNull()
	fb.builderMappingOffset.AppendNull()
	fb.builderMappingFileIndices.AppendNull()
	fb.builderMappingBuildIDIndices.AppendNull()

	fb.builderLocationAddress.AppendNull()
	fb.builderLocationFolded.AppendNull()
	fb.builderLocationLine.AppendNull()

	fb.builderFunctionStartLine.AppendNull()
	fb.builderFunctionNameIndices.AppendNull()
	fb.builderFunctionSystemNameIndices.AppendNull()
	fb.builderFunctionFilenameIndices.AppendNull()

	// The cumulative values is calculated and at the end set to the correct value.
	fb.builderCumulative.Append(0)
	fb.builderDiff.Append(0)

	return fb, nil
}

// NewRecord returns a new record from the builders.
// It adds the children to the children column and the labels intersection to the labels column.
// Finally, it assembles all columns from the builders into an arrow record.
func (fb *flamegraphBuilder) NewRecord() (arrow.Record, error) {
	cleanupArrs := make([]arrow.Array, 0, 26+(2*len(fb.builderLabelFields)))
	defer func() {
		for _, arr := range cleanupArrs {
			arr.Release()
		}
	}()
	// We have manually tracked the total cumulative value.
	// Now we set/overwrite the cumulative value for the root row (which is always the 0 row in our flame graphs).
	fb.builderCumulative.Set(0, fb.cumulative)
	fb.builderDiff.Set(0, fb.diff)

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
	fb.ensureLabelColumnsComplete()

	mappingBuildIDIndices := fb.builderMappingBuildIDIndices.NewArray()
	cleanupArrs = append(cleanupArrs, mappingBuildIDIndices)
	mappingBuildIDDict, err := fb.builderMappingBuildIDDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return nil, err
	}
	cleanupArrs = append(cleanupArrs, mappingBuildIDDict)
	mappingBuildIDType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
	mappingBuildID := array.NewDictionaryArray(mappingBuildIDType, mappingBuildIDIndices, mappingBuildIDDict)
	cleanupArrs = append(cleanupArrs, mappingBuildID)

	mappingFileIndices := fb.builderMappingFileIndices.NewArray()
	cleanupArrs = append(cleanupArrs, mappingFileIndices)
	mappingFileDict, err := fb.builderMappingFileDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return nil, err
	}
	cleanupArrs = append(cleanupArrs, mappingFileDict)
	mappingFileType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
	mappingFile := array.NewDictionaryArray(mappingFileType, mappingFileIndices, mappingFileDict)
	cleanupArrs = append(cleanupArrs, mappingFile)

	functionNameIndices := fb.builderFunctionNameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionNameIndices)
	functionNameDict, err := fb.builderFunctionNameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return nil, err
	}
	cleanupArrs = append(cleanupArrs, functionNameDict)
	functionNameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
	functionName := array.NewDictionaryArray(functionNameType, functionNameIndices, functionNameDict)
	cleanupArrs = append(cleanupArrs, functionName)

	functionSystemNameIndices := fb.builderFunctionSystemNameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionSystemNameIndices)
	functionSystemNameDict, err := fb.builderFunctionSystemNameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return nil, err
	}
	cleanupArrs = append(cleanupArrs, functionSystemNameDict)
	functionSystemNameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
	functionSystemName := array.NewDictionaryArray(functionSystemNameType, functionSystemNameIndices, functionSystemNameDict)
	cleanupArrs = append(cleanupArrs, functionSystemName)

	functionFilenameIndices := fb.builderFunctionFilenameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionFilenameIndices)
	functionFilenameDict, err := fb.builderFunctionFilenameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return nil, err
	}
	cleanupArrs = append(cleanupArrs, functionFilenameDict)
	functionFilenameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
	functionFilename := array.NewDictionaryArray(functionFilenameType, functionFilenameIndices, functionFilenameDict)
	cleanupArrs = append(cleanupArrs, functionFilename)

	// This has to be here, because after calling .NewArray() on the builder,
	// the builder is reset.
	numRows := fb.builderCumulative.Len()

	fields := []arrow.Field{
		{Name: FlamegraphFieldLabelsOnly, Type: arrow.FixedWidthTypes.Boolean},
		{Name: FlamegraphFieldMappingStart, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingLimit, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingOffset, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingFile, Type: mappingFileType},
		{Name: FlamegraphFieldMappingBuildID, Type: mappingBuildIDType},
		// Location
		{Name: FlamegraphFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldLocationFolded, Type: arrow.FixedWidthTypes.Boolean},
		{Name: FlamegraphFieldLocationLine, Type: arrow.PrimitiveTypes.Int64},
		// Function
		{Name: FlamegraphFieldFunctionStartLine, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldFunctionName, Type: functionNameType},
		{Name: FlamegraphFieldFunctionSystemName, Type: functionSystemNameType},
		{Name: FlamegraphFieldFunctionFileName, Type: functionFilenameType},
		// Values
		{Name: FlamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: FlamegraphFieldCumulative, Type: arrow.PrimitiveTypes.Int64},
		{Name: FlamegraphFieldDiff, Type: arrow.PrimitiveTypes.Int64, Nullable: true},
	}

	arrays := make([]arrow.Array, 16)
	arrays[0] = fb.builderLabelsOnly.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[0])
	arrays[1] = fb.builderMappingStart.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[1])
	arrays[2] = fb.builderMappingLimit.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[2])
	arrays[3] = fb.builderMappingOffset.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[3])
	arrays[4] = mappingFile
	arrays[5] = mappingBuildID
	arrays[6] = fb.builderLocationAddress.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[6])
	arrays[7] = fb.builderLocationFolded.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[7])
	arrays[8] = fb.builderLocationLine.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[8])
	arrays[9] = fb.builderFunctionStartLine.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[9])
	arrays[10] = functionName
	arrays[11] = functionSystemName
	arrays[12] = functionFilename
	arrays[13] = fb.builderChildren.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[13])
	arrays[14] = fb.builderCumulative.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[14])
	arrays[15] = fb.builderDiff.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[15])

	for i := range fb.builderLabelFields {
		if err := func() error {
			typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.String}
			fields = append(fields, arrow.Field{
				Name: fb.builderLabelFields[i].Name,
				Type: typ,
			})
			indices := fb.builderLabels[i].NewArray()
			cleanupArrs = append(cleanupArrs, indices)
			dict, err := fb.builderLabelsDictUnifiers[i].GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
			if err != nil {
				return err
			}
			cleanupArrs = append(cleanupArrs, dict)
			dictarray := array.NewDictionaryArray(typ, indices, dict)
			cleanupArrs = append(cleanupArrs, dictarray)
			arrays = append(arrays, dictarray)
			return nil
		}(); err != nil {
			return nil, err
		}
	}

	return array.NewRecord(
		arrow.NewSchema(fields, nil),
		arrays,
		int64(numRows)), nil
}

func (fb *flamegraphBuilder) Release() {
	fb.builderLabelsOnly.Release()
	fb.builderLabelsExist.Release()

	fb.builderMappingStart.Release()
	fb.builderMappingLimit.Release()
	fb.builderMappingOffset.Release()
	fb.builderMappingFileIndices.Release()
	fb.builderMappingFileDictUnifier.Release()
	fb.builderMappingBuildIDIndices.Release()
	fb.builderMappingBuildIDDictUnifier.Release()

	fb.builderLocationAddress.Release()
	fb.builderLocationFolded.Release()
	fb.builderLocationLine.Release()

	fb.builderFunctionStartLine.Release()
	fb.builderFunctionNameIndices.Release()
	fb.builderFunctionNameDictUnifier.Release()
	fb.builderFunctionSystemNameIndices.Release()
	fb.builderFunctionSystemNameDictUnifier.Release()
	fb.builderFunctionFilenameIndices.Release()
	fb.builderFunctionFilenameDictUnifier.Release()

	fb.builderChildren.Release()
	fb.builderChildrenValues.Release()
	fb.builderCumulative.Release()
	fb.builderDiff.Release()

	for i := range fb.builderLabelFields {
		fb.builderLabels[i].Release()
		fb.builderLabelsDictUnifiers[i].Release()
	}
}

func (fb *flamegraphBuilder) appendRow(
	r *profile.RecordReader,
	t *transpositions,
	builderToRecordIndexMapping []int,
	sampleRow, locationRow, lineRow int,
	row int,
) error {
	fb.height++

	fb.builderLabelsOnly.Append(false)

	// Mapping
	if r.Mapping.IsValid(locationRow) {
		fb.builderMappingStart.Append(r.MappingStart.Value(locationRow))
		fb.builderMappingLimit.Append(r.MappingLimit.Value(locationRow))
		fb.builderMappingOffset.Append(r.MappingOffset.Value(locationRow))
		fb.builderMappingFileIndices.Append(t.mappingFile.indices.Value(r.MappingFile.GetValueIndex(locationRow)))
		fb.builderMappingBuildIDIndices.Append(t.mappingBuildID.indices.Value(r.MappingBuildID.GetValueIndex(locationRow)))
	} else {
		fb.builderMappingStart.AppendNull()
		fb.builderMappingLimit.AppendNull()
		fb.builderMappingOffset.AppendNull()
		fb.builderMappingFileIndices.AppendNull()
		fb.builderMappingBuildIDIndices.AppendNull()
	}

	fb.builderLocationAddress.Append(r.Address.Value(locationRow))
	fb.builderLocationFolded.AppendSingle(false)

	if lineRow == -1 {
		fb.builderLocationLine.AppendNull()
		fb.builderFunctionStartLine.AppendNull()
		fb.builderFunctionNameIndices.AppendNull()
		fb.builderFunctionSystemNameIndices.AppendNull()
		fb.builderFunctionFilenameIndices.AppendNull()
	} else {
		// A non -1 lineRow means that the line is definitely valid, otherwise
		// something has already gone terribly wrong.
		fb.builderLocationLine.Append(r.LineNumber.Value(lineRow))

		if r.LineFunction.IsValid(lineRow) {
			fb.builderFunctionStartLine.Append(r.LineFunctionStartLine.Value(lineRow))
			fb.builderFunctionNameIndices.Append(t.functionName.indices.Value(r.LineFunctionName.GetValueIndex(lineRow)))
			fb.builderFunctionSystemNameIndices.Append(t.functionSystemName.indices.Value(r.LineFunctionSystemName.GetValueIndex(lineRow)))
			fb.builderFunctionFilenameIndices.Append(t.functionFilename.indices.Value(r.LineFunctionFilename.GetValueIndex(lineRow)))
		} else {
			fb.builderFunctionStartLine.AppendNull()
			fb.builderFunctionNameIndices.AppendNull()
			fb.builderFunctionSystemNameIndices.AppendNull()
			fb.builderFunctionFilenameIndices.AppendNull()
		}
	}

	// Values

	labelsExist := false
	for i, builderLabel := range fb.builderLabels {
		if recordIndex := builderToRecordIndexMapping[i]; recordIndex != -1 {
			lc := r.LabelColumns[recordIndex]
			if lc.Col.IsValid(sampleRow) && len(lc.Dict.Value(lc.Col.GetValueIndex(sampleRow))) > 0 {
				transposedIndex := t.labels[i].indices.Value(lc.Col.GetValueIndex(sampleRow))
				builderLabel.Append(transposedIndex)
				labelsExist = true
			} else {
				builderLabel.AppendNull()
			}
		} else {
			builderLabel.AppendNull()
		}
	}
	fb.builderLabelsExist.AppendSingle(labelsExist)

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

	fb.builderCumulative.Append(r.Value.Value(sampleRow))

	if r.Diff.Value(sampleRow) > 0 {
		fb.builderDiff.Append(r.Diff.Value(sampleRow))
	} else {
		fb.builderDiff.AppendNull()
	}

	return nil
}

func (fb *flamegraphBuilder) AppendLabelRow(
	r *profile.RecordReader,
	t *transpositions,
	builderToRecordIndexMapping []int,
	row int,
	sampleRow int,
) error {
	labelsExist := false
	for i, labelColumn := range fb.builderLabels {
		if recordIndex := builderToRecordIndexMapping[i]; recordIndex != -1 {
			lc := r.LabelColumns[recordIndex]
			if lc.Col.IsValid(sampleRow) && len(lc.Dict.Value(lc.Col.GetValueIndex(sampleRow))) > 0 {
				transposedIndex := t.labels[i].indices.Value(lc.Col.GetValueIndex(sampleRow))
				labelColumn.Append(transposedIndex)
				labelsExist = true
			} else {
				labelColumn.AppendNull()
			}
		} else {
			labelColumn.AppendNull()
		}
	}
	fb.builderLabelsExist.AppendSingle(labelsExist)

	if len(fb.children) == row {
		// We need to grow the children slice, so we'll do that here.
		// We'll double the capacity of the slice.
		newChildren := make([][]int, len(fb.children)*2)
		copy(newChildren, fb.children)
		fb.children = newChildren
	}
	// Add this label row to the root row's children.
	fb.children[0] = append(fb.children[0], row)

	fb.builderLabelsOnly.Append(true)
	fb.builderMappingStart.AppendNull()
	fb.builderMappingLimit.AppendNull()
	fb.builderMappingOffset.AppendNull()
	fb.builderMappingFileIndices.AppendNull()
	fb.builderMappingBuildIDIndices.AppendNull()
	fb.builderLocationAddress.AppendNull()
	fb.builderLocationFolded.AppendNull()
	fb.builderLocationLine.AppendNull()
	fb.builderFunctionStartLine.AppendNull()
	fb.builderFunctionNameIndices.AppendNull()
	fb.builderFunctionSystemNameIndices.AppendNull()
	fb.builderFunctionFilenameIndices.AppendNull()

	// Append both cumulative and diff values and overwrite them below.
	fb.builderCumulative.Append(0)
	fb.builderDiff.Append(0)
	fb.addRowValues(r, row, sampleRow)

	return nil
}

// addRowValues updates the existing row's values and potentially adding existing values on top.
func (fb *flamegraphBuilder) addRowValues(r *profile.RecordReader, row, sampleRow int) {
	fb.builderCumulative.Add(row, r.Value.Value(sampleRow))
	fb.builderDiff.Add(row, r.Diff.Value(sampleRow))
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
