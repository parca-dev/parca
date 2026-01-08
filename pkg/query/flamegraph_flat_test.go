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
	"testing"

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

func TestGenerateFlamegraphFlat(t *testing.T) {
	t.Parallel()

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
		Line: []pprofprofile.Line{{
			Function: functions[0],
		}},
	}, {
		ID:      2,
		Mapping: mappings[0],
		Line: []pprofprofile.Line{{
			Function: functions[1],
		}},
	}, {
		ID:      3,
		Mapping: mappings[0],
		Line: []pprofprofile.Line{{
			Function: functions[2],
		}},
	}, {
		ID:      4,
		Mapping: mappings[0],
		Line: []pprofprofile.Line{{
			Function: functions[3],
		}},
	}, {
		ID:      5,
		Mapping: mappings[0],
		Line: []pprofprofile.Line{{
			Function: functions[4],
		}},
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
	fg, err := GenerateFlamegraphFlat(ctx, tracer, op)
	require.NoError(t, err)

	require.True(t, proto.Equal(&pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metastorepb.Function{Id: "unknown-filename/7qd6hvHIBYXdQJ7V9xewooDjrmdIR_zZ0Jveuutjpyg=", Name: "1"},
				Line:     &metastorepb.Line{FunctionId: "unknown-filename/7qd6hvHIBYXdQJ7V9xewooDjrmdIR_zZ0Jveuutjpyg="},
				Location: &metastorepb.Location{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=/mq717xceAZPk_DwjoEV9l4Zea2W6gCPeVQE6xhupIyA=", MappingId: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A="},
				Mapping:  &metastorepb.Mapping{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=", File: "a"},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metastorepb.Function{Id: "unknown-filename/iba2LWuPfiWoW-U7bfl8Y_zmXJV0N22DMycYfD944AA=", Name: "2"},
					Line:     &metastorepb.Line{FunctionId: "unknown-filename/iba2LWuPfiWoW-U7bfl8Y_zmXJV0N22DMycYfD944AA="},
					Location: &metastorepb.Location{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=/nywr5cWVdwJb1cAWbrm2oKprBJKYoVpjEHbtKenHaGg=", MappingId: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A="},
					Mapping:  &metastorepb.Mapping{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=", File: "a"},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metastorepb.Function{Id: "unknown-filename/4CNM2O_LHZCLNRVCXHnlyun6GwI-Sv6rwgMbR7EaJQ4=", Name: "3"},
						Line:     &metastorepb.Line{FunctionId: "unknown-filename/4CNM2O_LHZCLNRVCXHnlyun6GwI-Sv6rwgMbR7EaJQ4="},
						Location: &metastorepb.Location{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=/Ykzo9tYbar2yhdULf19Jp20SZmCJLn1c5TLXLumKSKc=", MappingId: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A="},
						Mapping:  &metastorepb.Mapping{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=", File: "a"},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metastorepb.Function{Id: "unknown-filename/tIwpxw9EeOUPRuqj2MDOrI3yyqNRmHZPT_zpLCp5yhs=", Name: "4"},
							Line:     &metastorepb.Line{FunctionId: "unknown-filename/tIwpxw9EeOUPRuqj2MDOrI3yyqNRmHZPT_zpLCp5yhs="},
							Location: &metastorepb.Location{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=/cukmn1qTXWLpu2SG5ox8x8A2hZqNjS55baOmijXi3co=", MappingId: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A="},
							Mapping:  &metastorepb.Mapping{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=", File: "a"},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metastorepb.Function{Id: "unknown-filename/PwksWp7MLSZfdiTUSy4aP-r0Bjr_O_3-9VEJ0SE_4Yc=", Name: "5"},
							Line:     &metastorepb.Line{FunctionId: "unknown-filename/PwksWp7MLSZfdiTUSy4aP-r0Bjr_O_3-9VEJ0SE_4Yc="},
							Location: &metastorepb.Location{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=/FyW1Ts_USzmLVTuEbWNGW3bLt3vEf8Gn15goyXywkQw=", MappingId: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A="},
							Mapping:  &metastorepb.Mapping{Id: "9eZsDnwt8q-4ctrD1sXhPf2PILaD5V5-iXIfxEN_77A=", File: "a"},
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

	nodeTrimFraction := float32(0)
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	fileContent := MustReadAllGzip(t, "./testdata/profile1.pb.gz")
	p, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	pp, err := PprofToSymbolizedProfile(profile.Meta{}, p, 0, []string{})
	require.NoError(t, err)

	sp, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, pp)
	require.NoError(t, err)

	_, err = GenerateFlamegraphTable(ctx, tracer, sp, nodeTrimFraction, NewTableConverterPool())
	require.NoError(t, err)
}

func TestGenerateFlamegraphWithInlined(t *testing.T) {
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

	tracer := noop.NewTracerProvider().Tracer("")
	fg, err := GenerateFlamegraphFlat(ctx, tracer, op)
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

	fg, err := GenerateFlamegraphFlat(ctx, tracer, op)
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
