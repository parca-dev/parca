// Copyright 2024-2025 The Parca Authors
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

package normalizer

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/compute"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow/arrowutils"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/parca-dev/parca/pkg/profile"
)

const (
	MetadataSchemaVersion = "parca_write_schema_version"
)

const (
	MetadataSchemaVersionV1 = "v1"
)

func NewMetrics(reg prometheus.Registerer) *Metrics {
	return &Metrics{
		IncompleteLocations: promauto.With(reg).NewCounter(prometheus.CounterOpts{
			Name: "parca_normalizer_incomplete_locations_total",
			Help: "The total number of incomplete locations in the profile",
		}),
	}
}

type Metrics struct {
	IncompleteLocations prometheus.Counter
}

type arrowToInternalConverter struct {
	metrics *Metrics

	mem    memory.Allocator
	schema *dynparquet.Schema

	b *InternalRecordBuilderV1
}

func NewArrowToInternalConverter(
	mem memory.Allocator,
	schema *dynparquet.Schema,
	metrics *Metrics,
) *arrowToInternalConverter {
	return &arrowToInternalConverter{
		metrics: metrics,

		mem:    mem,
		schema: schema,

		b: &InternalRecordBuilderV1{},
	}
}

func (c *arrowToInternalConverter) Release() {
	c.b.Release()
}

func (c *arrowToInternalConverter) Validate() error {
	return c.b.validate()
}

func (c *arrowToInternalConverter) HasUnknownStacktraceIDs() (bool, error) {
	if c.b.StacktraceIDs == nil {
		return false, errors.New("missing stacktrace IDs column")
	}

	// We're not yet caching known stacktrace IDs.
	return true, nil
}

func (c *arrowToInternalConverter) UnknownStacktraceIDsRecord() (arrow.Record, error) {
	m := arrow.NewMetadata(
		[]string{MetadataSchemaVersion},
		[]string{MetadataSchemaVersionV1},
	)

	arr, ok := c.b.StacktraceIDs.Dictionary().(*array.Binary)
	if !ok {
		return nil, fmt.Errorf("expected stacktrace IDs column to be of type Binary, got %T", c.b.StacktraceIDs.Dictionary())
	}

	return array.NewRecord(
		arrow.NewSchema([]arrow.Field{{
			Name: "stacktrace_id",
			Type: arrow.BinaryTypes.Binary,
		}}, &m),
		[]arrow.Array{arr},
		int64(arr.Len()),
	), nil
}

func (c *arrowToInternalConverter) AddLocationsRecord(
	ctx context.Context,
	rec arrow.Record,
) error {
	value, ok := rec.Schema().Metadata().GetValue(MetadataSchemaVersion)
	if !ok {
		return fmt.Errorf("missing schema version in metadata")
	}

	switch value {
	case MetadataSchemaVersionV1:
		return c.AddLocationsRecordV1(ctx, rec)
	default:
		return fmt.Errorf("unsupported schema version %q", value)
	}
}

func (c *arrowToInternalConverter) AddSampleRecord(
	ctx context.Context,
	rec arrow.Record,
) error {
	value, ok := rec.Schema().Metadata().GetValue(MetadataSchemaVersion)
	if !ok {
		return fmt.Errorf("missing schema version in metadata")
	}

	switch value {
	case MetadataSchemaVersionV1:
		return c.AddSampleRecordV1(ctx, rec)
	default:
		return fmt.Errorf("unsupported schema version %q", value)
	}
}

func getBinaryDict(arr arrow.Array, fieldName string) (*array.Dictionary, *array.Binary, error) {
	dict, ok := arr.(*array.Dictionary)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be of type Dictionary, got %T", fieldName, arr)
	}

	binDict, ok := dict.Dictionary().(*array.Binary)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be a Dictionary with Values of type Binary, got %T", fieldName, dict.Dictionary())
	}

	return dict, binDict, nil
}

func getREEUint64(arr arrow.Array, fieldName string) (*array.RunEndEncoded, *array.Uint64, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}

	uint64Arr, ok := ree.Values().(*array.Uint64)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncoded with Uint64 Values, got %T", fieldName, arr)
	}

	return ree, uint64Arr, nil
}

