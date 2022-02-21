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
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
)

func TestMemSeries(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())

	app, err := s.Appender()
	require.NoError(t, err)

	ctx := context.Background()

	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid4 := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	uuid5 := uuid.MustParse("00000000-0000-0000-0000-000000000005")

	s11 := profile.MakeSample(1, []uuid.UUID{uuid2, uuid1})
	s12 := profile.MakeSample(2, []uuid.UUID{uuid4, uuid1})
	s12.Label = label
	s12.NumLabel = numLabel
	s12.NumUnit = numUnit

	k11 := uuid.MustParse("00000000-0000-0000-0000-000000000e11")
	k12 := uuid.MustParse("00000000-0000-0000-0000-000000000e12")

	fp1 := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{
			PeriodType: profile.ValueType{},
			SampleType: profile.ValueType{},
			Timestamp:  1000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		FlatSamples: map[string]*profile.Sample{
			string(k11[:]): s11,
			string(k12[:]): s12,
		},
	}

	err = app.AppendFlat(ctx, fp1)
	require.NoError(t, err)

	require.Len(t, s.samples, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1), s.samples[string(k11[:])][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0])

	s2 := profile.MakeSample(3, []uuid.UUID{uuid2, uuid1})
	fp2 := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{
			PeriodType: profile.ValueType{},
			SampleType: profile.ValueType{},
			Timestamp:  2000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		FlatSamples: map[string]*profile.Sample{
			string(k11[:]): s2,
		},
	}

	err = app.AppendFlat(ctx, fp2)
	require.NoError(t, err)

	require.Len(t, s.samples, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.samples[string(k11[:])][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0]) // sparse - nothing added

	// Add another sample with one new Location
	s3 := profile.MakeSample(4, []uuid.UUID{uuid3, uuid1})
	k3 := uuid.MustParse("00000000-0000-0000-0000-0000000000e3")

	fp3 := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{
			PeriodType: profile.ValueType{},
			SampleType: profile.ValueType{},
			Timestamp:  3000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		FlatSamples: map[string]*profile.Sample{
			string(k3[:]): s3,
		},
	}

	err = app.AppendFlat(ctx, fp3)
	require.NoError(t, err)

	require.Len(t, s.samples, 3)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.samples[string(k11[:])][0]) // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0])    // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.samples[string(k3[:])][0])

	// Merging another profileTree onto the existing one with one new Location
	s4 := profile.MakeSample(6, []uuid.UUID{uuid5, uuid2, uuid1})
	k4 := uuid.MustParse("00000000-0000-0000-0000-0000000000e4")

	fp4 := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{
			PeriodType: profile.ValueType{},
			SampleType: profile.ValueType{},
			Timestamp:  4000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		FlatSamples: map[string]*profile.Sample{
			string(k4[:]): s4,
		},
	}

	err = app.AppendFlat(ctx, fp4)
	require.NoError(t, err)

	require.Len(t, s.samples, 4)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.samples[string(k11[:])][0])  // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0])     // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.samples[string(k3[:])][0]) // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(3, 6), s.samples[string(k4[:])][0])

	// Merging another profileTree onto the existing one with one new Location
	s5 := profile.MakeSample(7, []uuid.UUID{uuid2, uuid1})
	fp5 := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{
			PeriodType: profile.ValueType{},
			SampleType: profile.ValueType{},
			Timestamp:  5000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		FlatSamples: map[string]*profile.Sample{
			string(k11[:]): s5,
		},
	}
	err = app.AppendFlat(ctx, fp5)
	require.NoError(t, err)

	require.Len(t, s.samples, 4)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3, 0, 0, 7), s.samples[string(k11[:])][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0])     // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.samples[string(k3[:])][0]) // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(3, 6), s.samples[string(k4[:])][0]) // sparse - nothing added

	require.Equal(t, uint16(5), s.numSamples)
	require.Equal(t, chunkenc.FromValuesDelta(1000, 2000, 3000, 4000, 5000), s.timestamps[0].chunk)
	require.Equal(t, chunkenc.FromValuesRLE(time.Second.Nanoseconds(), 5), s.durations[0])
	require.Equal(t, chunkenc.FromValuesRLE(time.Second.Nanoseconds(), 5), s.periods[0])
}

func TestMemSeriesMany(t *testing.T) {
	snano := time.Second.Nanoseconds()

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())

	app, err := s.Appender()
	require.NoError(t, err)

	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid4 := uuid.MustParse("00000000-0000-0000-0000-000000000004")

	s1 := profile.MakeSample(0, []uuid.UUID{uuid2, uuid1})
	s2 := profile.MakeSample(0, []uuid.UUID{uuid4, uuid1})

	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")

	ctx := context.Background()
	for i := 1; i < 200; i++ {
		s1.Value = int64(i)
		s2.Value = int64(2 * i)

		err = app.AppendFlat(ctx, &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{
				Timestamp: int64(i),
				Duration:  snano,
				Period:    snano,
			},
			FlatSamples: map[string]*profile.Sample{
				string(k1[:]): s1,
				string(k2[:]): s2,
			},
		})
		require.NoError(t, err)
	}

	it := NewMultiChunkIterator(s.root)
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, int64(2*i+i), it.At())
	}

	require.Len(t, s.samples, 2)

	it = NewMultiChunkIterator(s.samples[string(k1[:])])
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, int64(i), it.At())
	}
	it = NewMultiChunkIterator(s.samples[string(k2[:])])
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, int64(2*i), it.At())
	}
}

