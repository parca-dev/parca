package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestTreeStack(t *testing.T) {
	s := TreeStack{}
	s.Push(&TreeNode{Name: "a"})
	s.Push(&TreeNode{Name: "b"})

	require.Equal(t, 2, s.Size())

	e, hasMore := s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "b", e.Name)

	require.Equal(t, 1, s.Size())

	e, hasMore = s.Pop()
	require.True(t, hasMore)
	require.Equal(t, "a", e.Name)

	require.Equal(t, 0, s.Size())

	e, hasMore = s.Pop()
	require.False(t, hasMore)
}

func TestLinesToTreeNodes(t *testing.T) {
	outerMost, innerMost := linesToTreeNodes([]string{}, []profile.Line{
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

	require.Equal(t, &TreeNode{
		Name:     "log :0",
		FullName: "log :0",
		Cum:      2,
		Children: []*TreeNode{{
			Name:     "printf :0",
			FullName: "printf :0",
			Cum:      2,
			Children: []*TreeNode{{
				Name:     "memcpy :0",
				FullName: "memcpy :0",
				Cum:      2,
			}},
		}},
	}, outerMost)
	require.Equal(t, &TreeNode{
		Name:     "memcpy :0",
		FullName: "memcpy :0",
		Cum:      2,
	}, innerMost)
}

type fakeLocations struct {
	m map[uint64]*profile.Location
}

func (l *fakeLocations) GetLocationByID(id uint64) (*profile.Location, error) {
	return l.m[id], nil
}

func TestGenerateFlamegraph(t *testing.T) {
	pt := &ProfileTree{}
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))

	l := &fakeLocations{m: map[uint64]*profile.Location{
		1: &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: "1"}}}},
		2: &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: "2"}}}},
		3: &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: "3"}}}},
		4: &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: "4"}}}},
		5: &profile.Location{Line: []profile.Line{{Function: &profile.Function{Name: "5"}}}},
	}}

	fg, err := generateFlamegraph(l, pt.Iterator())
	require.NoError(t, err)
	require.Equal(t, &TreeNode{
		Name: "root",
		Cum:  6,
		Children: []*TreeNode{{
			Name:     "1 :0",
			FullName: "1 :0",
			Cum:      6,
			Children: []*TreeNode{{
				Name:     "2 :0",
				FullName: "2 :0",
				Cum:      6,
				Children: []*TreeNode{{
					Name:     "3 :0",
					FullName: "3 :0",
					Cum:      4,
					Children: []*TreeNode{{
						Name:     "4 :0",
						FullName: "4 :0",
						Cum:      3,
					}, {
						Name:     "5 :0",
						FullName: "5 :0",
						Cum:      1,
					}},
				}},
			}},
		}},
	},
		fg)
}

func testGenerateFlamegraphFromProfileTree(t *testing.T) *TreeNode {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(l)
	require.NoError(t, s.Append(p1))

	profileTree, err := s.prepareSamplesForInsert(p1)
	require.NoError(t, err)

	fg, err := generateFlamegraph(l, profileTree.Iterator())
	require.NoError(t, err)

	return fg
}

func TestGenerateFlamegraphFromProfileTree(t *testing.T) {
	testGenerateFlamegraphFromProfileTree(t)
}

func testGenerateFlamegraphFromInstantProfile(t *testing.T) *TreeNode {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(l)
	require.NoError(t, s.Append(p1))

	it := s.Iterator()
	require.True(t, it.Next())
	require.NoError(t, it.Err())
	instantProfile := it.At()

	fg, err := generateFlamegraph(l, instantProfile.ProfileTree().Iterator())
	require.NoError(t, err)
	return fg
}

func TestGenerateFlamegraphFromInstantProfile(t *testing.T) {
	testGenerateFlamegraphFromInstantProfile(t)
}

func TestFlamegraphConsistency(t *testing.T) {
	require.Equal(t, testGenerateFlamegraphFromProfileTree(t), testGenerateFlamegraphFromInstantProfile(t))
}

func BenchmarkGenerateFlamegraph(b *testing.B) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(l)
	require.NoError(b, s.Append(p1))

	profileTree, err := s.prepareSamplesForInsert(p1)
	require.NoError(b, err)

	b.ResetTimer()
	_, err = generateFlamegraph(l, profileTree.Iterator())
	require.NoError(b, err)
}