func getREEBinaryDict(arr arrow.Array, fieldName string) (*array.RunEndEncoded, *array.Dictionary, *array.Binary, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}

	dict, ok := ree.Values().(*array.Dictionary)
	if !ok {
		return nil, nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncedod with Dictionary Values, got %T", fieldName, arr)
	}

	binDict, ok := dict.Dictionary().(*array.Binary)
	if !ok {
		return nil, nil, nil, fmt.Errorf("expected column %q to be a RunEndEncoded with Dictionary Values of type Binary, got %T", fieldName, dict.Dictionary())
	}

	return ree, dict, binDict, nil
}

func expandREEBinaryDict(mem memory.Allocator, arr arrow.Array, fieldName string) (*array.Dictionary, *array.Binary, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}

	dict, ok := ree.Values().(*array.Dictionary)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be of type RunEndEncedod with Dictionary Values, got %T", fieldName, arr)
	}

	binDict, ok := dict.Dictionary().(*array.Binary)
	if !ok {
		return nil, nil, fmt.Errorf("expected column %q to be a RunEndEncoded with Dictionary Values of type Binary, got %T", fieldName, dict.Dictionary())
	}

	b := array.NewBuilder(mem, dict.DataType()).(*array.BinaryDictionaryBuilder)
	defer b.Release()

	runEnds := ree.RunEndsArr().(*array.Int32)
	prevEnd := int32(0)
	for i := 0; i < runEnds.Len(); i++ {
		for j := prevEnd; j < runEnds.Value(i); j++ {
			if dict.IsNull(int(i)) {
				b.AppendNull()
				continue
			}
			v := binDict.Value(dict.GetValueIndex(i))
			if len(v) == 0 {
				b.AppendNull()
				continue
			}

			if err := b.Append(v); err != nil {
				return nil, nil, fmt.Errorf("append value: %w", err)
			}
		}
		prevEnd = runEnds.Value(i)
	}

	res := b.NewArray().(*array.Dictionary)
	resDict := res.Dictionary().(*array.Binary)
	return res, resDict, nil
}

func expandREEInt64(mem memory.Allocator, arr arrow.Array, fieldName string) (*array.Int64, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}

	int64Arr, ok := ree.Values().(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type RunEndEncoded with Int64 Values, got %T", fieldName, arr)
	}

	b := array.NewBuilder(mem, int64Arr.DataType()).(*array.Int64Builder)
	defer b.Release()

	runEnds := ree.RunEndsArr().(*array.Int32)
	prevEnd := int32(0)
	for i := 0; i < runEnds.Len(); i++ {
		for j := prevEnd; j < runEnds.Value(i); j++ {
			b.Append(int64Arr.Value(i))
		}
		prevEnd = runEnds.Value(i)
	}

	return b.NewArray().(*array.Int64), nil
}

type InternalRecordBuilderV1 struct {
	Producer   *array.Dictionary
	SampleType *array.Dictionary
	SampleUnit *array.Dictionary
	PeriodType *array.Dictionary
	PeriodUnit *array.Dictionary

	Period    *array.Int64
	Duration  *array.Int64
	Timestamp *array.Int64
	TimeNanos *array.Int64
	Value     *array.Int64

	Labels      []arrow.Array
	LabelFields []arrow.Field

	Locations *array.List

	StacktraceIDs *array.Dictionary
}

func (r *InternalRecordBuilderV1) Release() {
	if r.Producer != nil {
		r.Producer.Release()
	}
	if r.SampleType != nil {
		r.SampleType.Release()
	}
	if r.SampleUnit != nil {
		r.SampleUnit.Release()
	}
	if r.PeriodType != nil {
		r.PeriodType.Release()
	}
	if r.PeriodUnit != nil {
		r.PeriodUnit.Release()
	}
	if r.Period != nil {
		r.Period.Release()
	}
	if r.Duration != nil {
		r.Duration.Release()
	}
	if r.Timestamp != nil {
		r.Timestamp.Release()
	}
	if r.TimeNanos != nil {
		r.TimeNanos.Release()
	}
	if r.Value != nil {
		r.Value.Release()
	}

	for _, l := range r.Labels {
		l.Release()
	}

	if r.Locations != nil {
		r.Locations.Release()
	}

	if r.StacktraceIDs != nil {
		r.StacktraceIDs.Release()
	}
}

