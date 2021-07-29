package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMergeProfileSimple(t *testing.T) {
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(2, []uint64{2, 1}))

	p1 := &Profile{
		tree: pt1,
		meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(1, []uint64{3, 1}))

	p2 := &Profile{
		tree: pt2,
		meta: InstantProfileMeta{
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
	WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
	})

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
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(3, []uint64{3, 3, 2}))
	pt1.Insert(makeSample(3, []uint64{6, 2}))
	pt1.Insert(makeSample(3, []uint64{2, 3}))
	pt1.Insert(makeSample(3, []uint64{1, 3}))

	p1 := &Profile{
		tree: pt1,
		meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(3, []uint64{3, 2, 2}))

	p2 := &Profile{
		tree: pt2,
		meta: InstantProfileMeta{
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
	WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
	})

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
	pt1 := &ProfileTree{}
	pt1.Insert(makeSample(2, []uint64{2, 1}))
	pt1.Insert(makeSample(1, []uint64{6, 3, 2, 1}))
	pt1.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt1.Insert(makeSample(3, []uint64{3, 3, 2}))
	pt1.Insert(makeSample(3, []uint64{6, 2}))

	p1 := &Profile{
		tree: pt1,
		meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := &ProfileTree{}
	pt2.Insert(makeSample(2, []uint64{2, 1}))
	pt2.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt2.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt2.Insert(makeSample(3, []uint64{3, 2, 2}))

	p2 := &Profile{
		tree: pt2,
		meta: InstantProfileMeta{
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
	WalkProfileTree(mp.ProfileTree(), func(n InstantProfileTreeNode) {
		res = append(res, sample{
			id:         n.LocationID(),
			flat:       n.FlatValues(),
			cumulative: n.CumulativeValue(),
		})
	})

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
