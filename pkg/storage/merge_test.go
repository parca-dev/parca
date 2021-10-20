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
	"fmt"
	"math"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
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

	mp, err := MergeProfiles(p1, p2)
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
			id:       n.LocationID(),
			flat:     n.FlatValues(),
			flatDiff: n.FlatDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
		},
		{
			id: uint64(1),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "2|1|0"},
				Value: int64(2),
			}},
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "3|1|0"},
				Value: int64(1),
			}},
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

	mp, err := MergeProfiles(p1, p2)
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
			id:       n.LocationID(),
			flat:     n.FlatValues(),
			flatDiff: n.FlatDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
		},
		{
			id: uint64(2),
		},
		{
			id: uint64(2),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "3|2|2|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "3|3|2|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "6|2|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(3),
		},
		{
			id: uint64(1),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "1|3|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "2|3|0"},
				Value: int64(3),
			}},
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

	mp, err := MergeProfiles(p1, p2)
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
			id:       n.LocationID(),
			flat:     n.FlatValues(),
			flatDiff: n.FlatDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uint64(0),
		},
		{
			id: uint64(1),
		},
		{
			id: uint64(2),
			flat: []*ProfileTreeValueNode{{
				Value: int64(4),
			}},
		},
		{
			id: uint64(3),
		},
		{
			id: uint64(4),
			flat: []*ProfileTreeValueNode{{
				Value: int64(6),
			}},
		},
		{
			id: uint64(5),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "5|3|2|1|0"},
				Value: int64(1),
			}},
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "6|3|2|1|0"},
				Value: int64(1),
			}},
		},
		{
			id: uint64(2),
		},
		{
			id: uint64(2),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "3|2|2|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(3),
		},
		{
			id: uint64(3),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "3|3|2|0"},
				Value: int64(3),
			}},
		},
		{
			id: uint64(6),
			flat: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "6|2|0"},
				Value: int64(3),
			}},
		},
	}, res)
}

func TestMergeSingle(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"mergesingle",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	prof, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)

	m, err := MergeProfiles(prof)
	require.NoError(t, err)
	CopyInstantProfileTree(m.ProfileTree())
}

func TestMergeMany(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"mergemany",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	prof, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)

	num := 1000
	profiles := make([]InstantProfile, 0, 1000)
	for i := 0; i < num; i++ {
		profiles = append(profiles, prof)
	}

	m, err := MergeProfiles(profiles...)
	require.NoError(t, err)
	CopyInstantProfileTree(m.ProfileTree())
}

type sample struct {
	id       uint64
	flat     []*ProfileTreeValueNode
	flatDiff []*ProfileTreeValueNode
}

func BenchmarkTreeMerge(b *testing.B) {
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
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"treemerge",
	)
	b.Cleanup(func() {
		l.Close()
	})
	require.NoError(b, err)
	profileTree1, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(b, err)
	profileTree2, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p2, 0)
	require.NoError(b, err)

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
	m, err := MergeProfiles(prof1, prof2)
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

func BenchmarkMergeMany(b *testing.B) {
	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				ctx := context.Background()
				f, err := os.Open("testdata/profile1.pb.gz")
				require.NoError(b, err)
				p, err := profile.Parse(f)
				require.NoError(b, err)
				require.NoError(b, f.Close())

				l, err := metastore.NewInMemorySQLiteProfileMetaStore(
					prometheus.NewRegistry(),
					trace.NewNoopTracerProvider().Tracer(""),
					"bencmergequery",
				)
				require.NoError(b, err)
				b.Cleanup(func() {
					l.Close()
				})
				prof, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
				require.NoError(b, err)

				profiles := make([]InstantProfile, 0, n)
				for i := 0; i < n; i++ {
					profiles = append(profiles, prof)
				}

				m, err := MergeProfiles(profiles...)
				require.NoError(b, err)
				CopyInstantProfileTree(m.ProfileTree())
			}
		})
	}
}
