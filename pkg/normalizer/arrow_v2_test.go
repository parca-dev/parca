// Copyright 2024-2026 The Parca Authors
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
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/extensions"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/profile"
)

// TestAddSampleRecordV2 builds a single-row v2 sample record and exercises the
// v2 conversion path end-to-end through NewRecord. The record schema mirrors the
// agent's SampleSchemaV2 in parca-agent/reporter/arrow_v2.go.
func TestAddSampleRecordV2(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	rec := buildV2SampleRecord(t, mem)
	defer rec.Release()

	dynSchema, err := profile.Schema()
	require.NoError(t, err)

	c := NewArrowToInternalConverter(mem, dynSchema, NewMetrics(prometheus.NewRegistry()))
	defer c.Release()

	require.NoError(t, c.AddSampleRecord(context.Background(), rec))
	require.NoError(t, c.Validate())

	out, err := c.NewRecord(context.Background())
	require.NoError(t, err)
	defer out.Release()

	require.Equal(t, int64(1), out.NumRows())

	schema := out.Schema()
	for _, name := range []string{
		profile.ColumnName,
		profile.ColumnSampleType,
		profile.ColumnSampleUnit,
		profile.ColumnPeriodType,
		profile.ColumnPeriodUnit,
		profile.ColumnTimestamp,
		profile.ColumnTimeNanos,
		profile.ColumnStacktrace,
		profile.ColumnDuration,
		profile.ColumnPeriod,
		profile.ColumnValue,
		profile.ColumnLabelsPrefix + "service",
	} {
		require.NotEmpty(t, schema.FieldIndices(name), "expected internal record to have column %q", name)
	}

	// Verify the timestamp ms / time_nanos split.
	timestampIdx := schema.FieldIndices(profile.ColumnTimestamp)[0]
	timeNanosIdx := schema.FieldIndices(profile.ColumnTimeNanos)[0]
	tsCol := out.Column(timestampIdx).(*array.Int64)
	tnCol := out.Column(timeNanosIdx).(*array.Int64)
	require.Equal(t, int64(2_000_000_000), tnCol.Value(0))
	require.Equal(t, int64(2_000), tsCol.Value(0))
}

