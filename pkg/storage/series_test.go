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
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
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

	s11 := makeSample(1, []uuid.UUID{uuid2, uuid1})
	s12 := makeSample(2, []uuid.UUID{uuid4, uuid1})
	s12.Label = label
	s12.NumLabel = numLabel
	s12.NumUnit = numUnit

	k11 := uuid.MustParse("00000000-0000-0000-0000-000000000e11")
	k12 := uuid.MustParse("00000000-0000-0000-0000-000000000e12")

	fp1 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{},
			SampleType: ValueType{},
			Timestamp:  1000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		samples: map[string]*Sample{
			string(k11[:]): s11,
			string(k12[:]): s12,
		},
	}

	err = app.AppendFlat(ctx, fp1)
	require.NoError(t, err)

	require.Len(t, s.samples, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1), s.samples[string(k11[:])][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0])

	s2 := makeSample(3, []uuid.UUID{uuid2, uuid1})
	fp2 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{},
			SampleType: ValueType{},
			Timestamp:  2000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		samples: map[string]*Sample{
			string(k11[:]): s2,
		},
	}

	err = app.AppendFlat(ctx, fp2)
	require.NoError(t, err)

	require.Len(t, s.samples, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.samples[string(k11[:])][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.samples[string(k12[:])][0]) // sparse - nothing added

	// Add another sample with one new Location
	s3 := makeSample(4, []uuid.UUID{uuid3, uuid1})
	k3 := uuid.MustParse("00000000-0000-0000-0000-0000000000e3")

	fp3 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{},
			SampleType: ValueType{},
			Timestamp:  3000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		samples: map[string]*Sample{
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
	s4 := makeSample(6, []uuid.UUID{uuid5, uuid2, uuid1})
	k4 := uuid.MustParse("00000000-0000-0000-0000-0000000000e4")

	fp4 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{},
			SampleType: ValueType{},
			Timestamp:  4000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		samples: map[string]*Sample{
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
	s5 := makeSample(7, []uuid.UUID{uuid2, uuid1})
	fp5 := &FlatProfile{
		Meta: InstantProfileMeta{
			PeriodType: ValueType{},
			SampleType: ValueType{},
			Timestamp:  5000,
			Duration:   time.Second.Nanoseconds(),
			Period:     time.Second.Nanoseconds(),
		},
		samples: map[string]*Sample{
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

	s1 := makeSample(0, []uuid.UUID{uuid2, uuid1})
	s2 := makeSample(0, []uuid.UUID{uuid4, uuid1})

	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")

	ctx := context.Background()
	for i := 1; i < 200; i++ {
		s1.Value = int64(i)
		s2.Value = int64(2 * i)

		err = app.AppendFlat(ctx, &FlatProfile{
			Meta: InstantProfileMeta{
				Timestamp: int64(i),
				Duration:  snano,
				Period:    snano,
			},
			samples: map[string]*Sample{
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

func TestMemSeriesTree(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	// Note: These keys are not unique per location.
	// For this test they simply seem to be.
	k2 := ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}
	k3 := ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000003|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}
	k4 := ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000004|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000", labels: `"foo"["bar" "baz"]`, numlabels: `"foo"[1 2][6279746573 6f626a65637473]`}
	k5 := ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000005|00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}

	s11 := makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})

	s12 := makeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	s12.Label = label
	s12.NumLabel = numLabel
	s12.NumUnit = numUnit

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())
	s.timestamps = append(s.timestamps, &timestampChunk{chunk: chunkenc.FromValuesDelta(1)})

	pt1 := NewProfileTree()
	pt1.Insert(s11)
	pt1.Insert(s12)
	err := s.seriesTree.Insert(0, pt1)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1), s.flatValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0])

	require.Len(t, s.labels, 1)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]string{k4: label}, s.labels)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]int64{k4: numLabel}, s.numLabels)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]string{k4: numUnit}, s.numUnits)

	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"), // root
			Children: []*MemSeriesTreeNode{{
				LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				}},
			}},
		},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one

	s2 := makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})

	pt2 := NewProfileTree()
	pt2.Insert(s2)

	s.timestamps[0] = &timestampChunk{chunk: chunkenc.FromValuesDelta(1, 2)}
	err = s.seriesTree.Insert(1, pt2)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0]) // sparse - nothing added

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"), // root
			Children: []*MemSeriesTreeNode{{
				LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				}},
			}},
		},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one with one new Location
	s3 := makeSample(4, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})

	pt3 := NewProfileTree()
	pt3.Insert(s3)

	s.timestamps[0] = &timestampChunk{chunk: chunkenc.FromValuesDelta(1, 2, 3)}
	err = s.seriesTree.Insert(2, pt3)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 3)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2][0])   // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0])      // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.flatValues[k3][0]) // new

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"), // root
			Children: []*MemSeriesTreeNode{{
				LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}, {
					keys:       []ProfileTreeValueNodeKey{k3},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				}},
			}},
		},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one with one new Location
	s4 := makeSample(6, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000005"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	pt4 := NewProfileTree()
	pt4.Insert(s4)

	s.timestamps[0] = &timestampChunk{chunk: chunkenc.FromValuesDelta(1, 2, 3, 4)}
	err = s.seriesTree.Insert(3, pt4)
	require.NoError(t, err)

	// Merging another profileTree onto the existing one with one new Location
	s5 := makeSample(7, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	pt5 := NewProfileTree()
	pt5.Insert(s5)

	s.timestamps[0] = &timestampChunk{chunk: chunkenc.FromValuesDelta(1, 2, 3, 4, 5)}
	err = s.seriesTree.Insert(4, pt5)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3, 0, 0, 7), s.flatValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0])
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.flatValues[k3][0])
	require.Equal(t, chunkenc.FromValuesXORAt(3, 6), s.flatValues[k5][0])

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"), // root
			Children: []*MemSeriesTreeNode{{
				LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					Children: []*MemSeriesTreeNode{{
						keys:       []ProfileTreeValueNodeKey{k5},
						LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000005"),
					}},
				}, {
					keys:       []ProfileTreeValueNodeKey{k3},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				}},
			}},
		},
	}, s.seriesTree)
}

