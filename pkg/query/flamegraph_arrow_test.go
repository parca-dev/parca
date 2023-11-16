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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/ipc"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

type flamegraphRow struct {
	LabelsOnly         bool
	MappingStart       uint64
	MappingLimit       uint64
	MappingOffset      uint64
	MappingFile        string
	MappingBuildID     string
	LocationAddress    uint64
	Inlined            bool
	LocationLine       uint8
	FunctionStartLine  uint8
	FunctionName       string
	FunctionSystemName string
	FunctionFilename   string
	Labels             map[string]string
	Children           []uint32
	Cumulative         uint8
	Diff               int8
}

type flamegraphColumns struct {
	labelsOnly          []bool
	mappingFiles        []string
	mappingBuildIDs     []string
	locationAddresses   []uint64
	inlined             []bool
	locationLines       []uint8
	functionStartLines  []uint8
	functionNames       []string
	functionSystemNames []string
	functionFileNames   []string
	labels              []map[string]string
	children            [][]uint32
	cumulative          []uint8
	diff                []int8
}

func rowsToColumn(rows []flamegraphRow) flamegraphColumns {
	columns := flamegraphColumns{}
	for _, row := range rows {
		columns.labelsOnly = append(columns.labelsOnly, row.LabelsOnly)
		columns.mappingFiles = append(columns.mappingFiles, row.MappingFile)
		columns.mappingBuildIDs = append(columns.mappingBuildIDs, row.MappingBuildID)
		columns.locationAddresses = append(columns.locationAddresses, row.LocationAddress)
		columns.locationLines = append(columns.locationLines, row.LocationLine)
		columns.inlined = append(columns.inlined, row.Inlined)
		columns.functionStartLines = append(columns.functionStartLines, row.FunctionStartLine)
		columns.functionNames = append(columns.functionNames, row.FunctionName)
		columns.functionSystemNames = append(columns.functionSystemNames, row.FunctionSystemName)
		columns.functionFileNames = append(columns.functionFileNames, row.FunctionFilename)
		columns.labels = append(columns.labels, row.Labels)
		columns.children = append(columns.children, row.Children)
		columns.cumulative = append(columns.cumulative, row.Cumulative)
		columns.diff = append(columns.diff, row.Diff)
	}
	return columns
}

func extractLabelColumns(t *testing.T, r arrow.Record) []map[string]string {
	pprofLabels := make([]map[string]string, r.NumRows())
	for i := 0; i < int(r.NumRows()); i++ {
		sampleLabels := map[string]string{}
		for j, f := range r.Schema().Fields() {
			if strings.HasPrefix(f.Name, profile.ColumnPprofLabelsPrefix) && r.Column(j).IsValid(i) {
				col := r.Column(r.Schema().FieldIndices(f.Name)[0]).(*array.Dictionary)
				dict := col.Dictionary().(*array.Binary)

				labelName := strings.TrimPrefix(f.Name, profile.ColumnPprofLabelsPrefix)
				sampleLabels[labelName] = string(dict.Value(col.GetValueIndex(i)))
			}
		}

		if len(sampleLabels) > 0 {
			pprofLabels[i] = sampleLabels
		}
	}

	return pprofLabels
}

func extractChildrenColumn(t *testing.T, r arrow.Record) [][]uint32 {
	children := make([][]uint32, r.NumRows())
	list := r.Column(r.Schema().FieldIndices(FlamegraphFieldChildren)[0]).(*array.List)
	listValues := list.ListValues().(*array.Uint32).Uint32Values()
	for i := 0; i < int(r.NumRows()); i++ {
		if !list.IsValid(i) {
			children[i] = nil
		} else {
			start, end := list.ValueOffsets(i)
			children[i] = listValues[start:end]
			// the children rows aren't sorted, so we sort them here to compare them
			if len(children[i]) > 0 {
				sort.Slice(children[i], func(j, k int) bool {
					return children[i][j] < children[i][k]
				})
			}
		}
	}

	return children
}

