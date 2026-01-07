// Copyright 2022-2026 The Parca Authors
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
	"math"
	"sort"
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/protobuf/proto"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlamegraphTable(t *testing.T) {
	ctx := context.Background()
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
		profile.Meta{},
		&pprofprofile.Profile{
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
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0), NewTableConverterPool())
	require.NoError(t, err)

	require.Equal(t, int32(5), fg.Height)
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(6), fg.Total)

	// Check if tables and thus deduplication was correct and deterministic

	require.Equal(t, []string{"", "a", "1", "2", "3", "5", "4"}, fg.StringTable)
	require.Equal(t, []*metastorepb.Location{
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 1}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 2}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 3}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 4}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 5}}},
	}, fg.Locations)
	require.Equal(t, []*metastorepb.Mapping{
		{BuildIdStringIndex: 0, FileStringIndex: 1},
	}, fg.Mapping)
	require.Equal(t, []*metastorepb.Function{
		{NameStringIndex: 2, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 3, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 4, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 5, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 6, SystemNameStringIndex: 0, FilenameStringIndex: 0},
	}, fg.Function)

	// Check the recursive flamegraph that references the tables above.

	expected := &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 6,
			Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 1},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 6,
				Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 2},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 4,
					Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 3},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 3,
						Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 5},
					}, {
						Cumulative: 1,
						Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 4},
					}},
				}},
			}},
		}},
	}
	require.Equal(t, expected, fg.Root)
	require.True(t, proto.Equal(expected, fg.Root))
}

func TestGenerateFlamegraphTableTrimming(t *testing.T) {
	ctx := context.Background()
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
		profile.Meta{},
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
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0.5), NewTableConverterPool())
	require.NoError(t, err)

	require.Equal(t, int32(5), fg.Height)
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(14), fg.Total)
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(14), fg.UntrimmedTotal)

	// Check if tables and thus deduplication was correct and deterministic

	require.Equal(t, []string{"", "a", "1", "2", "" /* 3 */, "" /* 5 */, "" /* 4 */}, fg.StringTable)
	require.Equal(t, []*metastorepb.Location{
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 1}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 2}}},
		// The following locations aren't referenced from the flame graph.
		nil, nil, nil,
	}, fg.Locations)
	require.Equal(t, []*metastorepb.Mapping{
		{BuildIdStringIndex: 0, FileStringIndex: 1},
	}, fg.Mapping)
	require.Equal(t, []*metastorepb.Function{
		{NameStringIndex: 2, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 3, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		// The following functions aren't referenced from the flame graph.
		nil, nil, nil,
	}, fg.Function)

	// Check the recursive flamegraph that references the tables above.

	expected := &pb.FlamegraphRootNode{
		Cumulative: 14,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 14,
			Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 1},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 14,
				Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 2},
			}},
		}},
	}
	require.Equal(t, expected, fg.Root)
	require.True(t, proto.Equal(expected, fg.Root))
}

