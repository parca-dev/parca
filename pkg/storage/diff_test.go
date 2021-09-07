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
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestDiffProfileSimple(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(3, []uint64{2, 1}))

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

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
	}, dp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(dp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:             n.LocationID(),
			flat:           n.FlatValues(),
			flatDiff:       n.FlatDiffValues(),
			cumulative:     n.CumulativeValues(),
			cumulativeDiff: n.CumulativeDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
			cumulative: []*ProfileTreeValueNode{{
				key: &ProfileTreeValueNodeKey{
					location: "0",
				},
				Value: int64(1),
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(-2),
			}},
		},
		{
			id: uint64(1),
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(1),
				key: &ProfileTreeValueNodeKey{
					location: "1|0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(-2),
			}},
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(1),
				key: &ProfileTreeValueNodeKey{
					location: "3|1|0",
				},
			}},
			flatDiff: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(1),
				key: &ProfileTreeValueNodeKey{
					location: "3|1|0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
		},
	}, res)
}

func TestDiffProfileDeep(t *testing.T) {
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

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
	}, dp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(dp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:             n.LocationID(),
			flat:           n.FlatValues(),
			flatDiff:       n.FlatDiffValues(),
			cumulative:     n.CumulativeValues(),
			cumulativeDiff: n.CumulativeDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(-9),
			}},
		},
		{
			id: uint64(2),
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "2|0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(-3),
			}},
		},
		{
			id: uint64(2),
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "2|2|0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "3|2|2|0",
				},
			}},
			flatDiff: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
			cumulative: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "3|2|2|0",
				},
			}},
			cumulativeDiff: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
		},
	}, res)
}

func BenchmarkDiff(b *testing.B) {
	ctx := context.Background()

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

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		trace.NewNoopTracerProvider().Tracer(""),
		"benchdiff",
	)
	require.NoError(b, err)
	b.Cleanup(func() {
		l.Close()
	})
	profileTree1 := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	profileTree2 := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p2, 0)

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

	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			d, err := NewDiffProfile(prof1, prof2)
			require.NoError(b, err)
			CopyInstantProfileTree(d.ProfileTree())
		}
	})
}
