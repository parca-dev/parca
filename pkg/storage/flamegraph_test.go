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

package storage

import (
	"context"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestTreeStack(t *testing.T) {
	s := TreeStack{}
	s.Push(&TreeStackEntry{nodes: []*pb.FlamegraphNode{{Meta: &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "a"}}}}})
	s.Push(&TreeStackEntry{nodes: []*pb.FlamegraphNode{{Meta: &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "b"}}}}})

	require.Equal(t, 2, s.Size())

	e, hasMore := s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "b", e.nodes[0].Meta.Function.Name)

	require.Equal(t, 1, s.Size())

	e, hasMore = s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "a", e.nodes[0].Meta.Function.Name)

	require.Equal(t, 0, s.Size())

	_, hasMore = s.Pop()
	require.False(t, hasMore)
}

func TestLinesToTreeNodes(t *testing.T) {
	nodes := linesToTreeNodes(&metastore.Location{}, uuid.MustParse("00000000-0000-0000-0000-000000000001"), &pb.Mapping{}, []metastore.LocationLine{
		{
			Function: &metastore.Function{
				FunctionKey: metastore.FunctionKey{
					Name: "memcpy",
				},
			},
		}, {
			Function: &metastore.Function{
				FunctionKey: metastore.FunctionKey{
					Name: "printf",
				},
			},
		}, {
			Function: &metastore.Function{
				FunctionKey: metastore.FunctionKey{
					Name: "log",
				},
			},
		},
	})

	require.Equal(t, []*pb.FlamegraphNode{{
		Meta: &pb.FlamegraphNodeMeta{
			Function: &pb.Function{
				Id:   "00000000-0000-0000-0000-000000000000",
				Name: "log",
			},
			Line: &pb.Line{
				LocationId: "00000000-0000-0000-0000-000000000000",
				FunctionId: "00000000-0000-0000-0000-000000000000",
			},
			Location: &pb.Location{
				Id:        "00000000-0000-0000-0000-000000000000",
				MappingId: "00000000-0000-0000-0000-000000000001",
			},
			Mapping: &pb.Mapping{},
		},
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Id:   "00000000-0000-0000-0000-000000000000",
					Name: "printf",
				},
				Line: &pb.Line{
					LocationId: "00000000-0000-0000-0000-000000000000",
					FunctionId: "00000000-0000-0000-0000-000000000000",
				},
				Location: &pb.Location{
					Id:        "00000000-0000-0000-0000-000000000000",
					MappingId: "00000000-0000-0000-0000-000000000001",
				},
				Mapping: &pb.Mapping{},
			},
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{
						Id:   "00000000-0000-0000-0000-000000000000",
						Name: "memcpy",
					},
					Line: &pb.Line{
						LocationId: "00000000-0000-0000-0000-000000000000",
						FunctionId: "00000000-0000-0000-0000-000000000000",
					},
					Location: &pb.Location{
						Id:        "00000000-0000-0000-0000-000000000000",
						MappingId: "00000000-0000-0000-0000-000000000001",
					},
					Mapping: &pb.Mapping{},
				},
			}},
		}},
	}, {
		Meta: &pb.FlamegraphNodeMeta{
			Function: &pb.Function{
				Id:   "00000000-0000-0000-0000-000000000000",
				Name: "printf",
			},
			Line: &pb.Line{
				LocationId: "00000000-0000-0000-0000-000000000000",
				FunctionId: "00000000-0000-0000-0000-000000000000",
			},
			Location: &pb.Location{
				Id:        "00000000-0000-0000-0000-000000000000",
				MappingId: "00000000-0000-0000-0000-000000000001",
			},
			Mapping: &pb.Mapping{},
		},
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Id:   "00000000-0000-0000-0000-000000000000",
					Name: "memcpy",
				},
				Line: &pb.Line{
					LocationId: "00000000-0000-0000-0000-000000000000",
					FunctionId: "00000000-0000-0000-0000-000000000000",
				},
				Location: &pb.Location{
					Id:        "00000000-0000-0000-0000-000000000000",
					MappingId: "00000000-0000-0000-0000-000000000001",
				},
				Mapping: &pb.Mapping{},
			},
		}},
	}, {
		Meta: &pb.FlamegraphNodeMeta{
			Function: &pb.Function{
				Id:   "00000000-0000-0000-0000-000000000000",
				Name: "memcpy",
			},
			Line: &pb.Line{
				LocationId: "00000000-0000-0000-0000-000000000000",
				FunctionId: "00000000-0000-0000-0000-000000000000",
			},
			Location: &pb.Location{
				Id:        "00000000-0000-0000-0000-000000000000",
				MappingId: "00000000-0000-0000-0000-000000000001",
			},
			Mapping: &pb.Mapping{},
		},
	}}, nodes)
}