func extractColumn(t *testing.T, r arrow.Record, field string) any {
	fi := r.Schema().FieldIndices(field)
	require.Equal(t, 1, len(fi))

	arr := r.Column(fi[0])
	switch arr := arr.(type) {
	case *array.Int64:
		return arr.Int64Values()
	case *array.Uint64:
		return arr.Uint64Values()
	case *array.Boolean:
		vals := make([]bool, r.NumRows())
		for i := 0; i < int(r.NumRows()); i++ {
			vals[i] = arr.Value(i)
		}

		return vals
	case *array.Uint8:
		return arr.Uint8Values()
	case *array.Int8:
		return arr.Int8Values()
	case *array.Dictionary:
		dict := arr.Dictionary()
		switch dict := dict.(type) {
		case *array.Binary:
			vals := make([]string, r.NumRows())
			for i := 0; i < int(r.NumRows()); i++ {
				if arr.IsValid(i) {
					vals[i] = string(dict.Value(arr.GetValueIndex(i)))
				} else {
					vals[i] = array.NullValueStr
				}
			}

			return vals
		case *array.String:
			vals := make([]string, r.NumRows())
			for i := 0; i < int(r.NumRows()); i++ {
				if arr.IsValid(i) {
					vals[i] = dict.Value(arr.GetValueIndex(i))
				} else {
					vals[i] = array.NullValueStr
				}
			}

			return vals
		default:
			t.Fatalf("unsupported type %T", arr)
			return nil
		}
	default:
		t.Fatalf("unsupported type %T", arr)
		return nil
	}
}