func (r *InternalRecordBuilderV1) validate() error {
	if r.Producer == nil {
		return fmt.Errorf("missing column %q", "producer")
	}
	if r.SampleType == nil {
		return fmt.Errorf("missing column %q", profile.ColumnSampleType)
	}
	if r.SampleUnit == nil {
		return fmt.Errorf("missing column %q", profile.ColumnSampleUnit)
	}
	if r.PeriodType == nil {
		return fmt.Errorf("missing column %q", profile.ColumnPeriodType)
	}
	if r.PeriodUnit == nil {
		return fmt.Errorf("missing column %q", profile.ColumnPeriodUnit)
	}
	if r.Period == nil {
		return fmt.Errorf("missing column %q", profile.ColumnPeriod)
	}
	if r.Duration == nil {
		return fmt.Errorf("missing column %q", profile.ColumnDuration)
	}
	if r.Timestamp == nil {
		return fmt.Errorf("missing column %q", profile.ColumnTimestamp)
	}
	if r.TimeNanos == nil {
		return fmt.Errorf("missing column %q", profile.ColumnTimeNanos)
	}
	if r.Value == nil {
		return fmt.Errorf("missing column %q", profile.ColumnValue)
	}
	if r.StacktraceIDs == nil {
		return fmt.Errorf("missing column %q", "stacktrace_id")
	}
	if r.Locations == nil {
		return fmt.Errorf("missing column %q", "locations")
	}

	return nil
}

func (c *arrowToInternalConverter) NewRecord(ctx context.Context) (arrow.Record, error) {
	newRecord := array.NewRecord(
		arrow.NewSchema(append(c.b.LabelFields, []arrow.Field{{
			Name: profile.ColumnName,
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: profile.ColumnSampleType,
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: profile.ColumnSampleUnit,
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: profile.ColumnPeriodType,
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: profile.ColumnPeriodUnit,
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: profile.ColumnTimestamp,
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: profile.ColumnTimeNanos,
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: profile.ColumnStacktrace,
			Type: arrow.ListOf(&arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary}),
		}, {
			Name: profile.ColumnDuration,
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: profile.ColumnPeriod,
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: profile.ColumnValue,
			Type: arrow.PrimitiveTypes.Int64,
		}}...), nil),
		append(c.b.Labels,
			[]arrow.Array{
				c.b.Producer,
				c.b.SampleType,
				c.b.SampleUnit,
				c.b.PeriodType,
				c.b.PeriodUnit,
				c.b.Timestamp,
				c.b.TimeNanos,
				c.b.Locations,
				c.b.Duration,
				c.b.Period,
				c.b.Value,
			}...),
		int64(c.b.Value.Len()),
	)

	sortingColDefs := c.schema.ColumnDefinitionsForSortingColumns()
	sortingColumns := make([]arrowutils.SortingColumn, 0, len(sortingColDefs))
	arrowSchema := newRecord.Schema()
	arrowFields := arrowSchema.Fields()
	for _, col := range c.schema.SortingColumns() {
		direction := arrowutils.Ascending
		if col.Descending() {
			direction = arrowutils.Descending
		}

		colDef, found := c.schema.ColumnByName(col.ColumnName())
		if !found {
			return nil, fmt.Errorf("sorting column %v not found in schema", col.ColumnName())
		}

		if colDef.Dynamic {
			for i, c := range arrowFields {
				if strings.HasPrefix(c.Name, colDef.Name) {
					sortingColumns = append(sortingColumns, arrowutils.SortingColumn{
						Index:      i,
						Direction:  direction,
						NullsFirst: col.NullsFirst(),
					})
				}
			}
		} else {
			indices := arrowSchema.FieldIndices(colDef.Name)
			for _, i := range indices {
				sortingColumns = append(sortingColumns, arrowutils.SortingColumn{
					Index:      i,
					Direction:  direction,
					NullsFirst: col.NullsFirst(),
				})
			}
		}
	}

	sortedIdxs, err := arrowutils.SortRecord(newRecord, sortingColumns)
	if err != nil {
		return nil, fmt.Errorf("failed to sort record: %w", err)
	}
	isSorted := true
	for i := 0; i < sortedIdxs.Len(); i++ {
		if sortedIdxs.Value(i) != int32(i) {
			isSorted = false
			break
		}
	}

	if isSorted {
		return newRecord, nil
	}

	// Release the record, since Take will allocate a new, sorted, record.
	defer newRecord.Release()
	return arrowutils.Take(compute.WithAllocator(ctx, c.mem), newRecord, sortedIdxs)
}

