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
	"context"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateTable(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		pp,
		0,
		[]string{},
	)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	rec, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, p)
	require.NoError(t, err)
	defer rec.Release()

	require.NotNil(t, rec)
	require.NotNil(t, cumulative)

	require.Equal(t, int64(310797348), cumulative)
	// require.Equal(t, 899, rec.NumRows())

	mappingFileColumn := rec.Column(rec.Schema().FieldIndices(TableFieldMappingFile)[0]).(*array.Dictionary)
	mappingFileColumnDict := mappingFileColumn.Dictionary().(*array.String)
	mappingBuildIDColumn := rec.Column(rec.Schema().FieldIndices(TableFieldMappingBuildID)[0]).(*array.Dictionary)
	locationAddressColumn := rec.Column(rec.Schema().FieldIndices(TableFieldLocationAddress)[0]).(*array.Uint64)
	locationLineColumn := rec.Column(rec.Schema().FieldIndices(TableFieldLocationLine)[0]).(*array.Int64)
	functionStartLineColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionStartLine)[0]).(*array.Int64)
	functionNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionName)[0]).(*array.Dictionary)
	functionNameColumnDict := functionNameColumn.Dictionary().(*array.String)
	functionSystemNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionSystemName)[0]).(*array.Dictionary)
	functionSystemNameColumnDict := functionSystemNameColumn.Dictionary().(*array.String)
	functionFileNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionFileName)[0]).(*array.Dictionary)
	functionFileNameColumnDict := functionFileNameColumn.Dictionary().(*array.String)
	cumulativeColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCumulative)[0]).(*array.Int64)
	cumulativeDiffColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCumulativeDiff)[0]).(*array.Int64)
	flatColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFlat)[0]).(*array.Int64)
	flatDiffColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFlatDiff)[0]).(*array.Int64)

	found := false
	for i := 0; i < int(rec.NumRows()); i++ {
		if locationAddressColumn.Value(i) == uint64(7578561) {
			// mapping
			require.Equal(t, "/bin/operator", mappingFileColumnDict.Value(mappingFileColumn.GetValueIndex(i)))
			require.True(t, mappingBuildIDColumn.IsNull(i))
			// location
			// address is already checked above
			require.Equal(t, int64(107), locationLineColumn.Value(i))
			// function
			require.Equal(t, int64(0), functionStartLineColumn.Value(i))
			require.Equal(t,
				"encoding/json.Unmarshal",
				functionNameColumnDict.Value(functionNameColumn.GetValueIndex(i)),
			)
			require.Equal(t,
				"encoding/json.Unmarshal",
				functionSystemNameColumnDict.Value(functionSystemNameColumn.GetValueIndex(i)),
			)
			require.Equal(t,
				"/opt/hostedtoolcache/go/1.14.10/x64/src/encoding/json/decode.go",
				functionFileNameColumnDict.Value(functionFileNameColumn.GetValueIndex(i)),
			)
			// values
			require.Equal(t, int64(3135531), cumulativeColumn.Value(i))
			require.Equal(t, int64(1251322), flatColumn.Value(i))
			// diff
			require.Equal(t, int64(0), cumulativeDiffColumn.Value(i))
			require.Equal(t, int64(0), flatDiffColumn.Value(i))

			found = true
		}
	}

	require.Truef(t, found, "expected to find the specific function")
}

func TestTableCallView(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	fileContent := MustReadAllGzip(t, "testdata/two-stacks.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		pp,
		0,
		[]string{},
	)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	rec, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, p)
	require.NoError(t, err)
	defer rec.Release()

	require.NotNil(t, rec)
	require.NotNil(t, cumulative)

	functionNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionName)[0]).(*array.Dictionary)
	functionNameColumnDict := functionNameColumn.Dictionary().(*array.String)

	callersColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCallers)[0]).(*array.List)
	calleesColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCallees)[0]).(*array.List)

	nodeIndex := -1
	child1Index := -1
	child2Index := -1

	for i := 0; i < int(rec.NumRows()); i++ {
		if functionNameColumnDict.Value(functionNameColumn.GetValueIndex(i)) == "unwind failed" {
			nodeIndex = i
		}
		if functionNameColumnDict.Value(functionNameColumn.GetValueIndex(i)) == "ChunkNotFound" {
			child1Index = i
		}
		if functionNameColumnDict.Value(functionNameColumn.GetValueIndex(i)) == "PcNotCovered" {
			child2Index = i
		}
	}

	callerValues := callersColumn.ListValues().(*array.Int64)
	calleeValues := calleesColumn.ListValues().(*array.Int64)

	beg, end := callersColumn.ValueOffsets(nodeIndex)

	require.Equal(t, 0, int(end-beg))

	beg, end = callersColumn.ValueOffsets(child1Index)
	require.Equal(t, 1, int(end-beg))
	require.Equal(t, "unwind failed", functionNameColumnDict.Value(functionNameColumn.GetValueIndex(int(callerValues.Value(int(beg))))))

	beg, end = callersColumn.ValueOffsets(child2Index)
	require.Equal(t, 1, int(end-beg))
	require.Equal(t, "unwind failed", functionNameColumnDict.Value(functionNameColumn.GetValueIndex(int(callerValues.Value(int(beg))))))

	beg, end = calleesColumn.ValueOffsets(nodeIndex)

	actualValues := []string{}
	for i := beg; i < end; i++ {
		actualValues = append(actualValues, functionNameColumnDict.Value(functionNameColumn.GetValueIndex(int(calleeValues.Value(int(i))))))
	}

	require.Equal(t, 2, int(end-beg))
	require.Contains(t, actualValues, "ChunkNotFound")
	require.Contains(t, actualValues, "PcNotCovered")

	beg, end = calleesColumn.ValueOffsets(child1Index)
	require.Equal(t, 0, int(end-beg))

	beg, end = calleesColumn.ValueOffsets(child2Index)
	require.Equal(t, 0, int(end-beg))
}

