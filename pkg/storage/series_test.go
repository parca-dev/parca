package storage

import (
	"os"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
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
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	pt := NewProfileTree()
	{
		s := makeSample(2, []uint64{2, 1})
		pt.Insert(s)
	}
	{
		s := makeSample(1, []uint64{5, 3, 2, 1})
		pt.Insert(s)
	}
	{
		s := makeSample(3, []uint64{4, 3, 2, 1})
		s.Label = label
		s.NumLabel = numLabel
		s.NumUnit = numUnit
		pt.Insert(s)
	}

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
							locationID: 4,
							cumulativeValues: []*ProfileTreeValueNode{{
								Value:    3,
								Label:    label,
								NumLabel: numLabel,
								NumUnit:  numUnit,
							}},
							flatValues: []*ProfileTreeValueNode{{
								Value:    3,
								Label:    label,
								NumLabel: numLabel,
								NumUnit:  numUnit,
							}},
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
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s := NewMemSeries(labels.FromStrings("a", "b"), 0)

	{
		pt := NewProfileTree()
		{
			s := makeSample(2, []uint64{2, 1})
			pt.Insert(s)
		}
		{
			s := makeSample(2, []uint64{4, 1})
			s.Label = label
			s.NumLabel = numLabel
			s.NumUnit = numUnit
			pt.Insert(s)
		}

		err := s.seriesTree.Insert(0, pt)
		require.NoError(t, err)
	}

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[4])

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(4), s.cumulativeValues[0])
	require.Equal(t, chunkenc.FromValuesXOR(4), s.cumulativeValues[1])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[4])

	require.Len(t, s.labels, 1)
	require.Equal(t, map[uint64]map[string][]string{4: label}, s.labels)
	require.Equal(t, map[uint64]map[string][]int64{4: numLabel}, s.numLabels)
	require.Equal(t, map[uint64]map[string][]string{4: numUnit}, s.numUnits)

	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			Children: []*MemSeriesTreeNode{{
				LocationID: 1,
				Children: []*MemSeriesTreeNode{
					{LocationID: 2},
					{LocationID: 4},
				},
			}}},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one
	pt2 := NewProfileTree()
	pt2.Insert(makeSample(3, []uint64{2, 1}))
	err := s.seriesTree.Insert(1, pt2)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), s.flatValues[2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[4]) // sparse - nothing added

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(4, 3), s.cumulativeValues[0])
	require.Equal(t, chunkenc.FromValuesXOR(4, 3), s.cumulativeValues[1])
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), s.cumulativeValues[2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[4]) // sparse - nothing added

	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			Children: []*MemSeriesTreeNode{{
				LocationID: 1,
				Children: []*MemSeriesTreeNode{
					{LocationID: 2},
					{LocationID: 4},
				},
			}}},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one with one new Location
	pt3 := NewProfileTree()
	pt3.Insert(makeSample(2, []uint64{3, 1}))
	err = s.seriesTree.Insert(2, pt3)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 3)
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), s.flatValues[2]) // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 2), s.flatValues[3])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[4]) // sparse - nothing added

	require.Len(t, s.cumulativeValues, 5)
	require.Equal(t, chunkenc.FromValuesXOR(4, 3, 2), s.cumulativeValues[0])
	require.Equal(t, chunkenc.FromValuesXOR(4, 3, 2), s.cumulativeValues[1])
	require.Equal(t, chunkenc.FromValuesXOR(2, 3), s.cumulativeValues[2]) // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 2), s.cumulativeValues[3])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[4]) // sparse - nothing added

	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			Children: []*MemSeriesTreeNode{{
				LocationID: 1,
				Children: []*MemSeriesTreeNode{
					{LocationID: 2},
					{LocationID: 3},
					{LocationID: 4},
				},
			}}},
	}, s.seriesTree)
}

