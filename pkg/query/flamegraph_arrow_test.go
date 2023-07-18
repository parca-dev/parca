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
	"testing"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

type flamegraphRow struct {
	MappingStart       uint64
	MappingLimit       uint64
	MappingOffset      uint64
	MappingFile        string
	MappingBuildID     string
	LocationAddress    uint64
	LocationFolded     bool
	LocationLine       int64
	FunctionStartLine  int64
	FunctionName       string
	FunctionSystemName string
	FunctionFilename   string
	Labels             map[string]string
	Children           []uint32
	Cumulative         int64
}

type flamegraphColumns struct {
	mappingStart        []uint64
	mappingLimit        []uint64
	mappingOffset       []uint64
	mappingFiles        []string
	mappingBuildIDs     []string
	locationAddresses   []uint64
	locationFolded      []bool
	locationLines       []int64
	functionStartLines  []int64
	functionNames       []string
	functionSystemNames []string
	functionFileNames   []string
	labels              []map[string]string
	children            [][]uint32
	cumulative          []int64
}

func rowsToColumn(rows []flamegraphRow) flamegraphColumns {
	columns := flamegraphColumns{}
	for _, row := range rows {
		columns.mappingStart = append(columns.mappingStart, row.MappingStart)
		columns.mappingLimit = append(columns.mappingLimit, row.MappingLimit)
		columns.mappingOffset = append(columns.mappingOffset, row.MappingOffset)
		columns.mappingFiles = append(columns.mappingFiles, row.MappingFile)
		columns.mappingBuildIDs = append(columns.mappingBuildIDs, row.MappingBuildID)
		columns.locationAddresses = append(columns.locationAddresses, row.LocationAddress)
		columns.locationFolded = append(columns.locationFolded, row.LocationFolded)
		columns.locationLines = append(columns.locationLines, row.LocationLine)
		columns.functionStartLines = append(columns.functionStartLines, row.FunctionStartLine)
		columns.functionNames = append(columns.functionNames, row.FunctionName)
		columns.functionSystemNames = append(columns.functionSystemNames, row.FunctionSystemName)
		columns.functionFileNames = append(columns.functionFileNames, row.FunctionFilename)
		columns.labels = append(columns.labels, row.Labels)
		columns.children = append(columns.children, row.Children)
		columns.cumulative = append(columns.cumulative, row.Cumulative)
	}
	return columns
}

func requireColumn(t *testing.T, r arrow.Record, field string, expected any) {
	switch expected.(type) {
	case []int64:
		require.Equal(t,
			expected,
			r.Column(r.Schema().FieldIndices(field)[0]).(*array.Int64).Int64Values(),
		)
	case []uint64:
		require.Equal(t,
			expected,
			r.Column(r.Schema().FieldIndices(field)[0]).(*array.Uint64).Uint64Values(),
		)
	}
}

func requireColumnDict(t *testing.T, r arrow.Record, field string, expected any) {
	dict := r.Column(r.Schema().FieldIndices(field)[0]).(*array.Dictionary)

	switch expected.(type) {
	case []string:
		mappingFilesString := dict.Dictionary().(*array.String)
		mappingFiles := make([]string, r.NumRows())
		for i := 0; i < int(r.NumRows()); i++ {
			mappingFiles[i] = mappingFilesString.Value(dict.GetValueIndex(i))
		}
		require.Equal(t, expected, mappingFiles)
	}
}

func requireColumnChildren(t *testing.T, record arrow.Record, expected [][]uint32) {
	children := make([][]uint32, record.NumRows())
	list := record.Column(record.Schema().FieldIndices(FlamegraphFieldChildren)[0]).(*array.List)
	listValues := list.ListValues().(*array.Uint32).Uint32Values()
	for i := 0; i < int(record.NumRows()); i++ {
		if !list.IsValid(i) {
			children[i] = nil
		} else {
			start, end := list.ValueOffsets(i)
			children[i] = listValues[start:end]
		}
	}
	require.Equal(t, expected, children)
}

