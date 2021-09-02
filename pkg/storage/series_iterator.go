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

import "github.com/parca-dev/parca/pkg/storage/chunkenc"

// MemSeriesValuesIterator is an abstraction on iterator over values from possible multiple chunks.
// It most likely is an abstraction like the MultiChunksIterator over []chunkenc.Chunk.
type MemSeriesValuesIterator interface {
	// Next iterates to the next value and returns true if there's more.
	Next() bool
	// At returns the current value.
	At() int64
	// Err returns the underlying errors. Next will return false when encountering errors.
	Err() error
}

type MemSeriesIterator struct {
	tree               *MemSeriesIteratorTree
	timestampsIterator MemSeriesValuesIterator
	durationsIterator  MemSeriesValuesIterator
	periodsIterator    MemSeriesValuesIterator

	series     *MemSeries
	numSamples uint16
}

func (s *MemSeries) Iterator() ProfileSeriesIterator {
	root := &MemSeriesIteratorTreeNode{}

	// TODO: this might be still wrong in case there are multiple roots with different labels?
	// We might be never reading roots with labels...
	rootKey := ProfileTreeValueNodeKey{location: "0"}
	s.mu.RLock()
	root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
		Values:   NewMultiChunkIterator(s.cumulativeValues[rootKey]),
		Label:    s.labels[rootKey],
		NumLabel: s.numLabels[rootKey],
		NumUnit:  s.numUnits[rootKey],
	})

	timestamps := make([]chunkenc.Chunk, 0, len(s.timestamps))
	for _, t := range s.timestamps {
		timestamps = append(timestamps, t.chunk)
	}
	s.mu.RUnlock()

	res := &MemSeriesIterator{
		tree: &MemSeriesIteratorTree{
			Roots: root,
		},
		timestampsIterator: NewMultiChunkIterator(timestamps),
		durationsIterator:  NewMultiChunkIterator(s.durations),
		periodsIterator:    NewMultiChunkIterator(s.periods),
		series:             s,
		numSamples:         s.numSamples,
	}

	memItStack := MemSeriesIteratorTreeStack{{
		node:  root,
		child: 0,
	}}

	it := s.seriesTree.Iterator()

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()

			n := &MemSeriesIteratorTreeNode{
				locationID: child.LocationID,
				Children:   make([]*MemSeriesIteratorTreeNode, 0, len(child.Children)),
			}

			s.mu.RLock()
			for _, key := range child.keys {
				if chunks, ok := s.flatValues[key]; ok {
					n.flatValues = append(n.flatValues, &MemSeriesIteratorTreeValueNode{
						Values:   NewMultiChunkIterator(chunks),
						Label:    s.labels[key],
						NumLabel: s.numLabels[key],
						NumUnit:  s.numUnits[key],
					})
				}
				if chunks, ok := s.cumulativeValues[key]; ok {
					n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
						Values:   NewMultiChunkIterator(chunks),
						Label:    s.labels[key],
						NumLabel: s.numLabels[key],
						NumUnit:  s.numUnits[key],
					})
				}
			}
			s.mu.RUnlock()

			cur := memItStack.Peek()
			cur.node.Children = append(cur.node.Children, n)

			memItStack.Push(&MemSeriesIteratorTreeStackEntry{
				node:  n,
				child: 0,
			})
			it.StepInto()
			continue
		}
		it.StepUp()
		memItStack.Pop()
	}

	return res
}

func (it *MemSeriesIterator) Next() bool {
	it.series.mu.RLock()
	defer it.series.mu.RUnlock()

	if it.numSamples == 0 {
		return false
	}

	if !it.timestampsIterator.Next() {
		return false
	}

	if !it.durationsIterator.Next() {
		return false
	}

	if !it.periodsIterator.Next() {
		return false
	}

	iit := NewMemSeriesIteratorTreeIterator(it.tree)
	for iit.HasMore() {
		if iit.NextChild() {
			child := iit.at()

			for _, v := range child.flatValues {
				v.Values.Next()
			}

			for _, v := range child.cumulativeValues {
				v.Values.Next()
			}

			iit.StepInto()
			continue
		}
		iit.StepUp()
	}

	it.numSamples--
	return true
}