type fakeLocations struct {
	m map[uuid.UUID]*metastore.Location
}

func (l *fakeLocations) GetLocationsByIDs(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]*metastore.Location, error) {
	return l.m, nil
}

func TestGenerateInlinedFunctionFlamegraph(t *testing.T) {
	ctx := context.Background()

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	m := &metastore.Mapping{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"),
	}
	l := &fakeLocations{m: map[uuid.UUID]*metastore.Location{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f1"),
					FunctionKey: metastore.FunctionKey{Name: "1"},
				},
			}},
		},
		uuid.MustParse("00000000-0000-0000-0000-000000000002"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f3"),
					FunctionKey: metastore.FunctionKey{Name: "3"},
				},
			}, {
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f2"),
					FunctionKey: metastore.FunctionKey{Name: "2"},
				},
			}},
		},
	}}

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: pt},
	)
	require.NoError(t, err)
	require.Equal(t, &pb.Flamegraph{Height: 3, Total: 2, Root: &pb.FlamegraphRootNode{
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Id:   "00000000-0000-0000-0000-0000000000f1",
					Name: "1",
				},
				Line: &pb.Line{
					LocationId: "00000000-0000-0000-0000-000000000001",
					FunctionId: "00000000-0000-0000-0000-0000000000f1",
				},
				Location: &pb.Location{
					Id:        "00000000-0000-0000-0000-000000000001",
					MappingId: "00000000-0000-0000-0000-0000000000a1",
				},
				Mapping: &pb.Mapping{
					Id: "00000000-0000-0000-0000-0000000000a1",
				},
			},
			Cumulative: 2,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{
						Id:   "00000000-0000-0000-0000-0000000000f2",
						Name: "2",
					},
					Line: &pb.Line{
						LocationId: "00000000-0000-0000-0000-000000000002",
						FunctionId: "00000000-0000-0000-0000-0000000000f2",
					},
					Location: &pb.Location{
						Id:        "00000000-0000-0000-0000-000000000002",
						MappingId: "00000000-0000-0000-0000-0000000000a1",
					},
					Mapping: &pb.Mapping{
						Id: "00000000-0000-0000-0000-0000000000a1",
					},
				},
				Cumulative: 2,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{
							Id:   "00000000-0000-0000-0000-0000000000f3",
							Name: "3",
						},
						Line: &pb.Line{
							LocationId: "00000000-0000-0000-0000-000000000002",
							FunctionId: "00000000-0000-0000-0000-0000000000f3",
						},
						Location: &pb.Location{
							Id:        "00000000-0000-0000-0000-000000000002",
							MappingId: "00000000-0000-0000-0000-0000000000a1",
						},
						Mapping: &pb.Mapping{
							Id: "00000000-0000-0000-0000-0000000000a1",
						},
					},
					Cumulative: 2,
				}},
			}},
		}},
	}}, fg)
}