func TestGenerateFlamegraphArrow(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewGoAllocator()
	var err error

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)

	metastore := metastore.NewInProcessClient(l)

	mres, err := metastore.GetOrCreateMappings(ctx, &metastorepb.GetOrCreateMappingsRequest{
		Mappings: []*metastorepb.Mapping{{Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildId: "aID"}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := metastore.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{
			{Name: "1", SystemName: "1", Filename: "1", StartLine: 1},
			{Name: "2", SystemName: "2", Filename: "2", StartLine: 2},
			{Name: "3", SystemName: "3", Filename: "3", StartLine: 3},
			{Name: "4", SystemName: "4", Filename: "4", StartLine: 4},
			{Name: "5", SystemName: "5", Filename: "5", StartLine: 5},
		},
	})
	require.NoError(t, err)
	f1 := fres.Functions[0]
	f2 := fres.Functions[1]
	f3 := fres.Functions[2]
	f4 := fres.Functions[3]
	f5 := fres.Functions[4]

	lres, err := metastore.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			MappingId: m.Id,
			Address:   0xa1,
			Lines: []*metastorepb.Line{{
				FunctionId: f1.Id,
				Line:       1,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa2,
			Lines: []*metastorepb.Line{{
				FunctionId: f2.Id,
				Line:       2,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa3,
			Lines: []*metastorepb.Line{{
				FunctionId: f3.Id,
				Line:       3,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa4,
			Lines: []*metastorepb.Line{{
				FunctionId: f4.Id,
				Line:       4,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa5,
			Lines: []*metastorepb.Line{{
				FunctionId: f5.Id,
				Line:       5,
			}},
		}},
	})
	require.NoError(t, err)
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]
	l3 := lres.Locations[2]
	l4 := lres.Locations[3]
	l5 := lres.Locations[4]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l4.Id, l3.Id, l2.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	s1 := sres.Stacktraces[0]
	s2 := sres.Stacktraces[1]
	s3 := sres.Stacktraces[2]

	tracer := trace.NewNoopTracerProvider().Tracer("")

	p, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
		Samples: []*parcaprofile.NormalizedSample{{
			StacktraceID: s1.Id,
			Value:        2,
			Label:        map[string]string{"goroutine": "1"},
		}, {
			StacktraceID: s2.Id,
			Value:        1,
			Label:        map[string]string{"goroutine": "1"},
		}, {
			StacktraceID: s3.Id,
			Value:        3,
			Label:        map[string]string{},
		}},
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		aggregate []string
		// expectations
		rows       []flamegraphRow
		cumulative int64
		height     int32
		trimmed    int64
	}{{
		name:      "aggregate-nothing", // raw
		aggregate: nil,
		// expectations
		cumulative: 6,
		height:     5,
		trimmed:    0, // TODO
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: nil, Children: []uint32{1, 3, 7}},                                   // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 2, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 2, Labels: nil, Children: nil},                                         // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{4}}, // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 1, Labels: nil, Children: []uint32{5}},                                 // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Labels: nil, Children: []uint32{6}},                                 // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Labels: nil, Children: nil},                                         // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: nil, Children: []uint32{8}},                                 // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: nil, Children: []uint32{9}},                                 // 8
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Labels: nil, Children: []uint32{10}},                                // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                         // 10
		},
	}, {
		name:      "aggregate-function-name",
		aggregate: []string{FlamegraphFieldFunctionName},
		// expectations
		cumulative: 6,
		height:     5,
		trimmed:    0, // TODO
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: nil, Children: []uint32{1}},                                         // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 6, Labels: nil, Children: []uint32{3}},                                 // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Labels: nil, Children: []uint32{4, 5}},                              // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Labels: nil, Children: nil},                                         // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                         // 5
		},
	}, {
		name:      "aggregate-pprof-labels",
		aggregate: []string{FlamegraphFieldLabels},
		// expectations
		cumulative: 6,
		height:     5,
		trimmed:    0, // TODO
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: nil, Children: []uint32{1, 5}}, // 0
			// all of these have the same labels, so they are aggregated
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: nil, Children: []uint32{3}},                                 // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Labels: nil, Children: []uint32{4}},                                 // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Labels: nil, Children: nil},                                         // 4
			// all of these have no labels, so they are kept separate
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: nil, Children: []uint32{6}}, // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: nil, Children: []uint32{7}}, // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Labels: nil, Children: []uint32{8}}, // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},         // 8
		},
	}, {
		name:      "aggregate-mapping-file",
		aggregate: []string{FlamegraphFieldMappingFile},
		// expectations
		cumulative: 6,
		height:     5,
		trimmed:    0, // TODO
		rows: []flamegraphRow{
			// This aggregates all the rows with the same mapping file, meaning that we only keep one flamegraphRow per stack depth in this example.
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: nil, Children: []uint32{1}},                                         // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 6, Labels: nil, Children: []uint32{3}},                                 // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Labels: nil, Children: []uint32{4}},                                 // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Labels: nil, Children: nil},                                         // 4
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, tc.aggregate, 0)
			require.NoError(t, err)

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(len(tc.rows)), fa.NumRows())
			require.Equal(t, int64(16), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			columns := rowsToColumn(tc.rows)

			requireColumn(t, fa, FlamegraphFieldMappingStart, columns.mappingStart)
			requireColumn(t, fa, FlamegraphFieldMappingLimit, columns.mappingLimit)
			requireColumn(t, fa, FlamegraphFieldMappingOffset, columns.mappingOffset)
			requireColumnDict(t, fa, FlamegraphFieldMappingFile, columns.mappingFiles)
			requireColumnDict(t, fa, FlamegraphFieldMappingBuildID, columns.mappingBuildIDs)
			requireColumn(t, fa, FlamegraphFieldLocationAddress, columns.locationAddresses)
			requireColumn(t, fa, FlamegraphFieldLocationFolded, columns.locationFolded)
			requireColumn(t, fa, FlamegraphFieldLocationLine, columns.locationLines)
			requireColumn(t, fa, FlamegraphFieldFunctionStartLine, columns.functionStartLines)
			requireColumnDict(t, fa, FlamegraphFieldFunctionName, columns.functionNames)
			requireColumnDict(t, fa, FlamegraphFieldFunctionSystemName, columns.functionSystemNames)
			requireColumnDict(t, fa, FlamegraphFieldFunctionFileName, columns.functionFileNames)
			requireColumn(t, fa, FlamegraphFieldCumulative, columns.cumulative)
			requireColumnChildren(t, fa, columns.children)

			labelsDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldLabels)[0]).(*array.Dictionary)
			labelsString := labelsDict.Dictionary().(*array.String)
			pprofLabels := make([]map[string]string, fa.NumRows())
			for i := 0; i < int(fa.NumRows()); i++ {
				if labelsDict.IsNull(i) {
					continue
				}
				ls := map[string]string{}
				err := json.Unmarshal([]byte(labelsString.Value(labelsDict.GetValueIndex(i))), &ls)
				require.NoError(t, err)
				pprofLabels[i] = ls
			}
			require.Equal(t, columns.labels, pprofLabels)

			require.Equal(t,
				len(tc.rows),
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldDiff)[0]).(*array.Int64).NullN(),
			)
		})
	}
}