func TestMemSeriesTreeMany(t *testing.T) {
	snano := time.Second.Nanoseconds()

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())

	app, err := s.Appender()
	require.NoError(t, err)

	for i := 1; i < 200; i++ {
		pt1 := NewProfileTree()
		pt1.Insert(makeSample(int64(i), []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000002"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}))
		pt1.Insert(makeSample(2*int64(i), []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000004"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}))

		err = app.Append(context.Background(), &Profile{
			Meta: InstantProfileMeta{
				Timestamp: int64(i),
				Duration:  snano,
				Period:    snano,
			},
			Tree: pt1,
		})
		require.NoError(t, err)
	}

	it := NewMultiChunkIterator(s.root)
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, int64(2*i+i), it.At())
	}

	require.Len(t, s.flatValues, 2)

	it = NewMultiChunkIterator(s.flatValues[ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}])
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, int64(i), it.At())
	}

	it = NewMultiChunkIterator(s.flatValues[ProfileTreeValueNodeKey{location: "00000000-0000-0000-0000-000000000004|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}])
	for i := 1; i < 200; i++ {
		require.True(t, it.Next())
		require.Equal(t, 2*int64(i), it.At())
	}

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"), // root
			Children: []*MemSeriesTreeNode{{
				LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{{location: "00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}, {
					keys:       []ProfileTreeValueNodeKey{{location: "00000000-0000-0000-0000-000000000004|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000"}},
					LocationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
				}},
			}},
		},
	}, s.seriesTree)
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
				require.NoError(t, app.AppendFlat(ctx, &FlatProfile{
					Meta: InstantProfileMeta{Timestamp: i},
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

	s1 := makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	k1 := makeStacktraceKey(s1)

	for i := int64(1); i < 500; i++ {
		require.NoError(t, app.AppendFlat(ctx, &FlatProfile{
			Meta: InstantProfileMeta{Timestamp: i},
			samples: map[string]*Sample{
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
		require.NoError(t, app.AppendFlat(ctx, &FlatProfile{
			Meta: InstantProfileMeta{Timestamp: i},
			samples: map[string]*Sample{
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
		require.NoError(t, app.AppendFlat(ctx, &FlatProfile{
			Meta: InstantProfileMeta{Timestamp: i},
			samples: map[string]*Sample{
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

	sample := makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	sampleKey := makeStacktraceKey(sample)

	p := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: map[string]*Sample{
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
