// Copyright 2021 The Parca Authors
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
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlamegraphFlat(t *testing.T) {
	t.Parallel()

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
		Mappings: []*metastorepb.Mapping{{
			File: "a",
		}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := metastore.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{{
			Name: "1",
		}, {
			Name: "2",
		}, {
			Name: "3",
		}, {
			Name: "4",
		}, {
			Name: "5",
		}},
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
			Lines: &metastorepb.LocationLines{
				Entries: []*metastorepb.Line{{
					FunctionId: f1.Id,
				}},
			},
		}, {
			MappingId: m.Id,
			Lines: &metastorepb.LocationLines{
				Entries: []*metastorepb.Line{{
					FunctionId: f2.Id,
				}},
			},
		}, {
			MappingId: m.Id,
			Lines: &metastorepb.LocationLines{
				Entries: []*metastorepb.Line{{
					FunctionId: f3.Id,
				}},
			},
		}, {
			MappingId: m.Id,
			Lines: &metastorepb.LocationLines{
				Entries: []*metastorepb.Line{{
					FunctionId: f4.Id,
				}},
			},
		}, {
			MappingId: m.Id,
			Lines: &metastorepb.LocationLines{
				Entries: []*metastorepb.Line{{
					FunctionId: f5.Id,
				}},
			},
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

	fg, err := GenerateFlamegraphFlat(ctx, tracer, p)
	require.NoError(t, err)

	require.True(t, proto.Equal(&pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metastorepb.Function{Id: f1.Id, Name: "1"},
				Line:     &metastorepb.Line{FunctionId: f1.Id},
				Location: &metastorepb.Location{Id: l1.Id, MappingId: m.Id},
				Mapping:  &metastorepb.Mapping{Id: m.Id, File: "a"},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metastorepb.Function{Id: f2.Id, Name: "2"},
					Line:     &metastorepb.Line{FunctionId: f2.Id},
					Location: &metastorepb.Location{Id: l2.Id, MappingId: m.Id},
					Mapping:  &metastorepb.Mapping{Id: m.Id, File: "a"},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metastorepb.Function{Id: f3.Id, Name: "3"},
						Line:     &metastorepb.Line{FunctionId: f3.Id},
						Location: &metastorepb.Location{Id: l3.Id, MappingId: m.Id},
						Mapping:  &metastorepb.Mapping{Id: m.Id, File: "a"},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metastorepb.Function{Id: f4.Id, Name: "4"},
							Line:     &metastorepb.Line{FunctionId: f4.Id},
							Location: &metastorepb.Location{Id: l4.Id, MappingId: m.Id},
							Mapping:  &metastorepb.Mapping{Id: m.Id, File: "a"},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metastorepb.Function{Id: f5.Id, Name: "5"},
							Line:     &metastorepb.Line{FunctionId: f5.Id},
							Location: &metastorepb.Location{Id: l5.Id, MappingId: m.Id},
							Mapping:  &metastorepb.Mapping{Id: m.Id, File: "a"},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}}, fg))
}

func TestGenerateFlamegraphFromProfile(t *testing.T) {
	t.Parallel()

	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		reg,
		tracer,
	)

	testGenerateFlamegraphFromProfile(t, metastore.NewInProcessClient(l))
}

func testGenerateFlamegraphFromProfile(t *testing.T, l metastorepb.MetastoreServiceClient) *pb.Flamegraph {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	fileContent := MustReadAllGzip(t, "./testdata/profile1.pb.gz")
	p := &pprofpb.Profile{}
	err := p.UnmarshalVT(fileContent)
	require.NoError(t, err)

	normalizer := parcacol.NewNormalizer(l)
	profiles, err := normalizer.NormalizePprof(ctx, "test", p, false)
	require.NoError(t, err)

	sp, err := parcacol.NewArrowToProfileConverter(tracer, l).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	fg, err := GenerateFlamegraphFlat(ctx, tracer, sp)
	require.NoError(t, err)

	return fg
}

