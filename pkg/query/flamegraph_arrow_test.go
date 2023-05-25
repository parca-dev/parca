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
	"context"
	"testing"

	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlamegraphArrow(t *testing.T) {
	ctx := context.Background()
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
		}, {
			StacktraceID: s2.Id,
			Value:        1,
		}, {
			StacktraceID: s3.Id,
			Value:        3,
		}},
	})
	require.NoError(t, err)

	type row struct {
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
		Children           []uint32
		Cumulative         int64
	}

	for _, tc := range []struct {
		name    string
		groupBy []string
		// expectations
		numRows int64
		rows    []row
	}{{
		name:    "aggregate-default",
		groupBy: nil,
		// expectations
		numRows: 6,
		rows: []row{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Children: []uint32{1}},            // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Children: []uint32{2}},    // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 6, Children: []uint32{3}},    // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Children: []uint32{4, 5}}, // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Children: nil},            // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Children: nil},            // 5
		},
	}, {
		name:    "aggregate-never",
		groupBy: []string{FlamegraphFieldMappingFile, FlamegraphFieldFunctionName},
		// expectations
		numRows: 11,
		rows: []row{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 6, Children: []uint32{1, 3, 7}},    // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 2, Children: []uint32{2}},  // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 2, Children: nil},          // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 1, Children: []uint32{4}},  // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 1, Children: []uint32{5}},  // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Children: []uint32{6}},  // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Children: nil},          // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Children: []uint32{8}},  // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Children: []uint32{9}},  // 8
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Children: []uint32{10}}, // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Children: nil},          // 10
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			fa, err := GenerateFlamegraphArrow(ctx, tracer, p, tc.groupBy, 0)
			require.NoError(t, err)

			require.Equal(t, tc.numRows, fa.NumRows())
			require.Equal(t, int64(15), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			columns := struct {
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
				children            [][]uint32
				cumulative          []int64
			}{}
			for _, row := range tc.rows {
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
				columns.children = append(columns.children, row.Children)
				columns.cumulative = append(columns.cumulative, row.Cumulative)
			}

			require.Equal(t,
				columns.mappingStart,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldMappingStart)[0]).(*array.Uint64).Uint64Values(),
			)
			require.Equal(t,
				columns.mappingLimit,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldMappingLimit)[0]).(*array.Uint64).Uint64Values(),
			)
			require.Equal(t,
				columns.mappingOffset,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldMappingOffset)[0]).(*array.Uint64).Uint64Values(),
			)

			mappingFilesDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldMappingFile)[0]).(*array.Dictionary)
			mappingFilesString := mappingFilesDict.Dictionary().(*array.String)
			mappingFiles := make([]string, fa.NumRows())
			for i := 0; i < int(fa.NumRows()); i++ {
				mappingFiles[i] = mappingFilesString.Value(mappingFilesDict.GetValueIndex(i))
			}
			require.Equal(t, columns.mappingFiles, mappingFiles)

			mappingBuildIDDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldMappingBuildID)[0]).(*array.Dictionary)
			mappingBuildIDString := mappingBuildIDDict.Dictionary().(*array.String)
			mappingBuildID := make([]string, fa.NumRows())
			for i := 0; i < int(fa.NumRows()); i++ {
				mappingBuildID[i] = mappingBuildIDString.Value(mappingBuildIDDict.GetValueIndex(i))
			}
			require.Equal(t, columns.mappingBuildIDs, mappingBuildID)

			require.Equal(t,
				columns.locationAddresses,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldLocationAddress)[0]).(*array.Uint64).Uint64Values(),
			)

			locationFolded := make([]bool, fa.NumRows())
			for i := 0; i < int(fa.NumRows()); i++ {
				locationFolded[i] = fa.Column(fa.Schema().FieldIndices(FlamegraphFieldLocationFolded)[0]).(*array.Boolean).Value(i)
			}
			require.Equal(t, columns.locationFolded, locationFolded)

			require.Equal(t,
				columns.locationLines,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldLocationLine)[0]).(*array.Int64).Int64Values(),
			)

			require.Equal(t,
				columns.functionStartLines,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldFunctionStartLine)[0]).(*array.Int64).Int64Values(),
			)

			functionNameDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldFunctionName)[0]).(*array.Dictionary)
			functionNameString := functionNameDict.Dictionary().(*array.String)
			functionSystemNameDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldFunctionSystemName)[0]).(*array.Dictionary)
			functionSystemNameString := functionSystemNameDict.Dictionary().(*array.String)
			functionFileNameDict := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldFunctionFileName)[0]).(*array.Dictionary)
			functionFileNameString := functionFileNameDict.Dictionary().(*array.String)

			functionNames := make([]string, fa.NumRows())
			functionSystemNames := make([]string, fa.NumRows())
			functionFileNames := make([]string, fa.NumRows())
			for i := 0; i < int(fa.NumRows()); i++ {
				functionNames[i] = functionNameString.Value(functionNameDict.GetValueIndex(i))
				functionSystemNames[i] = functionSystemNameString.Value(functionSystemNameDict.GetValueIndex(i))
				functionFileNames[i] = functionFileNameString.Value(functionFileNameDict.GetValueIndex(i))
			}
			require.Equal(t, columns.functionNames, functionNames)
			require.Equal(t, columns.functionSystemNames, functionSystemNames)
			require.Equal(t, columns.functionFileNames, functionFileNames)

			children := make([][]uint32, fa.NumRows())
			list := fa.Column(fa.Schema().FieldIndices(FlamegraphFieldChildren)[0]).(*array.List)
			listValues := list.ListValues().(*array.Uint32).Uint32Values()
			for i := 0; i < int(fa.NumRows()); i++ {
				if !list.IsValid(i) {
					children[i] = nil
				} else {
					start, end := list.ValueOffsets(i)
					children[i] = listValues[start:end]
				}
			}
			require.Equal(t, columns.children, children)

			require.Equal(t,
				columns.cumulative,
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldCumulative)[0]).(*array.Int64).Int64Values(),
			)
			require.Equal(t,
				int(tc.numRows),
				fa.Column(fa.Schema().FieldIndices(FlamegraphFieldDiff)[0]).(*array.Int64).NullN(),
			)
		})
	}
}