func TestGenerateFlamegraph(t *testing.T) {
	ctx := context.Background()

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))
	pt.Insert(makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000005"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))
	pt.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	m := &metastore.Mapping{
		ID: uuid.MustParse("00000000-0000-0000-0000-0000000000a1"),
	}
	l := &fakeLocations{m: map[uuid.UUID]*metastore.Location{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f1"),
					FunctionKey: metastore.FunctionKey{Name: "1"},
				},
			}},
		},
		uuid.MustParse("00000000-0000-0000-0000-000000000002"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f2"),
					FunctionKey: metastore.FunctionKey{Name: "2"},
				},
			}},
		},
		uuid.MustParse("00000000-0000-0000-0000-000000000003"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f3"),
					FunctionKey: metastore.FunctionKey{Name: "3"},
				},
			}},
		},
		uuid.MustParse("00000000-0000-0000-0000-000000000004"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000004"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f4"),
					FunctionKey: metastore.FunctionKey{Name: "4"},
				},
			}},
		},
		uuid.MustParse("00000000-0000-0000-0000-000000000005"): {
			ID:      uuid.MustParse("00000000-0000-0000-0000-000000000005"),
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{
					ID:          uuid.MustParse("00000000-0000-0000-0000-0000000000f5"),
					FunctionKey: metastore.FunctionKey{Name: "5"},
				},
			}},
		},
	}}

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: pt},
	)
	require.NoError(t, err)
	require.Equal(t, &pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Id:   "00000000-0000-0000-0000-0000000000f1",
					Name: "1",
				},
				Line: &pb.Line{
					LocationId: "00000000-0000-0000-0000-000000000001",
					FunctionId: "00000000-0000-0000-0000-0000000000f1",
				},
				Location: &pb.Location{
					Id:        "00000000-0000-0000-0000-000000000001",
					MappingId: "00000000-0000-0000-0000-0000000000a1",
				},
				Mapping: &pb.Mapping{
					Id: "00000000-0000-0000-0000-0000000000a1",
				},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{
						Id:   "00000000-0000-0000-0000-0000000000f2",
						Name: "2",
					},
					Line: &pb.Line{
						LocationId: "00000000-0000-0000-0000-000000000002",
						FunctionId: "00000000-0000-0000-0000-0000000000f2",
					},
					Location: &pb.Location{
						Id:        "00000000-0000-0000-0000-000000000002",
						MappingId: "00000000-0000-0000-0000-0000000000a1",
					},
					Mapping: &pb.Mapping{
						Id: "00000000-0000-0000-0000-0000000000a1",
					},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{
							Id:   "00000000-0000-0000-0000-0000000000f3",
							Name: "3",
						},
						Line: &pb.Line{
							LocationId: "00000000-0000-0000-0000-000000000003",
							FunctionId: "00000000-0000-0000-0000-0000000000f3",
						},
						Location: &pb.Location{
							Id:        "00000000-0000-0000-0000-000000000003",
							MappingId: "00000000-0000-0000-0000-0000000000a1",
						},
						Mapping: &pb.Mapping{
							Id: "00000000-0000-0000-0000-0000000000a1",
						},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{
								Id:   "00000000-0000-0000-0000-0000000000f4",
								Name: "4",
							},
							Line: &pb.Line{
								LocationId: "00000000-0000-0000-0000-000000000004",
								FunctionId: "00000000-0000-0000-0000-0000000000f4",
							},
							Location: &pb.Location{
								Id:        "00000000-0000-0000-0000-000000000004",
								MappingId: "00000000-0000-0000-0000-0000000000a1",
							},
							Mapping: &pb.Mapping{
								Id: "00000000-0000-0000-0000-0000000000a1",
							},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{
								Id:   "00000000-0000-0000-0000-0000000000f5",
								Name: "5",
							},
							Line: &pb.Line{
								LocationId: "00000000-0000-0000-0000-000000000005",
								FunctionId: "00000000-0000-0000-0000-0000000000f5",
							},
							Location: &pb.Location{
								Id:        "00000000-0000-0000-0000-000000000005",
								MappingId: "00000000-0000-0000-0000-0000000000a1",
							},
							Mapping: &pb.Mapping{
								Id: "00000000-0000-0000-0000-0000000000a1",
							},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}},
		fg)
}

func testGenerateFlamegraphFromProfileTree(t *testing.T, l metastore.ProfileMetaStore) *pb.Flamegraph {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	profileTree, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: profileTree, Meta: InstantProfileMeta{
			SampleType: ValueType{Unit: "count"},
		}},
	)
	require.NoError(t, err)

	return fg
}

func TestGenerateFlamegraphFromProfileTree(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		l.Close()
	})

	testGenerateFlamegraphFromProfileTree(t, l)
}

func testGenerateFlamegraphFromInstantProfile(t *testing.T, l metastore.ProfileMetaStore) *pb.Flamegraph {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	require.NoError(t, err)
	s := NewMemSeries(1, labels.Labels{{Name: "test_name", Value: "test_value"}}, func(int64) {}, newHeadChunkPool())
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	prof, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)
	require.NoError(t, app.Append(ctx, prof))

	it := s.Iterator()
	require.True(t, it.Next())
	require.NoError(t, it.Err())
	instantProfile := it.At()

	fg, err := GenerateFlamegraph(
		ctx,
		tracer,
		l,
		instantProfile,
	)
	require.NoError(t, err)
	return fg
}

func TestGenerateFlamegraphFromInstantProfile(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		l.Close()
	})

	testGenerateFlamegraphFromInstantProfile(t, l)
}

func TestFlamegraphConsistency(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		l.Close()
	})

	require.Equal(
		t,
		testGenerateFlamegraphFromProfileTree(t, l),
		testGenerateFlamegraphFromInstantProfile(t, l),
	)
}

func TestGenerateFlamegraphFromMergeProfile(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		l.Close()
	})

	testGenerateFlamegraphFromMergeProfile(t, l)
}

