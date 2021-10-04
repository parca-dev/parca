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
	"errors"
	"fmt"

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
)

// MemRangeSeries is an iterator that only queries certain chunks within the range and
// then only the samples within the range.
type MemRangeSeries struct {
	s    *MemSeries
	mint int64
	maxt int64
}

func (rs *MemRangeSeries) Labels() labels.Labels {
	return rs.s.Labels()
}

func (rs *MemRangeSeries) Iterator() ProfileSeriesIterator {
	rs.s.mu.RLock()
	defer rs.s.mu.RUnlock()

	chunkStart, chunkEnd := rs.s.timestamps.indexRange(rs.mint, rs.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range rs.s.timestamps[chunkStart:chunkEnd] {
		timestamps = append(timestamps, t.chunk)
	}

	it := NewMultiChunkIterator(timestamps)
	start, end, err := getIndexRange(it, rs.s.numSamples, rs.mint, rs.maxt)
	if err != nil {
		return &MemRangeSeriesIterator{err: err}
	}

	rootKey := ProfileTreeValueNodeKey{location: "0"}

	rootIt := NewMultiChunkIterator(rs.s.cumulativeValues[rootKey][chunkStart:chunkEnd])
	if start != 0 {
		rootIt.Seek(start)
	}

	root := &MemSeriesIteratorTreeNode{}
	root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
		Values:   rootIt,
		Label:    rs.s.labels[rootKey],
		NumLabel: rs.s.numLabels[rootKey],
		NumUnit:  rs.s.numUnits[rootKey],
	})

	memItStack := MemSeriesIteratorTreeStack{{
		node:  root,
		child: 0,
	}}

	treeIt := rs.s.seriesTree.Iterator()

	for treeIt.HasMore() {
		if treeIt.NextChild() {
			child := treeIt.At()

			n := &MemSeriesIteratorTreeNode{
				locationID: child.LocationID,
				Children:   make([]*MemSeriesIteratorTreeNode, 0, len(child.Children)),
			}

			for _, key := range child.keys {
				if chunks, ok := rs.s.flatValues[key]; ok {
					it := NewMultiChunkIterator(chunks[chunkStart:chunkEnd])
					if start != 0 {
						it.Seek(start)
					}
					n.flatValues = append(n.flatValues, &MemSeriesIteratorTreeValueNode{
						Values:   it,
						Label:    rs.s.labels[key],
						NumLabel: rs.s.numLabels[key],
						NumUnit:  rs.s.numUnits[key],
					})
				}
				if chunks, ok := rs.s.cumulativeValues[key]; ok {
					it := NewMultiChunkIterator(chunks[chunkStart:chunkEnd])
					if start != 0 {
						it.Seek(start) // We might need another interface with Seek(index uint64) for multi chunks.
					}
					n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
						Values:   it,
						Label:    rs.s.labels[key],
						NumLabel: rs.s.numLabels[key],
						NumUnit:  rs.s.numUnits[key],
					})
				}
			}

			cur := memItStack.Peek()
			cur.node.Children = append(cur.node.Children, n)

			memItStack.Push(&MemSeriesIteratorTreeStackEntry{
				node:  n,
				child: 0,
			})
			treeIt.StepInto()
			continue
		}
		treeIt.StepUp()
		memItStack.Pop()
	}

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
		tree: &MemSeriesIteratorTree{
			Roots: root,
		},
	}
}

type MemRangeSeriesIterator struct {
	s    *MemSeries
	mint int64
	maxt int64

	tree               *MemSeriesIteratorTree
	timestampsIterator MemSeriesValuesIterator
	durationsIterator  MemSeriesValuesIterator
	periodsIterator    MemSeriesValuesIterator

	numSamples uint64 // uint16 might not be enough for many chunks (~500+)
	err        error
}

func (it *MemRangeSeriesIterator) Next() bool {
	it.s.mu.RLock()
	defer it.s.mu.RUnlock()

	if it.numSamples == 0 {
		return false
	}

	if !it.timestampsIterator.Next() {
		it.err = errors.New("unexpected end of timestamps iterator")
		return false
	}

	if it.timestampsIterator.Err() != nil {
		it.err = fmt.Errorf("next timestamp: %w", it.timestampsIterator.Err())
		return false
	}

	if !it.durationsIterator.Next() {
		it.err = errors.New("unexpected end of durations iterator")
		return false
	}

	if it.durationsIterator.Err() != nil {
		it.err = fmt.Errorf("next duration: %w", it.durationsIterator.Err())
		return false
	}

	if !it.periodsIterator.Next() {
		it.err = errors.New("unexpected end of periods iterator")
		return false
	}

	if it.periodsIterator.Err() != nil {
		it.err = fmt.Errorf("next period: %w", it.periodsIterator.Err())
		return false
	}

	read := it.timestampsIterator.Read()

	if dread := it.durationsIterator.Read(); dread != read {
		it.err = fmt.Errorf("duration iterator in wrong iteration, expected %d got %d", read, dread)
		return false
	}
	if pread := it.periodsIterator.Read(); pread != read {
		it.err = fmt.Errorf("period iterator in wrong iteration, expected %d got %d", read, pread)
		return false
	}

	iit := NewMemSeriesIteratorTreeIterator(it.tree)
	for iit.HasMore() {
		if iit.NextChild() {
			child := iit.at()

			for _, v := range child.flatValues {
				if !v.Values.Next() {
					it.err = errors.New("unexpected end of flat value iterator")
					return false
				}

				if v.Values.Err() != nil {
					it.err = fmt.Errorf("next flat value: %w", v.Values.Err())
					return false
				}

				if vread := v.Values.Read(); vread != read {
					it.err = fmt.Errorf("flat value iterator in wrong iteration, expected %d got %d", read, vread)
					return false
				}
			}

			for _, v := range child.cumulativeValues {
				if !v.Values.Next() {
					it.err = errors.New("unexpected end of cumulative value iterator")
					return false
				}

				if v.Values.Err() != nil {
					it.err = fmt.Errorf("next cumulative value: %w", v.Values.Err())
					return false
				}

				if vread := v.Values.Read(); vread != read {
					it.err = fmt.Errorf("wrong iteration for cumulative value, expected: %d, got: %d", read, vread)
					return false
				}
			}

			iit.StepInto()
			continue
		}
		iit.StepUp()
	}

	it.numSamples--
	return true
}

func (it *MemRangeSeriesIterator) At() InstantProfile {
	return &MemSeriesInstantProfile{
		itt: it.tree,
		it: &MemSeriesIterator{
			tree:               it.tree,
			timestampsIterator: it.timestampsIterator,
			durationsIterator:  it.durationsIterator,
			periodsIterator:    it.periodsIterator,
			series:             it.s,
			numSamples:         uint16(it.numSamples - 1), // should be an uint64 eventually.
		},
	}
}

func (it *MemRangeSeriesIterator) Err() error {
	return it.err
}