func TestGenerateTableAggregateFlat(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	mappings := []*pprofprofile.Mapping{{
		ID:      1,
		Start:   1,
		Limit:   1,
		Offset:  1,
		File:    "1",
		BuildID: "1",
	}}

	locations := []*pprofprofile.Location{{
		ID:      1,
		Mapping: mappings[0],
		Address: 0x1,
	}, {
		ID:      2,
		Mapping: mappings[0],
		Address: 0x2,
	}, {
		ID:      3,
		Mapping: mappings[0],
		Address: 0x3,
	}, {
		ID:      4,
		Mapping: mappings[0],
		Address: 0x4,
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[1], locations[0]},
				Value:    []int64{1},
			}, {
				Location: []*pprofprofile.Location{locations[2], locations[0]},
				Value:    []int64{2},
			}, {
				Location: []*pprofprofile.Location{locations[3], locations[0]},
				Value:    []int64{3},
			}, {
				Location: []*pprofprofile.Location{locations[0]},
				Value:    []int64{4},
			}},
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	rec, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, p)
	require.NoError(t, err)
	defer rec.Release()

	require.Equal(t, int64(4), rec.NumRows())
	require.Equal(t, int64(10), cumulative)

	expectedColumns := tableColumns{
		mappingFile:        []string{"1", "1", "1", "1"},
		mappingBuildID:     []string{"1", "1", "1", "1"},
		locationAddress:    []uint64{2, 1, 3, 4},
		locationLine:       []int64{0, 0, 0, 0},
		functionStartLine:  []int64{0, 0, 0, 0},
		functionName:       []string{"(null)", "(null)", "(null)", "(null)"},
		functionSystemName: []string{"(null)", "(null)", "(null)", "(null)"},
		functionFileName:   []string{"(null)", "(null)", "(null)", "(null)"},
		cumulative:         []int64{1, 10, 2, 3},
		cumulativeDiff:     []int64{0, 0, 0, 0},
		flat:               []int64{1, 4, 2, 3},
		flatDiff:           []int64{0, 0, 0, 0},
	}
	actualColumns := tableRecordToColumns(t, rec)

	require.Equal(t, expectedColumns, actualColumns)
}

type tableColumns struct {
	mappingFile        []string
	mappingBuildID     []string
	locationAddress    []uint64
	locationLine       []int64
	functionStartLine  []int64
	functionName       []string
	functionSystemName []string
	functionFileName   []string
	cumulative         []int64
	cumulativeDiff     []int64
	flat               []int64
	flatDiff           []int64
}

func tableRecordToColumns(t *testing.T, r arrow.RecordBatch) tableColumns {
	return tableColumns{
		mappingFile:        extractColumn(t, r, TableFieldMappingFile).([]string),
		mappingBuildID:     extractColumn(t, r, TableFieldMappingBuildID).([]string),
		locationAddress:    extractColumn(t, r, TableFieldLocationAddress).([]uint64),
		locationLine:       extractColumn(t, r, TableFieldLocationLine).([]int64),
		functionStartLine:  extractColumn(t, r, TableFieldFunctionStartLine).([]int64),
		functionName:       extractColumn(t, r, TableFieldFunctionName).([]string),
		functionSystemName: extractColumn(t, r, TableFieldFunctionSystemName).([]string),
		functionFileName:   extractColumn(t, r, TableFieldFunctionFileName).([]string),
		cumulative:         extractColumn(t, r, TableFieldCumulative).([]int64),
		cumulativeDiff:     extractColumn(t, r, TableFieldCumulativeDiff).([]int64),
		flat:               extractColumn(t, r, TableFieldFlat).([]int64),
		flatDiff:           extractColumn(t, r, TableFieldFlatDiff).([]int64),
	}
}