func TestGenerateFlamegraphTableMergeMappings(t *testing.T) {
	ctx := context.Background()
	var err error

	mappings := []*pprofprofile.Mapping{{
		ID:   1,
		File: "a",
	}, {
		ID:   2,
		File: "b",
	}}

	functions := []*pprofprofile.Function{{
		ID:   1,
		Name: "1",
	}}

	locations := []*pprofprofile.Location{{
		ID:      1,
		Address: 0x1,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[0]}},
	}, {
		ID:      2,
		Address: 0x8,
		Mapping: mappings[1],
		Line:    []pprofprofile.Line{{Function: functions[0]}},
	}, {
		ID:      3,
		Address: 0x5,
		Mapping: mappings[1],
	}, {
		ID:      4,
		Address: 0x7,
		Mapping: mappings[1],
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[0]},
				Value:    []int64{2},
			}, {
				Location: []*pprofprofile.Location{locations[1]},
				Value:    []int64{2},
			}, {
				Location: []*pprofprofile.Location{locations[2]},
				Value:    []int64{2},
			}, {
				Location: []*pprofprofile.Location{locations[3]},
				Value:    []int64{1},
			}},
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0), NewTableConverterPool())
	require.NoError(t, err)

	require.Equal(t, int32(2), fg.Height)
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(7), fg.Total)

	// Check if tables and thus deduplication was correct and deterministic

	require.Equal(t, []string{"", "a", "1", "b"}, fg.StringTable)
	require.Equal(t, 4, len(fg.Locations))

	require.Equal(t, uint32(1), fg.Locations[0].MappingIndex)
	require.Equal(t, 1, len(fg.Locations[0].Lines))
	require.Equal(t, uint64(0x1), fg.Locations[0].Address)
	require.Equal(t, uint32(1), fg.Locations[0].Lines[0].FunctionIndex)

	require.Equal(t, uint32(0), fg.Locations[1].MappingIndex)
	require.Equal(t, 1, len(fg.Locations[1].Lines))
	require.Equal(t, uint64(0x8), fg.Locations[1].Address)
	require.Equal(t, uint32(1), fg.Locations[1].Lines[0].FunctionIndex)

	require.Equal(t, uint32(2), fg.Locations[2].MappingIndex)
	require.Equal(t, 0, len(fg.Locations[2].Lines))
	require.Equal(t, uint64(0x5), fg.Locations[2].Address)

	require.Equal(t, uint32(2), fg.Locations[3].MappingIndex)
	require.Equal(t, 0, len(fg.Locations[3].Lines))
	require.Equal(t, uint64(0x7), fg.Locations[3].Address)

	require.Equal(t, []*metastorepb.Mapping{
		{BuildIdStringIndex: 0, FileStringIndex: 1},
		{BuildIdStringIndex: 0, FileStringIndex: 3},
	}, fg.Mapping)
	require.Equal(t, []*metastorepb.Function{{
		NameStringIndex:       2,
		SystemNameStringIndex: 0,
		FilenameStringIndex:   0,
	}}, fg.Function)

	// Check the recursive flamegraph that references the tables above.

	expected := &pb.FlamegraphRootNode{
		Cumulative: 7,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 2,
			Meta: &pb.FlamegraphNodeMeta{
				LocationIndex: 3,
			},
		}, {
			Cumulative: 1,
			Meta: &pb.FlamegraphNodeMeta{
				LocationIndex: 4,
			},
		}, {
			Cumulative: 4,
			Meta: &pb.FlamegraphNodeMeta{
				LocationIndex: 2,
				LineIndex:     0,
			},
		}},
	}
	require.Equal(t, int64(7), fg.Root.Cumulative)
	require.Equal(t, 3, len(fg.Root.Children))

	require.Equal(t, int64(2), fg.Root.Children[0].Cumulative)
	require.Equal(t, uint32(3), fg.Root.Children[0].Meta.LocationIndex)

	require.Equal(t, int64(1), fg.Root.Children[1].Cumulative)
	require.Equal(t, uint32(4), fg.Root.Children[1].Meta.LocationIndex)

	require.Equal(t, int64(4), fg.Root.Children[2].Cumulative)
	require.Equal(t, uint32(2), fg.Root.Children[2].Meta.LocationIndex)
	require.Equal(t, uint32(0), fg.Root.Children[2].Meta.LineIndex)
	require.True(t, proto.Equal(expected, fg.Root))
}

func TestGenerateFlamegraphTableFromProfile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fileContent := MustReadAllGzip(t, "./testdata/profile1.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		pp,
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	_, err = GenerateFlamegraphTable(ctx, tracer, op, float32(0), NewTableConverterPool())
	require.NoError(t, err)
}

