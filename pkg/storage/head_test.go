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
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestHead_MaxTime(t *testing.T) {
	h := NewHead(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)

	ctx := context.Background()
	app, err := h.Appender(ctx, labels.FromStrings("foo", "bar"))
	require.NoError(t, err)

	pt := NewProfileTree()
	pt.Insert(makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	for i := int64(1); i < 500; i++ {
		require.NoError(t, app.Append(ctx, &Profile{
			Tree: pt,
			Meta: InstantProfileMeta{Timestamp: i},
		}))
		require.Equal(t, i, h.MaxTime())
	}

	require.Equal(t, int64(499), h.MaxTime())
	require.Equal(t, int64(1), h.MinTime())
}

func TestStripeSeries(t *testing.T) {
	chunkPool := newHeadChunkPool()
	ss := newStripeSeries(DefaultStripeSize, func(int64) {})

	numSeries := uint64(1_000_000)
	for ref := uint64(0); ref < numSeries; ref++ {
		lset := labels.FromStrings("ref", strconv.FormatUint(ref, 10))
		_, _ = ss.getOrCreateWithID(ref, lset.Hash(), lset, chunkPool)
	}

	for i := uint64(0); i < numSeries; i += 100 {
		lset := labels.FromStrings("ref", strconv.FormatUint(i, 10))
		s1 := ss.getByID(i)
		s2 := ss.getByHash(lset.Hash(), lset)
		require.Equal(t, i, s1.id)
		require.Equal(t, i, s2.id)
		require.Equal(t, s1, s2)
	}
}

func TestSeriesHashmap(t *testing.T) {
	chunkPool := newHeadChunkPool()
	sh := seriesHashmap{}
	for i := 0; i < 1000; i++ {
		s := NewMemSeries(uint64(i), labels.FromStrings("foo", "bar", "i", strconv.Itoa(i)), func(int64) {}, chunkPool)
		sh.set(s.lset.Hash(), s)
	}

	lset := labels.FromStrings("foo", "bar", "i", "555")
	series := sh.get(lset.Hash(), lset)
	require.Equal(t, uint64(555), series.id)

	// Overwrite the 100 series
	lset = labels.FromStrings("foo", "bar", "i", "100")
	sh.set(lset.Hash(), &MemSeries{id: 100, lset: lset, minTime: 100})
	series = sh.get(lset.Hash(), lset)
	require.Equal(t, uint64(100), series.id)
	require.Equal(t, int64(100), series.minTime)

	// Delete the 200 series
	lset = labels.FromStrings("foo", "bar", "i", "200")
	sh.del(lset.Hash(), lset)

	series = sh.get(lset.Hash(), lset)
	require.Nil(t, series)
}

func BenchmarkStripeSeries(b *testing.B) {
	chunkPool := newHeadChunkPool()
	ss := newStripeSeries(DefaultStripeSize, func(int64) {})

	for i := 0; i < b.N; i++ {
		lset := labels.FromStrings("foo", "bar", "id", strconv.Itoa(i))
		_, _ = ss.getOrCreateWithID(uint64(i), lset.Hash(), lset, chunkPool)
	}

	b.ResetTimer()
	b.SetParallelism(10)

	b.Run("ids", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ss.getByID(uint64(i))
		}
	})
	b.Run("hashes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			lset := labels.FromStrings("foo", "bar", "id", strconv.Itoa(i))
			ss.getByHash(lset.Hash(), lset)
		}
	})
}

func TestHead_Truncate(t *testing.T) {
	h := NewHead(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)

	pt := NewProfileTree()
	pt.Insert(makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	ctx := context.Background()
	{
		app, err := h.Appender(ctx, labels.FromStrings("a", "b"))
		require.NoError(t, err)

		for i := int64(1); i <= 500; i++ {
			require.NoError(t, app.Append(ctx, &Profile{
				Tree: pt,
				Meta: InstantProfileMeta{Timestamp: i},
			}))
		}
	}
	{
		app, err := h.Appender(ctx, labels.FromStrings("a", "c"))
		require.NoError(t, err)

		for i := int64(100); i < 768; i++ {
			require.NoError(t, app.Append(ctx, &Profile{
				Tree: pt,
				Meta: InstantProfileMeta{Timestamp: i},
			}))
		}
	}

	require.NoError(t, h.Truncate(420))

	require.Equal(t, int64(340), h.MinTime())
	require.Equal(t, int64(767), h.MaxTime())
}
