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
	"io"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
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
	LocationFolded     bool
	LocationLine       int64
	FunctionStartLine  int64
	FunctionName       string
	FunctionSystemName string
	FunctionFilename   string
	Labels             map[string]string
	Children           []uint32
	Cumulative         int64
	Diff               int64
}

type flamegraphColumns struct {
	labelsOnly          []bool
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
	diff                []int64
}

func (c flamegraphColumns) slice(start, end int) flamegraphColumns {
	return flamegraphColumns{
		labelsOnly:          c.labelsOnly[start:end],
		mappingStart:        c.mappingStart[start:end],
		mappingLimit:        c.mappingLimit[start:end],
		mappingOffset:       c.mappingOffset[start:end],
		mappingFiles:        c.mappingFiles[start:end],
		mappingBuildIDs:     c.mappingBuildIDs[start:end],
		locationAddresses:   c.locationAddresses[start:end],
		locationFolded:      c.locationFolded[start:end],
		locationLines:       c.locationLines[start:end],
		functionStartLines:  c.functionStartLines[start:end],
		functionNames:       c.functionNames[start:end],
		functionSystemNames: c.functionSystemNames[start:end],
		functionFileNames:   c.functionFileNames[start:end],
		labels:              c.labels[start:end],
		children:            c.children[start:end],
		cumulative:          c.cumulative[start:end],
		diff:                c.diff[start:end],
	}
}

func (c flamegraphColumns) swap(i, j int) {
	c.labelsOnly[i], c.labelsOnly[j] = c.labelsOnly[j], c.labelsOnly[i]
	c.mappingStart[i], c.mappingStart[j] = c.mappingStart[j], c.mappingStart[i]
	c.mappingLimit[i], c.mappingLimit[j] = c.mappingLimit[j], c.mappingLimit[i]
	c.mappingOffset[i], c.mappingOffset[j] = c.mappingOffset[j], c.mappingOffset[i]
	c.mappingFiles[i], c.mappingFiles[j] = c.mappingFiles[j], c.mappingFiles[i]
	c.mappingBuildIDs[i], c.mappingBuildIDs[j] = c.mappingBuildIDs[j], c.mappingBuildIDs[i]
	c.locationAddresses[i], c.locationAddresses[j] = c.locationAddresses[j], c.locationAddresses[i]
	c.locationFolded[i], c.locationFolded[j] = c.locationFolded[j], c.locationFolded[i]
	c.locationLines[i], c.locationLines[j] = c.locationLines[j], c.locationLines[i]
	c.functionStartLines[i], c.functionStartLines[j] = c.functionStartLines[j], c.functionStartLines[i]
	c.functionNames[i], c.functionNames[j] = c.functionNames[j], c.functionNames[i]
	c.functionSystemNames[i], c.functionSystemNames[j] = c.functionSystemNames[j], c.functionSystemNames[i]
	c.functionFileNames[i], c.functionFileNames[j] = c.functionFileNames[j], c.functionFileNames[i]
	c.labels[i], c.labels[j] = c.labels[j], c.labels[i]
	c.children[i], c.children[j] = c.children[j], c.children[i]
	c.cumulative[i], c.cumulative[j] = c.cumulative[j], c.cumulative[i]
	c.diff[i], c.diff[j] = c.diff[j], c.diff[i]
}

type flamegraphColumnSorter struct {
	columns flamegraphColumns
	less    func(c flamegraphColumns, a, b int) bool
	slices  [][2]int
}

func (s *flamegraphColumnSorter) Len() int           { return len(s.columns.labelsOnly) }
func (s *flamegraphColumnSorter) Swap(i, j int)      { s.columns.swap(i, j) }
func (s *flamegraphColumnSorter) Less(i, j int) bool { return s.less(s.columns, i, j) }

func (s *flamegraphColumnSorter) sort(columns flamegraphColumns) {
	for _, slice := range s.slices {
		s.columns = columns.slice(slice[0], slice[1])
		sort.Sort(s)
	}
}

func rowsToColumn(rows []flamegraphRow) flamegraphColumns {
	columns := flamegraphColumns{}
	for _, row := range rows {
		columns.labelsOnly = append(columns.labelsOnly, row.LabelsOnly)
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
		columns.diff = append(columns.diff, row.Diff)
	}
	return columns
}