func Benchmark_GenerateFlamegraphTable_FromProfile(b *testing.B) {
	ctx := context.Background()

	fileContent := MustReadAllGzip(b, "./testdata/profile1.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(b, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		pp,
		0,
		[]string{},
	)
	require.NoError(b, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(b, err)

	tracer := noop.NewTracerProvider().Tracer("")
	pool := NewTableConverterPool()

	var dontOptimise *pb.Flamegraph
	for i := 0; i < b.N; i++ {
		ctx, cancel := context.WithCancel(ctx)
		dontOptimise, err = GenerateFlamegraphTable(ctx, tracer, op, float32(0), pool)
		require.NoError(b, err)
		cancel()
	}
	_ = dontOptimise
}

func TestGenerateFlamegraphTableWithInlined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	functions := []*pprofprofile.Function{
		{ID: 1, Name: "net.(*netFD).accept", SystemName: "net.(*netFD).accept", Filename: "net/fd_unix.go"},
		{ID: 2, Name: "internal/poll.(*FD).Accept", SystemName: "internal/poll.(*FD).Accept", Filename: "internal/poll/fd_unix.go"},
		{ID: 3, Name: "internal/poll.(*pollDesc).waitRead", SystemName: "internal/poll.(*pollDesc).waitRead", Filename: "internal/poll/fd_poll_runtime.go"},
		{ID: 4, Name: "internal/poll.(*pollDesc).wait", SystemName: "internal/poll.(*pollDesc).wait", Filename: "internal/poll/fd_poll_runtime.go"},
	}
	locations := []*pprofprofile.Location{
		{ID: 1, Address: 94658718830132, Line: []pprofprofile.Line{{Line: 173, Function: functions[0]}}},
		{ID: 2, Address: 94658718611115, Line: []pprofprofile.Line{
			{Line: 89, Function: functions[1]},
			{Line: 402, Function: functions[2]},
		}},
		{ID: 3, Address: 94658718597969, Line: []pprofprofile.Line{{Line: 84, Function: functions[3]}}},
	}
	samples := []*pprofprofile.Sample{
		{
			Location: []*pprofprofile.Location{locations[2], locations[1], locations[0]},
			Value:    []int64{1},
		},
	}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{
			Name: "memory",
			SampleType: profile.ValueType{
				Type: "alloc_space",
				Unit: "bytes",
			},
			PeriodType: profile.ValueType{
				Type: "space",
				Unit: "bytes",
			},
		},
		&pprofprofile.Profile{
			SampleType: []*pprofprofile.ValueType{{Type: "alloc_space", Unit: "bytes"}},
			PeriodType: &pprofprofile.ValueType{Type: "space", Unit: "bytes"},
			Sample:     samples,
			Location:   locations,
			Function:   functions,
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0), NewTableConverterPool())
	require.NoError(t, err)

	require.Equal(t, []*metastorepb.Mapping{}, fg.GetMapping())

	require.Equal(t, []string{
		"",
		"net.(*netFD).accept",
		"net/fd_unix.go",
		"internal/poll.(*FD).Accept",
		"internal/poll/fd_unix.go",
		"internal/poll.(*pollDesc).waitRead",
		"internal/poll/fd_poll_runtime.go",
		"internal/poll.(*pollDesc).wait",
	}, fg.GetStringTable())

	require.Equal(t, []*metastorepb.Function{{
		NameStringIndex:       1,
		SystemNameStringIndex: 1,
		FilenameStringIndex:   2,
	}, {
		NameStringIndex:       3,
		SystemNameStringIndex: 3,
		FilenameStringIndex:   4,
	}, {
		NameStringIndex:       5,
		SystemNameStringIndex: 5,
		FilenameStringIndex:   6,
	}, {
		NameStringIndex:       7,
		SystemNameStringIndex: 7,
		FilenameStringIndex:   6,
	}}, fg.GetFunction())

	require.Equal(t, []*metastorepb.Location{{
		Address:      94658718830132,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          173,
			FunctionIndex: 1,
		}},
	}, {
		Address:      94658718611115,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          89,
			FunctionIndex: 2,
		}, {
			Line:          402,
			FunctionIndex: 3,
		}},
	}, {
		Address:      94658718597969,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          84,
			FunctionIndex: 4,
		}},
	}}, fg.GetLocations())

	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(1), fg.GetTotal())
	require.Equal(t, int32(4), fg.GetHeight())
	require.Equal(t, "bytes", fg.GetUnit())

	require.Equal(t, &pb.FlamegraphRootNode{
		Cumulative: 1,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 1,
			Meta: &pb.FlamegraphNodeMeta{
				LocationIndex: 1,
				LineIndex:     0,
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 1,
				Meta: &pb.FlamegraphNodeMeta{
					LocationIndex: 2,
					LineIndex:     1,
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 1,
					Meta: &pb.FlamegraphNodeMeta{
						LocationIndex: 2,
						LineIndex:     0,
					},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 1,
						Meta: &pb.FlamegraphNodeMeta{
							LocationIndex: 3,
							LineIndex:     0,
						},
					}},
				}},
			}},
		}},
	}, fg.Root)
}