type MemSeriesIteratorTree struct {
	Roots *MemSeriesIteratorTreeNode
}

type MemSeriesIteratorTreeNode struct {
	locationID       uint64
	flatValues       []*MemSeriesIteratorTreeValueNode
	cumulativeValues []*MemSeriesIteratorTreeValueNode
	Children         []*MemSeriesIteratorTreeNode
}

type MemSeriesIteratorTreeValueNode struct {
	Values   MemSeriesValuesIterator
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

func (n *MemSeriesIteratorTreeNode) LocationID() uint64 {
	return n.locationID
}

func (n *MemSeriesIteratorTreeNode) CumulativeValue() int64 {
	res := int64(0)
	for _, v := range n.cumulativeValues {
		if v.Values != nil {
			res += v.Values.At()
		}
	}
	return res
}

func (n *MemSeriesIteratorTreeNode) CumulativeDiffValue() int64 { return 0 }

func (n *MemSeriesIteratorTreeNode) CumulativeDiffValues() []*ProfileTreeValueNode { return nil }

func (n *MemSeriesIteratorTreeNode) CumulativeValues() []*ProfileTreeValueNode {
	if len(n.cumulativeValues) == 0 { // For consistency with other iterators
		return nil
	}

	res := make([]*ProfileTreeValueNode, 0, len(n.cumulativeValues))
	for _, v := range n.cumulativeValues {
		res = append(res, &ProfileTreeValueNode{
			Value:    v.Values.At(),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	return res
}

func (n *MemSeriesIteratorTreeNode) FlatDiffValues() []*ProfileTreeValueNode { return nil }

func (n *MemSeriesIteratorTreeNode) FlatValues() []*ProfileTreeValueNode {
	if len(n.flatValues) == 0 { // For consistency with other iterators
		return nil
	}

	res := make([]*ProfileTreeValueNode, 0, len(n.flatValues))
	for _, v := range n.flatValues {
		res = append(res, &ProfileTreeValueNode{
			Value:    v.Values.At(),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	return res
}

func getIndexRange(it MemSeriesValuesIterator, mint, maxt int64) (uint64, uint64, error) {
	// figure out the index of the first sample > mint and the last sample < maxt
	start := uint64(0)
	end := uint64(0)
	for it.Next() {
		t := it.At()
		if t < mint {
			start++
		}
		if t <= maxt {
			end++
		} else {
			break
		}
	}

	return start, end, it.Err()
}

type MemSeriesInstantProfile struct {
	itt *MemSeriesIteratorTree
	it  *MemSeriesIterator
}

type MemSeriesInstantProfileTree struct {
	itt *MemSeriesIteratorTree
}

func (t *MemSeriesInstantProfileTree) Iterator() InstantProfileTreeIterator {
	return NewMemSeriesIteratorTreeIterator(t.itt)
}

func (p *MemSeriesInstantProfile) ProfileTree() InstantProfileTree {
	return &MemSeriesInstantProfileTree{
		itt: p.itt,
	}
}

func (p *MemSeriesInstantProfile) ProfileMeta() InstantProfileMeta {
	return InstantProfileMeta{
		PeriodType: p.it.series.periodType,
		SampleType: p.it.series.sampleType,
		Timestamp:  p.it.timestampsIterator.At(),
		Duration:   p.it.durationsIterator.At(),
		Period:     p.it.periodsIterator.At(),
	}
}

func (it *MemSeriesIterator) At() InstantProfile {
	return &MemSeriesInstantProfile{
		itt: it.tree,
		it:  it,
	}
}

func (it *MemSeriesIterator) Err() error {
	return nil
}