func TestGenerateFlamegraphArrow(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)

	mc := metastore.NewInProcessClient(l)

	mres, err := mc.GetOrCreateMappings(ctx, &metastorepb.GetOrCreateMappingsRequest{
		Mappings: []*metastorepb.Mapping{
			{Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildId: "aID"},
			{Start: 2, Limit: 2, Offset: 0x1235, File: "b", BuildId: "bID"},
		},
	})
	require.NoError(t, err)
	m1 := mres.Mappings[0]
	m2 := mres.Mappings[1]

	fres, err := mc.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{
			{Name: "1", SystemName: "1", Filename: "1", StartLine: 1},
			{Name: "2", SystemName: "2", Filename: "2", StartLine: 2},
			{Name: "3", SystemName: "3", Filename: "3", StartLine: 3},
			{Name: "4", SystemName: "4", Filename: "4", StartLine: 4},
			{Name: "5", SystemName: "5", Filename: "5", StartLine: 5},
			{Name: "2", SystemName: "6", Filename: "6", StartLine: 6}, // gets merged with function name 2 but everything else differs.
		},
	})
	require.NoError(t, err)
	f1 := fres.Functions[0]
	f2 := fres.Functions[1]
	f3 := fres.Functions[2]
	f4 := fres.Functions[3]
	f5 := fres.Functions[4]
	f6 := fres.Functions[5]

	lres, err := mc.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			MappingId: m1.Id,
			Address:   0xa1,
			Lines: []*metastorepb.Line{{
				FunctionId: f1.Id,
				Line:       1,
			}},
		}, {
			MappingId: m1.Id,
			Address:   0xa2,
			Lines: []*metastorepb.Line{{
				FunctionId: f2.Id,
				Line:       2,
			}},
		}, {
			MappingId: m1.Id,
			Address:   0xa3,
			Lines: []*metastorepb.Line{{
				FunctionId: f3.Id,
				Line:       3,
			}},
		}, {
			MappingId: m1.Id,
			Address:   0xa4,
			Lines: []*metastorepb.Line{{
				FunctionId: f4.Id,
				Line:       4,
			}},
		}, {
			MappingId: m1.Id,
			Address:   0xa5,
			Lines: []*metastorepb.Line{{
				FunctionId: f5.Id,
				Line:       5,
			}},
		}, {
			MappingId: m2.Id,
			Address:   0xa6,
			Lines: []*metastorepb.Line{{
				FunctionId: f6.Id,
				Line:       6,
			}},
		}},
	})
	require.NoError(t, err)
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]
	l3 := lres.Locations[2]
	l4 := lres.Locations[3]
	l5 := lres.Locations[4]
	l6 := lres.Locations[5]

	sres, err := mc.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l4.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l6.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	s1 := sres.Stacktraces[0]
	s2 := sres.Stacktraces[1]
	s3 := sres.Stacktraces[2]
	s4 := sres.Stacktraces[3]
	s5 := sres.Stacktraces[4]

	tracer := trace.NewNoopTracerProvider().Tracer("")

	p, err := parcacol.NewProfileSymbolizer(tracer, mc).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
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
		}, {
			// this is the same stack as s2 but with a different label
			StacktraceID: s4.Id,
			Value:        4,
			Label:        map[string]string{"goroutine": "2"},
		}, {
			StacktraceID: s5.Id,
			Value:        1,
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
		name:      "aggregate-function-name",
		aggregate: []string{FlamegraphFieldFunctionName},
		// expectations
		cumulative: 11,
		height:     5,
		trimmed:    0,
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Labels: nil, Children: []uint32{1}}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 11, Labels: nil, Children: []uint32{2}},                                                                  // 1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0x0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Labels: nil, Children: []uint32{3}},              // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Labels: nil, Children: []uint32{4, 5}},                                                                // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                                                           // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Labels: nil, Children: nil},                                                                           // 5
		},
	}, {
		name:      "aggregate-pprof-labels",
		aggregate: []string{FlamegraphFieldLabels},
		// expectations
		cumulative: 11,
		height:     6,
		trimmed:    0,
		rows: []flamegraphRow{
			// root
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Labels: nil, Children: []uint32{1, 6, 11}}, // 0
			// stack 1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}, LabelsOnly: true}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{3}},                                                                          // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{4}},                                                                          // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{5}},                                                                          // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: nil},                                                                                  // 5
			// stack 2
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{7}, LabelsOnly: true}, // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{8}},                                                                          // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{9}},                                                                          // 8
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{10}},                                                                         // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: nil},                                                                                  // 10
			// stack 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Labels: nil, Children: []uint32{12}},                                                        // 11
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Labels: nil, Children: []uint32{13}}, // 12
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Labels: nil, Children: []uint32{14}},                                                        // 13
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                                                 // 14
		},
	}, {
		name:      "aggregate-mapping-file",
		aggregate: []string{FlamegraphFieldMappingFile},
		// expectations
		cumulative: 11,
		height:     5,
		trimmed:    0,
		rows: []flamegraphRow{
			// This aggregates all the rows with the same mapping file, meaning that we only keep one flamegraphRow per stack depth in this example.
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Labels: nil, Children: []uint32{1}}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 11, Labels: nil, Children: []uint32{2, 6}},                                                               // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 10, Labels: nil, Children: []uint32{3}},                                                                  // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Labels: nil, Children: []uint32{4, 5}},                                                                // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                                                           // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Labels: nil, Children: nil},                                                                           // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "b", MappingBuildID: "bID", LocationAddress: 0xa6, LocationLine: 6, FunctionStartLine: 6, FunctionName: "2", FunctionSystemName: "6", FunctionFilename: "6", Cumulative: 1, Labels: nil, Children: nil},                                                                           // 5
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			np, err := OldProfileToArrowProfile(p)
			require.NoError(t, err)

			np.Samples = []arrow.Record{
				np.Samples[0].NewSlice(0, 2),
				np.Samples[0].NewSlice(2, 5),
			}

			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, np, tc.aggregate, 0)
			require.NoError(t, err)
			defer fa.Release()

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(14), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			expectedColumns := rowsToColumn(tc.rows)

			fc := newFlamegraphComparer(t)
			fc.convert(fa)
			fc.compare(expectedColumns)
		})
	}
}

type flamegraphComparer struct {
	t      *testing.T
	stack  *flamegraphComparerStack
	actual flamegraphColumns
}

func newFlamegraphComparer(t *testing.T) *flamegraphComparer {
	return &flamegraphComparer{
		t:     t,
		stack: &flamegraphComparerStack{elements: []flamegraphCompareElement{{row: 0}}}, // start with the root
	}
}