func TestGenerateFlamegraphTableWithInlinedExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	functions := []*pprofprofile.Function{
		{ID: 1, Name: "net.(*netFD).accept", SystemName: "net.(*netFD).accept", Filename: "net/fd_unix.go"},
		{ID: 2, Name: "internal/poll.(*FD).Accept", SystemName: "internal/poll.(*FD).Accept", Filename: "internal/poll/fd_unix.go"},
		{ID: 3, Name: "internal/poll.(*pollDesc).waitRead", SystemName: "internal/poll.(*pollDesc).waitRead", Filename: "internal/poll/fd_poll_runtime.go"},
		{ID: 4, Name: "internal/poll.(*pollDesc).wait", SystemName: "internal/poll.(*pollDesc).wait", Filename: "internal/poll/fd_poll_runtime.go"},
	}
	locations := []*pprofprofile.Location{
		{ID: 1, Address: 94658718830132, Line: []pprofprofile.Line{{Line: 173, Function: functions[0]}}},
		{ID: 2, Address: 94658718611115, Line: []pprofprofile.Line{
			{Line: 89, Function: functions[1]},
			{Line: 402, Function: functions[2]},
		}},
		{ID: 3, Address: 94658718597969, Line: []pprofprofile.Line{{Line: 84, Function: functions[3]}}},
	}
	samples := []*pprofprofile.Sample{
		{
			Location: []*pprofprofile.Location{locations[2], locations[1], locations[0]},
			Value:    []int64{1},
		},
		{
			Location: []*pprofprofile.Location{locations[1], locations[0]},
			Value:    []int64{2},
		},
	}
	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			SampleType: []*pprofprofile.ValueType{{Type: "", Unit: ""}},
			PeriodType: &pprofprofile.ValueType{Type: "", Unit: ""},
			Sample:     samples,
			Location:   locations,
			Function:   functions,
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	tracer := noop.NewTracerProvider().Tracer("")
	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0), NewTableConverterPool())
	require.NoError(t, err)

	require.Equal(t, []*metastorepb.Mapping{}, fg.GetMapping())

	require.Equal(t, []string{
		"",
		"net.(*netFD).accept",
		"net/fd_unix.go",
		"internal/poll.(*FD).Accept",
		"internal/poll/fd_unix.go",
		"internal/poll.(*pollDesc).waitRead",
		"internal/poll/fd_poll_runtime.go",
		"internal/poll.(*pollDesc).wait",
	}, fg.GetStringTable())

	require.Equal(t, []*metastorepb.Function{{
		NameStringIndex:       1,
		SystemNameStringIndex: 1,
		FilenameStringIndex:   2,
	}, {
		NameStringIndex:       3,
		SystemNameStringIndex: 3,
		FilenameStringIndex:   4,
	}, {
		NameStringIndex:       5,
		SystemNameStringIndex: 5,
		FilenameStringIndex:   6,
	}, {
		NameStringIndex:       7,
		SystemNameStringIndex: 7,
		FilenameStringIndex:   6,
	}}, fg.GetFunction())

	require.Equal(t, []*metastorepb.Location{{
		Address:      94658718830132,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          173,
			FunctionIndex: 1,
		}},
	}, {
		Address:      94658718611115,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          89,
			FunctionIndex: 2,
		}, {
			Line:          402,
			FunctionIndex: 3,
		}},
	}, {
		Address:      94658718597969,
		MappingIndex: 0,
		Lines: []*metastorepb.Line{{
			Line:          84,
			FunctionIndex: 4,
		}},
	}}, fg.GetLocations())

	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(3), fg.GetTotal())
	require.Equal(t, int32(4), fg.GetHeight())
	require.Equal(t, "", fg.GetUnit())

	require.Equal(t, &pb.FlamegraphRootNode{
		Cumulative: 3,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 3,
			Meta: &pb.FlamegraphNodeMeta{
				LocationIndex: 1,
				LineIndex:     0,
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 3,
				Meta: &pb.FlamegraphNodeMeta{
					LocationIndex: 2,
					LineIndex:     1,
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 3,
					Meta: &pb.FlamegraphNodeMeta{
						LocationIndex: 2,
						LineIndex:     0,
					},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 1,
						Meta: &pb.FlamegraphNodeMeta{
							LocationIndex: 3,
							LineIndex:     0,
						},
					}},
				}},
			}},
		}},
	}, fg.Root)
}

