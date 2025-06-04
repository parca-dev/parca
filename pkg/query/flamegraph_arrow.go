// Copyright 2023-2025 The Parca Authors
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
	stdmath "math"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/olekukonko/tablewriter"
	"github.com/polarsignals/frostdb/pqarrow/builder"
	"github.com/zeebo/xxh3"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	compactDictionary "github.com/parca-dev/parca/pkg/compactdictionary"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	FlamegraphFieldLabelsOnly = "labels_only"

	FlamegraphFieldMappingFile    = "mapping_file"
	FlamegraphFieldMappingBuildID = "mapping_build_id"

	FlamegraphFieldLocationAddress = "location_address"
	FlamegraphFieldLocationLine    = "location_line"

	FlamegraphFieldInlined            = "inlined"
	FlamegraphFieldFunctionStartLine  = "function_startline"
	FlamegraphFieldFunctionName       = "function_name"
	FlamegraphFieldFunctionSystemName = "function_system_name"
	FlamegraphFieldFunctionFileName   = "function_file_name"

	FlamegraphFieldLabels     = "labels"
	FlamegraphFieldChildren   = "children"
	FlamegraphFieldParent     = "parent"
	FlamegraphFieldCumulative = "cumulative"
	FlamegraphFieldFlat       = "flat"
	FlamegraphFieldDiff       = "diff"

	FlamegraphFieldTimestamp   = "timestamp"
	FlamegraphFieldDepth       = "depth"
	FlamegraphFieldValueOffset = "value_offset"
)

func GenerateFlamegraphArrow(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
	groupBy []string,
	trimFraction float32,
) (*queryv1alpha1.FlamegraphArrow, int64, error) {
	ctx, span := tracer.Start(ctx, "GenerateFlamegraphArrow")
	defer span.End()

	record, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, groupBy, trimFraction)
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

	span.SetAttributes(attribute.Int("record_size", buf.Len()))
	if buf.Len() > 1<<24 { // 16MiB
		span.SetAttributes(attribute.String("record_stats", recordStats(record)))
	}

	return &queryv1alpha1.FlamegraphArrow{
		Record:  buf.Bytes(),
		Unit:    p.Meta.SampleType.Unit,
		Height:  height, // add one for the root
		Trimmed: trimmed,
	}, cumulative, nil
}

func generateFlamegraphArrowRecord(ctx context.Context, mem memory.Allocator, tracer trace.Tracer, p profile.Profile, groupBy []string, trimFraction float32) (arrow.Record, int64, int32, int64, error) {
	ctx, span := tracer.Start(ctx, "generateFlamegraphArrowRecord")
	defer span.End()

	totalRows := int64(0)
	for _, r := range p.Samples {
		totalRows += r.NumRows()
	}

	fb, err := newFlamegraphBuilder(mem, totalRows, groupBy)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("create flamegraph builder: %w", err)
	}
	defer fb.Release()

	// these change with every iteration below
	row := fb.builderCumulative.Len()

	profileReader := profile.NewReader(p)
	labelHasher := xxh3.New()
	for _, r := range profileReader.RecordReaders {
		fb.cumulative += math.Int64.Sum(r.Value)
		fb.diff += math.Int64.Sum(r.Diff)

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
		for i := 0; i < int(r.Record.NumRows()); i++ {
			beg, end := r.Locations.ValueOffsets(i)

			hasLabels := false
			labelHash := uint64(0)
			for _, field := range fb.builderLabelFields {
				if r.LabelColumns[fb.labelNameIndex[field.Name]].Col.IsValid(i) {
					hasLabels = true
					break
				}
			}

			rootRowChildren := fb.children[0]
			rootRow := 0
			if len(fb.aggregationConfig.aggregateByLabels) > 0 && hasLabels {
				labelHasher.Reset()

				for _, field := range fb.builderLabelFields {
					index := fb.labelNameIndex[field.Name]
					labelColumn := r.LabelColumns[index]
					if labelColumn.Col.IsValid(i) {
						_, _ = labelHasher.WriteString(r.LabelFields[index].Name)
						_, _ = labelHasher.Write(labelColumn.Dict.Value(int(labelColumn.Col.Value(i))))
					}
				}
				labelHash = labelHasher.Sum64()
				sampleLabelRow := row
				if row, ok := fb.rootsRow[labelHash]; ok {
					// We want to compare against this found label root's children.
					rootRowChildren = fb.children[row]
					rootRow = row
					fb.compareRows = rootRowChildren
					fb.addRowValues(r, row, i, false) // adds the cumulative and diff values to the existing row
				} else {
					rootRowChildren = map[uint64]int{}
					err := fb.AppendLabelRow(
						r,
						t,
						builderToRecordIndexMapping,
						sampleLabelRow,
						i,
						labelHash,
						rootRowChildren,
						false, // labels will never actually have a flat value themselves.
					)
					if err != nil {
						return nil, 0, 0, 0, fmt.Errorf("failed to inject label row: %w", err)
					}
					rootRow = sampleLabelRow
				}
				fb.maxHeight = max(fb.maxHeight, fb.height)
				fb.height = 1 // We start with root + label row.

				fb.parent.Set(rootRow)
				row = fb.builderCumulative.Len()
			}

			// every new sample resets the childRow to -1 indicating that we start with a leaf again.
			// pprof stores locations in reverse order, thus we loop over locations in reverse order.
			for j := int(end - 1); j >= int(beg); j-- {
				if r.Locations.ListValues().IsNull(j) {
					continue // skip null values; these have been filtered out.
				}
				// If the location has no lines, it's not symbolized.
				// We work with the location address instead.

				// This returns whether this location is a root of a stacktrace.
				locationRoot := isLocationRoot(beg, end, int64(j), r.Locations)
				// Depending on whether we aggregate the labels (and thus inject node labels), we either compare the rows or not.
				isRoot := locationRoot && !(len(fb.aggregationConfig.aggregateByLabels) > 0 && hasLabels)

				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)
				if !r.Lines.IsValid(j) || llOffsetEnd-llOffsetStart <= 0 {
					isLeaf := isFirstNonNil(i, j, r.Locations)

					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						fb.compareRows = rootRowChildren
						// append this row afterward to not compare to itself
						fb.parent.Reset()
						fb.maxHeight = max(fb.maxHeight, fb.height)
						fb.height = 0
					}

					key := r.Address.Value(j)
					merged, err := fb.mergeUnsymbolizedRows(
						r,
						t,
						recordLabelIndex,
						i, j, int(end),
						key,
						isLeaf,
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
						rootRowChildren[key] = row
						fb.childrenList[rootRow] = append(fb.childrenList[rootRow], row)
					}

					err = fb.appendRow(r, t, builderToRecordIndexMapping, i, j, -1, row, key, isLeaf, false)
					if err != nil {
						return nil, 0, 0, 0, err
					}

					fb.parent.Set(row)
					row = fb.builderCumulative.Len()
					continue
				}

				// just like locations, pprof stores lines in reverse order.
				for k := int(llOffsetEnd - 1); k >= int(llOffsetStart); k-- {
					isInlineRoot := isLocationRoot(llOffsetStart, llOffsetEnd, int64(k), r.Lines)
					isInlined := !isInlineRoot
					isLeaf := isFirstNonNil(i, j, r.Locations) && isFirstNonNil(j, k, r.Lines)

					isRoot = locationRoot && !(len(fb.aggregationConfig.aggregateByLabels) > 0 && hasLabels) && isInlineRoot
					// We only want to compare the rows if this is the root, and we don't aggregate the labels.
					if isRoot {
						fb.compareRows = rootRowChildren
						// append this row afterward to not compare to itself
						fb.parent.Reset()
						fb.maxHeight = max(fb.maxHeight, fb.height)
						fb.height = 0
					}

					translatedFunctionNameIndex := t.functionName.indices.Value(int(r.LineFunctionNameIndices.Value(k)))
					key := uint64(translatedFunctionNameIndex)

					if len(fb.aggregationConfig.aggregateByLabels) > 0 {
						key = hashCombine(key, labelHash)
					}
					if fb.aggregationConfig.aggregateByMappingFile {
						translatedMappingFileIndex := t.mappingFile.indices.Value(int(r.MappingFileIndices.Value(j)))
						key = hashCombine(key, uint64(translatedMappingFileIndex))
					}
					if fb.aggregationConfig.aggregateByLocationAddress {
						key = hashCombine(key, r.Address.Value(j))
					}
					if fb.aggregationConfig.aggregateByFunctionFilename {
						translatedFunctionFilenameIndex := t.functionFilename.indices.Value(int(r.LineFunctionFilenameIndices.Value(k)))
						key = hashCombine(key, uint64(translatedFunctionFilenameIndex))
					}

					merged, err := fb.mergeSymbolizedRows(
						r,
						t,
						recordLabelIndex,
						i,
						j,
						k,
						int(end),
						key,
						isLeaf,
						isInlined,
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
						rootRowChildren[key] = row
						fb.childrenList[rootRow] = append(fb.childrenList[rootRow], row)
					}

					err = fb.appendRow(r, t, builderToRecordIndexMapping, i, j, k, row, key, isLeaf, isInlined)
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

	// We need to set the first row's cumulative and diff values.
	// We unify the dictionaries unifiers and indices into actual dictionaries.
	// These are need for trimming and compaction later on.
	if err := fb.prepareNewRecord(); err != nil {
		return nil, 0, 0, 0, fmt.Errorf("failed to prepare the new record: %w", err)
	}

	// Trim only if we have more rows than the root row.
	if fb.builderCumulative.Len() > 1 {
		if err := fb.trim(ctx, tracer, trimFraction); err != nil {
			return nil, 0, 0, 0, fmt.Errorf("failed to trim flame graph: %w", err)
		}
	} else {
		fb.trimmedLocationLine = array.NewUint8Builder(fb.pool)
		fb.trimmedLocationLine.AppendNull()
		fb.trimmedFunctionStartLine = array.NewUint8Builder(fb.pool)
		fb.trimmedFunctionStartLine.AppendNull()
		fb.trimmedCumulative = array.NewUint8Builder(fb.pool)
		fb.trimmedCumulative.AppendNull()
		fb.trimmedFlat = array.NewUint8Builder(fb.pool)
		fb.trimmedFlat.AppendNull()
		fb.trimmedDiff = array.NewUint8Builder(fb.pool)
		fb.trimmedDiff.AppendNull()
		fb.trimmedTimestamp = builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64)
		fb.trimmedTimestamp.AppendNull()
		fb.trimmedDepth = array.NewUint8Builder(fb.pool)
		fb.trimmedDepth.AppendNull()
		fb.trimmedParent = array.NewInt32Builder(fb.pool)
		fb.trimmedParent.Append(-1) // -1 indicates no parent, which is the root row.
		fb.valueOffset = array.NewUint8Builder(fb.pool)
		fb.valueOffset.(*array.Uint8Builder).Append(0) // by definition the root row has a value offset of 0
	}

	_, spanNewRecord := tracer.Start(ctx, "NewRecord")
	defer spanNewRecord.End()

	record, err := fb.NewRecord()
	if err != nil {
		return nil, 0, 0, 0, err
	}
	spanNewRecord.SetAttributes(attribute.Int64("rows", record.NumRows()))

	return record, fb.cumulative, fb.maxHeight + 1, fb.trimmed, nil
}