func (c *flamegraphComparer) convert(r arrow.Record) {
	c.t.Helper()
	c.actual = flamegraphColumns{
		labelsOnly:          extractColumn(c.t, r, FlamegraphFieldLabelsOnly).([]bool),
		mappingFiles:        extractColumn(c.t, r, FlamegraphFieldMappingFile).([]string),
		mappingBuildIDs:     extractColumn(c.t, r, FlamegraphFieldMappingBuildID).([]string),
		locationAddresses:   extractColumn(c.t, r, FlamegraphFieldLocationAddress).([]uint64),
		inlined:             extractColumn(c.t, r, FlamegraphFieldInlined).([]bool),
		locationLines:       extractColumn(c.t, r, FlamegraphFieldLocationLine).([]uint8),
		functionStartLines:  extractColumn(c.t, r, FlamegraphFieldFunctionStartLine).([]uint8),
		functionNames:       extractColumn(c.t, r, FlamegraphFieldFunctionName).([]string),
		functionSystemNames: extractColumn(c.t, r, FlamegraphFieldFunctionSystemName).([]string),
		functionFileNames:   extractColumn(c.t, r, FlamegraphFieldFunctionFileName).([]string),
		labels:              extractLabelColumns(c.t, r),
		children:            extractChildrenColumn(c.t, r),
		cumulative:          extractColumn(c.t, r, FlamegraphFieldCumulative).([]uint8),
		diff:                extractColumn(c.t, r, FlamegraphFieldDiff).([]int8),
	}
}

func (c *flamegraphComparer) compare(expected flamegraphColumns) {
	c.t.Helper()

	order := make([]int, 0, len(c.actual.cumulative))
	sortedChildren := make([][]uint32, len(c.actual.cumulative))

	var i int
	for c.stack.Len() > 0 {
		r := c.stack.Pop()
		order = append(order, r.row)
		if r.row != 0 {
			sortedChildren[r.parent] = append(sortedChildren[r.parent], uint32(i))
		}

		children := c.actual.children[r.row]
		// This will sort the children by their values to guarantee a deterministic order for tests.
		sort.Slice(children, func(a, b int) bool {
			labelsOnlyA := c.actual.labelsOnly[children[a]]
			labelsOnlyB := c.actual.labelsOnly[children[b]]

			if labelsOnlyA && labelsOnlyB {
				labelsA := labels.FromMap(c.actual.labels[children[a]]).String()
				labelsB := labels.FromMap(c.actual.labels[children[b]]).String()
				return labelsA < labelsB
			}
			if labelsOnlyA && !labelsOnlyB {
				return true
			}
			if c.actual.functionNames[children[a]] < c.actual.functionNames[children[b]] {
				return true
			}
			if c.actual.functionNames[children[a]] != "" && c.actual.functionNames[children[b]] != "" {
				addrA := c.actual.locationAddresses[children[a]]
				addrB := c.actual.locationAddresses[children[b]]
				return addrA < addrB
			}

			return false
		})

		slices.Reverse(children) // since using a stack, we need to reverse the children to get the correct order
		for _, child := range children {
			c.stack.Push(flamegraphCompareElement{parent: i, row: int(child)})
		}
		i++
	}

	require.Equal(c.t, expected.labelsOnly, reorder(c.actual.labelsOnly, order))
	require.Equal(c.t, expected.mappingFiles, reorder(c.actual.mappingFiles, order))
	require.Equal(c.t, expected.mappingBuildIDs, reorder(c.actual.mappingBuildIDs, order))
	require.Equal(c.t, expected.locationAddresses, reorder(c.actual.locationAddresses, order))
	require.Equal(c.t, expected.inlined, reorder(c.actual.inlined, order))
	require.Equal(c.t, expected.locationLines, reorder(c.actual.locationLines, order))
	require.Equal(c.t, expected.functionStartLines, reorder(c.actual.functionStartLines, order))
	require.Equal(c.t, expected.functionNames, reorder(c.actual.functionNames, order))
	require.Equal(c.t, expected.functionSystemNames, reorder(c.actual.functionSystemNames, order))
	require.Equal(c.t, expected.functionFileNames, reorder(c.actual.functionFileNames, order))
	require.Equal(c.t, expected.labels, reorder(c.actual.labels, order))
	require.Equal(c.t, expected.cumulative, reorder(c.actual.cumulative, order), order)
	require.Equal(c.t, expected.diff, reorder(c.actual.diff, order))
	require.Equal(c.t, expected.children, sortedChildren)
}

func reorder[T any](slice []T, order []int) []T {
	res := make([]T, len(slice))
	for i, o := range order {
		res[i] = slice[o]
	}
	return res
}

type flamegraphCompareElement struct {
	parent int
	row    int
}

type flamegraphComparerStack struct {
	elements []flamegraphCompareElement
}

func (s *flamegraphComparerStack) Push(e flamegraphCompareElement) {
	s.elements = append(s.elements, e)
}

