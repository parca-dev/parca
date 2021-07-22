package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunk"
	"github.com/stretchr/testify/require"
)

//func TestChunk(t *testing.T) {
//	os.Remove("result-profile1.pb.gz")
//	os.Remove("result-profile2.pb.gz")
//
//	f, err := os.Open("testdata/profile1.pb.gz")
//	require.NoError(t, err)
//	p1, err := profile.Parse(f)
//	require.NoError(t, err)
//	require.NoError(t, f.Close())
//	f, err = os.Open("testdata/profile2.pb.gz")
//	require.NoError(t, err)
//	p2, err := profile.Parse(f)
//	require.NoError(t, err)
//	require.NoError(t, f.Close())
//
//	c := &Series{chunk: &chunk.Chunk{}}
//	require.NoError(t, c.Append(p1))
//	require.NoError(t, c.Append(p2))
//
//	it := c.Iterator()
//
//	require.Equal(t, 2, len(it.data.Timestamps))
//	require.Equal(t, 2, len(it.data.Durations))
//	require.Equal(t, 2, len(it.data.Periods))
//
//	f, err = os.Create("result-profile1.pb.gz")
//	defer os.Remove("result-profile1.pb.gz")
//	require.NoError(t, err)
//	require.True(t, it.Next())
//	resp1 := it.At()
//	require.Equal(t, len(p1.Sample), len(resp1.Sample))
//	require.NoError(t, resp1.Write(f))
//	require.NoError(t, f.Close())
//
//	f, err = os.Create("result-profile2.pb.gz")
//	defer os.Remove("result-profile2.pb.gz")
//	require.NoError(t, err)
//	require.True(t, it.Next())
//	resp2 := it.At()
//	require.Equal(t, len(p1.Sample), len(resp1.Sample))
//	require.NoError(t, resp2.Write(f))
//	require.NoError(t, f.Close())
//
//	require.False(t, it.Next())
//}

func TestProfileTree(t *testing.T) {
	pt := &ProfileTree{}
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))

	require.Equal(t, &ProfileTree{
		Roots: &ProfileTreeNode{
			CumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
			Children: []*ProfileTreeNode{{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
				Children: []*ProfileTreeNode{{
					LocationID:       2,
					CumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
					FlatValues:       []*ProfileTreeValueNode{{Value: 2}},
					Children: []*ProfileTreeNode{{
						LocationID:       3,
						CumulativeValues: []*ProfileTreeValueNode{{Value: 4}},
						Children: []*ProfileTreeNode{{
							LocationID:       4,
							CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
							FlatValues:       []*ProfileTreeValueNode{{Value: 3}},
						}, {
							LocationID:       5,
							CumulativeValues: []*ProfileTreeValueNode{{Value: 1}},
							FlatValues:       []*ProfileTreeValueNode{{Value: 1}},
						}},
					}},
				}},
			}}},
	}, pt)
}

func TestSeriesTree(t *testing.T) {
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(2, []uint64{4, 1}))

	st := &SeriesTree{}
	st.Insert(0, pt1)

	require.Equal(t, &SeriesTree{
		Roots: &SeriesTreeNode{
			CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(4)}},
			Children: []*SeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(4)}},
				Children: []*SeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
					FlatValues:       []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
				}, {
					LocationID:       4,
					CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
					FlatValues:       []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
				}},
			}}},
	}, st)

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(2, []uint64{3, 1}))

	st.Insert(1, pt2)

	require.Equal(t, &SeriesTree{
		Roots: &SeriesTreeNode{
			CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(4, 2)}},
			Children: []*SeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(4, 2)}},
				Children: []*SeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
					FlatValues:       []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
				}, {
					LocationID:       3,
					CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(0, 2)}},
					FlatValues:       []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(0, 2)}},
				}, {
					LocationID:       4,
					CumulativeValues: []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
					FlatValues:       []*SeriesTreeValueNode{{Values: chunk.MustFakeChunk(2)}},
				}},
			}}},
	}, st)
}

func TestRealInsert(t *testing.T) {
	os.Remove("result-profile1.pb.gz")
	os.Remove("result-profile2.pb.gz")

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

	s := &Series{}
	require.NoError(t, s.Append(p1))
	require.NoError(t, s.Append(p2))
}
