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
	"fmt"

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
)

// MemRootSeries is an iterator that only queries the cumulative values for the root of each series.
type MemRootSeries struct {
	s    *MemSeries
	mint int64
	maxt int64
}

func (rs *MemRootSeries) Labels() labels.Labels {
	return rs.s.Labels()
}

func (rs *MemRootSeries) Iterator() ProfileSeriesIterator {
	rs.s.mu.RLock()
	defer rs.s.mu.RUnlock()

	var numSamples uint64

	chunkStart, chunkEnd := rs.s.timestamps.indexRange(rs.mint, rs.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range rs.s.timestamps[chunkStart:chunkEnd] {
		numSamples += uint64(t.chunk.NumSamples())
		timestamps = append(timestamps, t.chunk)
	}

	it := NewMultiChunkIterator(timestamps)
	start, end, err := getIndexRange(it, numSamples, rs.mint, rs.maxt)
	if start == end {
		return &MemRootSeriesIterator{err: fmt.Errorf("no samples within the time range")}
	}
	if err != nil {
		return &MemRootSeriesIterator{err: err}
	}

	timestampIterator := NewMultiChunkIterator(timestamps)
	rootIterator := NewMultiChunkIterator(rs.s.root[chunkStart:chunkEnd])

	if start != 0 {
		timestampIterator.Seek(start)
		rootIterator.Seek(start)
	}

	// Set numSamples correctly if only subset selected.
	if end-start < numSamples {
		// -1 for length to index
		// -1 for exclusive first sample
		numSamples = end - start - 2
	}

	return &MemRootSeriesIterator{
		s:    rs.s,
		mint: rs.mint,
		maxt: rs.maxt,

		timestampsIterator: timestampIterator,
		rootIterator:       rootIterator,

		numSamples: numSamples,
	}
}

type MemRootSeriesIterator struct {
	s    *MemSeries
	mint int64
	maxt int64

	timestampsIterator MemSeriesValuesIterator
	rootIterator       MemSeriesValuesIterator

	numSamples uint64
	err        error
}

func (it *MemRootSeriesIterator) Next() bool {
	if it.err != nil || it.numSamples == 0 {
		return false
	}

	it.s.mu.RLock()
	defer it.s.mu.RUnlock()

	if !it.timestampsIterator.Next() {
		it.err = fmt.Errorf("unexpected end of timestamps iterator")
		return false
	}
	if it.timestampsIterator.Err() != nil {
		it.err = fmt.Errorf("next timestamp: %w", it.timestampsIterator.Err())
		return false
	}

	if !it.rootIterator.Next() {
		it.err = fmt.Errorf("unexpected end of root iterator")
		return false
	}
	if it.rootIterator.Err() != nil {
		it.err = fmt.Errorf("next root: %w", it.rootIterator.Err())
		return false
	}

	tr := it.timestampsIterator.Read()
	rr := it.rootIterator.Read()
	if tr != rr {
		it.err = fmt.Errorf("iteration mismatch for timestamps and roots: %d, got: %d", tr, rr)
		return false
	}

	it.numSamples--
	return true
}

func (it *MemRootSeriesIterator) At() InstantProfile {
	return &Profile{
		Meta: InstantProfileMeta{
			Timestamp:  it.timestampsIterator.At(),
			PeriodType: it.s.periodType,
			SampleType: it.s.sampleType,
		},
		Tree: &ProfileTree{
			Roots: &ProfileTreeRootNode{
				CumulativeValue: it.rootIterator.At(),
				ProfileTreeNode: &ProfileTreeNode{},
			},
		},
	}
}

func (it *MemRootSeriesIterator) Err() error {
	return it.err
}