func (s *flamegraphComparerStack) Pop() flamegraphCompareElement {
	e := s.elements[len(s.elements)-1]
	s.elements = s.elements[:len(s.elements)-1]
	return e
}

func (s *flamegraphComparerStack) Len() int {
	return len(s.elements)
}

func TestGenerateFlamegraphArrowEmpty(t *testing.T) {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	// empty profile
	// basically the same as querying a time range with no data.
	p := profile.Profile{}

	record, total, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, []string{FlamegraphFieldFunctionName}, 0)
	require.NoError(t, err)
	defer record.Release()

	require.Equal(t, int64(0), total)
	require.Equal(t, int32(1), height)
	require.Equal(t, int64(0), trimmed)
	require.Equal(t, int64(13), record.NumCols())
	require.Equal(t, int64(1), record.NumRows())
}

func TestGenerateFlamegraphArrowWithInlined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	counter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "parca_test_counter",
		Help: "parca_test_counter",
	})
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
			{Line: 200, Function: functions[2]},
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
	normalizer := parcacol.NewNormalizer(metastore, true, counter)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]string{}, p, false, nil)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.NewProfileSymbolizer(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	newProfile, err := OldProfileToArrowProfile(symbolizedProfile)
	require.NoError(t, err)

	record, total, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, newProfile, []string{FlamegraphFieldFunctionName}, 0)
	require.NoError(t, err)
	defer record.Release()

	fmt.Println(record)

	require.Equal(t, int64(1), total)
	require.Equal(t, int32(5), height)
	require.Equal(t, int64(0), trimmed)

	require.Equal(t, int64(13), record.NumCols())
	require.Equal(t, int64(5), record.NumRows())

	rows := []flamegraphRow{
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "(null)", FunctionSystemName: "(null)", FunctionFilename: "(null)", Cumulative: 1, Labels: nil, Children: []uint32{1}},                                                                                        // 0
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa1, LocationLine: 173, FunctionStartLine: 0, FunctionName: "net.(*netFD).accept", FunctionSystemName: "net.(*netFD).accept", FunctionFilename: "net/fd_unix.go", Cumulative: 1, Labels: nil, Children: []uint32{2}},                                                 // 1
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa2, LocationLine: 200, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).waitRead", FunctionSystemName: "internal/poll.(*pollDesc).waitRead", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: []uint32{3}}, // 2
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa2, LocationLine: 89, FunctionStartLine: 0, FunctionName: "internal/poll.(*FD).Accept", FunctionSystemName: "internal/poll.(*FD).Accept", FunctionFilename: "internal/poll/fd_unix.go", Cumulative: 1, Labels: nil, Children: []uint32{4}, Inlined: true},           // 3
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa3, LocationLine: 84, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).wait", FunctionSystemName: "internal/poll.(*pollDesc).wait", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: nil},                  // 4
	}
	expectedColumns := rowsToColumn(rows)

	fc := newFlamegraphComparer(t)
	fc.convert(record)
	fc.compare(expectedColumns)
}

func TestGenerateFlamegraphArrowUnsymbolized(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
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

	p, err := parcacol.NewProfileSymbolizer(tracer, metastore).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
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
		{
			name:      "aggregate-function-name",
			aggregate: []string{FlamegraphFieldFunctionName},
			// expectations
			cumulative: 6,
			height:     5,
			trimmed:    0,
			rows: []flamegraphRow{
				{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "(null)", MappingBuildID: "(null)", LocationAddress: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Children: []uint32{1}},    // 0
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Children: []uint32{2}},    // 1
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Children: []uint32{3}},    // 2
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Children: []uint32{4, 5}}, // 3
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 3, Children: nil},            // 4
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 1, Children: nil},            // 5
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			np, err := OldProfileToArrowProfile(p)
			require.NoError(t, err)
			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, np, tc.aggregate, 0)
			require.NoError(t, err)
			defer fa.Release()

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(len(tc.rows)), fa.NumRows())
			require.Equal(t, int64(13), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			expectedColumns := rowsToColumn(tc.rows)

			fc := newFlamegraphComparer(t)
			fc.convert(fa)
			fc.compare(expectedColumns)
		})
	}
}