func TestFlamegraphTrimming(t *testing.T) {
	fullGraph := &pb.Flamegraph{
		Total: 102,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 102,

			Children: []*pb.FlamegraphNode{
				{
					Cumulative: 101,
					Meta: &pb.FlamegraphNodeMeta{
						LocationIndex: 1,
						LineIndex:     0,
					},
					Children: []*pb.FlamegraphNode{
						{
							// This node is trimmed because it is below the threshold.
							Cumulative: 1,
						},
						{
							Cumulative: 30,
							Meta: &pb.FlamegraphNodeMeta{
								LocationIndex: 2,
								LineIndex:     1,
							},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 30,
								Meta: &pb.FlamegraphNodeMeta{
									LocationIndex: 2,
									LineIndex:     0,
								},
							}},
						},
						{
							Cumulative: 70,
							Meta: &pb.FlamegraphNodeMeta{
								LocationIndex: 2,
								LineIndex:     1,
							},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 70,
								Meta: &pb.FlamegraphNodeMeta{
									LocationIndex: 2,
									LineIndex:     0,
								},
							}},
						},
					},
				},
				{
					// This node is trimmed because it is below the threshold.
					Cumulative: 3,
					Meta: &pb.FlamegraphNodeMeta{
						LocationIndex: 3,
						LineIndex:     0,
					},
				},
			},
		},
	}
	// trim all children that have less than 10% cumulative value of the parent.
	trimmedGraph := TrimFlamegraph(context.Background(), noop.NewTracerProvider().Tracer(""), fullGraph, 0.1)
	require.Equal(t, &pb.Flamegraph{
		Total:          102,
		Trimmed:        4,
		UntrimmedTotal: 102,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 102,
			Children: []*pb.FlamegraphNode{
				{
					Cumulative: 101,
					Meta: &pb.FlamegraphNodeMeta{
						LocationIndex: 1,
						LineIndex:     0,
					},
					Children: []*pb.FlamegraphNode{
						{
							Cumulative: 30,
							Meta: &pb.FlamegraphNodeMeta{
								LocationIndex: 2,
								LineIndex:     1,
							},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 30,
								Meta: &pb.FlamegraphNodeMeta{
									LocationIndex: 2,
									LineIndex:     0,
								},
							}},
						},
						{
							Cumulative: 70,
							Meta: &pb.FlamegraphNodeMeta{
								LocationIndex: 2,
								LineIndex:     1,
							},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 70,
								Meta: &pb.FlamegraphNodeMeta{
									LocationIndex: 2,
									LineIndex:     0,
								},
							}},
						},
					},
				},
			},
		},
	}, trimmedGraph)
}

func TestFlamegraphTrimmingSingleNodeGraph(t *testing.T) {
	fullGraph := &pb.Flamegraph{
		Total: 100,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 100,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 100,
			}},
		},
	}
	trimmedGraph := TrimFlamegraph(context.Background(), noop.NewTracerProvider().Tracer(""), fullGraph, float32(0.02))
	require.Equal(t, &pb.Flamegraph{
		Total:          100,
		UntrimmedTotal: 100,
		Trimmed:        0,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 100,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 100,
			}},
		},
	}, trimmedGraph)
}

func TestFlamegraphTrimmingNodeWithFlatValues(t *testing.T) {
	fullGraph := &pb.Flamegraph{
		Total: 151,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 151,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 151,
				Children: []*pb.FlamegraphNode{{
					Cumulative: 100,
				}, {
					Cumulative: 1,
				}},
			}},
		},
	}
	trimmedGraph := TrimFlamegraph(context.Background(), noop.NewTracerProvider().Tracer(""), fullGraph, float32(0.02))
	require.Equal(t, &pb.Flamegraph{
		Total:          151,
		UntrimmedTotal: 151,
		Trimmed:        1,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 151,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 151,
				Children: []*pb.FlamegraphNode{{
					Cumulative: 100,
				}},
			}},
		},
	}, trimmedGraph)
}

