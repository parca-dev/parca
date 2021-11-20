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
	metapb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

func TestTreeStack(t *testing.T) {
	s := TreeStack{}
	s.Push(&TreeStackEntry{nodes: []*pb.FlamegraphNode{{Meta: &pb.FlamegraphNodeMeta{Function: &metapb.Function{Name: "a"}}}}})
	s.Push(&TreeStackEntry{nodes: []*pb.FlamegraphNode{{Meta: &pb.FlamegraphNodeMeta{Function: &metapb.Function{Name: "b"}}}}})

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

func uuidBytes(s string) []byte {
	u, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return u[:]
}

func TestLinesToTreeNodes(t *testing.T) {
	nodes := linesToTreeNodes(&metastore.Location{ID: uuid.Nil}, &metapb.Mapping{Id: uuidBytes("00000000-0000-0000-0000-000000000001")}, []metastore.LocationLine{
		{
			Function: &metapb.Function{
				Id:   uuid.Nil[:],
				Name: "memcpy",
			},
		}, {
			Function: &metapb.Function{
				Id:   uuid.Nil[:],
				Name: "printf",
			},
		}, {
			Function: &metapb.Function{
				Id:   uuid.Nil[:],
				Name: "log",
			},
		},
	})

	require.Equal(t, []*pb.FlamegraphNode{{
		Meta: &pb.FlamegraphNodeMeta{
			Function: &metapb.Function{
				Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
				Name: "log",
			},
			Line: &metapb.Line{
				FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
			},
			Location: &metapb.Location{
				Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
				MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
			Mapping: &metapb.Mapping{
				Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
		},
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
					Name: "printf",
				},
				Line: &metapb.Line{
					FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
				},
				Location: &metapb.Location{
					Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
					MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
				},
				Mapping: &metapb.Mapping{
					Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
				},
			},
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{
						Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
						Name: "memcpy",
					},
					Line: &metapb.Line{
						FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
					},
					Location: &metapb.Location{
						Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
						MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
					},
					Mapping: &metapb.Mapping{
						Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
					},
				},
			}},
		}},
	}, {
		Meta: &pb.FlamegraphNodeMeta{
			Function: &metapb.Function{
				Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
				Name: "printf",
			},
			Line: &metapb.Line{
				FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
			},
			Location: &metapb.Location{
				Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
				MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
			Mapping: &metapb.Mapping{
				Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
		},
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
					Name: "memcpy",
				},
				Line: &metapb.Line{
					FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
				},
				Location: &metapb.Location{
					Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
					MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
				},
				Mapping: &metapb.Mapping{
					Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
				},
			},
		}},
	}, {
		Meta: &pb.FlamegraphNodeMeta{
			Function: &metapb.Function{
				Id:   uuidBytes("00000000-0000-0000-0000-000000000000"),
				Name: "memcpy",
			},
			Line: &metapb.Line{
				FunctionId: uuidBytes("00000000-0000-0000-0000-000000000000"),
			},
			Location: &metapb.Location{
				Id:        uuidBytes("00000000-0000-0000-0000-000000000000"),
				MappingId: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
			Mapping: &metapb.Mapping{
				Id: uuidBytes("00000000-0000-0000-0000-000000000001"),
			},
		},
	}}, nodes)
}