func testGenerateFlamegraphFromMergeProfile(t *testing.T, l metastore.ProfileMetaStore) *pb.Flamegraph {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(t, err)
	p2, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	prof1, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)
	prof2, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p2, 0)
	require.NoError(t, err)

	m, err := NewMergeProfile(prof1, prof2)
	require.NoError(t, err)

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		m,
	)
	require.NoError(t, err)

	return fg
}

func TestControlGenerateFlamegraphFromMergeProfile(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/merge.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"controlgenerateflamegraphfrommergeprofile",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	profileTree, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: profileTree, Meta: InstantProfileMeta{
			SampleType: ValueType{Unit: "count"},
		}},
	)
	require.NoError(t, err)

	mfg := testGenerateFlamegraphFromMergeProfile(t, l)
	require.Equal(t, fg, mfg)
}

func BenchmarkGenerateFlamegraph(b *testing.B) {
	ctx := context.Background()

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"flamegraph",
	)
	b.Cleanup(func() {
		l.Close()
	})
	require.NoError(b, err)
	profileTree, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = GenerateFlamegraph(
			ctx,
			trace.NewNoopTracerProvider().Tracer(""),
			l,
			&Profile{Tree: profileTree},
		)
		require.NoError(b, err)
	}
}

func TestAggregateByFunction(t *testing.T) {
	fg := &pb.Flamegraph{Total: 12, Root: &pb.FlamegraphRootNode{
		Cumulative: 12,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Name: "1",
				},
				Line:     &pb.Line{},
				Location: &pb.Location{},
			},
			Cumulative: 12,
			Children: []*pb.FlamegraphNode{
				{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{
							Name: "2",
						},
						Line:     &pb.Line{},
						Location: &pb.Location{},
					},
					Cumulative: 6,
					Children: []*pb.FlamegraphNode{{
						Cumulative: 4,
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{
								Name: "3",
							},
							Line:     &pb.Line{},
							Location: &pb.Location{},
						},
						Children: []*pb.FlamegraphNode{{
							Meta: &pb.FlamegraphNodeMeta{
								Function: &pb.Function{
									Name: "4",
								},
								Line:     &pb.Line{},
								Location: &pb.Location{},
							},
							Cumulative: 3,
						}, {
							Meta: &pb.FlamegraphNodeMeta{
								Function: &pb.Function{
									Name: "5",
								},
								Line:     &pb.Line{},
								Location: &pb.Location{},
							},
							Cumulative: 1,
						}},
					}},
				},
				{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{
							Name: "2",
						},
						Line:     &pb.Line{},
						Location: &pb.Location{},
					},
					Cumulative: 6,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{
								Name: "3",
							},
							Line:     &pb.Line{},
							Location: &pb.Location{},
						},
						Cumulative: 4,
						Children: []*pb.FlamegraphNode{{
							Meta: &pb.FlamegraphNodeMeta{
								Function: &pb.Function{
									Name: "4",
								},
								Line:     &pb.Line{},
								Location: &pb.Location{},
							},
							Cumulative: 3,
						}, {
							Meta: &pb.FlamegraphNodeMeta{
								Function: &pb.Function{
									Name: "5",
								},
								Line:     &pb.Line{},
								Location: &pb.Location{},
							},
							Cumulative: 1,
						}},
					}},
				},
			},
		}, {
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Name: "1",
				},
				Line:     &pb.Line{},
				Location: &pb.Location{},
			},
			Cumulative: 2,
		}},
	}}

	afg := &pb.Flamegraph{Total: 12, Root: &pb.FlamegraphRootNode{
		Cumulative: 12,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Name: "1",
				},
				Location: &pb.Location{},
			},
			Cumulative: 14,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 12,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{
						Name: "2",
					},
					Location: &pb.Location{},
				},
				Children: []*pb.FlamegraphNode{
					{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{
								Name: "3",
							},
							Location: &pb.Location{},
						},
						Cumulative: 8,
						Children: []*pb.FlamegraphNode{
							{
								Meta: &pb.FlamegraphNodeMeta{
									Function: &pb.Function{
										Name: "4",
									},
									Location: &pb.Location{},
								},
								Cumulative: 6,
							}, {
								Meta: &pb.FlamegraphNodeMeta{
									Function: &pb.Function{
										Name: "5",
									},
									Location: &pb.Location{},
								},
								Cumulative: 2,
							},
						},
					},
				},
			},
			},
		}},
	}}

	require.Equal(t, afg, aggregateByFunction(fg))
}
