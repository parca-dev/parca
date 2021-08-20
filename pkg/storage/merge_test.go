package storage

import (
	"os"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestMergeProfileSimple(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(2, []uint64{2, 1}))

	p1 := &Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(1, []uint64{3, 1}))

	p2 := &Profile{
		Tree: pt2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	mp, err := NewMergeProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(3),
		},
		{
			id: uint64(1),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(3),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(2),
			}},
			cumulative: int64(2),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
			cumulative: int64(1),
		},
	}, res)
}

func TestMergeProfileDeep(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(3, []uint64{3, 3, 2}))
	pt1.Insert(makeSample(3, []uint64{6, 2}))
	pt1.Insert(makeSample(3, []uint64{2, 3}))
	pt1.Insert(makeSample(3, []uint64{1, 3}))

	p1 := &Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(3, []uint64{3, 2, 2}))

	p2 := &Profile{
		Tree: pt2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	mp, err := NewMergeProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(15),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(9),
		},
		{
			id:         uint64(2),
			cumulative: int64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id:         uint64(3),
			cumulative: int64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id:         uint64(3),
			cumulative: int64(6),
		},
		{
			id: uint64(1),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
	}, res)
}

func TestMergeProfile(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(1, []uint64{6, 3, 2, 1}))
	pt1.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt1.Insert(makeSample(3, []uint64{3, 3, 2}))
	pt1.Insert(makeSample(3, []uint64{6, 2}))

	p1 := &Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(2, []uint64{2, 1}))
	pt2.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt2.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt2.Insert(makeSample(3, []uint64{3, 2, 2}))

	p2 := &Profile{
		Tree: pt2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	mp, err := NewMergeProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(21),
		},
		{
			id: uint64(1),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(12),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(4),
			}},
			cumulative: int64(12),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(8),
		},
		{
			id: uint64(4),
			flat: []*ProfileTreeValueNode{{
				Value: int64(6),
			}},
			cumulative: int64(6),
		},
		{
			id: uint64(5),
			flat: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
			cumulative: int64(1),
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
			cumulative: int64(1),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(0),
			}},
			cumulative: int64(9),
		},
		{
			id:         uint64(2),
			cumulative: int64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id:         uint64(3),
			cumulative: int64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: int64(3),
		},
	}, res)
}

type sample struct {
	id         uint64
	flat       []*ProfileTreeValueNode
	cumulative int64
}

func BenchmarkTreeMerge(b *testing.B) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(b, err)
	p2, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree1 := ProfileTreeFromPprof(l, p1, 0)
	profileTree2 := ProfileTreeFromPprof(l, p2, 0)

	prof1 := &Profile{
		Tree: profileTree1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	prof2 := &Profile{
		Tree: profileTree2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	b.ResetTimer()
	m, err := NewMergeProfile(prof1, prof2)
	require.NoError(b, err)
	CopyInstantProfileTree(m.ProfileTree())
}

func BenchmarkMerge(b *testing.B) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(b, err)
	p2, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	b.ResetTimer()
	_, err = profile.Merge([]*profile.Profile{p1, p2})
	require.NoError(b, err)
}