// TestFlamegraphTrimmingAndFiltering tests that the flamegraph trimming and filtering at the same time.
// The filter removes half of the samples and the trimming removes all samples with less than 50% of the total.
// In the end just a single sample should be left.
func TestFlamegraphTrimmingAndFiltering(t *testing.T) {
	ctx := context.Background()
	var err error

	mappings := []*pprofprofile.Mapping{{
		ID:   1,
		File: "a",
	}}

	functions := []*pprofprofile.Function{{
		ID:   1,
		Name: "1.a",
	}, {
		ID:   2,
		Name: "2.a",
	}, {
		ID:   3,
		Name: "3.a",
	}, {
		ID:   4,
		Name: "4.b",
	}, {
		ID:   5,
		Name: "5.c",
	}, {
		ID:   6,
		Name: "6.b",
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
	}, {
		ID:      6,
		Mapping: mappings[0],
		Line:    []pprofprofile.Line{{Function: functions[5]}},
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			Mapping:  mappings,
			Function: functions,
			Location: locations,
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[1], locations[0]},
				Value:    []int64{2},
			}, {
				Location: []*pprofprofile.Location{locations[4], locations[2], locations[1], locations[0]},
				Value:    []int64{1},
			}, {
				// Only this sample will be in the final flamegraph.
				// The two above will be filtered and the last one will be trimmed.
				Location: []*pprofprofile.Location{locations[3], locations[2], locations[1], locations[0]},
				Value:    []int64{12},
			}, {
				Location: []*pprofprofile.Location{locations[5], locations[3], locations[2], locations[1], locations[0]},
				Value:    []int64{3},
			}},
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	var filtered int64
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	tracer := noop.NewTracerProvider().Tracer("")
	// Create stack filter for function "b" - should filter out the "5.c" function
	filters := []*pb.Filter{
		{
			Filter: &pb.Filter_StackFilter{
				StackFilter: &pb.StackFilter{
					Filter: &pb.StackFilter_Criteria{
						Criteria: &pb.FilterCriteria{
							FunctionName: &pb.StringCondition{
								Condition: &pb.StringCondition_Contains{
									Contains: "b",
								},
							},
						},
					},
				},
			},
		},
	}
	p.Samples, filtered, err = FilterProfileData(ctx, tracer, mem, p.Samples, filters)
	require.NoError(t, err)
	defer func() {
		for _, s := range p.Samples {
			s.Release()
		}
	}()

	op, err := parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	fg, err := GenerateFlamegraphTable(ctx, tracer, op, float32(0.5), NewTableConverterPool()) // 50% threshold
	require.NoError(t, err)

	require.Equal(t, int32(6), fg.Height)

	// The unfiltered flamegraph had 15+3 = 18 samples.
	// There were nodes that got trimmed with a cumulative value of 3.
	require.Equal(t, int64(3), filtered)
	require.Equal(t, int64(3), fg.Trimmed)
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int64(15), fg.Total)

	// Check if tables and thus deduplication was correct and deterministic
	require.Equal(t, []string{"", "a", "1.a", "2.a", "3.a", "4.b", "" /* 6.b*/}, fg.StringTable)
	require.Equal(t, []*metastorepb.Location{
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 1}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 2}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 3}}},
		{MappingIndex: 1, Lines: []*metastorepb.Line{{FunctionIndex: 4}}},
		// The location isn't referenced from the flame graph.
		nil,
	}, fg.Locations)
	require.Equal(t, []*metastorepb.Mapping{
		{BuildIdStringIndex: 0, FileStringIndex: 1},
	}, fg.Mapping)
	require.Equal(t, []*metastorepb.Function{
		{NameStringIndex: 2, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 3, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 4, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		{NameStringIndex: 5, SystemNameStringIndex: 0, FilenameStringIndex: 0},
		// The function isn't referenced from the flame graph.
		nil,
	}, fg.Function)

	// Check the recursive flamegraph that references the tables above.

	expected := &pb.FlamegraphRootNode{
		Cumulative: 15,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 15,
			Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 1},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 15,
				Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 2},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 15,
					Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 3},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 15,
						Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 4},
					}},
				}},
			}},
		}},
	}
	require.Equal(t, expected, fg.Root)
	require.True(t, proto.Equal(expected, fg.Root))
}

