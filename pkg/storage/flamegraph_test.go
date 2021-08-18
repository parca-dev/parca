package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestTreeStack(t *testing.T) {
	s := TreeStack{}
	s.Push(&TreeStackEntry{node: &pb.FlamegraphNode{Name: "a"}})
	s.Push(&TreeStackEntry{node: &pb.FlamegraphNode{Name: "b"}})

	require.Equal(t, 2, s.Size())

	e, hasMore := s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "b", e.node.Name)

	require.Equal(t, 1, s.Size())

	e, hasMore = s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "a", e.node.Name)

	require.Equal(t, 0, s.Size())

	e, hasMore = s.Pop()
	require.False(t, hasMore)
}

func TestLinesToTreeNodes(t *testing.T) {
	outerMost, innerMost := linesToTreeNodes([]profile.Line{
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
	}, 2)

	require.Equal(t, &pb.FlamegraphNode{
		Name:       "log :0",
		FullName:   "log :0",
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Name:       "printf :0",
			FullName:   "printf :0",
			Cumulative: 2,
			Children: []*pb.FlamegraphNode{{
				Name:       "memcpy :0",
				FullName:   "memcpy :0",
				Cumulative: 2,
			}},
		}},
	}, outerMost)
	require.Equal(t, &pb.FlamegraphNode{
		Name:       "memcpy :0",
		FullName:   "memcpy :0",
		Cumulative: 2,
	}, innerMost)
}

type fakeLocations struct {
	m map[uint64]*profile.Location
}

func (l *fakeLocations) GetLocationByID(id uint64) (*profile.Location, error) {
	return l.m[id], nil
}

func TestGenerateFlamegraph(t *testing.T) {
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

	fg, err := GenerateFlamegraph(l, &Profile{Tree: pt})
	require.NoError(t, err)
	require.Equal(t, &pb.Flamegraph{Total: 6, Root: &pb.FlamegraphNode{
		Name:       "root",
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Name:       "1 :0",
			FullName:   "1 :0",
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Name:       "2 :0",
				FullName:   "2 :0",
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Name:       "3 :0",
					FullName:   "3 :0",
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Name:       "4 :0",
						FullName:   "4 :0",
						Cumulative: 3,
					}, {
						Name:       "5 :0",
						FullName:   "5 :0",
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}},
		fg)
}

func testGenerateFlamegraphFromProfileTree(t *testing.T) *pb.Flamegraph {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree := ProfileTreeFromPprof(l, p1)

	fg, err := GenerateFlamegraph(l, &Profile{Tree: profileTree})
	require.NoError(t, err)

	return fg
}

func TestGenerateFlamegraphFromProfileTree(t *testing.T) {
	testGenerateFlamegraphFromProfileTree(t)
}

func testGenerateFlamegraphFromInstantProfile(t *testing.T) *pb.Flamegraph {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	require.NoError(t, app.Append(ProfileFromPprof(l, p1)))

	it := s.Iterator()
	require.True(t, it.Next())
	require.NoError(t, it.Err())
	instantProfile := it.At()

	fg, err := GenerateFlamegraph(l, instantProfile)
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

	l := NewInMemoryProfileMetaStore()
	prof1 := ProfileFromPprof(l, p1)
	prof2 := ProfileFromPprof(l, p2)

	m, err := NewMergeProfile(prof1, prof2)
	require.NoError(t, err)

	fg, err := GenerateFlamegraph(l, m)
	require.NoError(t, err)

	return fg
}

func TestControlGenerateFlamegraphFromMergeProfile(t *testing.T) {
	f, err := os.Open("testdata/merge.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree := ProfileTreeFromPprof(l, p1)

	fg, err := GenerateFlamegraph(l, &Profile{Tree: profileTree})
	require.NoError(t, err)

	mfg := testGenerateFlamegraphFromMergeProfile(t)
	require.Equal(t, fg, mfg)
}

func BenchmarkGenerateFlamegraph(b *testing.B) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree := ProfileTreeFromPprof(l, p1)

	b.ResetTimer()
	_, err = GenerateFlamegraph(l, &Profile{Tree: profileTree})
	require.NoError(b, err)
}

func TestAggregateByFunctionName(t *testing.T) {
	fg := &pb.Flamegraph{Total: 6, Root: &pb.FlamegraphNode{
		Name:       "root",
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Name:       "1 :0",
			FullName:   "1 :0",
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Name:       "2 :0",
				FullName:   "2 :0",
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Name:       "3 :0",
					FullName:   "3 :0",
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Name:       "4 :0",
						FullName:   "4 :0",
						Cumulative: 3,
					}, {
						Name:       "5 :0",
						FullName:   "5 :0",
						Cumulative: 1,
					}},
				}},
			},
				{
					Name:       "2 :0",
					FullName:   "2 :0",
					Cumulative: 6,
					Children: []*pb.FlamegraphNode{{
						Name:       "3 :0",
						FullName:   "3 :0",
						Cumulative: 4,
						Children: []*pb.FlamegraphNode{{
							Name:       "4 :0",
							FullName:   "4 :0",
							Cumulative: 3,
						}, {
							Name:       "5 :0",
							FullName:   "5 :0",
							Cumulative: 1,
						}},
					}},
				}},
		}},
	}}

	afg := &pb.Flamegraph{Total: 6, Root: &pb.FlamegraphNode{
		Name:       "root",
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Name:       "1 :0",
			FullName:   "1 :0",
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Name:       "2 :0",
				FullName:   "2 :0",
				Cumulative: 12,
				Children: []*pb.FlamegraphNode{
					{
						Name:       "3 :0",
						FullName:   "3 :0",
						Cumulative: 8,
						Children: []*pb.FlamegraphNode{
							{
								Name:       "4 :0",
								FullName:   "4 :0",
								Cumulative: 6,
							}, {
								Name:       "5 :0",
								FullName:   "5 :0",
								Cumulative: 2,
							}},
					},
				},
			},
			},
		}},
	}}

	require.Equal(t, aggregateByFunctionName(fg), afg)
}