// buildV2SampleRecord constructs a one-sample arrow record matching the v2
// schema, with one stacktrace containing a single location with one line.
func buildV2SampleRecord(t *testing.T, mem memory.Allocator) arrow.RecordBatch {
	t.Helper()

	// Function dictionary (one function).
	funcType := arrow.StructOf(
		arrow.Field{Name: "system_name", Type: arrow.BinaryTypes.StringView, Nullable: true},
		arrow.Field{Name: "filename", Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}, Nullable: true},
		arrow.Field{Name: "start_line", Type: arrow.PrimitiveTypes.Uint64, Nullable: false},
	)
	funcDictType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: funcType}

	funcStructB := array.NewStructBuilder(mem, funcType)
	defer funcStructB.Release()
	funcStructB.Append(true)
	funcStructB.FieldBuilder(0).(*array.StringViewBuilder).AppendString("do_thing")
	funcStructB.FieldBuilder(1).(*array.BinaryDictionaryBuilder).AppendString("thing.go")
	funcStructB.FieldBuilder(2).(*array.Uint64Builder).Append(1)
	funcValues := funcStructB.NewArray()
	defer funcValues.Release()

	funcIdxB := array.NewUint32Builder(mem)
	defer funcIdxB.Release()
	funcIdxB.Append(0) // line 0 -> function 0
	funcIdxArr := funcIdxB.NewArray()
	defer funcIdxArr.Release()

	funcDictArr := array.NewDictionaryArray(funcDictType, funcIdxArr, funcValues)
	defer funcDictArr.Release()

	// Line struct: { line, column, function }
	lineType := arrow.StructOf(
		arrow.Field{Name: "line", Type: arrow.PrimitiveTypes.Uint64},
		arrow.Field{Name: "column", Type: arrow.PrimitiveTypes.Uint64},
		arrow.Field{Name: "function", Type: funcDictType},
	)

	lineNumB := array.NewUint64Builder(mem)
	defer lineNumB.Release()
	lineNumB.Append(42)
	lineNumArr := lineNumB.NewArray()
	defer lineNumArr.Release()

	lineColB := array.NewUint64Builder(mem)
	defer lineColB.Release()
	lineColB.Append(7)
	lineColArr := lineColB.NewArray()
	defer lineColArr.Release()

	lineStructData := array.NewData(
		lineType,
		1,
		[]*memory.Buffer{nil},
		[]arrow.ArrayData{lineNumArr.Data(), lineColArr.Data(), funcDictArr.Data()},
		0, 0,
	)
	defer lineStructData.Release()

	// Lines ListView: one location with one line at offset 0.
	lineOffsetsB := array.NewInt32Builder(mem)
	defer lineOffsetsB.Release()
	lineOffsetsB.Append(0)
	lineOffsetsArr := lineOffsetsB.NewArray()
	defer lineOffsetsArr.Release()

	lineSizesB := array.NewInt32Builder(mem)
	defer lineSizesB.Release()
	lineSizesB.Append(1)
	lineSizesArr := lineSizesB.NewArray()
	defer lineSizesArr.Release()

	linesListType := arrow.ListViewOf(lineType)
	linesListData := array.NewData(
		linesListType,
		1,
		[]*memory.Buffer{nil, lineOffsetsArr.Data().Buffers()[1], lineSizesArr.Data().Buffers()[1]},
		[]arrow.ArrayData{lineStructData},
		0, 0,
	)
	defer linesListData.Release()

	// Location struct: { address, frame_type, mapping_file, mapping_build_id, lines }.
	stringDictType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String}
	locType := arrow.StructOf(
		arrow.Field{Name: "address", Type: arrow.PrimitiveTypes.Uint64},
		arrow.Field{Name: "frame_type", Type: stringDictType, Nullable: true},
		arrow.Field{Name: "mapping_file", Type: stringDictType, Nullable: true},
		arrow.Field{Name: "mapping_build_id", Type: stringDictType, Nullable: true},
		arrow.Field{Name: "lines", Type: linesListType, Nullable: true},
	)

	addressB := array.NewUint64Builder(mem)
	defer addressB.Release()
	addressB.Append(0xDEADBEEF)
	addressArr := addressB.NewArray()
	defer addressArr.Release()

	frameTypeB := array.NewBuilder(mem, stringDictType).(*array.BinaryDictionaryBuilder)
	defer frameTypeB.Release()
	frameTypeB.AppendString("native")
	frameTypeArr := frameTypeB.NewArray()
	defer frameTypeArr.Release()

	mappingFileB := array.NewBuilder(mem, stringDictType).(*array.BinaryDictionaryBuilder)
	defer mappingFileB.Release()
	mappingFileB.AppendString("/usr/bin/app")
	mappingFileArr := mappingFileB.NewArray()
	defer mappingFileArr.Release()

	buildIDB := array.NewBuilder(mem, stringDictType).(*array.BinaryDictionaryBuilder)
	defer buildIDB.Release()
	buildIDB.AppendString("build123")
	buildIDArr := buildIDB.NewArray()
	defer buildIDArr.Release()

	locStructData := array.NewData(
		locType,
		1,
		[]*memory.Buffer{nil},
		[]arrow.ArrayData{
			addressArr.Data(),
			frameTypeArr.Data(),
			mappingFileArr.Data(),
			buildIDArr.Data(),
			linesListData,
		},
		0, 0,
	)
	defer locStructData.Release()
	locStructArr := array.MakeFromData(locStructData)
	defer locStructArr.Release()

	// Stacktrace ListView[Dictionary[Uint32, LocationStruct]] referring to the
	// single location once.
	locDictType := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: locType}
	locIdxB := array.NewUint32Builder(mem)
	defer locIdxB.Release()
	locIdxB.Append(0)
	locIdxArr := locIdxB.NewArray()
	defer locIdxArr.Release()
	locDictArr := array.NewDictionaryArray(locDictType, locIdxArr, locStructArr)
	defer locDictArr.Release()

	stOffsetsB := array.NewInt32Builder(mem)
	defer stOffsetsB.Release()
	stOffsetsB.Append(0)
	stOffsetsArr := stOffsetsB.NewArray()
	defer stOffsetsArr.Release()

	stSizesB := array.NewInt32Builder(mem)
	defer stSizesB.Release()
	stSizesB.Append(1)
	stSizesArr := stSizesB.NewArray()
	defer stSizesArr.Release()

	stacktraceType := arrow.ListViewOf(locDictType)
	stacktraceData := array.NewData(
		stacktraceType,
		1,
		[]*memory.Buffer{nil, stOffsetsArr.Data().Buffers()[1], stSizesArr.Data().Buffers()[1]},
		[]arrow.ArrayData{locDictArr.Data()},
		0, 0,
	)
	defer stacktraceData.Release()
	stacktraceArr := array.MakeFromData(stacktraceData)
	defer stacktraceArr.Release()

	// Labels struct with a single REE[Dict[Uint32, String]] label.
	labelType := arrow.RunEndEncodedOf(
		arrow.PrimitiveTypes.Int32,
		stringDictType,
	)
	labelsStructType := arrow.StructOf(arrow.Field{Name: "service", Type: labelType, Nullable: true})

	serviceArr := buildREEStringDict(t, mem, []string{"my-service"})
	defer serviceArr.Release()

	labelsData := array.NewData(
		labelsStructType,
		1,
		[]*memory.Buffer{nil},
		[]arrow.ArrayData{serviceArr.Data()},
		0, 0,
	)
	defer labelsData.Release()
	labelsArr := array.MakeFromData(labelsData)
	defer labelsArr.Release()

	// Stacktrace ID UUID (16 bytes).
	uuidB := extensions.NewUUIDBuilder(mem)
	defer uuidB.Release()
	uuidB.AppendBytes([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	uuidArr := uuidB.NewArray()
	defer uuidArr.Release()

	// Scalar fields.
	valueB := array.NewInt64Builder(mem)
	defer valueB.Release()
	valueB.Append(1)
	valueArr := valueB.NewArray()
	defer valueArr.Release()

	producerArr := buildREEString(t, mem, []string{"parca_agent"})
	defer producerArr.Release()
	sampleTypeArr := buildREEString(t, mem, []string{"samples"})
	defer sampleTypeArr.Release()
	sampleUnitArr := buildREEString(t, mem, []string{"count"})
	defer sampleUnitArr.Release()
	periodTypeArr := buildREEString(t, mem, []string{"cpu"})
	defer periodTypeArr.Release()
	periodUnitArr := buildREEString(t, mem, []string{"nanoseconds"})
	defer periodUnitArr.Release()
	temporalityArr := buildREEString(t, mem, []string{"delta"})
	defer temporalityArr.Release()

	periodArr := buildREEInt64(t, mem, []int64{1_000_000})
	defer periodArr.Release()
	durationArr := buildREEUint64(t, mem, []uint64{1_000_000_000})
	defer durationArr.Release()

	tsType := &arrow.TimestampType{Unit: arrow.Nanosecond, TimeZone: "UTC"}
	tsB := array.NewBuilder(mem, tsType).(*array.TimestampBuilder)
	defer tsB.Release()
	tsB.Append(arrow.Timestamp(2_000_000_000))
	tsArr := tsB.NewArray()
	defer tsArr.Release()

	schema := arrow.NewSchema([]arrow.Field{
		{Name: "labels", Type: labelsStructType, Nullable: false},
		{Name: profile.ColumnStacktrace, Type: stacktraceType, Nullable: true},
		{Name: "stacktrace_id", Type: extensions.NewUUIDType()},
		{Name: profile.ColumnValue, Type: arrow.PrimitiveTypes.Int64},
		{Name: "producer", Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)},
		{Name: profile.ColumnSampleType, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)},
		{Name: profile.ColumnSampleUnit, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)},
		{Name: profile.ColumnPeriodType, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)},
		{Name: profile.ColumnPeriodUnit, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)},
		{Name: "temporality", Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String), Nullable: true},
		{Name: profile.ColumnPeriod, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.PrimitiveTypes.Int64)},
		{Name: profile.ColumnDuration, Type: arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.PrimitiveTypes.Uint64)},
		{Name: profile.ColumnTimestamp, Type: tsType},
	}, withV2Metadata())

	return array.NewRecordBatch(schema, []arrow.Array{
		labelsArr,
		stacktraceArr,
		uuidArr,
		valueArr,
		producerArr,
		sampleTypeArr,
		sampleUnitArr,
		periodTypeArr,
		periodUnitArr,
		temporalityArr,
		periodArr,
		durationArr,
		tsArr,
	}, 1)
}

