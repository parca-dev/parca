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
	"strconv"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestStripeSeries(t *testing.T) {
	ss := newStripeSeries(DefaultStripeSize)

	numSeries := uint64(1_000_000)
	for ref := uint64(0); ref < numSeries; ref++ {
		lset := labels.FromStrings("ref", strconv.FormatUint(ref, 10))
		_, _ = ss.getOrCreateWithID(ref, lset.Hash(), lset)
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
	sh := seriesHashmap{}
	for i := 0; i < 1000; i++ {
		s := NewMemSeries(labels.FromStrings("foo", "bar", "i", strconv.Itoa(i)), uint64(i))
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
	ss := newStripeSeries(DefaultStripeSize)

	for i := 0; i < b.N; i++ {
		lset := labels.FromStrings("foo", "bar", "id", strconv.Itoa(i))
		_, _ = ss.getOrCreateWithID(uint64(i), lset.Hash(), lset)
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
