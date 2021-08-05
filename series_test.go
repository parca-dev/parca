package storage

import (
	"os"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
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
	pt := NewProfileTree()
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
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(2, []uint64{4, 1}))

	st := &MemSeriesTree{}
	err := st.Insert(0, pt1)
	require.NoError(t, err)

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

	// Merging another profileTree onto the existing one
	pt2 := NewProfileTree()
	pt2.Insert(makeSample(3, []uint64{2, 1}))
	err = st.Insert(1, pt2)
	require.NoError(t, err)

	require.Equal(t, &MemSeriesTree{
		Roots: &MemSeriesTreeNode{
			CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 3)}},
			Children: []*MemSeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 3)}},
				Children: []*MemSeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2, 3)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2, 3)}},
				}, {
					LocationID:       4,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}},
			}}},
	}, st)

	// Merging another profileTree onto the existing one with one new Location
	pt3 := NewProfileTree()
	pt3.Insert(makeSample(2, []uint64{3, 1}))
	err = st.Insert(2, pt3)
	require.NoError(t, err)

	// These require.Equal assertions are exactly the same as below, although you know exactly what line breaks.
	require.Equal(t, chunkenc.FromValuesXOR(4, 3, 2), st.Roots.CumulativeValues[0].Values)
	require.Equal(t, chunkenc.FromValuesXOR(4, 3, 2), st.Roots.Children[0].CumulativeValues[0].Values)            // Location: 1
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), st.Roots.Children[0].Children[0].CumulativeValues[0].Values)   // Location: 2
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), st.Roots.Children[0].Children[0].FlatValues[0].Values)         // Location: 2
	require.Equal(t, chunkenc.FromValuesXORAt(2, 2), st.Roots.Children[0].Children[1].CumulativeValues[0].Values) // Location: 3
	require.Equal(t, chunkenc.FromValuesXORAt(2, 2), st.Roots.Children[0].Children[1].FlatValues[0].Values)       // Location: 3
	require.Equal(t, chunkenc.FromValuesXOR(2), st.Roots.Children[0].Children[2].CumulativeValues[0].Values)      // Location: 4
	require.Equal(t, chunkenc.FromValuesXOR(2), st.Roots.Children[0].Children[2].FlatValues[0].Values)            // Location: 4

	require.Equal(t, &MemSeriesTree{
		Roots: &MemSeriesTreeNode{
			CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 3, 2)}},
			Children: []*MemSeriesTreeNode{{
				LocationID:       1,
				CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(4, 3, 2)}},
				Children: []*MemSeriesTreeNode{{
					LocationID:       2,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2, 3)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2, 3)}},
				}, {
					LocationID:       3,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXORAt(2, 2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXORAt(2, 2)}},
				}, {
					LocationID:       4,
					CumulativeValues: []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
					FlatValues:       []*MemSeriesTreeValueNode{{Values: chunkenc.FromValuesXOR(2)}},
				}},
			}}},
	}, st)
}

func TestMemSeriesIterator(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(2, []uint64{4, 1}))

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(2, []uint64{3, 1}))
	pt2.Insert(makeSample(2, []uint64{4, 1}))

	st := &MemSeries{
		timestamps: chunkenc.FromValuesDelta(1, 2),
		durations:  chunkenc.FromValuesDelta(time.Second.Nanoseconds(), time.Second.Nanoseconds()),
		periods:    chunkenc.FromValuesDelta(100, 100),
	}
	st.append(pt1)
	st.append(pt2)

	it := st.Iterator()
	require.True(t, it.Next())
	require.NoError(t, it.Err())

	instantProfile := it.At()
	require.Equal(t, InstantProfileMeta{
		Timestamp: 1,
		Duration:  time.Second.Nanoseconds(),
		Period:    100,
	}, instantProfile.ProfileMeta())

	res := []uint64{}
	err := WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, n.LocationID())
		return nil
	})
	require.NoError(t, err)

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
	err = WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, n.LocationID())
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []uint64{0, 1, 2, 3, 4}, res)
	require.False(t, it.Next())
}

func TestIteratorConsistency(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s, err := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	profile := ProfileFromPprof(l, p1)
	require.NoError(t, s.Append(profile))

	profileTree := profile.Tree

	res1 := []uint64{}
	err = WalkProfileTree(profileTree, func(n InstantProfileTreeNode) error {
		res1 = append(res1, n.LocationID())
		return nil
	})
	require.NoError(t, err)

	sit := s.Iterator()
	require.True(t, sit.Next())
	require.NoError(t, sit.Err())

	res2 := []uint64{}
	err = WalkProfileTree(sit.At().ProfileTree(), func(n InstantProfileTreeNode) error {
		res2 = append(res2, n.LocationID())
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, res1, res2)
}

func TestRealInsert(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s, err := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	profile := ProfileFromPprof(l, p)
	require.NoError(t, s.Append(profile))
	require.Equal(t, len(p.Location), len(l.locations))
}

func TestRealInserts(t *testing.T) {
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

	l := NewInMemoryProfileMetaStore()
	s, err := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	require.NoError(t, s.Append(ProfileFromPprof(l, p1)))
	require.NoError(t, s.Append(ProfileFromPprof(l, p2)))
}