func TestTableConverterLocation(t *testing.T) {
	tc := &tableConverter{locationsIndex: map[string]uint32{}}
	id := "foo"
	address := uint64(0x1234)
	index := tc.AddLocation(&metastorepb.Location{Id: id, Address: address})
	l := tc.GetLocation(index)
	require.Equal(t, id, l.Id)
	require.Equal(t, address, l.Address)

	// doesn't exist
	require.Nil(t, tc.GetLocation(0))
	require.Nil(t, tc.GetLocation(2))
}

func TestTableConverterMapping(t *testing.T) {
	tc := &tableConverter{
		stringsIndex:  map[string]uint32{},
		mappingsIndex: map[string]uint32{},
	}
	tc.AddString("")

	in := &metastorepb.Mapping{Id: "foo", File: "file", BuildId: "build"}
	index := tc.AddMapping(in)
	out := tc.GetMapping(index)
	require.Equal(t, in, out)
}

func TestTableConverterFunction(t *testing.T) {
	tc := &tableConverter{
		stringsIndex:   map[string]uint32{},
		functionsIndex: map[string]uint32{},
	}
	tc.AddString("")

	in := &metastorepb.Function{
		Id:         "foo",
		StartLine:  12,
		Name:       "name",
		SystemName: "systemname",
		Filename:   "filename",
	}
	index := tc.AddFunction(in)
	out := tc.GetFunction(index)
	require.Equal(t, in, out)
}

func TestAddGetString(t *testing.T) {
	tc := &tableConverter{stringsIndex: map[string]uint32{}}
	tc.AddString("")

	require.Equal(t, "foo", tc.GetString(tc.AddString("foo")))
	require.Equal(t, "bar", tc.GetString(tc.AddString("bar")))
	require.Equal(t, "foo", tc.GetString(tc.AddString("foo")))
	require.Equal(t, "", tc.GetString(tc.AddString("")))
	// doesn't exist
	require.Equal(t, "", tc.GetString(3))
}

func TestGenerateFlamegraphTrimmingStringTablesCompare(t *testing.T) {
	// Generate a flamegraph with a threshold of 0. This disables trimming.
	original := testGenerateFlamegraphFromProfile(t, 0)
	// Generate a flamegraph with a threshold that enables trimming but so small it doesn't actually trim anything.
	trimmed := testGenerateFlamegraphFromProfile(t, math.SmallestNonzeroFloat32)

	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, original.Total, trimmed.Total)
	require.Equal(t, original.Height, trimmed.Height)
	require.Equal(t, original.Unit, trimmed.Unit)
	require.Equal(t, original.Trimmed, trimmed.Trimmed)

	// Check if table converter has the same number of entries for each type.
	require.Len(t, trimmed.StringTable, len(trimmed.StringTable))
	require.Len(t, trimmed.Locations, len(trimmed.Locations))
	require.Len(t, trimmed.Mapping, len(trimmed.Mapping))
	require.Len(t, trimmed.Function, len(trimmed.Function))

	// sort the tables as trimming is not fully equal but the sorted tables should be equal.
	sort.Strings(original.StringTable)
	sort.Strings(trimmed.StringTable)
	require.Equal(t, original.StringTable, trimmed.StringTable)

	require.Equal(t, original.Root.Cumulative, trimmed.Root.Cumulative)
}

func testGenerateFlamegraphFromProfile(t *testing.T, nodeTrimFraction float32) *pb.Flamegraph {
	t.Helper()

	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	fileContent := MustReadAllGzip(t, "./testdata/profile1.pb.gz")
	p, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	pp, err := PprofToSymbolizedProfile(profile.Meta{}, p, 0, []string{})
	require.NoError(t, err)

	sp, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, pp)
	require.NoError(t, err)

	fg, err := GenerateFlamegraphTable(ctx, tracer, sp, nodeTrimFraction, NewTableConverterPool())
	require.NoError(t, err)

	return fg
}