func TestGenerateFlamegraphWithInlined(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
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
	b := bytes.NewBuffer(nil)
	err := (&pprofprofile.Profile{
		SampleType: []*profile.ValueType{{Type: "alloc_space", Unit: "bytes"}},
		PeriodType: &profile.ValueType{Type: "space", Unit: "bytes"},
		Sample:     samples,
		Location:   locations,
		Function:   functions,
	}).Write(b)
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	err = p.UnmarshalVT(MustDecompressGzip(t, b.Bytes()))
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(store)
	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", p, false)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	fg, err := GenerateFlamegraphFlat(ctx, tracer, symbolizedProfile)
	require.NoError(t, err)

	require.Equal(t, &pb.Flamegraph{
		Total:  1,
		Height: 4,
		Unit:   "bytes",
		Root: &pb.FlamegraphRootNode{
			Cumulative: 1,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 1,
				Meta: &pb.FlamegraphNodeMeta{
					Location: &metastorepb.Location{
						Id:      "unknown-mapping/wJmUgWzHpt_1Bzsh-bnkPo913VmZj2rOa1tl20PcJvA=",
						Address: 94658718830132,
					},
					Line: &metastorepb.Line{
						FunctionId: "X41EE15Xxty3PrlqUcutB78Ky065QN8ikr3Oe3lJCyk=/TwSdYgG7s5EyzoYpXaSLH1hgWDHZAx71F2B7rfxyvLc=",
						Line:       173,
					},
					Function: &metastorepb.Function{
						Id:         "X41EE15Xxty3PrlqUcutB78Ky065QN8ikr3Oe3lJCyk=/TwSdYgG7s5EyzoYpXaSLH1hgWDHZAx71F2B7rfxyvLc=",
						StartLine:  0,
						Name:       "net.(*netFD).accept",
						SystemName: "net.(*netFD).accept",
						Filename:   "net/fd_unix.go",
					},
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 1,
					Meta: &pb.FlamegraphNodeMeta{
						Location: &metastorepb.Location{
							Id:      "unknown-mapping/JzDHFIpGrx8A4p04YPmWW5GMV-OZyMuUXJtG-4-psFQ=",
							Address: 94658718611115,
						},
						Line: &metastorepb.Line{
							FunctionId: "vMl-yJEz1SsY8o6W-Lxuzeq9jsKV_ED-k-qkZ4MwL7Y=/_wsHOkzMl1Lpbx9VXgtine4hJArOr2seSnHY62sf-Q8=",
							Line:       89,
						},
						Function: &metastorepb.Function{
							Id:         "vMl-yJEz1SsY8o6W-Lxuzeq9jsKV_ED-k-qkZ4MwL7Y=/_wsHOkzMl1Lpbx9VXgtine4hJArOr2seSnHY62sf-Q8=",
							StartLine:  0,
							Name:       "internal/poll.(*FD).Accept",
							SystemName: "internal/poll.(*FD).Accept",
							Filename:   "internal/poll/fd_unix.go",
						},
					},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 1,
						Meta: &pb.FlamegraphNodeMeta{
							Location: &metastorepb.Location{
								Id:      "unknown-mapping/JzDHFIpGrx8A4p04YPmWW5GMV-OZyMuUXJtG-4-psFQ=",
								Address: 94658718611115,
							},
							Function: &metastorepb.Function{
								Id:         "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/S7CG45dizCdxVa6kkqIIlY8FYFla8TBKHXog0LoR85Q=",
								Name:       "internal/poll.(*pollDesc).waitRead",
								SystemName: "internal/poll.(*pollDesc).waitRead",
								Filename:   "internal/poll/fd_poll_runtime.go",
							},
							Line: &metastorepb.Line{
								FunctionId: "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/S7CG45dizCdxVa6kkqIIlY8FYFla8TBKHXog0LoR85Q=",
								Line:       402,
							},
						},
						Children: []*pb.FlamegraphNode{{
							Cumulative: 1,
							Meta: &pb.FlamegraphNodeMeta{
								Location: &metastorepb.Location{
									Id:      "unknown-mapping/ErbUKNq4N3aXRlqxBjLFiUkt-1eWmB7Bj4rz5qcACpc=",
									Address: 94658718597969,
								},
								Function: &metastorepb.Function{
									Id:         "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/QsXGa5kGpsxygKpOahq1YmSBkx2dUch-Nmr0l3-AkEQ=",
									Name:       "internal/poll.(*pollDesc).wait",
									SystemName: "internal/poll.(*pollDesc).wait",
									Filename:   "internal/poll/fd_poll_runtime.go",
								},
								Line: &metastorepb.Line{
									FunctionId: "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/QsXGa5kGpsxygKpOahq1YmSBkx2dUch-Nmr0l3-AkEQ=",
									Line:       84,
								},
							},
							Children: nil,
						}},
					}},
				}},
			}},
		},
	}, fg)
}

