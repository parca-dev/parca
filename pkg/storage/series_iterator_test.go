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
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestMemSeriesIterator(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())

	s.timestamps = []*timestampChunk{{chunk: chunkenc.FromValuesDelta(1, 2)}}
	s.durations = []chunkenc.Chunk{chunkenc.FromValuesRLE(time.Second.Nanoseconds(), 2)}
	s.periods = []chunkenc.Chunk{chunkenc.FromValuesRLE(100, 2)}

	{
		pt := NewProfileTree()
		{
			s := makeSample(1, []uint64{2, 1})
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
			pt.Insert(makeSample(3, []uint64{3, 1}))
		}
		{
			pt.Insert(makeSample(4, []uint64{4, 1}))
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
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       2,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 1}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 1}},
			},
			{
				LocationID:       3,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0}},
			},
			{
				LocationID:       4,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 0}},
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
			FlatValues       []*ProfileTreeValueNode
		}{
			{
				LocationID:       0,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 7}},
			},
			{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 7}},
			},
			{
				LocationID:       2,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0}},
			},
			{
				LocationID:       3,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       4,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 4}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 4}},
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
		require.Equal(t, 5, i)
	}

	// No more iterations
	require.False(t, it.Next())
}

func TestIteratorConsistency(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"iteratorconsistency",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	s := NewMemSeries(1, labels.Labels{{Name: "test_name", Value: "test_value"}}, func(int64) {}, newHeadChunkPool())
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	profile, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)
	require.NoError(t, app.Append(ctx, profile))

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

func TestGetIndexRange(t *testing.T) {
	c := chunkenc.FromValuesDelta(2, 4, 6, 7, 8)

	start, end, err := getIndexRange(c.Iterator(nil), 5, 1, 9)
	require.NoError(t, err)
	require.Equal(t, uint64(0), start)
	require.Equal(t, uint64(5), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 2, 9)
	require.NoError(t, err)
	require.Equal(t, uint64(0), start)
	require.Equal(t, uint64(5), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 3, 6)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)
	require.Equal(t, uint64(3), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 3, 7)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)
	require.Equal(t, uint64(4), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 3, 8)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)
	require.Equal(t, uint64(5), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 3, 9)
	require.NoError(t, err)
	require.Equal(t, uint64(1), start)
	require.Equal(t, uint64(5), end)

	start, end, err = getIndexRange(c.Iterator(nil), 5, 5, 7)
	require.NoError(t, err)
	require.Equal(t, uint64(2), start)
	require.Equal(t, uint64(4), end)
}

func TestIteratorRangeSum(t *testing.T) {
	c := chunkenc.FromValuesDelta(2, 4, 6, 7, 8)
	start, end, err := getIndexRange(c.Iterator(nil), 5, 3, 6)
	require.NoError(t, err)

	sum, err := iteratorRangeSum(c.Iterator(nil), start, end)
	require.NoError(t, err)
	require.Equal(t, int64(10), sum)
}

func TestIteratorRangeMax(t *testing.T) {
	c := chunkenc.FromValuesDelta(10, 4, 12, 7, 8)
	max, err := iteratorRangeMax(c.Iterator(nil), 1, 5)
	require.NoError(t, err)
	require.Equal(t, int64(12), max)
}