// Go translation of boost's hash_combine function. Read here why these values
// are used and good choices: https://stackoverflow.com/questions/35985960/c-why-is-boosthash-combine-the-best-way-to-combine-hash-values
func hashCombine(lhs, rhs uint64) uint64 {
	return lhs ^ (rhs + 0x9e3779b9 + (lhs << 6) + (lhs >> 2))
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
	foundCount := 0
	for i := range fields {
		if _, found := fb.labelNameIndex[fields[i].Name]; found {
			recordToBuilder[i] = foundCount
			builderToRecord[foundCount] = i
			foundCount++
		} else {
			// We ignore all label columns that were project but won't be grouped by.
			recordToBuilder[i] = -1
		}
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
	for i, labelField := range fb.builderLabelFields {
		labelColumn := r.LabelColumns[fb.labelNameIndex[labelField.Name]]
		labelTranspositionData, labelTransposition, err := transpositionFromDict(fb.builderLabelsDictUnifiers[i], labelColumn.Dict)
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
	indices := array.NewInt32Data(data)
	return data, indices, nil
}

func (fb *flamegraphBuilder) ensureLabelColumns(fields []arrow.Field) error {
	for i, field := range fields {
		if fb.labelExists(field.Name) {
			continue
		}
		if _, exists := fb.aggregationConfig.aggregateByLabels[field.Name]; !exists {
			// In production, we shouldn't have columns that aren't aggregated by.
			// This is just making sure we don't return columns if we don't group by them - for tests.
			continue
		}

		fb.addLabelColumn(field, i)
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

func (fb *flamegraphBuilder) addLabelColumn(field arrow.Field, index int) {
	// We need to make sure the field has an int32 index for now.
	field.Type = &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.builderLabelsDictUnifiers = append(fb.builderLabelsDictUnifiers, array.NewBinaryDictionaryUnifier(fb.pool))
	fb.builderLabels = append(fb.builderLabels, builder.NewOptInt32Builder(arrow.PrimitiveTypes.Int32))
	fb.builderLabelFields = append(fb.builderLabelFields, field)
	fb.labelNameIndex[field.Name] = index
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
	key uint64,
	leaf bool,
	inlined bool,
) (bool, error) {
	if cr, found := fb.compareRows[key]; found {
		// If we don't group by the labels, we intersect the values before adding them to the flame graph.
		if len(fb.aggregationConfig.aggregateByLabels) == 0 {
			fb.intersectLabels(r, t, recordLabelIndex, sampleIndex, cr)
		}

		// Aggregating by timestamp usually means that we're rendering a flame chart and not graph.
		if fb.aggregationConfig.aggregateByTimestamp {
			if fb.parent.Has() && fb.lastChildMerged[fb.parent] != cr {
				return false, nil
			}
			merge, err := matchRowsByTimestamp(
				fb.builderTimestamp.Value(cr),
				fb.builderCumulative.Value(cr),
				r.Timestamp.Value(sampleIndex),
				r.Period.Value(sampleIndex),
			)
			if err != nil {
				return false, err
			}
			if !merge {
				return false, nil
			}
		}

		// Compare the existing row's metadata values with the one we're merging.
		// If these values differ we need to set the row's metadata column to null.
		{
			if !fb.builderMappingFileIndices.IsNull(cr) {
				oldIndex := int(r.MappingFileIndices.Value(locationIndex))
				if t.mappingFile.indices.IsNull(oldIndex) {
					fb.builderMappingFileIndices.SetNull(cr)
				} else {
					a := fb.builderMappingFileIndices.Value(cr)
					b := t.mappingFile.indices.Value(oldIndex)
					if a != b {
						fb.builderMappingFileIndices.SetNull(cr)
					}
				}
			}
		}
		{
			if !fb.builderMappingBuildIDIndices.IsNull(cr) {
				oldIndex := int(r.MappingBuildIDIndices.Value(locationIndex))
				if t.mappingBuildID.indices.IsNull(oldIndex) {
					fb.builderMappingBuildIDIndices.SetNull(cr)
				} else {
					a := fb.builderMappingBuildIDIndices.Value(cr)
					b := t.mappingBuildID.indices.Value(oldIndex)
					if a != b {
						fb.builderMappingBuildIDIndices.SetNull(cr)
					}
				}
			}
		}
		{
			if !fb.builderLocationAddress.IsNull(cr) {
				if r.Address.IsNull(locationIndex) {
					fb.builderLocationAddress.SetNull(cr)
				} else {
					a := fb.builderLocationAddress.Value(cr)
					b := r.Address.Value(locationIndex)
					if a != b {
						fb.builderLocationAddress.SetNull(cr)
					}
				}
			}
		}
		{
			if fb.builderInlined.IsValid(cr) && fb.builderInlined.Value(cr) != inlined {
				fb.builderInlined.SetNull(cr)
			}
		}
		{
			if fb.builderLocationLine.IsValid(cr) {
				if r.LineNumber.IsNull(lineIndex) {
					fb.builderLocationLine.SetNull(cr)
				} else {
					a := fb.builderLocationLine.Value(cr)
					b := r.LineNumber.Value(lineIndex)
					if a != b {
						fb.builderLocationLine.SetNull(cr)
					}
				}
			}
		}
		{
			if fb.builderFunctionStartLine.IsValid(cr) {
				if r.LineFunctionStartLine.IsNull(lineIndex) {
					fb.builderFunctionStartLine.SetNull(cr)
				} else {
					a := fb.builderFunctionStartLine.Value(cr)
					b := r.LineFunctionStartLine.Value(lineIndex)
					if a != b {
						fb.builderFunctionStartLine.SetNull(cr)
					}
				}
			}
		}
		{
			if !fb.builderFunctionSystemNameIndices.IsNull(cr) {
				oldIndex := int(r.LineFunctionSystemNameIndices.Value(lineIndex))
				if t.functionSystemName.indices.IsNull(oldIndex) {
					fb.builderFunctionSystemNameIndices.SetNull(cr)
				} else {
					a := fb.builderFunctionSystemNameIndices.Value(cr)
					b := t.functionSystemName.indices.Value(oldIndex)
					if a != b {
						fb.builderFunctionSystemNameIndices.SetNull(cr)
					}
				}
			}
		}
		{
			if !fb.builderFunctionFilenameIndices.IsNull(cr) {
				oldIndex := int(r.LineFunctionFilenameIndices.Value(lineIndex))
				if t.functionFilename.indices.IsNull(oldIndex) {
					fb.builderFunctionFilenameIndices.SetNull(cr)
				} else {
					a := fb.builderFunctionFilenameIndices.Value(cr)
					b := t.functionFilename.indices.Value(oldIndex)
					if a != b {
						fb.builderFunctionFilenameIndices.SetNull(cr)
					}
				}
			}
		}

		if fb.parent.Has() {
			fb.lastChildMerged[fb.parent] = cr
		}
		// All fields match, so we can aggregate this new row with the existing one.
		fb.addRowValues(r, cr, sampleIndex, leaf)
		// Continue with this row as the parent for the next iteration and compare to its children.
		fb.parent.Set(cr)
		fb.compareRows = fb.children[cr]
		return true, nil
	}
	// reset the compare rows
	// if there are no matching rows here, we don't want to merge their children either.
	fb.compareRows = nil
	return false, nil
}

// mergeUnsymbolizedRows compares the addresses only and ignores potential function names as they are not available.
func (fb *flamegraphBuilder) mergeUnsymbolizedRows(
	r *profile.RecordReader,
	t *transpositions,
	recordLabelIndex []int,
	sampleIndex, locationIndex, end int,
	key uint64,
	leaf bool,
) (bool, error) {
	if cr, found := fb.compareRows[key]; found {
		// If we don't group by the labels, we only keep those labels that are identical.
		if len(fb.aggregationConfig.aggregateByLabels) == 0 {
			fb.intersectLabels(r, t, recordLabelIndex, sampleIndex, cr)
		}

		// Aggregating by timestamp usually means that we're rendering a flame chart and not graph.
		if fb.aggregationConfig.aggregateByTimestamp {
			if fb.parent.Has() && fb.lastChildMerged[fb.parent] != cr {
				return false, nil
			}
			merge, err := matchRowsByTimestamp(
				fb.builderTimestamp.Value(cr),
				fb.builderCumulative.Value(cr),
				r.Timestamp.Value(sampleIndex),
				r.Period.Value(sampleIndex),
			)
			if err != nil {
				return false, err
			}
			if !merge {
				return false, nil
			}
		}

		value := r.Value.Value(sampleIndex)

		fb.builderCumulative.Add(cr, value)
		if leaf {
			fb.builderFlat.Add(cr, value)
		}

		fb.parent.Set(cr)
		fb.compareRows = fb.children[cr]
		return true, nil
	}
	// reset the compare rows
	// if there are no matching rows here, we don't want to merge their children either.
	fb.compareRows = nil
	return false, nil
}

func matchRowsByTimestamp(compareTimestamp, compareCumulative, timestamp, period int64) (bool, error) {
	if compareTimestamp > timestamp {
		return false, fmt.Errorf("compareTimestamp > timestamp: %d > %d", compareTimestamp, timestamp)
	}
	if compareTimestamp == timestamp {
		return false, fmt.Errorf("multiple samples for the same timestamp is not allowed: %d", timestamp)
	}

	difference := time.Duration(timestamp - (compareTimestamp + compareCumulative))
	// We truncate 10% jitter. We use duration which usually is the period.
	// For example, for 19hz sampling rate, we'll get a duration of 1000ms/19hz = 52.63ms
	// and 10% are 5.2ms jitter that gets truncated.
	jitter := time.Duration(period / 10)
	truncated := difference - jitter
	return truncated <= 0, nil
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
		transposedLabelIndex := t.labels[fieldIndex].indices.Value(int(recordLabelColumn.Col.Value(sampleIndex)))
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

type flamegraphBuilder struct {
	pool memory.Allocator

	aggregationConfig aggregationConfig

	// This keeps track of the total cumulative value so that we can set the first row's cumulative value at the end.
	cumulative int64
	// This keeps track of the total diff values so that we can set the first row's diff value at the end.
	diff int64
	// This keeps track of the max height of the flame graph.
	maxHeight int32
	// trimmed keeps track of the values that were trimmed from the flame graph.
	trimmed int64
	// parent keeps track of the parent of a row. This is used to build the children array.
	parent parent
	// This keeps track of a row's children and will be converted to an arrow array of lists at the end.
	// Allocating for an average of 8 children per stacktrace upfront.
	children        []map[uint64]int
	childrenList    [][]int
	lastChildMerged []int

	// This keeps track of the root rows indexed by the labels string.
	// If the stack trace has no labels, we use the empty string as the key.
	// This will be the root row's children, which is always our row 0 in flame graphs.
	rootsRow map[uint64]int
	// compareRows are the rows that we compare to the current location against and potentially merge.
	compareRows map[uint64]int
	// height keeps track of the current stack trace's height of the flame graph.
	height int32

	builderLabelsOnly                    *array.BooleanBuilder
	builderMappingFileIndices            *array.Int32Builder
	builderMappingFileDictUnifier        array.DictionaryUnifier
	builderMappingBuildIDIndices         *array.Int32Builder
	builderMappingBuildIDDictUnifier     array.DictionaryUnifier
	builderLocationAddress               *array.Uint64Builder
	builderLocationLine                  *builder.OptInt64Builder
	builderInlined                       *builder.OptBooleanBuilder
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
	builderParent                        *array.Int32Builder
	builderCumulative                    *builder.OptInt64Builder
	builderFlat                          *builder.OptInt64Builder
	builderDiff                          *builder.OptInt64Builder
	builderTimestamp                     *builder.OptInt64Builder
	builderDepth                         *array.Uint32Builder

	// Only at the last step when preparing the new record these are populated.
	// They are also used to create compacted dictionaries and after that replaced by them.
	mappingBuildID            *array.Dictionary
	mappingBuildIDIndices     *array.Int32
	mappingFile               *array.Dictionary
	mappingFileIndices        *array.Int32
	functionName              *array.Dictionary
	functionNameIndices       *array.Int32
	functionSystemName        *array.Dictionary
	functionSystemNameIndices *array.Int32
	functionFilename          *array.Dictionary
	functionFilenameIndices   *array.Int32
	labels                    []*array.Dictionary
	labelsIndices             []*array.Int32
	trimmedChildren           [][]int

	labelNameIndex map[string]int

	trimmedLocationLine      array.Builder
	trimmedFunctionStartLine array.Builder
	trimmedCumulative        array.Builder
	trimmedFlat              array.Builder
	trimmedDiff              array.Builder

	trimmedTimestamp *builder.OptInt64Builder
	trimmedDepth     array.Builder
	trimmedParent    *array.Int32Builder
	valueOffset      array.Builder
}

type aggregationConfig struct {
	aggregateByLabels           map[string]struct{}
	aggregateByMappingFile      bool
	aggregateByLocationAddress  bool
	aggregateByFunctionFilename bool
	aggregateByTimestamp        bool
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func newFlamegraphBuilder(
	pool memory.Allocator,
	rows int64,
	groupBy []string,
) (*flamegraphBuilder, error) {
	builderChildren := builder.NewListBuilder(pool, arrow.PrimitiveTypes.Uint32)

	fb := &flamegraphBuilder{
		pool: pool,

		parent: parent(-1),

		// ensuring that we always have space to set the first row below
		children:        make([]map[uint64]int, maxInt64(rows, 1)),
		childrenList:    make([][]int, maxInt64(rows, 1)),
		lastChildMerged: []int{-1}, // The first row is the root row, so we start with -1 to indicate no children merged yet.
		labelNameIndex:  map[string]int{},

		builderLabelsOnly:  array.NewBooleanBuilder(pool),
		builderLabelsExist: builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean),

		builderMappingFileIndices:        array.NewInt32Builder(pool),
		builderMappingFileDictUnifier:    array.NewBinaryDictionaryUnifier(pool),
		builderMappingBuildIDIndices:     array.NewInt32Builder(pool),
		builderMappingBuildIDDictUnifier: array.NewBinaryDictionaryUnifier(pool),

		builderLocationAddress: array.NewUint64Builder(pool),
		builderLocationLine:    builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),

		builderInlined:                       builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean),
		builderFunctionStartLine:             builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderFunctionNameIndices:           array.NewInt32Builder(pool),
		builderFunctionNameDictUnifier:       array.NewBinaryDictionaryUnifier(pool),
		builderFunctionSystemNameIndices:     array.NewInt32Builder(pool),
		builderFunctionSystemNameDictUnifier: array.NewBinaryDictionaryUnifier(pool),
		builderFunctionFilenameIndices:       array.NewInt32Builder(pool),
		builderFunctionFilenameDictUnifier:   array.NewBinaryDictionaryUnifier(pool),

		builderTimestamp: builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderDepth:     array.NewUint32Builder(pool),

		builderChildren:       builderChildren,
		builderChildrenValues: builderChildren.ValueBuilder().(*array.Uint32Builder),
		builderParent:         array.NewInt32Builder(pool),
		builderCumulative:     builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderFlat:           builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
		builderDiff:           builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64),
	}

	fb.aggregationConfig = aggregationConfig{aggregateByLabels: map[string]struct{}{}}

	for _, f := range groupBy {
		if strings.HasPrefix(f, FlamegraphFieldLabels) {
			fb.aggregationConfig.aggregateByLabels[f] = struct{}{}
		}
		if f == FlamegraphFieldMappingFile {
			fb.aggregationConfig.aggregateByMappingFile = true
		}
		if f == FlamegraphFieldLocationAddress {
			fb.aggregationConfig.aggregateByLocationAddress = true
		}
		if f == FlamegraphFieldFunctionFileName {
			fb.aggregationConfig.aggregateByFunctionFilename = true
		}
		if f == FlamegraphFieldTimestamp {
			fb.aggregationConfig.aggregateByTimestamp = true
		}
	}

	rootRow := map[uint64]int{}
	fb.children[0] = rootRow
	fb.rootsRow = rootRow

	// The very first row is the root row. It doesn't contain any metadata.
	// It only contains the root cumulative value and list of children (which are actual roots).
	fb.builderLabelsExist.AppendSingle(false)
	fb.builderLabelsOnly.AppendNull()
	fb.builderMappingFileIndices.AppendNull()
	fb.builderMappingBuildIDIndices.AppendNull()

	fb.builderLocationAddress.AppendNull()
	fb.builderLocationLine.AppendNull()

	fb.builderInlined.AppendSingle(false)
	fb.builderFunctionStartLine.AppendNull()
	fb.builderFunctionNameIndices.AppendNull()
	fb.builderFunctionSystemNameIndices.AppendNull()
	fb.builderFunctionFilenameIndices.AppendNull()

	// The cumulative values is calculated and at the end set to the correct value.
	fb.builderCumulative.Append(0)
	fb.builderDiff.Append(0)
	// the root will never have a flat value
	fb.builderFlat.Append(0)

	fb.builderTimestamp.Append(0)
	fb.builderDepth.Append(0)

	// The root has no parent
	fb.builderParent.Append(-1)

	return fb, nil
}

func (fb *flamegraphBuilder) prepareNewRecord() error {
	// TODO: Do we want to clean up the builders too?
	cleanupArrs := make([]compactDictionary.Releasable, 0, 10)
	defer func() {
		for _, arr := range cleanupArrs {
			arr.Release()
		}
	}()

	// We have manually tracked the total cumulative value.
	// Now we set/overwrite the cumulative value for the root row (which is always the 0 row in our flame graphs).
	// We don't care about a global flat value, therefore it's omitted here.
	fb.builderCumulative.Set(0, fb.cumulative)
	fb.builderDiff.Set(0, fb.diff)

	// We want to unify the dictionaries after having created the flame graph now.
	// They are going to be trimmed and compacted in the next step.

	mappingBuildIDIndices := fb.builderMappingBuildIDIndices.NewArray()
	cleanupArrs = append(cleanupArrs, mappingBuildIDIndices)
	mappingBuildIDDict, err := fb.builderMappingBuildIDDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return err
	}
	cleanupArrs = append(cleanupArrs, mappingBuildIDDict)
	mappingBuildIDType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.mappingBuildID = array.NewDictionaryArray(mappingBuildIDType, mappingBuildIDIndices, mappingBuildIDDict)
	fb.mappingBuildIDIndices = fb.mappingBuildID.Indices().(*array.Int32)

	mappingFileIndices := fb.builderMappingFileIndices.NewArray()
	cleanupArrs = append(cleanupArrs, mappingFileIndices)
	mappingFileDict, err := fb.builderMappingFileDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return err
	}
	cleanupArrs = append(cleanupArrs, mappingFileDict)
	mappingFileType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.mappingFile = array.NewDictionaryArray(mappingFileType, mappingFileIndices, mappingFileDict)
	fb.mappingFileIndices = fb.mappingFile.Indices().(*array.Int32)

	functionNameIndices := fb.builderFunctionNameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionNameIndices)
	functionNameDict, err := fb.builderFunctionNameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return err
	}
	cleanupArrs = append(cleanupArrs, functionNameDict)
	functionNameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.functionName = array.NewDictionaryArray(functionNameType, functionNameIndices, functionNameDict)
	fb.functionNameIndices = fb.functionName.Indices().(*array.Int32)

	functionSystemNameIndices := fb.builderFunctionSystemNameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionSystemNameIndices)
	functionSystemNameDict, err := fb.builderFunctionSystemNameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return err
	}
	cleanupArrs = append(cleanupArrs, functionSystemNameDict)
	functionSystemNameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.functionSystemName = array.NewDictionaryArray(functionSystemNameType, functionSystemNameIndices, functionSystemNameDict)
	fb.functionSystemNameIndices = fb.functionSystemName.Indices().(*array.Int32)

	functionFilenameIndices := fb.builderFunctionFilenameIndices.NewArray()
	cleanupArrs = append(cleanupArrs, functionFilenameIndices)
	functionFilenameDict, err := fb.builderFunctionFilenameDictUnifier.GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
	if err != nil {
		return err
	}
	cleanupArrs = append(cleanupArrs, functionFilenameDict)
	functionFilenameType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Int32, ValueType: arrow.BinaryTypes.Binary}
	fb.functionFilename = array.NewDictionaryArray(functionFilenameType, functionFilenameIndices, functionFilenameDict)
	fb.functionFilenameIndices = fb.functionFilename.Indices().(*array.Int32)

	fb.ensureLabelColumnsComplete()

	for i := range fb.builderLabels {
		// Ignore the label column if we don't group by it
		if _, ok := fb.aggregationConfig.aggregateByLabels[fb.builderLabelFields[i].Name]; !ok {
			continue
		}

		indices := fb.builderLabels[i].NewArray()
		cleanupArrs = append(cleanupArrs, indices)
		dict, err := fb.builderLabelsDictUnifiers[i].GetResultWithIndexType(arrow.PrimitiveTypes.Int32)
		if err != nil {
			return err
		}
		cleanupArrs = append(cleanupArrs, dict)
		typ := &arrow.DictionaryType{IndexType: indices.DataType(), ValueType: dict.DataType()}
		fb.labels = append(fb.labels, array.NewDictionaryArray(typ, indices, dict))
		fb.labelsIndices = append(fb.labelsIndices, fb.labels[i].Indices().(*array.Int32))
	}

	// If there is only one root row, we need to populate the trimmedChildren
	// to not panic when building the NewRecord. Using cumulative here as the
	// children's array is preallocated, so it doesn't reflect the true number
	// of rows in the result.
	if fb.builderCumulative.Len() == 1 {
		fb.trimmedChildren = make([][]int, 1)
	}

	return nil
}