func TestGenerateFlamegraphArrowWithInlined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mem := memory.NewGoAllocator()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	store := metastoretest.NewTestMetastore(t, logger, reg, tracer)

	functions := []*pprofprofile.Function{
		{ID: 1, Name: "net.(*netFD).accept", SystemName: "net.(*netFD).accept", Filename: "net/fd_unix.go"},
		{ID: 2, Name: "internal/poll.(*FD).Accept", SystemName: "internal/poll.(*FD).Accept", Filename: "internal/poll/fd_unix.go"},
		{ID: 3, Name: "internal/poll.(*pollDesc).waitRead", SystemName: "internal/poll.(*pollDesc).waitRead", Filename: "internal/poll/fd_poll_runtime.go"},
		{ID: 4, Name: "internal/poll.(*pollDesc).wait", SystemName: "internal/poll.(*pollDesc).wait", Filename: "internal/poll/fd_poll_runtime.go"},
	}
	locations := []*pprofprofile.Location{
		{ID: 1, Address: 0xa1, Line: []pprofprofile.Line{{Line: 173, Function: functions[0]}}},
		{ID: 2, Address: 0xa2, Line: []pprofprofile.Line{
			{Line: 89, Function: functions[1]},
			{Line: 402, Function: functions[2]},
		}},
		{ID: 3, Address: 0xa3, Line: []pprofprofile.Line{{Line: 84, Function: functions[3]}}},
	}
	samples := []*pprofprofile.Sample{
		{
			Location: []*pprofprofile.Location{locations[2], locations[1], locations[0]},
			Value:    []int64{1},
		},
	}
	b := bytes.NewBuffer(nil)
	err := (&pprofprofile.Profile{
		SampleType: []*pprofprofile.ValueType{{Type: "alloc_space", Unit: "bytes"}},
		PeriodType: &pprofprofile.ValueType{Type: "space", Unit: "bytes"},
		Sample:     samples,
		Location:   locations,
		Function:   functions,
	}).Write(b)
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	err = p.UnmarshalVT(MustDecompressGzip(t, b.Bytes()))
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(store)
	normalizer := parcacol.NewNormalizer(metastore, true)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]string{}, p, false)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	record, total, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, symbolizedProfile, []string{FlamegraphFieldFunctionName}, 0)
	require.NoError(t, err)

	require.Equal(t, int64(1), total)
	require.Equal(t, int32(4), height)
	require.Equal(t, int64(0), trimmed)

	require.Equal(t, int64(16), record.NumCols())
	require.Equal(t, int64(5), record.NumRows())

	rows := []flamegraphRow{
		{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "net.(*netFD).accept", FunctionSystemName: "net.(*netFD).accept", FunctionFilename: "net/fd_unix.go", Cumulative: 1, Labels: nil, Children: []uint32{1}},                                                           // 0
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 173, FunctionStartLine: 0, FunctionName: "net.(*netFD).accept", FunctionSystemName: "net.(*netFD).accept", FunctionFilename: "net/fd_unix.go", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}},                 // 1
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 402, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).waitRead", FunctionSystemName: "internal/poll.(*pollDesc).waitRead", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: []uint32{3}}, // 2
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 89, FunctionStartLine: 0, FunctionName: "internal/poll.(*FD).Accept", FunctionSystemName: "internal/poll.(*FD).Accept", FunctionFilename: "internal/poll/fd_unix.go", Cumulative: 1, Labels: nil, Children: []uint32{4}},                          // 3
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 84, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).wait", FunctionSystemName: "internal/poll.(*pollDesc).wait", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: nil},                  // 4
	}
	columns := rowsToColumn(rows)

	// mapping fields are all null here
	requireColumn(t, record, FlamegraphFieldLocationAddress, columns.locationAddresses)
	requireColumn(t, record, FlamegraphFieldLocationFolded, columns.locationFolded)
	requireColumn(t, record, FlamegraphFieldLocationLine, columns.locationLines)
	requireColumn(t, record, FlamegraphFieldFunctionStartLine, columns.functionStartLines)
	requireColumnDict(t, record, FlamegraphFieldFunctionName, columns.functionNames)
	requireColumnDict(t, record, FlamegraphFieldFunctionSystemName, columns.functionSystemNames)
	requireColumnDict(t, record, FlamegraphFieldFunctionFileName, columns.functionFileNames)
	requireColumn(t, record, FlamegraphFieldCumulative, columns.cumulative)
	requireColumnChildren(t, record, columns.children)
}