func TestGenerateInlinedFunctionFlamegraph(t *testing.T) {
	ctx := context.Background()
	var err error

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)

	m := &metapb.Mapping{
		File: "a",
	}
	m.Id, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metapb.Function{
		Name: "1",
	}
	f1.Id, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metapb.Function{
		Name: "2",
	}
	f2.Id, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metapb.Function{
		Name: "3",
	}
	f3.Id, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1ID, err := l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l1.ID, err = uuid.FromBytes(l1ID)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f3,
			},
			{
				Function: f2,
			},
		},
	}
	l2ID, err := l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	l2.ID, err = uuid.FromBytes(l2ID)
	require.NoError(t, err)

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		l2.ID,
		l1.ID,
	}))

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: pt},
	)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.Flamegraph{Height: 3, Total: 2, Root: &pb.FlamegraphRootNode{
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Id:   f1.Id,
					Name: "1",
				},
				Line: &metapb.Line{
					FunctionId: f1.Id,
				},
				Location: &metapb.Location{
					Id:        l1.ID[:],
					MappingId: m.Id,
				},
				Mapping: &metapb.Mapping{
					Id:   m.Id,
					File: "a",
				},
			},
			Cumulative: 2,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{
						Id:   f2.Id,
						Name: "2",
					},
					Line: &metapb.Line{
						FunctionId: f2.Id,
					},
					Location: &metapb.Location{
						Id:        l2.ID[:],
						MappingId: m.Id,
					},
					Mapping: &metapb.Mapping{
						Id:   m.Id,
						File: "a",
					},
				},
				Cumulative: 2,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{
							Id:   f3.Id,
							Name: "3",
						},
						Line: &metapb.Line{
							FunctionId: f3.Id,
						},
						Location: &metapb.Location{
							Id:        l2.ID[:],
							MappingId: m.Id,
						},
						Mapping: &metapb.Mapping{
							Id:   m.Id,
							File: "a",
						},
					},
					Cumulative: 2,
				}},
			}},
		}},
	}}, fg))
}

func TestGenerateFlamegraph(t *testing.T) {
	ctx := context.Background()
	var err error
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)

	m := &metapb.Mapping{
		File: "a",
	}
	m.Id, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metapb.Function{
		Name: "1",
	}
	f1.Id, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metapb.Function{
		Name: "2",
	}
	f2.Id, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metapb.Function{
		Name: "3",
	}
	f3.Id, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	f4 := &metapb.Function{
		Name: "4",
	}
	f4.Id, err = l.CreateFunction(ctx, f4)
	require.NoError(t, err)

	f5 := &metapb.Function{
		Name: "5",
	}
	f5.Id, err = l.CreateFunction(ctx, f5)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1ID, err := l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l1.ID, err = uuid.FromBytes(l1ID)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f2,
			},
		},
	}
	l2ID, err := l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	l2.ID, err = uuid.FromBytes(l2ID)
	require.NoError(t, err)

	l3 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f3,
			},
		},
	}
	l3ID, err := l.CreateLocation(ctx, l3)
	require.NoError(t, err)

	l3.ID, err = uuid.FromBytes(l3ID)
	require.NoError(t, err)

	l4 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f4,
			},
		},
	}
	l4ID, err := l.CreateLocation(ctx, l4)
	require.NoError(t, err)

	l4.ID, err = uuid.FromBytes(l4ID)
	require.NoError(t, err)

	l5 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f5,
			},
		},
	}
	l5ID, err := l.CreateLocation(ctx, l5)
	require.NoError(t, err)

	l5.ID, err = uuid.FromBytes(l5ID)
	require.NoError(t, err)

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		l2.ID,
		l1.ID,
	}))
	pt.Insert(makeSample(1, []uuid.UUID{
		l5.ID,
		l3.ID,
		l2.ID,
		l1.ID,
	}))
	pt.Insert(makeSample(3, []uuid.UUID{
		l4.ID,
		l3.ID,
		l2.ID,
		l1.ID,
	}))

	fg, err := GenerateFlamegraph(
		ctx,
		trace.NewNoopTracerProvider().Tracer(""),
		l,
		&Profile{Tree: pt},
	)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Id:   f1.Id,
					Name: "1",
				},
				Line: &metapb.Line{
					FunctionId: f1.Id,
				},
				Location: &metapb.Location{
					Id:        l1.ID[:],
					MappingId: m.Id,
				},
				Mapping: &metapb.Mapping{
					Id:   m.Id,
					File: "a",
				},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{
						Id:   f2.Id,
						Name: "2",
					},
					Line: &metapb.Line{
						FunctionId: f2.Id,
					},
					Location: &metapb.Location{
						Id:        l2.ID[:],
						MappingId: m.Id,
					},
					Mapping: &metapb.Mapping{
						Id:   m.Id,
						File: "a",
					},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{
							Id:   f3.Id,
							Name: "3",
						},
						Line: &metapb.Line{
							FunctionId: f3.Id,
						},
						Location: &metapb.Location{
							Id:        l3.ID[:],
							MappingId: m.Id,
						},
						Mapping: &metapb.Mapping{
							Id:   m.Id,
							File: "a",
						},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{
								Id:   f4.Id,
								Name: "4",
							},
							Line: &metapb.Line{
								FunctionId: f4.Id,
							},
							Location: &metapb.Location{
								Id:        l4.ID[:],
								MappingId: m.Id,
							},
							Mapping: &metapb.Mapping{
								Id:   m.Id,
								File: "a",
							},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{
								Id:   f5.Id,
								Name: "5",
							},
							Line: &metapb.Line{
								FunctionId: f5.Id,
							},
							Location: &metapb.Location{
								Id:        l5.ID[:],
								MappingId: m.Id,
							},
							Mapping: &metapb.Mapping{
								Id:   m.Id,
								File: "a",
							},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}}, fg))
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
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
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
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})

	testGenerateFlamegraphFromInstantProfile(t, l)
}