// NewRecord returns a new record from the builders.
// It adds the children to the children column and the labels intersection to the labels column.
// Finally, it assembles all columns from the builders into an arrow record.
func (fb *flamegraphBuilder) NewRecord() (arrow.Record, error) {
	fields := []arrow.Field{
		// Location
		{Name: FlamegraphFieldLabelsOnly, Type: arrow.FixedWidthTypes.Boolean},
		{Name: FlamegraphFieldLocationAddress, Type: arrow.PrimitiveTypes.Uint64},
		{Name: FlamegraphFieldMappingFile, Type: fb.mappingFile.DataType()},
		{Name: FlamegraphFieldMappingBuildID, Type: fb.mappingBuildID.DataType()},
		// Function
		{Name: FlamegraphFieldLocationLine, Type: fb.trimmedLocationLine.Type()},
		{Name: FlamegraphFieldInlined, Type: arrow.FixedWidthTypes.Boolean, Nullable: true},
		{Name: FlamegraphFieldFunctionStartLine, Type: fb.trimmedFunctionStartLine.Type()},
		{Name: FlamegraphFieldFunctionName, Type: fb.functionName.DataType()},
		{Name: FlamegraphFieldFunctionSystemName, Type: fb.functionSystemName.DataType()},
		{Name: FlamegraphFieldFunctionFileName, Type: fb.functionFilename.DataType()},
		// Values
		{Name: FlamegraphFieldChildren, Type: arrow.ListOf(arrow.PrimitiveTypes.Uint32)},
		{Name: FlamegraphFieldParent, Type: arrow.PrimitiveTypes.Int32},
		{Name: FlamegraphFieldCumulative, Type: fb.trimmedCumulative.Type()},
		{Name: FlamegraphFieldFlat, Type: fb.trimmedFlat.Type()},
		{Name: FlamegraphFieldDiff, Type: fb.trimmedDiff.Type()},
		// Timestamp
		{Name: FlamegraphFieldTimestamp, Type: arrow.PrimitiveTypes.Int64},
		// Depth and ValueOffset are required for parallelized frontend rendering.
		{Name: FlamegraphFieldDepth, Type: fb.trimmedDepth.Type()},
		{Name: FlamegraphFieldValueOffset, Type: fb.valueOffset.Type()},
	}

	numCols := len(fields)

	cleanupArrs := make([]arrow.Array, 0, numCols+1+(2*len(fb.builderLabelFields)))
	defer func() {
		for _, arr := range cleanupArrs {
			arr.Release()
		}
	}()

	// We have manually tracked each row's children.
	// So now we need to iterate over all rows in the record and append their children.
	// We cannot do this while building the rows as we need to append the children while iterating over the rows.
	for i := 0; i < fb.trimmedCumulative.Len(); i++ {
		if len(fb.trimmedChildren[i]) == 0 {
			fb.builderChildren.AppendNull() // leaf
		} else {
			fb.builderChildren.Append(true)
			for _, child := range fb.trimmedChildren[i] {
				fb.builderChildrenValues.Append(uint32(child))
			}
		}
	}

	// This has to be here, because after calling .NewArray() on the builder,
	// the builder is reset.
	numRows := fb.trimmedCumulative.Len()

	arrays := make([]arrow.Array, numCols+len(fb.labels))
	arrays[0] = fb.builderLabelsOnly.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[0])
	arrays[1] = fb.builderLocationAddress.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[1])
	arrays[2] = fb.mappingFile
	arrays[3] = fb.mappingBuildID
	arrays[4] = fb.trimmedLocationLine.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[4])
	arrays[5] = fb.builderInlined.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[5])
	arrays[6] = fb.trimmedFunctionStartLine.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[6])
	arrays[7] = fb.functionName
	arrays[8] = fb.functionSystemName
	arrays[9] = fb.functionFilename
	arrays[10] = fb.builderChildren.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[10])

	// Handle parent array - use trimmedParent if available, otherwise use builderParent
	if fb.trimmedParent != nil {
		arrays[11] = fb.trimmedParent.NewArray()
	} else {
		arrays[11] = fb.builderParent.NewArray()
	}
	cleanupArrs = append(cleanupArrs, arrays[11])
	arrays[12] = fb.trimmedCumulative.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[12])
	arrays[13] = fb.trimmedFlat.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[13])
	arrays[14] = fb.trimmedDiff.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[14])
	arrays[15] = fb.trimmedTimestamp.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[15])
	arrays[16] = fb.trimmedDepth.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[16])
	arrays[17] = fb.valueOffset.NewArray()
	cleanupArrs = append(cleanupArrs, arrays[17])

	for i, field := range fb.builderLabelFields {
		field.Type = fb.labels[i].DataType() // overwrite for variable length uint types
		fields = append(fields, field)
		arrays[numCols+i] = fb.labels[i]
	}

	return array.NewRecord(
		arrow.NewSchema(fields, nil),
		arrays,
		int64(numRows),
	), nil
}