func TestGenerateFlamegraphArrowUnsymbolized(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewGoAllocator()
	var err error

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)

	metastore := metastore.NewInProcessClient(l)

	mres, err := metastore.GetOrCreateMappings(ctx, &metastorepb.GetOrCreateMappingsRequest{
		Mappings: []*metastorepb.Mapping{{Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildId: "aID"}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	lres, err := metastore.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{
			{MappingId: m.Id, Address: 0xa1},
			{MappingId: m.Id, Address: 0xa2},
			{MappingId: m.Id, Address: 0xa3},
			{MappingId: m.Id, Address: 0xa4},
			{MappingId: m.Id, Address: 0xa5},
		},
	})
	require.NoError(t, err)
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]
	l3 := lres.Locations[2]
	l4 := lres.Locations[3]
	l5 := lres.Locations[4]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l4.Id, l3.Id, l2.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	s1 := sres.Stacktraces[0]
	s2 := sres.Stacktraces[1]
	s3 := sres.Stacktraces[2]

	tracer := trace.NewNoopTracerProvider().Tracer("")

	p, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
		Samples: []*parcaprofile.NormalizedSample{{
			StacktraceID: s1.Id,
			Value:        2,
		}, {
			StacktraceID: s2.Id,
			Value:        1,
		}, {
			StacktraceID: s3.Id,
			Value:        3,
		}},
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		aggregate []string
		// expectations
		rows       []flamegraphRow
		cumulative int64
		height     int32
		trimmed    int64
	}{
		// Aggregating by nothing or by function name yields the same result without function names.
		{
			name:      "aggregate-nothing", // raw
			aggregate: nil,
			// expectations
			cumulative: 6,
			height:     5,
			trimmed:    0, // TODO
			rows: []flamegraphRow{
				{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, Cumulative: 6, Children: []uint32{1}},            // 0
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, Cumulative: 6, Children: []uint32{2}},    // 1
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, Cumulative: 6, Children: []uint32{3}},    // 2
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, Cumulative: 4, Children: []uint32{4, 5}}, // 3
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, Cumulative: 1, Children: nil},            // 4
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, Cumulative: 3, Children: nil},            // 5
			},
		},
		{
			name:      "aggregate-function-name",
			aggregate: []string{FlamegraphFieldFunctionName},
			// expectations
			cumulative: 6,
			height:     5,
			trimmed:    0, // TODO
			rows: []flamegraphRow{
				{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, Cumulative: 6, Children: []uint32{1}},            // 0
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, Cumulative: 6, Children: []uint32{2}},    // 1
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, Cumulative: 6, Children: []uint32{3}},    // 2
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, Cumulative: 4, Children: []uint32{4, 5}}, // 3
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, Cumulative: 1, Children: nil},            // 4
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, Cumulative: 3, Children: nil},            // 5
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, tc.aggregate, 0)
			require.NoError(t, err)

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(len(tc.rows)), fa.NumRows())
			require.Equal(t, int64(16), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			columns := rowsToColumn(tc.rows)

			requireColumn(t, fa, FlamegraphFieldMappingStart, columns.mappingStart)
			requireColumn(t, fa, FlamegraphFieldMappingLimit, columns.mappingLimit)
			requireColumn(t, fa, FlamegraphFieldMappingOffset, columns.mappingOffset)
			requireColumnDict(t, fa, FlamegraphFieldMappingFile, columns.mappingFiles)
			requireColumnDict(t, fa, FlamegraphFieldMappingBuildID, columns.mappingBuildIDs)
			requireColumn(t, fa, FlamegraphFieldLocationAddress, columns.locationAddresses)
			requireColumn(t, fa, FlamegraphFieldLocationFolded, columns.locationFolded)
			requireColumn(t, fa, FlamegraphFieldCumulative, columns.cumulative)
			requireColumnChildren(t, fa, columns.children)
		})
	}
}

func TestParents(t *testing.T) {
	p := parent(-1)
	require.Equal(t, -1, p.Get())
	require.False(t, p.Has())
	p.Reset()
	require.Equal(t, -1, p.Get())
	require.False(t, p.Has())
	p.Set(1)
	require.Equal(t, 1, p.Get())
	require.True(t, p.Has())
	p.Set(2)
	require.Equal(t, 2, p.Get())
	require.True(t, p.Has())
	p.Reset()
	require.Equal(t, -1, p.Get())
	require.False(t, p.Has())
}
