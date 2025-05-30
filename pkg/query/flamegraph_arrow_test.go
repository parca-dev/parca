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
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/memory"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/m1gwings/treedrawer/tree"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	compactDictionary "github.com/parca-dev/parca/pkg/compactdictionary"
	"github.com/parca-dev/parca/pkg/profile"
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
	Parent             int32
	Cumulative         uint8
	Flat               uint8
	Diff               int8
	Depth              uint8
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
	parent              []int32
	cumulative          []uint8
	flat                []uint8
	diff                []int8
	depth               []uint8
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
		columns.parent = append(columns.parent, row.Parent)
		columns.cumulative = append(columns.cumulative, row.Cumulative)
		columns.flat = append(columns.flat, row.Flat)
		columns.diff = append(columns.diff, row.Diff)
		columns.depth = append(columns.depth, row.Depth)
	}
	return columns
}

func extractLabelColumns(t *testing.T, r arrow.Record) []map[string]string {
	pprofLabels := make([]map[string]string, r.NumRows())
	for i := 0; i < int(r.NumRows()); i++ {
		sampleLabels := map[string]string{}
		for j, f := range r.Schema().Fields() {
			if strings.HasPrefix(f.Name, profile.ColumnLabelsPrefix) && r.Column(j).IsValid(i) {
				col := r.Column(r.Schema().FieldIndices(f.Name)[0]).(*array.Dictionary)
				dict := col.Dictionary().(*array.Binary)

				labelName := strings.TrimPrefix(f.Name, profile.ColumnLabelsPrefix)
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

func extractParentColumn(t *testing.T, r arrow.Record) []int32 {
	parents := make([]int32, r.NumRows())
	col := r.Column(r.Schema().FieldIndices(FlamegraphFieldParent)[0]).(*array.Int32)
	for i := 0; i < int(r.NumRows()); i++ {
		parents[i] = col.Value(i)
	}
	return parents
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
	case *array.Float64:
		return arr.Float64Values()
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

	mappings := []*pprofprofile.Mapping{{
		ID:      1,
		Start:   1,
		Limit:   1,
		Offset:  0x1234,
		File:    "a",
		BuildID: "aID",
	}, {
		ID:      2,
		Start:   2,
		Limit:   2,
		Offset:  0x1235,
		File:    "b",
		BuildID: "bID",
	}}
	function := []*pprofprofile.Function{{
		ID:         1,
		Name:       "1",
		SystemName: "1",
		Filename:   "1",
		StartLine:  1,
	}, {
		ID:         2,
		Name:       "2",
		SystemName: "2",
		Filename:   "2",
		StartLine:  2,
	}, {
		ID:         3,
		Name:       "3",
		SystemName: "3",
		Filename:   "3",
		StartLine:  3,
	}, {
		ID:         4,
		Name:       "4",
		SystemName: "4",
		Filename:   "4",
		StartLine:  4,
	}, {
		ID:         5,
		Name:       "5",
		SystemName: "5",
		Filename:   "5",
		StartLine:  5,
	}, {
		ID:         6,
		Name:       "2",
		SystemName: "6",
		Filename:   "6",
		StartLine:  6,
	}}
	locations := []*pprofprofile.Location{{
		ID:      1,
		Mapping: mappings[0],
		Address: 0xa1,
		Line: []pprofprofile.Line{{
			Function: function[0],
			Line:     1,
		}},
	}, {
		ID:      2,
		Mapping: mappings[0],
		Address: 0xa2,
		Line: []pprofprofile.Line{{
			Function: function[1],
			Line:     2,
		}},
	}, {
		ID:      3,
		Mapping: mappings[0],
		Address: 0xa3,
		Line: []pprofprofile.Line{{
			Function: function[2],
			Line:     3,
		}},
	}, {
		ID:      4,
		Mapping: mappings[0],
		Address: 0xa4,
		Line: []pprofprofile.Line{{
			Function: function[3],
			Line:     4,
		}},
	}, {
		ID:      5,
		Mapping: mappings[0],
		Address: 0xa5,
		Line: []pprofprofile.Line{{
			Function: function[4],
			Line:     5,
		}},
	}, {
		ID:      6,
		Mapping: mappings[1],
		Address: 0xa6,
		Line: []pprofprofile.Line{{
			Function: function[5],
			Line:     6,
		}},
	}}
	loc1 := locations[0]
	loc2 := locations[1]
	loc3 := locations[2]
	loc4 := locations[3]
	loc5 := locations[4]
	loc6 := locations[5]

	p := &pprofprofile.Profile{
		Mapping:  mappings,
		Function: function,
		Location: locations,
		Sample: []*pprofprofile.Sample{{
			Location: []*pprofprofile.Location{loc2, loc1},
			Value:    []int64{2},
			Label:    map[string][]string{"goroutine": {"app"}, "cpu": {"0"}},
		}, {
			Location: []*pprofprofile.Location{loc5, loc3, loc2, loc1},
			Value:    []int64{1},
			Label:    map[string][]string{"goroutine": {"app"}, "cpu": {"1"}},
		}, {
			Location: []*pprofprofile.Location{loc4, loc3, loc2, loc1},
			Value:    []int64{3},
			Label:    map[string][]string{},
		}, {
			// this is the same stack as s2 but with a different label
			Location: []*pprofprofile.Location{loc5, loc3, loc2, loc1},
			Value:    []int64{4},
			Label:    map[string][]string{"goroutine": {"container"}},
		}, {
			Location: []*pprofprofile.Location{loc6, loc1},
			Value:    []int64{1},
			Label:    map[string][]string{},
		}},
	}

	tracer := noop.NewTracerProvider().Tracer("")

	for _, tc := range []struct {
		name      string
		aggregate []string
		// expectations
		rows       []flamegraphRow
		cumulative int64
		height     int32
		cols       int64
		trimmed    int64
	}{{
		name:      "aggregate-function-name",
		aggregate: []string{FlamegraphFieldFunctionName},
		// expectations
		cumulative: 11,
		height:     5,
		trimmed:    0,
		cols:       18,
		rows: []flamegraphRow{
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{1}, Parent: -1, Depth: 0}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{2}, Parent: 0, Depth: 1},                                                                   // 1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0x0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 3, Labels: nil, Children: []uint32{3}, Parent: 1, Depth: 2},               // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Flat: 0, Labels: nil, Children: []uint32{4, 5}, Parent: 2, Depth: 3},                                                                 // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Flat: 3, Labels: nil, Children: nil, Parent: 3, Depth: 4},                                                                            // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Flat: 5, Labels: nil, Children: nil, Parent: 3, Depth: 4},                                                                            // 5
		},
	}, {
		name:      "aggregate-labels-first-of-two",
		aggregate: []string{"labels.goroutine"},
		// expectations
		cumulative: 11,
		height:     6,
		trimmed:    0,
		cols:       19,
		rows: []flamegraphRow{
			// root
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{1, 6, 11}, Parent: -1, Depth: 0}, // 0
			// stack 1 -- labels: goroutine=app
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 3, Flat: 0, Labels: map[string]string{"goroutine": "app"}, Children: []uint32{2}, LabelsOnly: true, Parent: 0, Depth: 1}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 3, Flat: 0, Labels: map[string]string{"goroutine": "app"}, Children: []uint32{3}, Parent: 1, Depth: 2},                                                                          // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 3, Flat: 2, Labels: map[string]string{"goroutine": "app"}, Children: []uint32{4}, Parent: 4, Depth: 3},                                                                          // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Flat: 0, Labels: map[string]string{"goroutine": "app"}, Children: []uint32{5}, Parent: 7, Depth: 4},                                                                          // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Flat: 1, Labels: map[string]string{"goroutine": "app"}, Children: nil, Parent: 10, Depth: 5},                                                                                 // 5
			// stack 2 -- labels: goroutine=container
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{7}, LabelsOnly: true, Parent: 0, Depth: 1}, // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{8}, Parent: 3, Depth: 2},                                                                          // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{9}, Parent: 6, Depth: 3},                                                                          // 8
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{10}, Parent: 9, Depth: 4},                                                                         // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Flat: 4, Labels: map[string]string{"goroutine": "container"}, Children: nil, Parent: 12, Depth: 5},                                                                                 // 10
			// stack 3 -- no labels
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Flat: 0, Labels: nil, Children: []uint32{12}, Parent: 0, Depth: 1},                                                        // 11
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Flat: 1, Labels: nil, Children: []uint32{13}, Parent: 2, Depth: 2}, // 12
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Flat: 0, Labels: nil, Children: []uint32{14}, Parent: 5, Depth: 3},                                                        // 13
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Flat: 3, Labels: nil, Children: nil, Parent: 8, Depth: 4},                                                                 // 14
		},
	}, {
		name:      "aggregate-labels-last-of-two",
		aggregate: []string{"labels.cpu"},
		// expectations
		cumulative: 11,
		height:     6,
		trimmed:    0,
		cols:       19,
		rows: []flamegraphRow{
			// root
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{1, 4, 9}, Parent: -1, Depth: 0}, // 0
			// stack 1 -- labels: cpu=0
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 2, Flat: 0, Labels: map[string]string{"cpu": "0"}, Children: []uint32{2}, LabelsOnly: true, Parent: 0, Depth: 1}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 2, Flat: 0, Labels: map[string]string{"cpu": "0"}, Children: []uint32{3}, Parent: 1, Depth: 2},                                                                          // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 2, Flat: 2, Labels: map[string]string{"cpu": "0"}, Children: nil, Parent: 4, Depth: 3},                                                                                  // 3
			// stack 2 -- labels: cpu=1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1"}, Children: []uint32{5}, LabelsOnly: true, Parent: 0, Depth: 1}, // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1"}, Children: []uint32{6}, Parent: 2, Depth: 2},                                                                          // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1"}, Children: []uint32{7}, Parent: 5, Depth: 3},                                                                          // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1"}, Children: []uint32{8}, Parent: 8, Depth: 4},                                                                          // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Flat: 1, Labels: map[string]string{"cpu": "1"}, Children: nil, Parent: 10, Depth: 5},                                                                                 // 8
			// stack 3 -- no labels
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 8, Flat: 0, Labels: nil, Children: []uint32{10}, Parent: 0, Depth: 1},                                                        // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 8, Flat: 1, Labels: nil, Children: []uint32{11}, Parent: 3, Depth: 2}, // 10
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 7, Flat: 0, Labels: nil, Children: []uint32{12, 13}, Parent: 6, Depth: 3},                                                    // 11
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Flat: 3, Labels: nil, Children: nil, Parent: 9, Depth: 4},                                                                 // 12
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Flat: 4, Labels: nil, Children: nil, Parent: 9, Depth: 4},                                                                 // 13
		},
	}, {
		name:      "aggregate-labels-two-of-two",
		aggregate: []string{"labels.cpu", "labels.goroutine"},
		// expectations
		cumulative: 11,
		height:     6,
		trimmed:    0,
		cols:       20,
		rows: []flamegraphRow{
			// root
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{1, 4, 9, 14}, Parent: -1, Depth: 0}, // 0
			// stack 1 -- labels: cpu=0, goroutine=1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 2, Flat: 0, Labels: map[string]string{"cpu": "0", "goroutine": "app"}, Children: []uint32{2}, LabelsOnly: true, Parent: 0, Depth: 1}, // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 2, Flat: 0, Labels: map[string]string{"cpu": "0", "goroutine": "app"}, Children: []uint32{3}, Parent: 1, Depth: 2},                                                                          // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 2, Flat: 2, Labels: map[string]string{"cpu": "0", "goroutine": "app"}, Children: nil, Parent: 5, Depth: 3},                                                                                  // 3
			// stack 1 -- labels: cpu=1, goroutine=1
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1", "goroutine": "app"}, Children: []uint32{5}, LabelsOnly: true, Parent: 0, Depth: 1}, // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1", "goroutine": "app"}, Children: []uint32{6}, Parent: 2, Depth: 2},                                                                          // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1", "goroutine": "app"}, Children: []uint32{7}, Parent: 6, Depth: 3},                                                                          // 6
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 1, Flat: 0, Labels: map[string]string{"cpu": "1", "goroutine": "app"}, Children: []uint32{8}, Parent: 10, Depth: 4},                                                                         // 7
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 1, Flat: 1, Labels: map[string]string{"cpu": "1", "goroutine": "app"}, Children: nil, Parent: 13, Depth: 5},                                                                                 // 8
			// stack 3 -- labels: goroutine=2
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: `(null)`, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{10}, LabelsOnly: true, Parent: 0, Depth: 1}, // 9
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{11}, Parent: 4, Depth: 2},                                                                          // 10
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{12}, Parent: 8, Depth: 3},                                                                          // 11
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 4, Flat: 0, Labels: map[string]string{"goroutine": "container"}, Children: []uint32{13}, Parent: 12, Depth: 4},                                                                         // 12
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 4, Flat: 4, Labels: map[string]string{"goroutine": "container"}, Children: nil, Parent: 15, Depth: 5},                                                                                  // 13
			// stack 4 -- no labels
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 4, Flat: 0, Labels: nil, Children: []uint32{15}, Parent: 0, Depth: 1},                                                        // 14
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "2", FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Flat: 1, Labels: nil, Children: []uint32{16}, Parent: 3, Depth: 2}, // 15 - the nulls in this are due to merging with different underlying values
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 3, Flat: 0, Labels: nil, Children: []uint32{17}, Parent: 7, Depth: 3},                                                        // 16
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Flat: 3, Labels: nil, Children: nil, Parent: 11, Depth: 4},                                                                // 17
		},
	}, {
		name:      "aggregate-mapping-file",
		aggregate: []string{FlamegraphFieldMappingFile},
		// expectations
		cumulative: 11,
		height:     5,
		trimmed:    0,
		cols:       18,
		rows: []flamegraphRow{
			// This aggregates all the rows with the same mapping file, meaning that we only keep one flamegraphRow per stack depth in this example.
			{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{1}, Parent: -1, Depth: 0}, // 0
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Cumulative: 11, Flat: 0, Labels: nil, Children: []uint32{2, 6}, Parent: 0, Depth: 1},                                                                // 1
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Cumulative: 10, Flat: 2, Labels: nil, Children: []uint32{3}, Parent: 1, Depth: 2},                                                                   // 2
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Cumulative: 8, Flat: 0, Labels: nil, Children: []uint32{4, 5}, Parent: 2, Depth: 3},                                                                 // 3
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Cumulative: 3, Flat: 3, Labels: nil, Children: nil, Parent: 4, Depth: 4},                                                                            // 4
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Cumulative: 5, Flat: 5, Labels: nil, Children: nil, Parent: 4, Depth: 4},                                                                            // 5
			{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "b", MappingBuildID: "bID", LocationAddress: 0xa6, LocationLine: 6, FunctionStartLine: 6, FunctionName: "2", FunctionSystemName: "6", FunctionFilename: "6", Cumulative: 1, Flat: 1, Labels: nil, Children: nil, Parent: 1, Depth: 2},                                                                            // 6
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			sp, err := PprofToSymbolizedProfile(profile.Meta{Duration: (10 * time.Second).Nanoseconds()}, p, 0, tc.aggregate)
			require.NoError(t, err)

			sp.Samples = []arrow.Record{
				sp.Samples[0].NewSlice(0, 2),
				sp.Samples[0].NewSlice(2, 5),
			}

			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, sp, tc.aggregate, 0)
			require.NoError(t, err)
			defer fa.Release()

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, tc.cols, fa.NumCols())

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
		parent:              extractParentColumn(c.t, r),
		cumulative:          extractColumn(c.t, r, FlamegraphFieldCumulative).([]uint8),
		flat:                extractColumn(c.t, r, FlamegraphFieldFlat).([]uint8),
		diff:                extractColumn(c.t, r, FlamegraphFieldDiff).([]int8),
		depth:               extractColumn(c.t, r, FlamegraphFieldDepth).([]uint8),
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
	require.Equal(c.t, expected.cumulative, reorder(c.actual.cumulative, order))
	require.Equal(c.t, expected.flat, reorder(c.actual.flat, order))
	require.Equal(c.t, expected.diff, reorder(c.actual.diff, order))
	require.Equal(c.t, expected.depth, reorder(c.actual.depth, order))
	require.Equal(c.t, expected.children, sortedChildren)
	require.Equal(c.t, expected.parent, reorder(c.actual.parent, order))
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
	tracer := noop.NewTracerProvider().Tracer("")

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
	require.Equal(t, int64(18), record.NumCols())
	require.Equal(t, int64(1), record.NumRows())
}