func (fb *flamegraphBuilder) Release() {
	fb.builderLabelsOnly.Release()
	fb.builderLabelsExist.Release()

	fb.builderMappingFileIndices.Release()
	fb.builderMappingFileDictUnifier.Release()
	fb.builderMappingBuildIDIndices.Release()
	fb.builderMappingBuildIDDictUnifier.Release()

	fb.builderLocationAddress.Release()
	fb.builderLocationLine.Release()
	fb.builderInlined.Release()

	fb.builderFunctionStartLine.Release()
	fb.builderFunctionNameIndices.Release()
	fb.builderFunctionNameDictUnifier.Release()
	fb.builderFunctionSystemNameIndices.Release()
	fb.builderFunctionSystemNameDictUnifier.Release()
	fb.builderFunctionFilenameIndices.Release()
	fb.builderFunctionFilenameDictUnifier.Release()

	fb.builderTimestamp.Release()
	fb.builderDepth.Release()

	fb.builderChildren.Release()
	fb.builderParent.Release()
	fb.builderCumulative.Release()
	fb.builderFlat.Release()
	fb.builderDiff.Release()

	if fb.trimmedLocationLine != nil {
		fb.trimmedLocationLine.Release()
	}

	if fb.trimmedFunctionStartLine != nil {
		fb.trimmedFunctionStartLine.Release()
	}

	if fb.trimmedCumulative != nil {
		fb.trimmedCumulative.Release()
	}

	if fb.trimmedFlat != nil {
		fb.trimmedFlat.Release()
	}

	if fb.trimmedDiff != nil {
		fb.trimmedDiff.Release()
	}

	if fb.trimmedDepth != nil {
		fb.trimmedDepth.Release()
	}

	if fb.trimmedParent != nil {
		fb.trimmedParent.Release()
	}

	for i := range fb.builderLabelFields {
		fb.builderLabels[i].Release()
		fb.builderLabelsDictUnifiers[i].Release()
	}

	if fb.mappingBuildID != nil {
		fb.mappingBuildID.Release()
	}

	if fb.mappingFile != nil {
		fb.mappingFile.Release()
	}

	if fb.functionName != nil {
		fb.functionName.Release()
	}

	if fb.functionSystemName != nil {
		fb.functionSystemName.Release()
	}

	if fb.functionFilename != nil {
		fb.functionFilename.Release()
	}

	for _, r := range fb.labels {
		r.Release()
	}
}