func TestMemSeries_truncateChunksBefore(t *testing.T) {
	testcases := []struct {
		before int64

		truncated  int
		left       int
		minTime    int64
		maxTime    int64
		numSamples uint16
	}{
		{before: 10, truncated: 0, left: 5, minTime: 1, maxTime: 500, numSamples: 500},
		{before: 50, truncated: 0, left: 5, minTime: 1, maxTime: 500, numSamples: 500},
		{before: 123, truncated: 1, left: 4, minTime: 121, maxTime: 500, numSamples: 380},
		{before: 256, truncated: 2, left: 3, minTime: 241, maxTime: 500, numSamples: 260},
		{before: 490, truncated: 4, left: 1, minTime: 481, maxTime: 500, numSamples: 20},
		{before: 1_000, truncated: 5, left: 0, minTime: math.MaxInt64, maxTime: math.MinInt64, numSamples: 500},
	}

	chunkPool := newHeadChunkPool()

	for _, tc := range testcases {
		t.Run(fmt.Sprintf("truncate-%d", tc.before), func(t *testing.T) {
			ctx := context.Background()
			s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, chunkPool)

			app, err := s.Appender()
			require.NoError(t, err)

			for i := int64(1); i <= 500; i++ {
				require.NoError(t, app.AppendFlat(ctx, &profile.FlatProfile{
					Meta: profile.InstantProfileMeta{Timestamp: i},
				}))
			}

			require.Equal(t, tc.truncated, s.truncateChunksBefore(tc.before))

			require.Equal(t, tc.minTime, s.minTime)
			require.Equal(t, tc.maxTime, s.maxTime)
			require.Equal(t, tc.numSamples, s.numSamples)

			require.Equal(t, tc.left, len(s.timestamps))
			require.Equal(t, tc.left, len(s.durations))
			require.Equal(t, tc.left, len(s.periods))

			for _, c := range s.samples {
				require.Equal(t, tc.left, len(c))
			}
		})
	}
}

func TestMemSeries_truncateFlatChunksBeforeConcurrent(t *testing.T) {
	ctx := context.Background()
	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(i int64) {}, newHeadChunkPool())

	app, err := s.Appender()
	require.NoError(t, err)

	s1 := profile.MakeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	k1 := profile.MakeStacktraceKey(s1)

	for i := int64(1); i < 500; i++ {
		require.NoError(t, app.AppendFlat(ctx, &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{Timestamp: i},
			FlatSamples: map[string]*profile.Sample{
				string(k1): s1,
			},
		}))
	}

	// Truncating won't do anything here.
	require.Equal(t, 0, s.truncateChunksBefore(75))
	require.Equal(t, int64(1), s.minTime)
	require.Equal(t, int64(499), s.maxTime)

	// Truncate the first two chunks.
	require.Equal(t, 2, s.truncateChunksBefore(256))

	require.Equal(t, int64(241), s.minTime)
	require.Equal(t, int64(499), s.maxTime)

	// Test for appending working correctly after truncating.
	for i := int64(500); i < 1_000; i++ {
		require.NoError(t, app.AppendFlat(ctx, &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{Timestamp: i},
			FlatSamples: map[string]*profile.Sample{
				string(k1): s1,
			},
		}))
	}

	require.Equal(t, int64(241), s.minTime)
	require.Equal(t, int64(999), s.maxTime)

	// Truncate all chunks.
	require.Equal(t, 7, s.truncateChunksBefore(1_234))
	require.Equal(t, int64(math.MaxInt64), s.minTime)
	require.Equal(t, int64(math.MinInt64), s.maxTime)

	// Append more profiles after truncating all chunks.
	for i := int64(1_100); i < 1_234; i++ {
		require.NoError(t, app.AppendFlat(ctx, &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{Timestamp: i},
			FlatSamples: map[string]*profile.Sample{
				string(k1): s1,
			},
		}))
	}

	require.Equal(t, int64(1_100), s.minTime)
	require.Equal(t, int64(1_233), s.maxTime)
}

// for i in {1..10}; do go test -bench=BenchmarkMemSeries_truncateChunksBefore --benchtime=100000x ./pkg/storage >> ./pkg/storage/benchmark/series-truncate.txt; done

func BenchmarkMemSeries_truncateChunksBefore(b *testing.B) {
	ctx := context.Background()
	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())
	app, err := s.Appender()
	require.NoError(b, err)

	sample := profile.MakeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	sampleKey := profile.MakeStacktraceKey(sample)

	p := &profile.FlatProfile{
		Meta: profile.InstantProfileMeta{},
		FlatSamples: map[string]*profile.Sample{
			string(sampleKey): sample,
		},
	}

	for i := 1; i <= b.N; i++ {
		p.Meta.Timestamp = int64(i)
		_ = app.AppendFlat(ctx, p)
	}

	// Truncate the first roughly 2/3 of all chunks.

	mint := int64(float64(b.N) / 3 * 2)

	b.ReportAllocs()
	b.ResetTimer()

	s.truncateChunksBefore(mint)
}