func (c *arrowToInternalConverter) AddLocationsRecordV1(
	ctx context.Context,
	rec arrow.Record,
) error {
	schema := rec.Schema()
	if schema.NumFields() != 3 {
		return fmt.Errorf("expected record to have 2 field, got %d", rec.Schema().NumFields())
	}

	var (
		stacktraceIDs *array.Binary
		locations     *array.List
		isComplete    *array.Boolean

		ok bool
	)

	for i, field := range schema.Fields() {
		switch field.Name {
		case "stacktrace_id":
			stacktraceIDs, ok = rec.Column(i).(*array.Binary)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Binary, got %T", field.Name, rec.Column(i))
			}

		case "locations":
			locations, ok = rec.Column(i).(*array.List)
			if !ok {
				return fmt.Errorf("expected column %q to be of type List, got %T", field.Name, rec.Column(i))
			}
		}

		if field.Name == "is_complete" {
			isComplete, ok = rec.Column(i).(*array.Boolean)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Boolean, got %T", field.Name, rec.Column(i))
			}
		}
	}

	if stacktraceIDs == nil {
		return errors.New("missing column stacktrace_id")
	}

	if locations == nil {
		return errors.New("missing column locations")
	}

	if isComplete == nil {
		return errors.New("missing column is_complete")
	}

	numIncomplete := 0
	for i := 0; i < isComplete.Len(); i++ {
		if !isComplete.Value(i) {
			numIncomplete++
		}
	}
	c.metrics.IncompleteLocations.Add(float64(numIncomplete))

	stacktraceIDIndex := make(map[string]int, stacktraceIDs.Len())
	for i := 0; i < stacktraceIDs.Len(); i++ {
		stacktraceID := string(stacktraceIDs.Value(i))
		stacktraceIDIndex[stacktraceID] = i
	}

	r, err := getLocationsReader(locations)
	if err != nil {
		return fmt.Errorf("get locations reader: %w", err)
	}

	stacktraceIDsDictionary, ok := c.b.StacktraceIDs.Dictionary().(*array.Binary)
	if !ok {
		return fmt.Errorf("expected stacktrace IDs column to be of type Binary, got %T", c.b.StacktraceIDs.Dictionary())
	}

	typ := arrow.ListOf(&arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary})
	b := array.NewListBuilderWithField(c.mem, typ.ElemField())
	defer b.Release()
	lv := b.ValueBuilder().(*array.BinaryDictionaryBuilder)

	for i := 0; i < c.b.StacktraceIDs.Len(); i++ {
		stacktraceID := stacktraceIDsDictionary.Value(c.b.StacktraceIDs.GetValueIndex(i))
		row, ok := stacktraceIDIndex[unsafeString(stacktraceID)]
		if !ok {
			return fmt.Errorf("missing stacktrace ID %q in stacktrace IDs column", string(stacktraceID))
		}

		lOffsetStart, lOffsetEnd := locations.ValueOffsets(row)
		if lOffsetEnd-lOffsetStart == 0 {
			b.Append(false)
			continue
		}
		b.Append(true)

		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			address := r.Address.Value(j)
			hasMapping := r.MappingStart.IsValid(j)
			mappingStart := r.MappingStartValues.Value(r.MappingStart.GetPhysicalIndex(j))
			mappingLimit := r.MappingLimitValues.Value(r.MappingLimit.GetPhysicalIndex(j))
			mappingOffset := r.MappingOffsetValues.Value(r.MappingOffset.GetPhysicalIndex(j))
			mappingFile := r.MappingFileDictValues.Value(r.MappingFileDict.GetValueIndex(r.MappingFile.GetPhysicalIndex(j)))

			var mappingBuildID []byte
			if r.MappingBuildIDDict.IsValid(r.MappingBuildID.GetPhysicalIndex(j)) {
				mappingBuildID = r.MappingBuildIDDictValues.Value(r.MappingBuildIDDict.GetValueIndex(r.MappingBuildID.GetPhysicalIndex(j)))
			}

			linesOffsetStart, linesOffsetEnd := r.Lines.ValueOffsets(j)
			if err := lv.Append(profile.EncodeArrowLocation(
				address,
				hasMapping,
				mappingStart,
				mappingLimit,
				mappingOffset,
				mappingFile,
				mappingBuildID,
				int(linesOffsetStart),
				int(linesOffsetEnd),
				r.Lines,
				r.Line,
				r.LineNumber,
				r.LineFunctionName,
				r.LineFunctionNameDict,
				r.LineFunctionSystemName,
				r.LineFunctionSystemNameDict,
				r.LineFunctionFilename,
				r.LineFunctionFilenameDict,
				r.LineFunctionFilenameDictValues,
				r.LineFunctionStartLine,
			)); err != nil {
				return fmt.Errorf("append location: %w", err)
			}
		}
	}

	c.b.Locations = b.NewListArray()
	return nil
}

