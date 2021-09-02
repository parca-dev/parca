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

// MemMergeSeries is an iterator that sums up all values while iterating that are within the range.
// In the end it returns a slice iterator with only the merge profile in it.
type MemMergeSeries struct {
	s    *MemSeries
	mint int64
	maxt int64
}

func (ms *MemMergeSeries) Labels() labels.Labels {
	return ms.s.Labels()
}

func (ms *MemMergeSeries) Iterator() ProfileSeriesIterator {
	ms.s.mu.RLock()
	defer ms.s.mu.RUnlock()

	chunkStart, chunkEnd := ms.s.timestamps.indexRange(ms.mint, ms.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range ms.s.timestamps[chunkStart:chunkEnd] {
		timestamps = append(timestamps, t.chunk)
	}

	sl := &SliceProfileSeriesIterator{i: -1}

	start, end, err := getIndexRange(NewMultiChunkIterator(timestamps), ms.mint, ms.maxt)
	if err != nil {
		sl.err = err
		return sl
	}

	it := NewMultiChunkIterator(timestamps)
	it.Seek(start)
	it.Next()
	minTimestamp := it.At()

	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.durations[chunkStart:chunkEnd])
	duration, err := iteratorRangeSum(it, start, end)
	if err != nil {
		sl.err = err
		return sl
	}

	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.periods[chunkStart:chunkEnd])
	period, err := iteratorRangeMax(it, start, end)
	if err != nil {
		sl.err = err
		return sl
	}

	p := &Profile{
		Meta: InstantProfileMeta{
			Duration:   duration,
			Period:     period,
			Timestamp:  minTimestamp,
			PeriodType: ms.s.periodType,
			SampleType: ms.s.sampleType,
		},
	}

	rootKey := ProfileTreeValueNodeKey{location: "0"}

	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.cumulativeValues[rootKey][chunkStart:chunkEnd])
	sum, err := iteratorRangeSum(it, start, end)
	if err != nil {
		sl.err = err
		return sl
	}

	cur := &ProfileTreeNode{
		cumulativeValues: []*ProfileTreeValueNode{{
			Value: sum,
		}},
	}

	tree := &ProfileTree{Roots: cur}
	p.Tree = tree
	sl.samples = append(sl.samples, p)

	stack := ProfileTreeStack{{node: cur}}
	treeIt := ms.s.seriesTree.Iterator()

	if !treeIt.HasMore() {
		return sl
	}
	if !treeIt.NextChild() {
		return sl
	}

	treeIt.StepInto()

	for {
		hasMore := treeIt.HasMore()
		if !hasMore {
			break
		}
		nextChild := treeIt.NextChild()
		if nextChild {
			child := treeIt.At()

			n := &ProfileTreeNode{
				locationID: child.LocationID,
				Children:   make([]*ProfileTreeNode, 0, len(child.Children)),
			}

			for _, key := range child.keys {
				if chunks, ok := ms.s.flatValues[key]; ok {
					it.Reset(chunks[chunkStart:chunkEnd])
					sum, err := iteratorRangeSum(it, start, end)
					if err != nil {
						sl.err = err
						return sl
					}
					if sum > 0 {
						n.flatValues = append(n.flatValues, &ProfileTreeValueNode{
							Value:    sum,
							Label:    ms.s.labels[key],
							NumLabel: ms.s.numLabels[key],
							NumUnit:  ms.s.numUnits[key],
						})
					}
				}
				if chunks, ok := ms.s.cumulativeValues[key]; ok {
					it.Reset(chunks[chunkStart:chunkEnd])
					sum, err := iteratorRangeSum(it, start, end)
					if err != nil {
						sl.err = err
						return sl
					}
					n.cumulativeValues = append(n.cumulativeValues, &ProfileTreeValueNode{
						Value:    sum,
						Label:    ms.s.labels[key],
						NumLabel: ms.s.numLabels[key],
						NumUnit:  ms.s.numUnits[key],
					})
				}
			}

			cur := stack.Peek()
			cur.node.Children = append(cur.node.Children, n)

			stack.Push(&ProfileTreeStackEntry{
				node: n,
			})
			treeIt.StepInto()
			continue
		}
		treeIt.StepUp()
		stack.Pop()
	}
	return sl
}

func iteratorRangeMax(it MemSeriesValuesIterator, start, end uint64) (int64, error) {
	max := int64(0)
	i := uint64(0)
	for it.Next() {
		if i >= end {
			break
		}
		cur := it.At()
		if i >= start && cur > max {
			max = cur
		}
		i++
	}
	return max, it.Err()
}

func iteratorRangeSum(it MemSeriesValuesIterator, start, end uint64) (int64, error) {
	sum := int64(0)
	i := uint64(0)
	for it.Next() {
		if i >= end {
			break
		}
		if i >= start {
			sum += it.At()
		}
		i++
	}
	return sum, it.Err()
}