func fgRecordToColumns(t *testing.T, r arrow.Record) flamegraphColumns {
	return flamegraphColumns{
		labelsOnly:          extractColumn(t, r, FlamegraphFieldLabelsOnly).([]bool),
		mappingStart:        extractColumn(t, r, FlamegraphFieldMappingStart).([]uint64),
		mappingLimit:        extractColumn(t, r, FlamegraphFieldMappingLimit).([]uint64),
		mappingOffset:       extractColumn(t, r, FlamegraphFieldMappingOffset).([]uint64),
		mappingFiles:        extractColumn(t, r, FlamegraphFieldMappingFile).([]string),
		mappingBuildIDs:     extractColumn(t, r, FlamegraphFieldMappingBuildID).([]string),
		locationAddresses:   extractColumn(t, r, FlamegraphFieldLocationAddress).([]uint64),
		locationFolded:      extractColumn(t, r, FlamegraphFieldLocationFolded).([]bool),
		locationLines:       extractColumn(t, r, FlamegraphFieldLocationLine).([]int64),
		functionStartLines:  extractColumn(t, r, FlamegraphFieldFunctionStartLine).([]int64),
		functionNames:       extractColumn(t, r, FlamegraphFieldFunctionName).([]string),
		functionSystemNames: extractColumn(t, r, FlamegraphFieldFunctionSystemName).([]string),
		functionFileNames:   extractColumn(t, r, FlamegraphFieldFunctionFileName).([]string),
		labels:              extractLabelColumns(t, r),
		children:            extractChildrenColumn(t, r),
		cumulative:          extractColumn(t, r, FlamegraphFieldCumulative).([]int64),
		diff:                extractColumn(t, r, FlamegraphFieldDiff).([]int64),
	}
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
	case *array.Dictionary:
		dict := arr.Dictionary()
		switch dict := dict.(type) {
		case *array.Binary:
			vals := make([]string, r.NumRows())
			for i := 0; i < int(r.NumRows()); i++ {
				if arr.IsValid(i) {
					vals[i] = string(dict.Value(arr.GetValueIndex(i)))
				} else {
					vals[i] = "(null)"
				}
			}

			return vals
		case *array.String:
			vals := make([]string, r.NumRows())
			for i := 0; i < int(r.NumRows()); i++ {
				if arr.IsValid(i) {
					vals[i] = dict.Value(arr.GetValueIndex(i))
				} else {
					vals[i] = "(null)"
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
		Mappings: []*metastorepb.Mapping{{Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildId: "aID"}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := mc.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
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

	lres, err := mc.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
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

	sres, err := mc.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l4.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	s1 := sres.Stacktraces[0]
	s2 := sres.Stacktraces[1]
	s3 := sres.Stacktraces[2]
	s4 := sres.Stacktraces[3]

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
		}},
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		name      string
		aggregate []string
		// expectations
		rows       []flamegraphRow
		sorter     *flamegraphColumnSorter
		cumulative int64
		height     int32
		trimmed    int64
	}{{
		name:      "aggregate-function-name",
		aggregate: []string{FlamegraphFieldFunctionName},
		// expectations
		cumulative: 10,
		height:     5,
		trimmed:    0,
		sorter: &flamegraphColumnSorter{
			slices: [][2]int{{4, 6}}, // lines 4 and 5 are not in stable order in the result so we need to sort them for the test to be deterministic, so the range we want to sort is [4, 6)
			less: func(columns flamegraphColumns, i, j int) bool {
				return columns.functionNames[i] > columns.functionNames[j]
			},
		},
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 10, Labels: nil, Children: []uint32{1}}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 10, Labels: nil, Children: []uint32{2}},                                                                  // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 10, Labels: nil, Children: []uint32{3}},                                                                  // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Labels: nil, Children: []uint32{4, 5}},                                                                // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Labels: nil, Children: nil},                                                                           // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                                                           // 5
		},
	}, {
		name:      "aggregate-pprof-labels",
		aggregate: []string{FlamegraphFieldLabels},
		// expectations
		cumulative: 10,
		height:     6,
		trimmed:    0,
		rows: []flamegraphRow{
			// level 0 - root
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 10, Labels: nil, Children: []uint32{1, 2, 3}}, // 0
			// level 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: nil, Children: []uint32{4}},                                                                                                          // 1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{5}, LabelsOnly: true}, // 2
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{6}, LabelsOnly: true}, // 3
			// level 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: nil, Children: []uint32{7}},                                 // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{8}}, // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{9}}, // 5
			// level 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Labels: nil, Children: []uint32{10}},                                 // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{11}}, // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{12}}, // 8
			// level 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                          // 10
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: []uint32{13}}, // 12
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{14}}, // 11
			// level 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Labels: map[string]string{"goroutine": "2"}, Children: nil}, // 13
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: nil}, // 14
		},
	}, {
		name:      "aggregate-mapping-file",
		aggregate: []string{FlamegraphFieldMappingFile},
		// expectations
		cumulative: 10,
		height:     5,
		trimmed:    0, // TODO
		sorter: &flamegraphColumnSorter{
			slices: [][2]int{{4, 6}}, // lines 4 and 5 are not in stable order in the result so we need to sort them for the test to be deterministic, so the range we want to sort is [4, 6)
			less: func(columns flamegraphColumns, i, j int) bool {
				return columns.functionNames[i] > columns.functionNames[j]
			},
		},
		rows: []flamegraphRow{
			// This aggregates all the rows with the same mapping file, meaning that we only keep one flamegraphRow per stack depth in this example.
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 10, Labels: nil, Children: []uint32{1}}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 10, Labels: nil, Children: []uint32{2}},                                                                  // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 10, Labels: nil, Children: []uint32{3}},                                                                  // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Labels: nil, Children: []uint32{4, 5}},                                                                // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Labels: nil, Children: nil},                                                                           // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Labels: nil, Children: nil},                                                                           // 4
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "aggregate-pprof-labels" {
				t.Skip("TODO: requires custom comparison logic due to ordering")
			}
			np, err := OldProfileToArrowProfile(p)
			require.NoError(t, err)

			np.Samples = []arrow.Record{
				np.Samples[0].NewSlice(0, 2),
				np.Samples[0].NewSlice(2, 4),
			}

			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, np, tc.aggregate, 0)
			require.NoError(t, err)
			defer fa.Release()

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(17), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			expectedColumns := rowsToColumn(tc.rows)
			actualColumns := fgRecordToColumns(t, fa)

			if tc.sorter != nil {
				tc.sorter.sort(actualColumns)
			}

			require.Equal(t, expectedColumns.labelsOnly, actualColumns.labelsOnly)
			require.Equal(t, expectedColumns.mappingStart, actualColumns.mappingStart)
			require.Equal(t, expectedColumns.mappingLimit, actualColumns.mappingLimit)
			require.Equal(t, expectedColumns.mappingOffset, actualColumns.mappingOffset)
			require.Equal(t, expectedColumns.mappingFiles, actualColumns.mappingFiles)
			require.Equal(t, expectedColumns.mappingBuildIDs, actualColumns.mappingBuildIDs)
			require.Equal(t, expectedColumns.locationAddresses, actualColumns.locationAddresses)
			require.Equal(t, expectedColumns.locationFolded, actualColumns.locationFolded)
			require.Equal(t, expectedColumns.locationLines, actualColumns.locationLines)
			require.Equal(t, expectedColumns.functionStartLines, actualColumns.functionStartLines)
			require.Equal(t, expectedColumns.functionNames, actualColumns.functionNames)
			require.Equal(t, expectedColumns.functionSystemNames, actualColumns.functionSystemNames)
			require.Equal(t, expectedColumns.functionFileNames, actualColumns.functionFileNames)
			require.Equal(t, expectedColumns.labels, actualColumns.labels)
			require.Equal(t, expectedColumns.children, actualColumns.children)
			require.Equal(t, expectedColumns.cumulative, actualColumns.cumulative)
			require.Equal(t, expectedColumns.diff, actualColumns.diff)
		})
	}
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
	require.Equal(t, int64(16), record.NumCols())
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

	require.Equal(t, int64(1), total)
	require.Equal(t, int32(5), height)
	require.Equal(t, int64(0), trimmed)

	require.Equal(t, int64(16), record.NumCols())
	require.Equal(t, int64(5), record.NumRows())

	rows := []flamegraphRow{
		{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0, LocationFolded: false, LocationLine: 0, FunctionStartLine: 0, FunctionName: "(null)", FunctionSystemName: "(null)", FunctionFilename: "(null)", Cumulative: 1, Labels: nil, Children: []uint32{1}},                                                                                             // 0
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 173, FunctionStartLine: 0, FunctionName: "net.(*netFD).accept", FunctionSystemName: "net.(*netFD).accept", FunctionFilename: "net/fd_unix.go", Cumulative: 1, Labels: map[string]string{"goroutine": "1"}, Children: []uint32{2}},                 // 1
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 402, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).waitRead", FunctionSystemName: "internal/poll.(*pollDesc).waitRead", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: []uint32{3}}, // 2
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 89, FunctionStartLine: 0, FunctionName: "internal/poll.(*FD).Accept", FunctionSystemName: "internal/poll.(*FD).Accept", FunctionFilename: "internal/poll/fd_unix.go", Cumulative: 1, Labels: nil, Children: []uint32{4}},                          // 3
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 84, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).wait", FunctionSystemName: "internal/poll.(*pollDesc).wait", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Labels: nil, Children: nil},                  // 4
	}
	expectedColumns := rowsToColumn(rows)
	actualColumns := fgRecordToColumns(t, record)

	// mapping fields are all null here
	require.Equal(t, expectedColumns.locationAddresses, actualColumns.locationAddresses)
	require.Equal(t, expectedColumns.locationFolded, actualColumns.locationFolded)
	require.Equal(t, expectedColumns.locationLines, actualColumns.locationLines)
	require.Equal(t, expectedColumns.functionStartLines, actualColumns.functionStartLines)
	require.Equal(t, expectedColumns.functionNames, actualColumns.functionNames)
	require.Equal(t, expectedColumns.functionSystemNames, actualColumns.functionSystemNames)
	require.Equal(t, expectedColumns.functionFileNames, actualColumns.functionFileNames)
	require.Equal(t, expectedColumns.cumulative, actualColumns.cumulative)
	require.Equal(t, expectedColumns.children, actualColumns.children)
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
		sorter     *flamegraphColumnSorter
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
			trimmed:    0, // TODO
			sorter: &flamegraphColumnSorter{
				slices: [][2]int{{4, 6}}, // lines 4 and 5 are not in stable order in the result so we need to sort them for the test to be deterministic, so the range we want to sort is [4, 6)
				less: func(columns flamegraphColumns, i, j int) bool {
					return columns.locationAddresses[i] > columns.locationAddresses[j]
				},
			},
			rows: []flamegraphRow{
				{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "(null)", MappingBuildID: "(null)", LocationAddress: 0, LocationFolded: false, LocationLine: 0, Cumulative: 6, Children: []uint32{1}},    // 0
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, Cumulative: 6, Children: []uint32{2}},    // 1
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, Cumulative: 6, Children: []uint32{3}},    // 2
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, Cumulative: 4, Children: []uint32{4, 5}}, // 3
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, Cumulative: 1, Children: nil},            // 4
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, Cumulative: 3, Children: nil},            // 5
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
			require.Equal(t, int64(16), fa.NumCols())

			// Convert the numRows to columns for easier access when testing below.
			expectedColumns := rowsToColumn(tc.rows)
			actualColumns := fgRecordToColumns(t, fa)

			if tc.sorter != nil {
				tc.sorter.sort(actualColumns)
			}

			require.Equal(t, expectedColumns.mappingStart, actualColumns.mappingStart)
			require.Equal(t, expectedColumns.mappingLimit, actualColumns.mappingLimit)
			require.Equal(t, expectedColumns.mappingOffset, actualColumns.mappingOffset)
			require.Equal(t, expectedColumns.mappingFiles, actualColumns.mappingFiles)
			require.Equal(t, expectedColumns.mappingBuildIDs, actualColumns.mappingBuildIDs)
			require.Equal(t, expectedColumns.locationAddresses, actualColumns.locationAddresses)
			require.Equal(t, expectedColumns.locationFolded, actualColumns.locationFolded)
			require.Equal(t, expectedColumns.cumulative, actualColumns.cumulative)
			require.Equal(t, expectedColumns.children, actualColumns.children)
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
	require.Equal(t, int64(16), fa.NumCols())

	rows := []flamegraphRow{
		{FunctionName: "(null)", Cumulative: 14, Children: []uint32{1}}, // 0
		{FunctionName: "1", Cumulative: 14, Children: []uint32{2}},      // 1
		{FunctionName: "2", Cumulative: 14, Children: nil},              // 2
	}
	expectedColumns := rowsToColumn(rows)
	actualColumns := fgRecordToColumns(t, fa)

	require.Equal(t, expectedColumns.functionNames, actualColumns.functionNames)
	require.Equal(t, expectedColumns.cumulative, actualColumns.cumulative)
	require.Equal(t, expectedColumns.diff, actualColumns.diff)
	require.Equal(t, expectedColumns.children, actualColumns.children)
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