func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

func (c *arrowToInternalConverter) AddSampleRecordV1(
	ctx context.Context,
	rec arrow.Record,
) error {
	var (
		ok  bool
		err error
	)

	s := rec.Schema()
	for i, field := range s.Fields() {
		switch field.Name {
		case profile.ColumnDuration:
			c.b.Duration, err = expandREEInt64(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case "producer":
			c.b.Producer, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnPeriod:
			c.b.Period, err = expandREEInt64(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnPeriodType:
			c.b.PeriodType, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnPeriodUnit:
			c.b.PeriodUnit, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnSampleType:
			c.b.SampleType, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnSampleUnit:
			c.b.SampleUnit, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case "stacktrace_id":
			c.b.StacktraceIDs, _, err = expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnTimestamp:
			timestamp, err := expandREEInt64(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return fmt.Errorf("expected column %q to be of type Int64, got %T", field.Name, rec.Column(i))
			}

			b := array.NewBuilder(c.mem, arrow.PrimitiveTypes.Int64).(*array.Int64Builder)
			defer b.Release()

			for i := 0; i < timestamp.Len(); i++ {
				// The protocol reports nanosecond timestamps, but the database wants milliseconds.
				b.Append(timestamp.Value(i) / time.Millisecond.Nanoseconds())
			}

			c.b.Timestamp = b.NewInt64Array()
			c.b.TimeNanos = timestamp
		case profile.ColumnValue:
			c.b.Value, ok = rec.Column(i).(*array.Int64)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Int64, got %T", field.Name, rec.Column(i))
			}
			c.b.Value.Retain()
		default:
			if strings.HasPrefix(field.Name, profile.ColumnLabelsPrefix) {
				dict, _, err := expandREEBinaryDict(c.mem, rec.Column(i), field.Name)
				if err != nil {
					return err
				}
				c.b.Labels = append(c.b.Labels, dict)
				c.b.LabelFields = append(c.b.LabelFields, arrow.Field{Name: field.Name, Type: dict.DataType()})
			}

			// Ignore other columns.
		}
	}

	return nil
}

type locationsReader struct {
	Locations                      *array.List
	Location                       *array.Struct
	Address                        *array.Uint64
	MappingStart                   *array.RunEndEncoded
	MappingStartValues             *array.Uint64
	MappingLimit                   *array.RunEndEncoded
	MappingLimitValues             *array.Uint64
	MappingOffset                  *array.RunEndEncoded
	MappingOffsetValues            *array.Uint64
	MappingFile                    *array.RunEndEncoded
	MappingFileDict                *array.Dictionary
	MappingFileDictValues          *array.Binary
	MappingBuildID                 *array.RunEndEncoded
	MappingBuildIDDict             *array.Dictionary
	MappingBuildIDDictValues       *array.Binary
	Lines                          *array.List
	Line                           *array.Struct
	LineNumber                     *array.Int64
	LineFunctionName               *array.Dictionary
	LineFunctionNameDict           *array.Binary
	LineFunctionSystemName         *array.Dictionary
	LineFunctionSystemNameDict     *array.Binary
	LineFunctionFilename           *array.RunEndEncoded
	LineFunctionFilenameDict       *array.Dictionary
	LineFunctionFilenameDictValues *array.Binary
	LineFunctionStartLine          *array.Int64
}

func getLocationsReader(locations *array.List) (*locationsReader, error) {
	location, ok := locations.ListValues().(*array.Struct)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type Struct, got %T", "locations", locations.ListValues())
	}

	const expectedLocationFields = 8
	if location.NumField() != expectedLocationFields {
		return nil, fmt.Errorf("expected location struct column to have %d fields, got %d", expectedLocationFields, location.NumField())
	}

	address, ok := location.Field(0).(*array.Uint64)
	if !ok {
		return nil, fmt.Errorf("expected column address to be of type Uint64, got %T", location.Field(0))
	}

	// skipping 1 field which is the frame type
	mappingStart, mappingStartValues, err := getREEUint64(location.Field(2), "mapping_start")
	if err != nil {
		return nil, err
	}

	mappingLimit, mappingLimitValues, err := getREEUint64(location.Field(3), "mapping_limit")
	if err != nil {
		return nil, err
	}

	mappingOffset, mappingOffsetValues, err := getREEUint64(location.Field(4), "mapping_offset")
	if err != nil {
		return nil, err
	}

	mappingFile, mappingFileDict, mappingFileDictValues, err := getREEBinaryDict(location.Field(5), "mapping_file")
	if err != nil {
		return nil, err
	}

	mappingBuildID, mappingBuildIDDict, mappingBuildIDValues, err := getREEBinaryDict(location.Field(6), "mapping_build_id")
	if err != nil {
		return nil, err
	}

	lines, ok := location.Field(7).(*array.List)
	if !ok {
		return nil, fmt.Errorf("expected column lines to be of type List, got %T", location.Field(7))
	}

	line, ok := lines.ListValues().(*array.Struct)
	if !ok {
		return nil, fmt.Errorf("expected column line to be of type Struct, got %T", lines.ListValues())
	}

	const expectedLineFields = 5
	if line.NumField() != expectedLineFields {
		return nil, fmt.Errorf("expected line struct column to have %d fields, got %d", expectedLineFields, line.NumField())
	}

	lineNumber, ok := line.Field(0).(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("expected column line_number to be of type Int64, got %T", line.Field(0))
	}

	lineFunctionName, lineFunctionNameDict, err := getBinaryDict(line.Field(1), "line_function_name")
	if err != nil {
		return nil, err
	}

	lineFunctionSystemName, lineFunctionSystemNameDict, err := getBinaryDict(line.Field(2), "line_function_system_name")
	if err != nil {
		return nil, err
	}

	lineFunctionFilename, lineFunctionFilenameDict, lineFunctionFilenameDictValues, err := getREEBinaryDict(line.Field(3), "line_function_filename")
	if err != nil {
		return nil, err
	}

	lineFunctionStartLine, ok := line.Field(4).(*array.Int64)
	if !ok {
		return nil, fmt.Errorf("expected column line_function_start_line to be of type Int64, got %T", line.Field(4))
	}

	return &locationsReader{
		Locations:                      locations,
		Location:                       location,
		Address:                        address,
		MappingStart:                   mappingStart,
		MappingStartValues:             mappingStartValues,
		MappingLimit:                   mappingLimit,
		MappingLimitValues:             mappingLimitValues,
		MappingOffset:                  mappingOffset,
		MappingOffsetValues:            mappingOffsetValues,
		MappingFile:                    mappingFile,
		MappingFileDict:                mappingFileDict,
		MappingFileDictValues:          mappingFileDictValues,
		MappingBuildID:                 mappingBuildID,
		MappingBuildIDDict:             mappingBuildIDDict,
		MappingBuildIDDictValues:       mappingBuildIDValues,
		Lines:                          lines,
		Line:                           line,
		LineNumber:                     lineNumber,
		LineFunctionName:               lineFunctionName,
		LineFunctionNameDict:           lineFunctionNameDict,
		LineFunctionSystemName:         lineFunctionSystemName,
		LineFunctionSystemNameDict:     lineFunctionSystemNameDict,
		LineFunctionFilename:           lineFunctionFilename,
		LineFunctionFilenameDict:       lineFunctionFilenameDict,
		LineFunctionFilenameDictValues: lineFunctionFilenameDictValues,
		LineFunctionStartLine:          lineFunctionStartLine,
	}, nil
}