func TestGenerateFlamegraphArrowTrimming(t *testing.T) {
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
		Mappings: []*metastorepb.Mapping{{
			File: "a",
		}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := metastore.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{
			{Name: "1"},
			{Name: "2"},
			{Name: "3"},
			{Name: "4"},
			{Name: "5"},
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
			Lines: []*metastorepb.Line{{
				FunctionId: f1.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f2.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f3.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f4.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f5.Id,
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

	p, err := parcacol.NewProfileSymbolizer(tracer, metastore).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
		Samples: []*parcaprofile.NormalizedSample{{
			StacktraceID: s1.Id,
			Value:        10,
		}, {
			// The following two samples are trimmed from the flamegraph.
			StacktraceID: s2.Id,
			Value:        1,
		}, {
			StacktraceID: s3.Id,
			Value:        3,
		}},
	})
	require.NoError(t, err)

	np, err := OldProfileToArrowProfile(p)
	require.NoError(t, err)

	fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, np, []string{FlamegraphFieldFunctionName}, float32(0.5))
	require.NoError(t, err)

	require.Equal(t, int64(14), cumulative)
	require.Equal(t, int32(5), height)
	require.Equal(t, int64(4), trimmed)
	require.Equal(t, int64(3), fa.NumRows())
	require.Equal(t, int64(13), fa.NumCols())

	// TODO: MappingBuildID and FunctionSystemNames shouldn't be "" but null?
	rows := []flamegraphRow{
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 14, Children: []uint32{1}}, // 0
		{MappingFile: "a", MappingBuildID: "", FunctionName: "1", FunctionSystemName: "", FunctionFilename: "", Cumulative: 14, Children: []uint32{2}},                                                                               // 1
		{MappingFile: "a", MappingBuildID: "", FunctionName: "2", FunctionSystemName: "", FunctionFilename: "", Cumulative: 14, Children: nil},                                                                                       // 2
	}
	expectedColumns := rowsToColumn(rows)

	fc := newFlamegraphComparer(t)
	fc.convert(fa)
	fc.compare(expectedColumns)
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

func TestMapsIntersection(t *testing.T) {
	// empty
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{}))
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{{}}))
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{{}, {}}))
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{{}, {"thread": "1"}}))
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{{"thread": "1"}, {}}))
	// one
	require.Equal(t, map[string]string{"thread": "1"}, mapsIntersection([]map[string]string{{"thread": "1"}}))
	require.Equal(t, map[string]string{"thread": "1"}, mapsIntersection([]map[string]string{
		{"thread": "1"},
		{"thread": "1"},
	}))
	require.Equal(t, map[string]string{"thread": "1"}, mapsIntersection([]map[string]string{
		{"thread": "1"},
		{"thread": "1"},
		{"thread": "1"},
	}))
	// two
	require.Equal(t, map[string]string{"thread": "1", "thread_name": "name"}, mapsIntersection([]map[string]string{
		{"thread": "1", "thread_name": "name"},
		{"thread": "1", "thread_name": "name"},
	}))
	// different
	require.Equal(t, map[string]string{}, mapsIntersection([]map[string]string{
		{"thread": "1"},
		{"thread": "2"},
	}))
	require.Equal(t, map[string]string{"thread_name": "name"}, mapsIntersection([]map[string]string{
		{"thread": "1", "thread_name": "name"},
		{"thread": "2", "thread_name": "name"},
	}))
}

func BenchmarkArrowFlamegraph(b *testing.B) {
	fileContent, err := os.ReadFile("testdata/profile-with-labels.pb.gz")
	require.NoError(b, err)

	gz, err := gzip.NewReader(bytes.NewBuffer(fileContent))
	require.NoError(b, err)

	decompressed, err := io.ReadAll(gz)
	require.NoError(b, err)

	p := &pprofpb.Profile{}
	require.NoError(b, p.UnmarshalVT(decompressed))

	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(b, err)

	np, err := PprofToSymbolizedProfile(parcaprofile.MetaFromPprof(p, "memory", 0), pp, 0)
	require.NoError(b, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := GenerateFlamegraphArrow(
			context.Background(),
			memory.DefaultAllocator,
			tracer,
			np,
			nil,
			0,
		)
		require.NoError(b, err)
	}
}