func TestGenerateFlamegraphWithInlinedExisting(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	store := metastoretest.NewTestMetastore(t, logger, reg, tracer)
	metastore := metastore.NewInProcessClient(store)

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
		{ID: 3, Address: 94658718597969, Line: []profile.Line{{Line: 84, Function: functions[3]}}},
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
	b := bytes.NewBuffer(nil)
	err := (&pprofprofile.Profile{
		SampleType: []*profile.ValueType{{Type: "", Unit: ""}},
		PeriodType: &profile.ValueType{Type: "", Unit: ""},
		Sample:     samples,
		Location:   locations,
		Function:   functions,
	}).Write(b)
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	err = p.UnmarshalVT(MustDecompressGzip(t, b.Bytes()))
	require.NoError(t, err)

	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "", p, false)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	fg, err := GenerateFlamegraphFlat(ctx, tracer, symbolizedProfile)
	require.NoError(t, err)

	expected := &pb.Flamegraph{
		Total:  3,
		Height: 4,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 3,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 3,
				Meta: &pb.FlamegraphNodeMeta{
					Location: &metastorepb.Location{
						Id:      "unknown-mapping/wJmUgWzHpt_1Bzsh-bnkPo913VmZj2rOa1tl20PcJvA=",
						Address: 94658718830132,
					},
					Line: &metastorepb.Line{
						FunctionId: "X41EE15Xxty3PrlqUcutB78Ky065QN8ikr3Oe3lJCyk=/TwSdYgG7s5EyzoYpXaSLH1hgWDHZAx71F2B7rfxyvLc=",
						Line:       173,
					},
					Function: &metastorepb.Function{
						Id:         "X41EE15Xxty3PrlqUcutB78Ky065QN8ikr3Oe3lJCyk=/TwSdYgG7s5EyzoYpXaSLH1hgWDHZAx71F2B7rfxyvLc=",
						StartLine:  0,
						Name:       "net.(*netFD).accept",
						SystemName: "net.(*netFD).accept",
						Filename:   "net/fd_unix.go",
					},
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 3,
					Meta: &pb.FlamegraphNodeMeta{
						Location: &metastorepb.Location{
							Id:      "unknown-mapping/JzDHFIpGrx8A4p04YPmWW5GMV-OZyMuUXJtG-4-psFQ=",
							Address: 94658718611115,
						},
						Line: &metastorepb.Line{
							FunctionId: "vMl-yJEz1SsY8o6W-Lxuzeq9jsKV_ED-k-qkZ4MwL7Y=/_wsHOkzMl1Lpbx9VXgtine4hJArOr2seSnHY62sf-Q8=",
							Line:       89,
						},
						Function: &metastorepb.Function{
							Id:         "vMl-yJEz1SsY8o6W-Lxuzeq9jsKV_ED-k-qkZ4MwL7Y=/_wsHOkzMl1Lpbx9VXgtine4hJArOr2seSnHY62sf-Q8=",
							StartLine:  0,
							Name:       "internal/poll.(*FD).Accept",
							SystemName: "internal/poll.(*FD).Accept",
							Filename:   "internal/poll/fd_unix.go",
						},
					},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 3,
						Meta: &pb.FlamegraphNodeMeta{
							Location: &metastorepb.Location{
								Id:      "unknown-mapping/JzDHFIpGrx8A4p04YPmWW5GMV-OZyMuUXJtG-4-psFQ=",
								Address: 94658718611115,
							},
							Function: &metastorepb.Function{
								Id:         "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/S7CG45dizCdxVa6kkqIIlY8FYFla8TBKHXog0LoR85Q=",
								Name:       "internal/poll.(*pollDesc).waitRead",
								SystemName: "internal/poll.(*pollDesc).waitRead",
								Filename:   "internal/poll/fd_poll_runtime.go",
							},
							Line: &metastorepb.Line{
								FunctionId: "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/S7CG45dizCdxVa6kkqIIlY8FYFla8TBKHXog0LoR85Q=",
								Line:       402,
							},
						},
						Children: []*pb.FlamegraphNode{{
							Cumulative: 1,
							Meta: &pb.FlamegraphNodeMeta{
								Location: &metastorepb.Location{
									Id:      "unknown-mapping/ErbUKNq4N3aXRlqxBjLFiUkt-1eWmB7Bj4rz5qcACpc=",
									Address: 94658718597969,
								},
								Function: &metastorepb.Function{
									Id:         "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/QsXGa5kGpsxygKpOahq1YmSBkx2dUch-Nmr0l3-AkEQ=",
									Name:       "internal/poll.(*pollDesc).wait",
									SystemName: "internal/poll.(*pollDesc).wait",
									Filename:   "internal/poll/fd_poll_runtime.go",
								},
								Line: &metastorepb.Line{
									FunctionId: "di0EZIvkHha8U8He-ZgW0DFTfFynx34ltT5cbHWvtXY=/QsXGa5kGpsxygKpOahq1YmSBkx2dUch-Nmr0l3-AkEQ=",
									Line:       84,
								},
							},
							Children: nil,
						}},
					}},
				}},
			}},
		},
	}

	require.Equal(t, expected, fg)
}