func (fb *flamegraphBuilder) appendRow(
	r *profile.RecordReader,
	t *transpositions,
	builderToRecordIndexMapping []int,
	sampleRow, locationRow, lineRow int,
	row int,
	key uint64,
	leaf bool,
	inlined bool,
) error {
	fb.height++

	fb.builderLabelsOnly.Append(false)

	if r.MappingFileIndices.IsValid(locationRow) {
		fb.builderMappingFileIndices.Append(
			t.mappingFile.indices.Value(
				int(r.MappingFileIndices.Value(locationRow)),
			),
		)
	} else {
		fb.builderMappingFileIndices.AppendNull()
	}

	if r.MappingBuildIDIndices.IsValid(locationRow) {
		fb.builderMappingBuildIDIndices.Append(
			t.mappingBuildID.indices.Value(
				int(r.MappingBuildIDIndices.Value(locationRow)),
			),
		)
	} else {
		fb.builderMappingBuildIDIndices.AppendNull()
	}

	fb.builderLocationAddress.Append(r.Address.Value(locationRow))
	fb.builderInlined.AppendSingle(inlined)

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

		if r.LineFunctionNameIndices.IsValid(lineRow) && t.functionSystemName.indices.Len() > 0 {
			fb.builderFunctionStartLine.Append(r.LineFunctionStartLine.Value(lineRow))
			fb.builderFunctionNameIndices.Append(t.functionName.indices.Value(int(r.LineFunctionNameIndices.Value(lineRow))))
			fb.builderFunctionSystemNameIndices.Append(t.functionSystemName.indices.Value(int(r.LineFunctionSystemNameIndices.Value(lineRow))))
			fb.builderFunctionFilenameIndices.Append(t.functionFilename.indices.Value(int(r.LineFunctionFilenameIndices.Value(lineRow))))
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
			if lc.Col.IsValid(sampleRow) && len(lc.Dict.Value(int(lc.Col.Value(sampleRow)))) > 0 {
				transposedIndex := t.labels[i].indices.Value(int(lc.Col.Value(sampleRow)))
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
		newChildren := make([]map[uint64]int, len(fb.children)*2)
		newChildrenList := make([][]int, len(fb.children)*2)
		copy(newChildren, fb.children)
		copy(newChildrenList, fb.childrenList)
		fb.children = newChildren
		fb.childrenList = newChildrenList
	}
	if len(fb.lastChildMerged) == row {
		newLastChildMerged := make([]int, len(fb.lastChildMerged)*2)
		copy(newLastChildMerged, fb.lastChildMerged)
		// Set all new lastChildMerged to -1, meaning that we haven't merged any children yet.
		for i := len(fb.lastChildMerged); i < len(newLastChildMerged); i++ {
			newLastChildMerged[i] = -1
		}
		fb.lastChildMerged = newLastChildMerged
	}
	// If there is a parent for this stack the parent is not -1 but the parent's row number.
	if fb.parent.Has() {
		// this is the first time we see this parent have a child, so we need to initialize the slice
		if fb.children[fb.parent.Get()] == nil {
			fb.children[fb.parent.Get()] = map[uint64]int{key: row}
			fb.childrenList[fb.parent.Get()] = []int{row}
		} else {
			// otherwise we can just append this row's number to the parent's slice
			fb.children[fb.parent.Get()][key] = row
			fb.childrenList[fb.parent.Get()] = append(fb.childrenList[fb.parent.Get()], row)
		}

		fb.lastChildMerged[fb.parent.Get()] = row
	}

	fb.builderTimestamp.Append(r.Timestamp.Value(sampleRow))
	fb.builderDepth.Append(uint32(fb.height))
	fb.builderParent.Append(int32(fb.parent.Get()))

	value := r.Value.Value(sampleRow)

	fb.builderCumulative.Append(value)

	if leaf { // leaf
		fb.builderFlat.Append(value)
	} else {
		// It's possible that these get merged later on and we don't want to set these to null
		fb.builderFlat.Append(0)
	}

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
	labelHash uint64,
	children map[uint64]int,
	leaf bool,
) error {
	// TODO Check and not merge when the timestamps are apart
	labelsExist := false
	for i, labelColumn := range fb.builderLabels {
		if recordIndex := builderToRecordIndexMapping[i]; recordIndex != -1 {
			lc := r.LabelColumns[recordIndex]
			if lc.Col.IsValid(sampleRow) && len(lc.Dict.Value(int(lc.Col.Value(sampleRow)))) > 0 {
				transposedIndex := t.labels[i].indices.Value(int(lc.Col.Value(sampleRow)))
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
		newChildren := make([]map[uint64]int, len(fb.children)*2)
		newChildrenList := make([][]int, len(fb.children)*2)
		copy(newChildren, fb.children)
		copy(newChildrenList, fb.childrenList)
		fb.children = newChildren
		fb.childrenList = newChildrenList
	}
	if len(fb.lastChildMerged) == row {
		newLastChildMerged := make([]int, len(fb.lastChildMerged)*2)
		copy(newLastChildMerged, fb.lastChildMerged)
		// Set all new lastChildMerged to -1, meaning that we haven't merged any children yet.
		for i := len(fb.lastChildMerged); i < len(newLastChildMerged); i++ {
			newLastChildMerged[i] = -1
		}
		fb.lastChildMerged = newLastChildMerged
	}
	fb.rootsRow[labelHash] = row
	fb.childrenList[0] = append(fb.childrenList[0], row)
	fb.children[row] = children
	fb.lastChildMerged[0] = row

	fb.builderLabelsOnly.Append(true)
	fb.builderMappingFileIndices.AppendNull()
	fb.builderMappingBuildIDIndices.AppendNull()
	fb.builderLocationAddress.AppendNull()
	fb.builderInlined.AppendNull()
	fb.builderLocationLine.AppendNull()
	fb.builderFunctionStartLine.AppendNull()
	fb.builderFunctionNameIndices.AppendNull()
	fb.builderFunctionSystemNameIndices.AppendNull()
	fb.builderFunctionFilenameIndices.AppendNull()

	fb.builderTimestamp.Append(0)
	fb.builderDepth.Append(1) // Label rows are direct children of root, so depth is 1

	// Label rows are always children of root (row 0)
	fb.builderParent.Append(0)

	// Append both cumulative and diff values and overwrite them below.
	fb.builderCumulative.Append(0)
	fb.builderDiff.Append(0)
	fb.builderFlat.Append(0)
	fb.addRowValues(r, row, sampleRow, leaf)

	return nil
}

// addRowValues updates the existing row's values and potentially adding existing values on top.
func (fb *flamegraphBuilder) addRowValues(r *profile.RecordReader, row, sampleRow int, leaf bool) {
	value := r.Value.Value(sampleRow)

	fb.builderCumulative.Add(row, value)
	fb.builderDiff.Add(row, r.Diff.Value(sampleRow))

	if leaf {
		fb.builderFlat.Add(row, value)
	}
}

func (fb *flamegraphBuilder) trim(ctx context.Context, tracer trace.Tracer, threshold float32) error {
	_, span := tracer.Start(ctx, "trim")
	defer span.End()

	releasers := make([]compactDictionary.Releasable, 0, 10+2*len(fb.labels))
	defer func() {
		for _, r := range releasers {
			r.Release()
		}
	}()

	// initialize the queue with the root rows' children. It usually has the most amount of children.
	trimmingQueue := queue{elements: make([]trimmingElement, 0, len(fb.children[0]))}
	trimmingQueue.push(trimmingElement{row: 0})

	row := -1
	largestLocationLine := uint64(0)
	largestFunctionStartLine := uint64(0)
	largestCumulativeValue := uint64(0)
	largestFlatValue := uint64(0)
	largestDiffValue := int64(0)
	smallestDiffValue := int64(0)
	largestDepth := uint64(0)
	for trimmingQueue.len() > 0 {
		// pop the first item from the queue
		te := trimmingQueue.pop()
		row++

		// The following two will never be null.
		locationLine := uint64(fb.builderLocationLine.Value(te.row))
		if locationLine > largestLocationLine {
			largestLocationLine = locationLine
		}
		functionStartLine := uint64(fb.builderFunctionStartLine.Value(te.row))
		if functionStartLine > largestFunctionStartLine {
			largestFunctionStartLine = functionStartLine
		}
		cum := uint64(fb.builderCumulative.Value(te.row))

		largestCumulativeValue = max(largestCumulativeValue, cum)
		largestFlatValue = max(largestFlatValue, uint64(fb.builderFlat.Value(te.row)))
		diff := fb.builderDiff.Value(te.row)
		largestDiffValue = max(largestDiffValue, diff)
		smallestDiffValue = max(smallestDiffValue, diff)
		largestDepth = max(largestDepth, uint64(fb.builderDepth.Value(te.row)))

		cumThreshold := float32(cum) * threshold

		for _, cr := range fb.childrenList[te.row] {
			if v := fb.builderCumulative.Value(cr); v > int64(cumThreshold) {
				// this row is above the threshold, so we need to keep it
				// add this row to the queue to check its children.
				trimmingQueue.push(trimmingElement{row: cr, parent: row})
			}
		}
	}

	trimmedLabelsOnly := array.NewBooleanBuilder(fb.pool)
	trimmedLabelsExist := builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean)
	trimmedMappingFileIndices := array.NewInt32Builder(fb.pool)
	trimmedMappingBuildIDIndices := array.NewInt32Builder(fb.pool)
	trimmedLocationAddress := array.NewUint64Builder(fb.pool)
	trimmedLocationLineType := smallestUnsignedTypeFor(largestLocationLine)
	trimmedLocationLine := array.NewBuilder(fb.pool, trimmedLocationLineType)
	trimmedInlined := builder.NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean)
	trimmedFunctionStartLineType := smallestUnsignedTypeFor(largestFunctionStartLine)
	trimmedFunctionStartLine := array.NewBuilder(fb.pool, trimmedFunctionStartLineType)
	trimmedFunctionNameIndices := array.NewInt32Builder(fb.pool)
	trimmedFunctionSystemNameIndices := array.NewInt32Builder(fb.pool)
	trimmedFunctionFilenameIndices := array.NewInt32Builder(fb.pool)
	trimmedCumulativeType := smallestUnsignedTypeFor(largestCumulativeValue)
	trimmedCumulative := array.NewBuilder(fb.pool, trimmedCumulativeType)
	trimmedFlatType := smallestUnsignedTypeFor(largestFlatValue)
	trimmedFlat := array.NewBuilder(fb.pool, trimmedFlatType)
	trimmedDiffType := smallestSignedTypeFor(smallestDiffValue, largestDiffValue)
	trimmedDiff := array.NewBuilder(fb.pool, trimmedDiffType)

	trimmedTimestamps := builder.NewOptInt64Builder(arrow.PrimitiveTypes.Int64)
	trimmedDepthType := smallestUnsignedTypeFor(largestDepth)
	trimmedDepth := array.NewBuilder(fb.pool, trimmedDepthType)

	// Parent indices use int32 to allow for -1 (null parent)
	trimmedParent := array.NewInt32Builder(fb.pool)

	valueOffset := array.NewBuilder(fb.pool, trimmedCumulativeType)

	releasers = append(releasers,
		trimmedMappingFileIndices,
		trimmedMappingBuildIDIndices,
		trimmedFunctionNameIndices,
		trimmedFunctionSystemNameIndices,
		trimmedFunctionFilenameIndices,
	)

	var trimmedLabelsIndices []*array.Int32Builder
	for range fb.labels {
		ib := array.NewInt32Builder(fb.pool)
		trimmedLabelsIndices = append(trimmedLabelsIndices, ib)
		releasers = append(releasers, ib)
	}

	trimmedChildren := make([][]int, len(fb.children))

	trimmedLabelsOnly.Reserve(row)
	trimmedLabelsExist.Reserve(row)
	trimmedMappingFileIndices.Reserve(row)
	trimmedMappingBuildIDIndices.Reserve(row)
	trimmedLocationAddress.Reserve(row)
	trimmedLocationLine.Reserve(row)
	trimmedInlined.Reserve(row)
	trimmedFunctionStartLine.Reserve(row)
	trimmedFunctionNameIndices.Reserve(row)
	trimmedFunctionSystemNameIndices.Reserve(row)
	trimmedFunctionFilenameIndices.Reserve(row)
	trimmedCumulative.Reserve(row)
	trimmedFlat.Reserve(row)
	trimmedDiff.Reserve(row)
	trimmedTimestamps.Reserve(row)
	trimmedDepth.Reserve(row)
	trimmedParent.Reserve(row)
	valueOffset.Reserve(row)

	for _, l := range trimmedLabelsIndices {
		l.Reserve(row)
	}

	trimmingQueue.elements = trimmingQueue.elements[:0]
	trimmingQueue.push(trimmingElement{row: 0})

	// keep processing new elements until the queue is empty
	for trimmingQueue.len() > 0 {
		// pop the first item from the queue
		te := trimmingQueue.pop()

		copyBoolBuilderValue(fb.builderLabelsOnly, trimmedLabelsOnly, te.row)
		copyOptBooleanBuilderValue(fb.builderLabelsExist, trimmedLabelsExist, te.row)
		appendDictionaryIndexInt32(fb.mappingFileIndices, trimmedMappingFileIndices, te.row)
		appendDictionaryIndexInt32(fb.mappingBuildIDIndices, trimmedMappingBuildIDIndices, te.row)
		copyUint64BuilderValue(fb.builderLocationAddress, trimmedLocationAddress, te.row)
		copyOptBooleanBuilderValue(fb.builderInlined, trimmedInlined, te.row)
		copyInt64BuilderValueToUnknownUnsigned(fb.builderLocationLine, trimmedLocationLine, te.row)
		copyInt64BuilderValueToUnknownUnsigned(fb.builderFunctionStartLine, trimmedFunctionStartLine, te.row)
		appendDictionaryIndexInt32(fb.functionNameIndices, trimmedFunctionNameIndices, te.row)
		appendDictionaryIndexInt32(fb.functionSystemNameIndices, trimmedFunctionSystemNameIndices, te.row)
		appendDictionaryIndexInt32(fb.functionFilenameIndices, trimmedFunctionFilenameIndices, te.row)
		copyOptInt64BuilderValue(fb.builderTimestamp, trimmedTimestamps, te.row)
		for i := range fb.labels {
			appendDictionaryIndexInt32(fb.labelsIndices[i], trimmedLabelsIndices[i], te.row)
		}

		// The following two will never be null.
		cum := fb.builderCumulative.Value(te.row)
		switch b := trimmedCumulative.(type) {
		case *array.Uint64Builder:
			b.Append(uint64(cum))
		case *array.Uint32Builder:
			b.Append(uint32(cum))
		case *array.Uint16Builder:
			b.Append(uint16(cum))
		case *array.Uint8Builder:
			b.Append(uint8(cum))
		default:
			panic(fmt.Errorf("unsupported type %T", b))
		}

		switch b := trimmedFlat.(type) {
		case *array.Uint64Builder:
			b.Append(uint64(fb.builderFlat.Value(te.row)))
		case *array.Uint32Builder:
			b.Append(uint32(fb.builderFlat.Value(te.row)))
		case *array.Uint16Builder:
			b.Append(uint16(fb.builderFlat.Value(te.row)))
		case *array.Uint8Builder:
			b.Append(uint8(fb.builderFlat.Value(te.row)))
		default:
			panic(fmt.Errorf("unsupported type %T", b))
		}

		switch b := trimmedDiff.(type) {
		case *array.Int64Builder:
			b.Append(fb.builderDiff.Value(te.row))
		case *array.Int32Builder:
			b.Append(int32(fb.builderDiff.Value(te.row)))
		case *array.Int16Builder:
			b.Append(int16(fb.builderDiff.Value(te.row)))
		case *array.Int8Builder:
			b.Append(int8(fb.builderDiff.Value(te.row)))
		default:
			panic(fmt.Errorf("unsupported type %T", b))
		}

		switch b := trimmedDepth.(type) {
		case *array.Uint64Builder:
			b.Append(uint64(fb.builderDepth.Value(te.row)))
		case *array.Uint32Builder:
			b.Append(fb.builderDepth.Value(te.row))
		case *array.Uint16Builder:
			b.Append(uint16(fb.builderDepth.Value(te.row)))
		case *array.Uint8Builder:
			b.Append(uint8(fb.builderDepth.Value(te.row)))
		default:
			panic(fmt.Errorf("unsupported type %T", b))
		}

		// This gets the newly inserted row's index.
		// It is used further down as the children's parent value when added to the trimmingQueue.
		row := trimmedCumulative.Len() - 1

		// When trimming, te.parent is the parent index in the trimmed structure
		// For the root row (te.row == 0), parent should be -1
		if row == 0 {
			trimmedParent.Append(-1)
		} else {
			trimmedParent.Append(int32(te.parent))
		}

		switch b := valueOffset.(type) {
		case *array.Uint64Builder:
			b.Append(uint64(te.valueOffset))
		case *array.Uint32Builder:
			b.Append(uint32(te.valueOffset))
		case *array.Uint16Builder:
			b.Append(uint16(te.valueOffset))
		case *array.Uint8Builder:
			b.Append(uint8(te.valueOffset))
		default:
			panic(fmt.Errorf("unsupported type %T", b))
		}

		// Add this new row as child to its parent if not the root row (index 0).
		if row != 0 {
			if len(trimmedChildren[te.parent]) == 0 {
				trimmedChildren[te.parent] = []int{row}
			} else {
				trimmedChildren[te.parent] = append(trimmedChildren[te.parent], row)
			}
		}

		cumThreshold := float32(cum) * threshold

		valueOffset := te.valueOffset
		for _, cr := range fb.childrenList[te.row] {
			if v := fb.builderCumulative.Value(cr); v > int64(cumThreshold) {
				// this row is above the threshold, so we need to keep it
				// add this row to the queue to check its children.
				trimmingQueue.push(trimmingElement{
					row:         cr,
					parent:      row,
					valueOffset: valueOffset,
				})
				valueOffset += fb.builderCumulative.Value(cr)
			} else {
				// this row is below the threshold, so we need to trim it
				fb.trimmed += v
			}
		}
	}

	// Next we just keep the values in the dictionaries that we need after trimming.
	var err error
	trimmedMappingBuildIDIndicesArray := trimmedMappingBuildIDIndices.NewArray()
	releasers = append(releasers, trimmedMappingBuildIDIndicesArray)

	mbid, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
		fb.mappingBuildID.DataType(),
		trimmedMappingBuildIDIndicesArray,
		fb.mappingBuildID.Dictionary(),
	))
	if err != nil {
		return err
	}
	release(fb.mappingBuildID)
	fb.mappingBuildID = mbid

	trimmedMappingFileIndicesArray := trimmedMappingFileIndices.NewArray()
	releasers = append(releasers, trimmedMappingFileIndicesArray)
	mf, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
		fb.mappingFile.DataType(),
		trimmedMappingFileIndicesArray,
		fb.mappingFile.Dictionary(),
	))
	if err != nil {
		return err
	}
	release(fb.mappingFile)
	fb.mappingFile = mf

	trimmedFunctionNameIndicesArray := trimmedFunctionNameIndices.NewArray()
	releasers = append(releasers, trimmedFunctionNameIndicesArray)
	fn, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
		fb.functionName.DataType(),
		trimmedFunctionNameIndicesArray,
		fb.functionName.Dictionary(),
	))
	if err != nil {
		return err
	}
	release(fb.functionName)
	fb.functionName = fn

	trimmedFunctionSystemNameIndicesArray := trimmedFunctionSystemNameIndices.NewArray()
	releasers = append(releasers, trimmedFunctionSystemNameIndicesArray)
	sn, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
		fb.functionSystemName.DataType(),
		trimmedFunctionSystemNameIndicesArray,
		fb.functionSystemName.Dictionary(),
	))
	if err != nil {
		return err
	}
	release(fb.functionSystemName)
	fb.functionSystemName = sn

	trimmedFunctionFilenameIndicesArray := trimmedFunctionFilenameIndices.NewArray()
	releasers = append(releasers, trimmedFunctionFilenameIndicesArray)
	ffn, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
		fb.functionFilename.DataType(),
		trimmedFunctionFilenameIndicesArray,
		fb.functionFilename.Dictionary(),
	))
	if err != nil {
		return err
	}
	release(fb.functionFilename)
	fb.functionFilename = ffn

	trimmedLabels := make([]*array.Dictionary, 0, len(fb.labels))
	for i, index := range trimmedLabelsIndices {
		trimmedIndexArray := index.NewArray()
		releasers = append(releasers, trimmedIndexArray)
		tl, err := compactDictionary.CompactDictionary(fb.pool, array.NewDictionaryArray(
			&arrow.DictionaryType{IndexType: trimmedIndexArray.DataType(), ValueType: fb.labels[i].Dictionary().DataType()},
			trimmedIndexArray,
			fb.labels[i].Dictionary(),
		))
		if err != nil {
			return err
		}
		trimmedLabels = append(trimmedLabels, tl)
	}
	for _, r := range fb.labels {
		r.Release()
	}
	fb.labels = trimmedLabels

	release(
		fb.builderLabelsOnly,
		fb.builderLabelsExist,
		fb.builderLocationAddress,
		fb.builderInlined,
		fb.builderLocationLine,
		fb.builderFunctionStartLine,
		fb.builderCumulative,
		fb.builderDiff,
		fb.builderLocationLine,
		fb.builderFunctionStartLine,
	)
	fb.builderLabelsOnly = trimmedLabelsOnly
	fb.builderLabelsExist = trimmedLabelsExist
	fb.builderLocationAddress = trimmedLocationAddress
	fb.builderInlined = trimmedInlined
	fb.trimmedLocationLine = trimmedLocationLine
	fb.trimmedFunctionStartLine = trimmedFunctionStartLine
	fb.trimmedCumulative = trimmedCumulative
	fb.trimmedFlat = trimmedFlat
	fb.trimmedDiff = trimmedDiff
	fb.trimmedChildren = trimmedChildren
	fb.trimmedTimestamp = trimmedTimestamps
	fb.trimmedDepth = trimmedDepth
	fb.trimmedParent = trimmedParent
	fb.valueOffset = valueOffset

	return nil
}

