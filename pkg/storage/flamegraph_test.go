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
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

func TestTreeStack(t *testing.T) {
	s := TreeStack{}
	s.Push(&TreeStackEntry{node: &pb.FlamegraphNode{Meta: &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "a"}}}})
	s.Push(&TreeStackEntry{node: &pb.FlamegraphNode{Meta: &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "b"}}}})

	require.Equal(t, 2, s.Size())

	e, hasMore := s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "b", e.node.Meta.Function.Name)

	require.Equal(t, 1, s.Size())

	e, hasMore = s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "a", e.node.Meta.Function.Name)

	require.Equal(t, 0, s.Size())

	_, hasMore = s.Pop()
	require.False(t, hasMore)
}

func TestLinesToTreeNodes(t *testing.T) {
	outerMost, innerMost := linesToTreeNodes(&profile.Location{}, uint64(1), &pb.Mapping{}, []profile.Line{
		{
			Function: &profile.Function{
				Name: "memcpy",
			},
		}, {
			Function: &profile.Function{
				Name: "printf",
			},
		}, {
			Function: &profile.Function{
				Name: "log",
			},
		},
	}, 2, 0)

	require.Equal(t, &pb.FlamegraphNode{
		Cumulative: 2,
		Meta: &pb.FlamegraphNodeMeta{
			Function: &pb.Function{
				Name: "log",
			},
			Line:     &pb.Line{},
			Location: &pb.Location{MappingId: 1},
			Mapping:  &pb.Mapping{},
		},
		Children: []*pb.FlamegraphNode{{
			Cumulative: 2,
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{
					Name: "printf",
				},
				Line:     &pb.Line{},
				Location: &pb.Location{MappingId: 1},
				Mapping:  &pb.Mapping{},
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 2,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{
						Name: "memcpy",
					},
					Line:     &pb.Line{},
					Location: &pb.Location{MappingId: 1},
					Mapping:  &pb.Mapping{},
				},
			}},
		}},
	}, outerMost)
	require.Equal(t, &pb.FlamegraphNode{
		Cumulative: 2,
		Meta: &pb.FlamegraphNodeMeta{
			Function: &pb.Function{
				Name: "memcpy",
			},
			Line:     &pb.Line{},
			Location: &pb.Location{MappingId: 1},
			Mapping:  &pb.Mapping{},
		},
	}, innerMost)
}

type fakeLocations struct {
	m map[uint64]*profile.Location
}

func (l *fakeLocations) GetLocationsByIDs(ctx context.Context, ids ...uint64) (map[uint64]*profile.Location, error) {
	return l.m, nil
}

func TestGenerateFlamegraph(t *testing.T) {
	ctx := context.Background()

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))

	l := &fakeLocations{m: map[uint64]*profile.Location{
		1: {Line: []profile.Line{{Function: &profile.Function{Name: "1"}}}},
		2: {Line: []profile.Line{{Function: &profile.Function{Name: "2"}}}},
		3: {Line: []profile.Line{{Function: &profile.Function{Name: "3"}}}},
		4: {Line: []profile.Line{{Function: &profile.Function{Name: "4"}}}},
		5: {Line: []profile.Line{{Function: &profile.Function{Name: "5"}}}},
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
					Name: "1",
				},
				Line:     &pb.Line{},
				Location: &pb.Location{},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
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
			}},
		}},
	}},
		fg)
}

func testGenerateFlamegraphFromProfileTree(t *testing.T) *pb.Flamegraph {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"generateflamegraphfromprofiletree",
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

	return fg
}

func TestGenerateFlamegraphFromProfileTree(t *testing.T) {
	testGenerateFlamegraphFromProfileTree(t)
}

func testGenerateFlamegraphFromInstantProfile(t *testing.T) *pb.Flamegraph {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	t.Cleanup(func() {
		l.Close()
	})
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
	testGenerateFlamegraphFromInstantProfile(t)
}

func TestFlamegraphConsistency(t *testing.T) {
	require.Equal(t, testGenerateFlamegraphFromProfileTree(t), testGenerateFlamegraphFromInstantProfile(t))
}

func TestGenerateFlamegraphFromMergeProfile(t *testing.T) {
	testGenerateFlamegraphFromMergeProfile(t)
}

func testGenerateFlamegraphFromMergeProfile(t *testing.T) *pb.Flamegraph {
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

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"generateflamegraphfrommergeprofile",
	)
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

	mfg := testGenerateFlamegraphFromMergeProfile(t)
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
	fg := &pb.Flamegraph{
		Total: 12,
		Root: &pb.FlamegraphRootNode{
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
		},
	}

	afg := &pb.Flamegraph{
		Total: 12,
		Root: &pb.FlamegraphRootNode{
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
		},
	}

	require.Equal(t, afg, aggregateByFunction(fg))
}

func TestAggregateByFunction2(t *testing.T) {
	in := &pb.Flamegraph{
		Total:  30_000_000,
		Height: 6,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 30_000_000,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 30_000_000,
				Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcBgMarkWorker"}},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 30_000_000,
					Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.systemstack"}},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 30_000_000,
						Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcBgMarkWorker.func2"}},
						Children: []*pb.FlamegraphNode{{
							Cumulative: 20_000_000,
							Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcDrain"}},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 10_000_000,
								Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.heapBits.bits"}},
							}, {
								Cumulative: 10_000_000,
								Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.scanobject"}},
								Children: []*pb.FlamegraphNode{{
									Cumulative: 10_000_000,
									Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.markBits.isMarked"}},
								}},
							}},
						}, {
							Cumulative: 10_000_000,
							Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcDrain"}},
						}},
					}},
				}},
			}},
		},
	}

	result := aggregateByFunction(in)

	expected := &pb.Flamegraph{
		Total:  30_000_000,
		Height: 6,
		Root: &pb.FlamegraphRootNode{
			Cumulative: 30_000_000,
			Children: []*pb.FlamegraphNode{{
				Cumulative: 30_000_000,
				Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcBgMarkWorker"}},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 30_000_000,
					Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.systemstack"}},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 30_000_000,
						Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcBgMarkWorker.func2"}},
						Children: []*pb.FlamegraphNode{{
							Cumulative: 30_000_000,
							Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.gcDrain"}},
							Children: []*pb.FlamegraphNode{{
								Cumulative: 10_000_000,
								Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.heapBits.bits"}},
							}, {
								Cumulative: 10_000_000,
								Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.scanobject"}},
								Children: []*pb.FlamegraphNode{{
									Cumulative: 10_000_000,
									Meta:       &pb.FlamegraphNodeMeta{Function: &pb.Function{Name: "runtime.markBits.isMarked"}},
								}},
							}},
						}},
					}},
				}},
			}},
		},
	}

	require.Equal(t, expected, result)
}