func TestCompactDictionary(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	builder := array.NewStringBuilder(mem)
	builder.AppendValues([]string{"a", "b", "c"}, nil)
	values := builder.NewArray()
	defer values.Release()
	defer builder.Release()

	// Test two values and a single null.
	index1Builder := array.NewInt32Builder(mem)
	index1Builder.AppendValues([]int32{0, 0}, nil)
	index1Builder.AppendNull()
	index1Builder.AppendValues([]int32{0, 1}, nil)
	index1 := index1Builder.NewArray()
	compArr, err := compactDictionary(mem, array.NewDictionaryArray(
		&arrow.DictionaryType{IndexType: index1.DataType(), ValueType: values.DataType()},
		index1,
		values,
	))
	require.NoError(t, err)
	require.Equal(t, 2, compArr.Dictionary().Len()) // make sure we actually compact values.
	require.Equal(t, "a", compArr.Dictionary().ValueStr(compArr.GetValueIndex(0)))
	require.Equal(t, "a", compArr.Dictionary().ValueStr(compArr.GetValueIndex(1)))
	require.True(t, compArr.IsNull(2))
	require.Equal(t, "a", compArr.Dictionary().ValueStr(compArr.GetValueIndex(3)))
	require.Equal(t, "b", compArr.Dictionary().ValueStr(compArr.GetValueIndex(4)))
	index1Builder.Release()
	index1.Release()
	compArr.Release()

	// Just one single underlying value.
	index2Builder := array.NewInt32Builder(mem)
	index2Builder.Append(2)
	index2 := index2Builder.NewArray()
	compArr, err = compactDictionary(mem, array.NewDictionaryArray(
		&arrow.DictionaryType{IndexType: index2.DataType(), ValueType: values.DataType()},
		index2,
		values,
	))
	require.NoError(t, err)
	require.Equal(t, 1, compArr.Dictionary().Len()) // make sure we actually compact values.
	require.Equal(t, "c", compArr.Dictionary().ValueStr(compArr.GetValueIndex(0)))
	index2Builder.Release()
	index2.Release()
	compArr.Release()

	// Just one single null, no actual values.
	index3Builder := array.NewInt32Builder(mem)
	index3Builder.AppendNull()
	index3 := index3Builder.NewArray()
	compArr, err = compactDictionary(mem, array.NewDictionaryArray(
		&arrow.DictionaryType{IndexType: index3.DataType(), ValueType: values.DataType()},
		index3,
		values,
	))
	require.NoError(t, err)
	require.Equal(t, 0, compArr.Dictionary().Len()) // make sure we actually compact values.
	require.True(t, compArr.IsNull(0))
	index3Builder.Release()
	index3.Release()
	compArr.Release()
}

func TestRecordStats(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	gz, err := gzip.NewReader(bytes.NewBuffer(fileContent))
	require.NoError(t, err)

	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(decompressed))

	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	np, err := PprofToSymbolizedProfile(parcaprofile.MetaFromPprof(p, "memory", 0), pp, 0)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")

	record, _, _, _, err := generateFlamegraphArrowRecord(
		context.Background(),
		mem,
		tracer,
		np,
		nil,
		0,
	)
	require.NoError(t, err)
	defer record.Release()

	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(mem),
	)
	defer w.Close()

	err = w.Write(record)
	require.NoError(t, err)

	fmt.Println("Encoded:", buf.Len())
	fmt.Println(recordStats(record))
}

func TestAllFramesFiltered(t *testing.T) {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	fileContent, err := os.ReadFile("testdata/no-python.pb.gz")
	require.NoError(t, err)

	gz, err := gzip.NewReader(bytes.NewBuffer(fileContent))
	require.NoError(t, err)

	decompressed, err := io.ReadAll(gz)
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(decompressed))

	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	np, err := PprofToSymbolizedProfile(parcaprofile.MetaFromPprof(p, "cpu", 0), pp, 0)
	require.NoError(t, err)

	// This is a regression test, what we want to achieve here is the input
	// data being multiple samples, but all frames are filtered out. What
	// happened is the input data contains no python frames, but only python
	// frames were requested.
	np.Samples, _, err = FilterProfileData(ctx, tracer, mem, np.Samples, "", &pb.RuntimeFilter{
		ShowInterpretedOnly: true,
	})
	require.NoError(t, err)

	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		nil,
		0,
	)
	require.NoError(t, err)
	defer record.Release()

	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(mem),
	)
	defer w.Close()

	err = w.Write(record)
	require.NoError(t, err)
}