func smallestUnsignedTypeFor(largestValue uint64) arrow.DataType {
	if largestValue < stdmath.MaxUint8 {
		return arrow.PrimitiveTypes.Uint8
	} else if largestValue < stdmath.MaxUint16 {
		return arrow.PrimitiveTypes.Uint16
	} else if largestValue < stdmath.MaxUint32 {
		return arrow.PrimitiveTypes.Uint32
	} else {
		return arrow.PrimitiveTypes.Uint64
	}
}

func smallestSignedTypeFor(min, max int64) arrow.DataType {
	if max < stdmath.MaxInt8 && min > stdmath.MinInt8 {
		return arrow.PrimitiveTypes.Int8
	} else if max < stdmath.MaxInt16 && min > stdmath.MinInt16 {
		return arrow.PrimitiveTypes.Int16
	} else if max < stdmath.MaxInt32 && min > stdmath.MinInt32 {
		return arrow.PrimitiveTypes.Int32
	} else {
		return arrow.PrimitiveTypes.Int64
	}
}

func copyInt64BuilderValueToUnknownUnsigned(old *builder.OptInt64Builder, new array.Builder, row int) {
	if old.IsNull(row) {
		new.AppendNull()
		return
	}
	switch b := new.(type) {
	case *array.Uint8Builder:
		b.Append(uint8(old.Value(row)))
	case *array.Uint16Builder:
		b.Append(uint16(old.Value(row)))
	case *array.Uint32Builder:
		b.Append(uint32(old.Value(row)))
	case *array.Uint64Builder:
		b.Append(uint64(old.Value(row)))
	default:
		panic(fmt.Errorf("unknown builder type %T", new))
	}
}

