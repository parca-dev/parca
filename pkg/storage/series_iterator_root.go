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

	chunkStart, chunkEnd := rs.s.timestamps.indexRange(rs.mint, rs.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range rs.s.timestamps[chunkStart:chunkEnd] {
		timestamps = append(timestamps, t.chunk)
	}

	it := NewMultiChunkIterator(timestamps)
	start, end, err := getIndexRange(it, rs.mint, rs.maxt)
	if err != nil {
		return &MemRangeSeriesIterator{err: err}
	}

	rootKey := ProfileTreeValueNodeKey{location: "0"}
	it.Reset(rs.s.cumulativeValues[rootKey][chunkStart:chunkEnd])
	if start != 0 {
		it.Seek(start)
	}

	root := &MemSeriesIteratorTreeNode{}
	root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
		Values:   it,
		Label:    rs.s.labels[rootKey],
		NumLabel: rs.s.numLabels[rootKey],
		NumUnit:  rs.s.numUnits[rootKey],
	})

	// As an implementation detail of the tree stack iterator,
	// we need to have the root as child of the root once more.
	// We need to recreate the root and not simply append it itself, as that creates an endless recursion.
	root.Children = append(root.Children, &MemSeriesIteratorTreeNode{
		cumulativeValues: []*MemSeriesIteratorTreeValueNode{{
			Values:   it,
			Label:    rs.s.labels[rootKey],
			NumLabel: rs.s.numLabels[rootKey],
			NumUnit:  rs.s.numUnits[rootKey],
		}},
	})

	timestampIterator := NewMultiChunkIterator(timestamps)
	durationsIterator := NewMultiChunkIterator(rs.s.durations[chunkStart:chunkEnd])
	periodsIterator := NewMultiChunkIterator(rs.s.periods[chunkStart:chunkEnd])

	if start != 0 {
		timestampIterator.Seek(start)
		durationsIterator.Seek(start)
		periodsIterator.Seek(start)
	}

	numSamples := uint64(rs.s.numSamples)
	if end-start < numSamples {
		numSamples = end - start - 1
	}

	return &MemRangeSeriesIterator{
		s:    rs.s,
		mint: rs.mint,
		maxt: rs.maxt,

		numSamples:         numSamples,
		timestampsIterator: timestampIterator,
		durationsIterator:  durationsIterator,
		periodsIterator:    periodsIterator,

		tree: &MemSeriesIteratorTree{Roots: root},
	}
}