func TestFlamegraphConsistency(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
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
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
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

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})
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

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	b.Cleanup(func() {
		l.Close()
	})
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
				Function: &metapb.Function{
					Name: "1",
				},
				Line:     &metapb.Line{},
				Location: &metapb.Location{},
			},
			Cumulative: 12,
			Children: []*pb.FlamegraphNode{
				{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{
							Name: "2",
						},
						Line:     &metapb.Line{},
						Location: &metapb.Location{},
					},
					Cumulative: 6,
					Children: []*pb.FlamegraphNode{{
						Cumulative: 4,
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{
								Name: "3",
							},
							Line:     &metapb.Line{},
							Location: &metapb.Location{},
						},
						Children: []*pb.FlamegraphNode{{
							Meta: &pb.FlamegraphNodeMeta{
								Function: &metapb.Function{
									Name: "4",
								},
								Line:     &metapb.Line{},
								Location: &metapb.Location{},
							},
							Cumulative: 3,
						}, {
							Meta: &pb.FlamegraphNodeMeta{
								Function: &metapb.Function{
									Name: "5",
								},
								Line:     &metapb.Line{},
								Location: &metapb.Location{},
							},
							Cumulative: 1,
						}},
					}},
				},
				{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{
							Name: "2",
						},
						Line:     &metapb.Line{},
						Location: &metapb.Location{},
					},
					Cumulative: 6,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{
								Name: "3",
							},
							Line:     &metapb.Line{},
							Location: &metapb.Location{},
						},
						Cumulative: 4,
						Children: []*pb.FlamegraphNode{{
							Meta: &pb.FlamegraphNodeMeta{
								Function: &metapb.Function{
									Name: "4",
								},
								Line:     &metapb.Line{},
								Location: &metapb.Location{},
							},
							Cumulative: 3,
						}, {
							Meta: &pb.FlamegraphNodeMeta{
								Function: &metapb.Function{
									Name: "5",
								},
								Line:     &metapb.Line{},
								Location: &metapb.Location{},
							},
							Cumulative: 1,
						}},
					}},
				},
			},
		}, {
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Name: "1",
				},
				Line:     &metapb.Line{},
				Location: &metapb.Location{},
			},
			Cumulative: 2,
		}},
	}}

	afg := &pb.Flamegraph{Total: 12, Root: &pb.FlamegraphRootNode{
		Cumulative: 12,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{
					Name: "1",
				},
				Location: &metapb.Location{},
			},
			Cumulative: 14,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 12,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{
						Name: "2",
					},
					Location: &metapb.Location{},
				},
				Children: []*pb.FlamegraphNode{
					{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{
								Name: "3",
							},
							Location: &metapb.Location{},
						},
						Cumulative: 8,
						Children: []*pb.FlamegraphNode{
							{
								Meta: &pb.FlamegraphNodeMeta{
									Function: &metapb.Function{
										Name: "4",
									},
									Location: &metapb.Location{},
								},
								Cumulative: 6,
							}, {
								Meta: &pb.FlamegraphNodeMeta{
									Function: &metapb.Function{
										Name: "5",
									},
									Location: &metapb.Location{},
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