func copyUint64BuilderValue(old, new *array.Uint64Builder, row int) {
	if old.IsNull(row) {
		new.AppendNull()
		return
	}
	new.Append(old.Value(row))
}

func copyOptInt64BuilderValue(old, new *builder.OptInt64Builder, row int) {
	if old.IsNull(row) {
		new.AppendNull()
		return
	}
	new.Append(old.Value(row))
}

func copyOptBooleanBuilderValue(old, new *builder.OptBooleanBuilder, row int) {
	if old.IsNull(row) {
		new.AppendNull()
		return
	}
	new.AppendSingle(old.Value(row))
}

func copyBoolBuilderValue(old, new *array.BooleanBuilder, row int) {
	if old.IsNull(row) {
		new.AppendNull()
		return
	}
	new.Append(old.Value(row))
}

func appendDictionaryIndexInt32(dict *array.Int32, index *array.Int32Builder, row int) {
	if dict.IsNull(row) {
		index.AppendNull()
		return
	}
	index.Append(dict.Value(row))
}

func isLocationRoot(beg, end, i int64, list *array.List) bool {
	for j := end - 1; j >= beg; j-- {
		if !list.ListValues().IsNull(int(j)) {
			return j == i
		}
	}
	return false
}

// parent stores the parent's row number of a stack.
type parent int

