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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestMemRangeSeries_Iterator(t *testing.T) {
	ctx := context.Background()
	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())
	app, err := s.Appender()
	require.NoError(t, err)

	s1 := profile.MakeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	k1 := profile.MakeStacktraceKey(s1)

	for i := 1; i <= 500; i++ {
		s1.Value = int64(i)
		p := &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{
				Timestamp: int64(i),
				Duration:  time.Second.Nanoseconds(),
				Period:    time.Second.Nanoseconds(),
			},
			FlatSamples: map[string]*profile.Sample{
				string(k1): s1,
			},
		}
		err = app.AppendFlat(ctx, p)
		require.NoError(t, err)
	}

	it := (&MemRangeSeries{s: s, mint: 74, maxt: 420}).Iterator()

	seen := int64(75)
	for it.Next() {
		p := it.At()
		require.Equal(t, seen, p.ProfileMeta().Timestamp)
		for _, sample := range p.Samples() {
			require.Equal(t, seen, sample.Value)
		}
		seen++
	}

	require.NoError(t, it.Err())
	require.Equal(t, int64(421), seen) // 421 would be seen next but 420 was the last value.
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

	start, end, err = getIndexRange(NewMultiChunkIterator([]chunkenc.Chunk{c}), 123, 1, 12)
	require.NoError(t, err)
	require.Equal(t, uint64(0), start)
	require.Equal(t, uint64(5), end)
}