func withV2Metadata() *arrow.Metadata {
	m := arrow.NewMetadata([]string{MetadataSchemaVersion}, []string{MetadataSchemaVersionV2})
	return &m
}

func buildREEString(t *testing.T, mem memory.Allocator, values []string) arrow.Array {
	t.Helper()
	typ := arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.BinaryTypes.String)
	b := array.NewBuilder(mem, typ).(*array.RunEndEncodedBuilder)
	defer b.Release()
	for _, v := range values {
		b.Append(1)
		b.ValueBuilder().(*array.StringBuilder).Append(v)
	}
	return b.NewArray()
}

func buildREEStringDict(t *testing.T, mem memory.Allocator, values []string) arrow.Array {
	t.Helper()
	typ := arrow.RunEndEncodedOf(
		arrow.PrimitiveTypes.Int32,
		&arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.String},
	)
	b := array.NewBuilder(mem, typ).(*array.RunEndEncodedBuilder)
	defer b.Release()
	for _, v := range values {
		b.Append(1)
		require.NoError(t, b.ValueBuilder().(*array.BinaryDictionaryBuilder).AppendString(v))
	}
	return b.NewArray()
}

func buildREEInt64(t *testing.T, mem memory.Allocator, values []int64) arrow.Array {
	t.Helper()
	typ := arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.PrimitiveTypes.Int64)
	b := array.NewBuilder(mem, typ).(*array.RunEndEncodedBuilder)
	defer b.Release()
	for _, v := range values {
		b.Append(1)
		b.ValueBuilder().(*array.Int64Builder).Append(v)
	}
	return b.NewArray()
}

func buildREEUint64(t *testing.T, mem memory.Allocator, values []uint64) arrow.Array {
	t.Helper()
	typ := arrow.RunEndEncodedOf(arrow.PrimitiveTypes.Int32, arrow.PrimitiveTypes.Uint64)
	b := array.NewBuilder(mem, typ).(*array.RunEndEncodedBuilder)
	defer b.Release()
	for _, v := range values {
		b.Append(1)
		b.ValueBuilder().(*array.Uint64Builder).Append(v)
	}
	return b.NewArray()
}