func TestMemSeriesIterator(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s := NewMemSeries(labels.FromStrings("a", "b"), 0)

	s.timestamps = chunkenc.FromValuesDelta(1, 2)
	s.durations = chunkenc.FromValuesRLE(time.Second.Nanoseconds(), 2)
	s.periods = chunkenc.FromValuesRLE(100, 2)

	{
		pt := NewProfileTree()
		{
			s := makeSample(2, []uint64{2, 1})
			pt.Insert(s)
		}
		{
			s := makeSample(2, []uint64{4, 1})
			s.Label = label
			s.NumLabel = numLabel
			s.NumUnit = numUnit
			pt.Insert(s)
		}

		err := s.appendTree(pt)
		s.numSamples++
		require.NoError(t, err)
	}
	{
		pt := NewProfileTree()
		{
			pt.Insert(makeSample(2, []uint64{3, 1}))
		}
		{
			pt.Insert(makeSample(2, []uint64{4, 1}))
		}

		err := s.appendTree(pt)
		s.numSamples++
		require.NoError(t, err)
	}

	it := s.Iterator()

	// First iteration
	{
		require.True(t, it.Next())
		require.NoError(t, it.Err())

		instantProfile := it.At()
		require.Equal(t, InstantProfileMeta{
			Timestamp: 1,
			Duration:  time.Second.Nanoseconds(),
			Period:    100,
		}, instantProfile.ProfileMeta())

		expected := []struct {
			LocationID       uint64
			CumulativeValues []*ProfileTreeValueNode
			FlatValues       []*ProfileTreeValueNode
		}{
			{
				LocationID:       0,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 4}},
				FlatValues:       []*ProfileTreeValueNode{},
			},
			{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 4}},
				FlatValues:       []*ProfileTreeValueNode{},
			},
			{
				LocationID:       2,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 2}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 2}},
			},
			{
				LocationID:       3,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0}},
			},
			{
				LocationID:       4,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}},
			},
		}

		i := 0
		err := WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
			require.Equal(t, expected[i].LocationID, n.LocationID())
			require.Equal(t, expected[i].CumulativeValues, n.CumulativeValues())
			require.Equal(t, expected[i].FlatValues, n.FlatValues())
			i++
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 5, i) // seen 5 nodes
	}

	// Second iteration
	{
		require.True(t, it.Next())
		require.NoError(t, it.Err())

		instantProfile := it.At()
		require.Equal(t, InstantProfileMeta{
			Timestamp: 2,
			Duration:  time.Second.Nanoseconds(),
			Period:    100,
		}, instantProfile.ProfileMeta())

		expected := []struct {
			LocationID       uint64
			CumulativeValues []*ProfileTreeValueNode
		}{
			{LocationID: 0, CumulativeValues: []*ProfileTreeValueNode{{Value: 4}}},
			{LocationID: 1, CumulativeValues: []*ProfileTreeValueNode{{Value: 4}}},
			{LocationID: 2, CumulativeValues: []*ProfileTreeValueNode{{Value: 2}}}, // TODO: Fix this iterator! It should be a sparse 0
			{LocationID: 3, CumulativeValues: []*ProfileTreeValueNode{{Value: 2}}},
			{LocationID: 4, CumulativeValues: []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}}},
		}

		i := 0
		err := WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
			require.Equal(t, expected[i].LocationID, n.LocationID())
			require.Equal(t, expected[i].CumulativeValues, n.CumulativeValues())
			i++
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 5, i)
	}

	// No more iterations
	require.False(t, it.Next())
}

func TestIteratorConsistency(t *testing.T) {
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
	profile := ProfileFromPprof(l, p1, 0)
	require.NoError(t, app.Append(profile))

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
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	profile := ProfileFromPprof(l, p, 0)
	require.NoError(t, app.Append(profile))
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
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	prof1 := ProfileFromPprof(l, p1, 0)
	prof2 := ProfileFromPprof(l, p2, 0)
	require.NoError(t, app.Append(prof1))
	require.NoError(t, app.Append(prof2))

	it := s.Iterator()
	require.True(t, it.Next())
	require.Equal(t, int64(1626013307085), it.At().ProfileMeta().Timestamp)
	require.True(t, it.Next())
	require.Equal(t, int64(1626014267084), it.At().ProfileMeta().Timestamp)
	require.False(t, it.Next())
}