func (p *parent) Set(i int) { *p = parent(i) }

func (p *parent) Reset() { *p = -1 }

func (p *parent) Get() int { return int(*p) }

func (p *parent) Has() bool { return *p > -1 }

type trimmingElement struct {
	row         int
	parent      int
	valueOffset int64
}

// queue is a small wrapper around a []trimmingElement used as queue.
type queue struct{ elements []trimmingElement }

func (q *queue) len() int { return len(q.elements) }

func (q *queue) push(i trimmingElement) { q.elements = append(q.elements, i) }

// pops the first element from the queue.
func (q *queue) pop() trimmingElement {
	v := q.elements[0]
	q.elements = q.elements[1:]
	return v
}

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

func release(releasers ...compactDictionary.Releasable) {
	for _, r := range releasers {
		if r != nil {
			r.Release()
		}
	}
}

func recordStats(r arrow.Record) string {
	var totalBytes int
	type fieldStat struct {
		valueBytes  int
		indexBytes  int
		bitmapBytes int
		countValues int
		countIndex  int
	}
	fieldStats := make([]fieldStat, r.NumCols())

	if r.NumRows() == 0 {
		b := &strings.Builder{}
		_, _ = fmt.Fprintf(b, "Cols: %d\n", r.NumCols())
		_, _ = fmt.Fprintf(b, "Rows: %d\n", r.NumRows())
		return b.String()
	}

	fields := r.Schema().Fields()
	for i, f := range fields {
		switch f.Type.(type) {
		case *arrow.BooleanType, *arrow.Int64Type, *arrow.Uint64Type, *arrow.Int32Type, *arrow.Uint32Type, *arrow.Int16Type, *arrow.Uint16Type, *arrow.Uint8Type, *arrow.Int8Type:
			data := r.Column(i).Data()
			fieldStats[i].countValues = data.Len()
			totalBytes += data.Len()
			bufs := data.Buffers()
			for j, buf := range bufs {
				if j == 0 {
					fieldStats[i].bitmapBytes += buf.Len()
					totalBytes += buf.Len()
					continue
				}
				fieldStats[i].valueBytes += buf.Len()
				totalBytes += buf.Len()
			}
		case *arrow.DictionaryType:
			data := r.Column(i).Data()
			fieldStats[i].countIndex = data.Len()
			totalBytes += data.Len()
			for j, buf := range data.Buffers() {
				if buf == nil {
					continue
				}
				if j == 0 {
					fieldStats[i].bitmapBytes += buf.Len()
					totalBytes += buf.Len()
					continue
				}
				fieldStats[i].indexBytes += buf.Len()
				totalBytes += buf.Len()
			}
			dict := r.Column(i).Data().Dictionary()
			fieldStats[i].countValues += dict.Len()
			totalBytes += dict.Len()
			for j, buf := range dict.Buffers() {
				if buf == nil {
					continue
				}
				if j == 0 {
					fieldStats[i].bitmapBytes += buf.Len()
					totalBytes += buf.Len()
					continue
				}
				fieldStats[i].valueBytes += buf.Len()
				totalBytes += buf.Len()
			}
		case *arrow.ListType:
			data := r.Column(i).Data()
			fieldStats[i].countIndex = data.Len()
			totalBytes += data.Len()
			for j, buf := range data.Buffers() {
				if j == 0 {
					fieldStats[i].bitmapBytes += buf.Len()
					totalBytes += buf.Len()
					continue
				}
				fieldStats[i].indexBytes += buf.Len()
				totalBytes += buf.Len()
			}
			for _, child := range data.Children() {
				fieldStats[i].countValues += child.Len()
				totalBytes += child.Len()
				for j, buf := range child.Buffers() {
					if j == 0 {
						fieldStats[i].bitmapBytes += buf.Len()
						totalBytes += buf.Len()
						continue
					}
					fieldStats[i].valueBytes += buf.Len()
					totalBytes += buf.Len()
				}
			}
		}
	}

	b := &strings.Builder{}
	table := tablewriter.NewWriter(b)
	table.SetAutoWrapText(false)
	table.SetColumnAlignment([]int{
		tablewriter.ALIGN_DEFAULT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_RIGHT,
		tablewriter.ALIGN_DEFAULT,
	})
	table.SetHeader([]string{
		"Name",
		"Bytes",
		"Bitmap Bytes",
		"Bytes Percent",
		"Count",
		"Type",
	})

	for i, s := range fieldStats {
		size := strconv.Itoa(s.valueBytes)
		if s.indexBytes > 0 {
			size = size + ", " + strconv.Itoa(s.indexBytes)
		}
		bytesPercent := fmt.Sprintf("%.2f%%",
			(100*float64(s.valueBytes+s.indexBytes+s.bitmapBytes))/float64(totalBytes),
		)
		count := strconv.Itoa(s.countValues)
		if s.countIndex > 0 {
			count = count + ", " + strconv.Itoa(s.countIndex)
		}

		table.Append([]string{
			fields[i].Name,
			size,
			strconv.Itoa(s.bitmapBytes),
			bytesPercent,
			count,
			fields[i].Type.String(),
		})
	}

	_, _ = fmt.Fprintf(b, "Bytes: %d\n", totalBytes)
	_, _ = fmt.Fprintf(b, "Cols: %d\n", r.NumCols())
	_, _ = fmt.Fprintf(b, "Rows: %d\n", r.NumRows())
	table.Render()

	return b.String()
}
