package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunk"
	"github.com/parca-dev/storage/chunkenc"
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

func TestProfileTreeInsert(t *testing.T) {
	pt := &ProfileTree{}
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))

	require.Equal(t, &ProfileTree{
		Roots: &ProfileTreeNode{
			cumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
			// Roots always have the LocationID 0.
			locationID: 0,
			Children: []*ProfileTreeNode{{
				locationID:       1,
				cumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
				Children: []*ProfileTreeNode{{
					locationID:       2,
					cumulativeValues: []*ProfileTreeValueNode{{Value: 6}},
					flatValues:       []*ProfileTreeValueNode{{Value: 2}},
					Children: []*ProfileTreeNode{{
						locationID:       3,
						cumulativeValues: []*ProfileTreeValueNode{{Value: 4}},
						Children: []*ProfileTreeNode{{
							locationID:       4,
							cumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
							flatValues:       []*ProfileTreeValueNode{{Value: 3}},
						}, {
							locationID:       5,
							cumulativeValues: []*ProfileTreeValueNode{{Value: 1}},
							flatValues:       []*ProfileTreeValueNode{{Value: 1}},
						}},
					}},
				}},
			}}},
	}, pt)
}

func TestMemSeriesTree(t *testing.T) {
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(2, []uint64{4, 1}))

	st := &MemSeriesTree{}
	st.Insert(0, pt1)

	require.Equal(t, &MemSeriesTree{
		Roots: &MemSeriesTreeNode{
			CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4)}},
			Children: []*MemSeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4)}},
				Children: []*MemSeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}, {
					LocationID:       4,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}},
			}}},
	}, st)

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(2, []uint64{3, 1}))

	st.Insert(1, pt2)

	require.Equal(t, &MemSeriesTree{
		Roots: &MemSeriesTreeNode{
			CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 2)}},
			Children: []*MemSeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 2)}},
				Children: []*MemSeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}, {
					LocationID:       3,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(0, 2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(0, 2)}},
					// 0,2,0,0,0,0,0,0,0,0,255,0,0,0,0,8
				}, {
					LocationID:       4,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}},
			}}},
	}, st)
}

func TestMemSeriesIterator(t *testing.T) {
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(2, []uint64{4, 1}))

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(2, []uint64{3, 1}))
	pt2.Insert(makeSample(2, []uint64{4, 1}))

	st := &MemSeries{
		timestamps: &chunk.FakeChunk{Values: []int64{1, 2}},
		durations:  &chunk.FakeChunk{Values: []int64{1e9, 1e9}},
		periods:    &chunk.FakeChunk{Values: []int64{100, 100}},
	}
	st.append(pt1)
	st.append(pt2)

	it := st.Iterator()
	require.True(t, it.Next())
	require.NoError(t, it.Err())

	instantProfile := it.At()
	require.Equal(t, InstantProfileMeta{
		Timestamp: 1,
		Duration:  1e9,
		Period:    100,
	}, instantProfile.ProfileMeta())

	res := []uint64{}
	WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) {
		res = append(res, n.LocationID())
	})

	require.Equal(t, []uint64{0, 1, 2, 3, 4}, res)

	require.True(t, it.Next())
	require.NoError(t, it.Err())

	instantProfile = it.At()
	require.Equal(t, InstantProfileMeta{
		Timestamp: 2,
		Duration:  1e9,
		Period:    100,
	}, instantProfile.ProfileMeta())

	res = []uint64{}
	WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) {
		res = append(res, n.LocationID())
	})

	require.Equal(t, []uint64{0, 1, 2, 3, 4}, res)
	require.False(t, it.Next())
}

func TestIteratorConsistency(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	s := &MemSeries{}
	require.NoError(t, s.Append(p1))

	profileTree, err := s.prepareSamplesForInsert(p1)
	require.NoError(t, err)

	res1 := []uint64{}
	WalkProfileTree(profileTree, func(n InstantProfileTreeNode) {
		res1 = append(res1, n.LocationID())
	})

	sit := s.Iterator()
	require.True(t, sit.Next())
	require.NoError(t, sit.Err())

	res2 := []uint64{}
	WalkProfileTree(sit.At().ProfileTree(), func(n InstantProfileTreeNode) {
		res2 = append(res2, n.LocationID())
	})

	require.Equal(t, res1, res2)
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

	s := &MemSeries{}
	require.NoError(t, s.Append(p1))
	require.NoError(t, s.Append(p2))
}
