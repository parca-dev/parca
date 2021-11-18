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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
)

func TestMergeFlatProfileSimple(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	s1 := makeSample(2, []uuid.UUID{uuid2, uuid1})
	k1 := makeStacktraceKey(s1)

	p1 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k1): s1,
		},
	}

	s2 := makeSample(1, []uuid.UUID{uuid3, uuid1})
	k2 := makeStacktraceKey(s2)

	p2 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k2): s2,
		},
	}

	//mp, err := MergeProfiles(p1, p2)
	mp, err := NewMergeProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	merged := mp.Samples()
	require.Len(t, merged, 2)

	require.Equal(t, &Sample{
		Value:    2,
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid1}},
	}, merged[string(k1)])
	require.Equal(t, &Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid1}},
	}, merged[string(k2)])
}

func TestMergeFlatProfileDeep(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid6 := uuid.MustParse("00000000-0000-0000-0000-000000000006")

	s1 := makeSample(3, []uuid.UUID{uuid1, uuid3})
	s2 := makeSample(3, []uuid.UUID{uuid2, uuid3})
	s3 := makeSample(3, []uuid.UUID{uuid3, uuid3, uuid2})
	s4 := makeSample(3, []uuid.UUID{uuid6, uuid2})

	k1 := makeStacktraceKey(s1)
	k2 := makeStacktraceKey(s2)
	k3 := makeStacktraceKey(s3)
	k4 := makeStacktraceKey(s4)

	p1 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k1): s1,
			string(k2): s2,
			string(k3): s3,
			string(k4): s4,
		},
	}

	s5 := makeSample(3, []uuid.UUID{uuid3, uuid2, uuid2})
	k5 := makeStacktraceKey(s5)

	p2 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k5): s5,
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

	merged := mp.Samples()

	require.Len(t, merged, 5)

	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid1}, {ID: uuid3}},
	}, merged[string(k1)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid3}},
	}, merged[string(k2)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid3}, {ID: uuid2}},
	}, merged[string(k3)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid2}},
	}, merged[string(k4)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid2}, {ID: uuid2}},
	}, merged[string(k5)])
}

func TestMergeFlatProfile(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid4 := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	uuid5 := uuid.MustParse("00000000-0000-0000-0000-000000000005")
	uuid6 := uuid.MustParse("00000000-0000-0000-0000-000000000006")

	s1 := makeSample(2, []uuid.UUID{uuid2, uuid1})
	s2 := makeSample(1, []uuid.UUID{uuid6, uuid3, uuid2, uuid1})
	s3 := makeSample(3, []uuid.UUID{uuid4, uuid3, uuid2, uuid1})
	s4 := makeSample(3, []uuid.UUID{uuid3, uuid3, uuid2})
	s5 := makeSample(3, []uuid.UUID{uuid6, uuid2})

	k1 := makeStacktraceKey(s1)
	k2 := makeStacktraceKey(s2)
	k3 := makeStacktraceKey(s3)
	k4 := makeStacktraceKey(s4)
	k5 := makeStacktraceKey(s5)

	p1 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k1): s1,
			string(k2): s2,
			string(k3): s3,
			string(k4): s4,
			string(k5): s5,
		},
	}

	s6 := makeSample(1, []uuid.UUID{uuid5, uuid3, uuid2, uuid1})
	s7 := makeSample(3, []uuid.UUID{uuid3, uuid2, uuid2})

	k6 := makeStacktraceKey(s6)
	k7 := makeStacktraceKey(s7)

	p2 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		samples: map[string]*Sample{
			string(k1): s1,
			string(k3): s3,
			string(k6): s6,
			string(k7): s7,
		},
	}

	//mp, err := MergeProfiles(p1, p2)
	mp, err := NewMergeProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	merged := mp.Samples()
	require.Len(t, merged, 7)

	require.Equal(t, &Sample{
		Value:    4, // 2 + 2
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid1}},
	}, merged[string(k1)])
	require.Equal(t, &Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k2)])
	require.Equal(t, &Sample{
		Value:    6, // 3 + 3
		Location: []*metastore.Location{{ID: uuid4}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k3)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid3}, {ID: uuid2}},
	}, merged[string(k4)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid2}},
	}, merged[string(k5)])
	require.Equal(t, &Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid5}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k6)])
	require.Equal(t, &Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid2}, {ID: uuid2}},
	}, merged[string(k7)])
}