func TestGenerateFlamegraphArrowWithInlined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	tracer := noop.NewTracerProvider().Tracer("")

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

	p, err := PprofToSymbolizedProfile(profile.Meta{Duration: (10 * time.Second).Nanoseconds()}, &pprofprofile.Profile{
		SampleType: []*pprofprofile.ValueType{{Type: "alloc_space", Unit: "bytes"}},
		PeriodType: &pprofprofile.ValueType{Type: "space", Unit: "bytes"},
		Sample:     samples,
		Location:   locations,
		Function:   functions,
	}, 0, []string{})
	require.NoError(t, err)

	record, total, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, []string{FlamegraphFieldFunctionName}, 0)
	require.NoError(t, err)
	defer record.Release()

	require.Equal(t, int64(1), total)
	require.Equal(t, int32(5), height)
	require.Equal(t, int64(0), trimmed)

	require.Equal(t, int64(18), record.NumCols())
	require.Equal(t, int64(5), record.NumRows())

	rows := []flamegraphRow{
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0, LocationLine: 0, FunctionStartLine: 0, FunctionName: "(null)", FunctionSystemName: "(null)", FunctionFilename: "(null)", Cumulative: 1, Flat: 0, Labels: nil, Children: []uint32{1}, Parent: -1, Depth: 0},                                                                                       // 0
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa1, LocationLine: 173, FunctionStartLine: 0, FunctionName: "net.(*netFD).accept", FunctionSystemName: "net.(*netFD).accept", FunctionFilename: "net/fd_unix.go", Cumulative: 1, Flat: 0, Labels: nil, Children: []uint32{2}, Parent: 0, Depth: 1},                                                 // 1
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa2, LocationLine: 200, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).waitRead", FunctionSystemName: "internal/poll.(*pollDesc).waitRead", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Flat: 0, Labels: nil, Children: []uint32{3}, Parent: 1, Depth: 2}, // 2
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa2, LocationLine: 89, FunctionStartLine: 0, FunctionName: "internal/poll.(*FD).Accept", FunctionSystemName: "internal/poll.(*FD).Accept", FunctionFilename: "internal/poll/fd_unix.go", Cumulative: 1, Flat: 0, Labels: nil, Children: []uint32{4}, Inlined: true, Parent: 2, Depth: 3},           // 3
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, LocationAddress: 0xa3, LocationLine: 84, FunctionStartLine: 0, FunctionName: "internal/poll.(*pollDesc).wait", FunctionSystemName: "internal/poll.(*pollDesc).wait", FunctionFilename: "internal/poll/fd_poll_runtime.go", Cumulative: 1, Flat: 1, Labels: nil, Children: nil, Parent: 3, Depth: 4},                  // 4
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

	mappings := []*pprofprofile.Mapping{
		{ID: 1, Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildID: "aID"},
	}

	locations := []*pprofprofile.Location{{
		ID:      1,
		Mapping: mappings[0],
		Address: 0xa1,
	}, {
		ID:      2,
		Mapping: mappings[0],
		Address: 0xa2,
	}, {
		ID:      3,
		Mapping: mappings[0],
		Address: 0xa3,
	}, {
		ID:      4,
		Mapping: mappings[0],
		Address: 0xa4,
	}, {
		ID:      5,
		Mapping: mappings[0],
		Address: 0xa5,
	}}

	p, err := PprofToSymbolizedProfile(profile.Meta{Duration: (10 * time.Second).Nanoseconds()}, &pprofprofile.Profile{
		Mapping:  mappings,
		Location: locations,
		Sample: []*pprofprofile.Sample{{
			Location: []*pprofprofile.Location{locations[1], locations[0]},
			Value:    []int64{2},
		}, {
			Location: []*pprofprofile.Location{locations[4], locations[2], locations[1], locations[0]},
			Value:    []int64{1},
		}, {
			Location: []*pprofprofile.Location{locations[3], locations[2], locations[1], locations[0]},
			Value:    []int64{3},
		}},
	}, 0, []string{})
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")

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
				{MappingStart: 0, MappingLimit: 0, MappingOffset: 0, MappingFile: "(null)", MappingBuildID: "(null)", LocationAddress: 0, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Flat: 0, Children: []uint32{1}, Parent: -1, Depth: 0},   // 0
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Flat: 0, Children: []uint32{2}, Parent: 0, Depth: 1},    // 1
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 6, Flat: 2, Children: []uint32{3}, Parent: 1, Depth: 2},    // 2
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 4, Flat: 0, Children: []uint32{4, 5}, Parent: 2, Depth: 3}, // 3
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 3, Flat: 3, Children: nil, Parent: 3, Depth: 4},            // 4
				{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 1, Flat: 1, Children: nil, Parent: 3, Depth: 4},            // 5
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, tc.aggregate, 0)
			require.NoError(t, err)
			defer fa.Release()

			require.Equal(t, tc.cumulative, cumulative)
			require.Equal(t, tc.height, height)
			require.Equal(t, tc.trimmed, trimmed)
			require.Equal(t, int64(len(tc.rows)), fa.NumRows())
			require.Equal(t, int64(18), fa.NumCols())

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

	mappings := []*pprofprofile.Mapping{{
		ID:   1,
		File: "a",
	}}

	functions := []*pprofprofile.Function{{
		ID:   1,
		Name: "1",
	}, {
		ID:   2,
		Name: "2",
	}, {
		ID:   3,
		Name: "3",
	}, {
		ID:   4,
		Name: "4",
	}, {
		ID:   5,
		Name: "5",
	}}

	locations := []*pprofprofile.Location{{
		ID:      1,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[0]}},
	}, {
		ID:      2,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[1]}},
	}, {
		ID:      3,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[2]}},
	}, {
		ID:      4,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[3]}},
	}, {
		ID:      5,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[4]}},
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{Duration: (10 * time.Second).Nanoseconds()},
		&pprofprofile.Profile{
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[1], locations[0]},
				Value:    []int64{10},
			}, {
				Location: []*pprofprofile.Location{locations[4], locations[2], locations[1], locations[0]},
				Value:    []int64{1},
			}, {
				Location: []*pprofprofile.Location{locations[3], locations[2], locations[1], locations[0]},
				Value:    []int64{3},
			}},
		},
		0, []string{},
	)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	fa, cumulative, height, trimmed, err := generateFlamegraphArrowRecord(ctx, mem, tracer, p, []string{FlamegraphFieldFunctionName}, float32(0.5))
	require.NoError(t, err)

	require.Equal(t, int64(14), cumulative)
	require.Equal(t, int32(5), height)
	require.Equal(t, int64(4), trimmed)
	require.Equal(t, int64(3), fa.NumRows())
	require.Equal(t, int64(18), fa.NumCols())

	// TODO: MappingBuildID and FunctionSystemNames shouldn't be "" but null?
	rows := []flamegraphRow{
		{MappingFile: array.NullValueStr, MappingBuildID: array.NullValueStr, FunctionName: array.NullValueStr, FunctionSystemName: array.NullValueStr, FunctionFilename: array.NullValueStr, Cumulative: 14, Flat: 0, Children: []uint32{1}, Parent: -1, Depth: 0}, // 0
		{MappingFile: "a", MappingBuildID: "", FunctionName: "1", FunctionSystemName: "", FunctionFilename: "", Cumulative: 14, Flat: 0, Children: []uint32{2}, Parent: 0, Depth: 1},                                                                                // 1
		{MappingFile: "a", MappingBuildID: "", FunctionName: "2", FunctionSystemName: "", FunctionFilename: "", Cumulative: 14, Flat: 10, Children: nil, Parent: 1, Depth: 2},                                                                                       // 2
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

	np, err := PprofToSymbolizedProfile(profile.MetaFromPprof(p, "memory", 0), pp, 0, []string{})
	require.NoError(b, err)

	tracer := noop.NewTracerProvider().Tracer("")

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
	compArr, err := compactDictionary.CompactDictionary(mem, array.NewDictionaryArray(
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
	compArr, err = compactDictionary.CompactDictionary(mem, array.NewDictionaryArray(
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
	compArr, err = compactDictionary.CompactDictionary(mem, array.NewDictionaryArray(
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

	np, err := PprofToSymbolizedProfile(profile.MetaFromPprof(p, "memory", 0), pp, 0, []string{})
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")

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
	tracer := noop.NewTracerProvider().Tracer("")

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

	np, err := PprofToSymbolizedProfile(profile.MetaFromPprof(p, "cpu", 0), pp, 0, []string{})
	require.NoError(t, err)

	// This is a regression test, what we want to achieve here is the input
	// data being multiple samples, but all frames are filtered out. What
	// happened is the input data contains no python frames, but only python
	// frames were requested.
	np.Samples, _, err = FilterProfileData(ctx, tracer, mem, np.Samples, "", false, map[string]struct{}{})
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

func TestFlameChartMultipleSamplesForSameTimestamp(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
main;func_fib 10 1000 10
main;func_fib 10 2000 10
main;func_fib 10 3000 10
main;func_add 10 3000 10
`))
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
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)

	require.Error(t, err)
	require.Nil(t, record)
}

func TestFlameChartGroupByTimestamp(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
main;func_fib 10 1000 10
runtime;gc 10 2000 10
main;func_fib 10 3000 10
`))
	require.NoError(t, err)
	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	// Group by function_name, timestamp, duration
	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)
	require.NoError(t, err)
	defer record.Release()

	row := 0
	schema := record.Schema()
	childrenColIdx := schema.FieldIndices("children")[0]
	childrenCol := record.Column(childrenColIdx).(*array.List)
	ChildrenValues := childrenCol.ListValues().(*array.Uint32)
	timestampColIdx := schema.FieldIndices("timestamp")[0]
	timestampCol := record.Column(timestampColIdx).(*array.Int64)

	offsetStart, offsetEnd := childrenCol.ValueOffsets(row)

	nums := offsetEnd - offsetStart

	expectedTs := []int64{1000, 2000, 3000}

	for j := int64(0); j < nums; j++ {
		row = int(ChildrenValues.Value(int(offsetStart + j)))
		ts := timestampCol.Value(row)
		require.Equal(t, expectedTs[j], ts)
	}
}

func TestFlameChartMergeNeighbouringStacksWithSameRoot(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
main;func_fib 10 10 1
main;func_fib 10 20 1
main;func_add 10 30 1
runtime;gc 10 40 1
main;func_fib 10 50 1
main;func_fib 10 60 1
`))
	require.NoError(t, err)
	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	// Group by function_name, timestamp, duration
	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)
	require.NoError(t, err)
	defer record.Release()

	row := 0
	schema := record.Schema()
	childrenColIdx := schema.FieldIndices("children")[0]
	childrenCol := record.Column(childrenColIdx).(*array.List)
	ChildrenValues := childrenCol.ListValues().(*array.Uint32)
	timestampColIdx := schema.FieldIndices("timestamp")[0]
	timestampCol := record.Column(timestampColIdx).(*array.Int64)

	offsetStart, offsetEnd := childrenCol.ValueOffsets(row)

	nums := offsetEnd - offsetStart

	require.Equal(t, int64(3), nums)

	expectedTs := []int64{10, 40, 50}

	for j := int64(0); j < nums; j++ {
		row = int(ChildrenValues.Value(int(offsetStart + j)))
		ts := timestampCol.Value(row)
		require.Equal(t, expectedTs[j], ts)
	}
}

func TestFlameChartDoNotMergeSamplesTooFarApart(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
a;b 52 1000 52
a;c 52 1158 52
a;d 52 1211 52
`))

	// 1-2 diff in ts: 158, which means the first two samples are not merged
	// 2-3 diff in ts: 53, which means the last two samples are merged
	require.NoError(t, err)
	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	// Group by function_name, timestamp, duration
	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)
	require.NoError(t, err)
	defer record.Release()
	// root -> a -> b
	//      -> a -> c
	//         ┗ -> d
	require.Equal(t, int64(6), record.NumRows())
}

func TestFlameChartDoNotMergeWithSampleInbetween(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
a;b 52 1549 52
a;c 52 1601 52
a;b 52 1654 52
`))

	// 1-2 diff in ts: 158, which means the first two samples are not merged
	// 2-3 diff in ts: 53, which means the last two samples are merged
	require.NoError(t, err)
	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	// Group by function_name, timestamp, duration
	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)
	require.NoError(t, err)
	defer record.Release()
	// root -> a -> b
	//      -> a -> c
	//      -> a -> b
	require.Equal(t, int64(5), record.NumRows())
}

func TestFlameChartDoNotMergeWithDistance(t *testing.T) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	np, err := foldedStacksWithTsToProfile(mem, []byte(`
a;b 52 1549 52
a;b 52 1654 52
`))

	require.NoError(t, err)
	defer func() {
		for _, r := range np.Samples {
			r.Release()
		}
	}()

	// Group by function_name, timestamp, duration
	record, _, _, _, err := generateFlamegraphArrowRecord(
		ctx,
		mem,
		tracer,
		np,
		[]string{
			FlamegraphFieldFunctionName,
			FlamegraphFieldTimestamp,
		},
		0,
	)
	require.NoError(t, err)
	defer record.Release()
	// root -> a -> b
	//      -> a -> b
	require.Equal(t, int64(5), record.NumRows())
}

// split the line into 4 parts: stack, value, timestamp, duration
//
// example line:
//
// main;do;some;work 123 1732617178462 duration
// end.
func splitLine(line string) (string, string, string, string, error) {
	parts := strings.Split(line, " ")
	if len(parts) != 4 {
		return "", "", "", "", fmt.Errorf("invalid line format")
	}

	return parts[0], parts[1], parts[2], parts[3], nil
}

func foldedStacksWithTsToProfile(pool memory.Allocator, input []byte) (profile.Profile, error) {
	w := profile.NewWriter(pool, nil)
	defer w.RecordBuilder.Release()

	for n, line := range strings.Split(string(input), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		stack, valueStr, timstampStr, periodStr, err := splitLine(line)
		if err != nil {
			return profile.Profile{}, err
		}

		val, err := strconv.ParseInt(valueStr, 10, 64)
		if err != nil {
			return profile.Profile{}, fmt.Errorf("bad line: %d: %q: %s", n, line, err)
		}

		ts, err := strconv.ParseInt(timstampStr, 10, 64)
		if err != nil {
			return profile.Profile{}, fmt.Errorf("bad line: %d: %q: %s", n, line, err)
		}

		period, err := strconv.ParseInt(periodStr, 10, 64)
		if err != nil {
			return profile.Profile{}, fmt.Errorf("bad line: %d: %q: %s", n, line, err)
		}

		stackFrames := strings.Split(stack, ";")
		if len(stackFrames) == 0 {
			return profile.Profile{}, fmt.Errorf("bad line: %d: %q: no stack frames", n, line)
		}

		w.Value.Append(val)
		w.Diff.Append(0)
		w.TimeNanos.Append(ts)
		w.Period.Append(period)
		w.LocationsList.Append(true)

		for i := len(stackFrames) - 1; i >= 0; i-- {
			w.Locations.Append(true)
			w.Addresses.Append(0)
			w.MappingStart.AppendNull()
			w.MappingLimit.AppendNull()
			w.MappingOffset.AppendNull()
			w.MappingFile.AppendNull()
			w.MappingBuildID.AppendNull()
			w.Lines.Append(true)
			w.Line.Append(true)
			w.LineNumber.Append(0)
			w.FunctionName.Append([]byte(stackFrames[i]))
			w.FunctionSystemName.Append([]byte(""))
			w.FunctionFilename.Append([]byte(""))
			w.FunctionStartLine.Append(0)
		}
	}

	return profile.Profile{
		Samples: []arrow.Record{w.RecordBuilder.NewRecord()},
	}, nil
}

//lint:ignore U1000 Used for debugging purposes
func drawFlamegraphToConsole(testing *testing.T, record arrow.Record) {
	schema := record.Schema()
	childrenColIdx := schema.FieldIndices("children")[0]
	functionNameColIdx := schema.FieldIndices("function_name")[0]
	childrenCol := record.Column(childrenColIdx).(*array.List)
	functionNameCol := record.Column(functionNameColIdx).(*array.Dictionary)
	ChildrenValues := childrenCol.ListValues().(*array.Uint32)
	cumulativeColIdx := schema.FieldIndices("cumulative")[0]
	cumulativeCol := record.Column(cumulativeColIdx).(*array.Uint8)
	timestampColIdx := schema.FieldIndices("timestamp")[0]
	timestampCol := record.Column(timestampColIdx).(*array.Int64)

	t := tree.NewTree(tree.NodeString("root" + " " + fmt.Sprint(timestampCol.Value(0))))

	var populateChild func(t *tree.Tree, row int)

	populateChild = func(t *tree.Tree, row int) {
		offsetStart, offsetEnd := childrenCol.ValueOffsets(row)
		nums := offsetEnd - offsetStart
		for j := int64(0); j < nums; j++ {
			child := ChildrenValues.Value(int(offsetStart + j))
			cumulativeValue := cumulativeCol.Value(int(child))
			fBytes := functionNameCol.ValueStr(int(child))
			funcName, err := base64.StdEncoding.DecodeString(fBytes)
			if err != nil {
				funcName = []byte("N/A")
			}
			ts := timestampCol.Value(int(child))
			childT := t.AddChild(tree.NodeString(string(funcName) + " " + fmt.Sprint(ts) + " " + fmt.Sprint(cumulativeValue)))
			populateChild(childT, int(child))
		}
	}

	populateChild(t, 0)
	fmt.Println(t)
}

func Test_matchRowsByTimestamp(t *testing.T) {
	second := time.Second.Nanoseconds()

	tests := []struct {
		name         string
		ct, cd, t, d int64
		match        bool
	}{
		{"0", 0, second, second, second, true},
		{"1/100", 0, second, second + second/100, second, true},
		{"1/10", 0, second, second + second/10, second, true},
		{"1/5", 0, second, second + second/5, second, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, _ := matchRowsByTimestamp(tc.ct, tc.cd, tc.t, tc.d)
			require.Equal(t, tc.match, m)
		})
	}
}
