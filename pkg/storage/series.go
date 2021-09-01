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
	"math"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
)

var (
	ErrOutOfOrderSample = errors.New("out of order sample")
)

type MemSeriesTreeNode struct {
	keys []ProfileTreeValueNodeKey

	LocationID uint64
	Children   []*MemSeriesTreeNode
}

func (n *MemSeriesTreeNode) addKey(key ProfileTreeValueNodeKey) {
	for _, k := range n.keys {
		if k.Equals(key) {
			return
		}
	}
	n.keys = append(n.keys, key)
}

type MemSeriesTree struct {
	s     *MemSeries
	Roots *MemSeriesTreeNode
}

func (t *MemSeriesTree) Iterator() *MemSeriesTreeIterator {
	return NewMemSeriesTreeIterator(t)
}

func (t *MemSeriesTree) Insert(index uint16, profileTree *ProfileTree) error {
	if t.Roots == nil {
		t.Roots = &MemSeriesTreeNode{}
	}

	pit := profileTree.Iterator()
	sit := t.Iterator()

	for pit.HasMore() {
		if pit.NextChild() {
			profileTreeChild := pit.At()
			pId := profileTreeChild.LocationID()

			done := false
			for {
				if !sit.NextChild() {
					node := sit.Node()
					seriesTreeChild := &MemSeriesTreeNode{
						LocationID: pId,
					}

					for _, n := range profileTreeChild.FlatValues() {
						if n.key == nil {
							n.Key(profileTreeChild.LocationID())
						}

						t.s.mu.Lock()
						if t.s.flatValues[*n.key] == nil {
							// Create the needed amount of chunks based on how many timestamp chunks there are.
							t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
							for i := 0; i < len(t.s.timestamps); i++ {
								t.s.flatValues[*n.key][i] = chunkenc.NewXORChunk()
							}
						}
						app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
						if err != nil {
							t.s.mu.Unlock()
							return fmt.Errorf("failed to open flat appender: %w", err)
						}
						app.AppendAt(index, n.Value)
						t.s.mu.Unlock()

						// We need to keep track of the node keys.
						seriesTreeChild.addKey(*n.key)

						if len(n.Label) > 0 {
							t.s.mu.Lock()
							if t.s.labels[*n.key] == nil {
								t.s.labels[*n.key] = n.Label
							}

							if t.s.numLabels[*n.key] == nil {
								t.s.numLabels[*n.key] = n.NumLabel
							}

							if t.s.numUnits[*n.key] == nil {
								t.s.numUnits[*n.key] = n.NumUnit
							}
							t.s.mu.Unlock()
						}
					}

					for _, n := range profileTreeChild.CumulativeValues() {
						if n.key == nil {
							n.Key(profileTreeChild.LocationID())
						}

						t.s.mu.Lock()
						if t.s.cumulativeValues[*n.key] == nil {
							// Create the needed amount of chunks based on how many timestamp chunks there are.
							t.s.cumulativeValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
							for i := 0; i < len(t.s.timestamps); i++ {
								t.s.cumulativeValues[*n.key][i] = chunkenc.NewXORChunk()
							}
						}
						app, err := t.s.cumulativeValues[*n.key][len(t.s.cumulativeValues[*n.key])-1].Appender()
						if err != nil {
							t.s.mu.Unlock()
							return fmt.Errorf("failed to open cumulative appender: %w", err)
						}
						app.AppendAt(index, n.Value)
						t.s.mu.Unlock()

						// We need to keep track of the node keys.
						seriesTreeChild.addKey(*n.key)
					}

					node.Children = append(node.Children, seriesTreeChild)

					pit.StepInto()
					sit.StepInto()
					done = true
					break
				}
				sId := sit.At().LocationID
				if pId == sId || pId < sId {
					break
				}
			}
			if done {
				continue
			}

			seriesTreeChild := sit.At()
			sId := seriesTreeChild.LocationID

			// The node with the location id in the profile-tree is the same (except Location ID 0 - the root),
			// this means this node present in the series-tree, so we need add the new values to the existing node.
			if pId == sId {
				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					// Even if the location exists.
					// labels can be different and then the key is different, so we need check.
					t.s.mu.Lock()
					if t.s.flatValues[*n.key] == nil {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.flatValues[*n.key][i] = chunkenc.NewXORChunk()
						}
					}
					app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node IDs.
					seriesTreeChild.addKey(*n.key)
				}

				for _, n := range profileTreeChild.CumulativeValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if len(t.s.cumulativeValues[*n.key]) == 0 {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.cumulativeValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.cumulativeValues[*n.key][i] = chunkenc.NewXORChunk()
						}
					}
					app, err := t.s.cumulativeValues[*n.key][len(t.s.cumulativeValues[*n.key])-1].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open cumulative appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					seriesTreeChild.addKey(*n.key)
				}

				pit.StepInto()
				sit.StepInto()
				continue
			}

			// The node with the location id in the profile-tree is smaller,
			// this means this node is not present yet in the series-tree, so it has to be added at the current child position.
			if pId < sId {
				node := sit.Node()
				childIndex := sit.ChildIndex()
				newChildren := make([]*MemSeriesTreeNode, len(node.Children)+1)
				copy(newChildren, node.Children[:childIndex])
				newChild := &MemSeriesTreeNode{
					LocationID: pId,
				}

				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if t.s.flatValues[*n.key] == nil {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.flatValues[*n.key][i] = chunkenc.NewXORChunk()
						}
					}
					app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					newChild.addKey(*n.key)
				}

				for _, n := range profileTreeChild.CumulativeValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if t.s.cumulativeValues[*n.key] == nil {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.cumulativeValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.cumulativeValues[*n.key][i] = chunkenc.NewXORChunk()
						}
					}
					app, err := t.s.cumulativeValues[*n.key][len(t.s.cumulativeValues[*n.key])-1].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open cumulative appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					newChild.addKey(*n.key)
				}

				newChildren[childIndex] = newChild
				copy(newChildren[childIndex+1:], node.Children[childIndex:])
				node.Children = newChildren

				pit.StepInto()
				sit.StepInto()
				continue
			}
		}
		pit.StepUp()
		sit.StepUp()
	}

	return nil
}

type ProfileTreeNode struct {
	locationID           uint64
	flatValues           []*ProfileTreeValueNode
	flatDiffValues       []*ProfileTreeValueNode
	cumulativeValues     []*ProfileTreeValueNode
	cumulativeDiffValues []*ProfileTreeValueNode
	Children             []*ProfileTreeNode
}

func (n *ProfileTreeNode) LocationID() uint64 {
	return n.locationID
}

func (n *ProfileTreeNode) CumulativeValue() int64 {
	res := int64(0)
	for _, cv := range n.cumulativeValues {
		res += cv.Value
	}

	return res
}

func (n *ProfileTreeNode) CumulativeDiffValue() int64 {
	res := int64(0)
	for _, cv := range n.cumulativeDiffValues {
		res += cv.Value
	}

	return res
}

func (n *ProfileTreeNode) CumulativeDiffValues() []*ProfileTreeValueNode {
	return n.cumulativeDiffValues
}

func (n *ProfileTreeNode) FlatDiffValues() []*ProfileTreeValueNode {
	return n.flatDiffValues
}

func (n *ProfileTreeNode) CumulativeValues() []*ProfileTreeValueNode {
	return n.cumulativeValues
}

func (n *ProfileTreeNode) FlatValues() []*ProfileTreeValueNode {
	return n.flatValues
}

type ProfileTree struct {
	Roots *ProfileTreeNode
}

func NewProfileTree() *ProfileTree {
	return &ProfileTree{
		Roots: &ProfileTreeNode{
			cumulativeValues: []*ProfileTreeValueNode{{
				Value: 0,
			}},
		},
	}
}

func (t *ProfileTree) Iterator() InstantProfileTreeIterator {
	return NewProfileTreeIterator(t)
}

func (t *ProfileTree) Insert(sample *profile.Sample) {
	cur := t.Roots
	locations := sample.Location

	locationIDs := make([]uint64, 0, len(sample.Location)+1)
	for _, l := range sample.Location {
		locationIDs = append(locationIDs, l.ID)
	}
	locationIDs = append(locationIDs, 0) // add the root

	for i := len(locations) - 1; i >= 0; i-- {
		nextId := locations[i].ID

		var child *ProfileTreeNode

		// Binary search for child in list. If it exists continue to use the existing one.
		index := sort.Search(len(cur.Children), func(i int) bool { return cur.Children[i].LocationID() >= nextId })
		if index < len(cur.Children) && cur.Children[index].LocationID() == nextId {
			// Child with this ID already exists.
			child = cur.Children[index]
		} else {
			// No child with ID exists, but it should be inserted at `index`.
			newChildren := make([]*ProfileTreeNode, len(cur.Children)+1)
			copy(newChildren, cur.Children[:index])
			child = &ProfileTreeNode{
				locationID: nextId,
			}
			newChildren[index] = child
			copy(newChildren[index+1:], cur.Children[index:])
			cur.Children = newChildren
		}

		// Nodes that might only have cumulativeValues
		if cur.cumulativeValues == nil {
			cur.cumulativeValues = []*ProfileTreeValueNode{{}}
		}
		cur.cumulativeValues[0].Value += sample.Value[0]

		for _, cv := range cur.cumulativeValues {
			// Populate the keys with the current subset of locations.
			// i+1 because we additionally have the root in locationIDs.
			cv.Key(locationIDs[i+1:]...)
		}

		cur = child
	}

	if cur.cumulativeValues == nil {
		cur.cumulativeValues = []*ProfileTreeValueNode{{}}
	}

	cur.cumulativeValues[0].Value += sample.Value[0]
	// TODO: We probably need to merge labels, numLabels and numUnits
	cur.cumulativeValues[0].Label = sample.Label
	cur.cumulativeValues[0].NumLabel = sample.NumLabel
	cur.cumulativeValues[0].NumUnit = sample.NumUnit

	for _, cv := range cur.cumulativeValues {
		cv.Key(locationIDs...) // populate the keys
	}

	if cur.flatValues == nil {
		cur.flatValues = []*ProfileTreeValueNode{{}}
	}
	cur.flatValues[0].Value += sample.Value[0]
	// TODO: We probably need to merge labels, numLabels and numUnits
	cur.flatValues[0].Label = sample.Label
	cur.flatValues[0].NumLabel = sample.NumLabel
	cur.flatValues[0].NumUnit = sample.NumUnit

	for _, fv := range cur.flatValues {
		fv.Key(locationIDs...) //populate the keys
	}
}

type MemSeries struct {
	lset labels.Labels
	id   uint64

	periodType ValueType
	sampleType ValueType

	minTime, maxTime int64
	timestamps       timestampChunks
	durations        []chunkenc.Chunk
	periods          []chunkenc.Chunk

	// TODO: Might be worth combining behind some struct?
	// Or maybe not because it's easier to serialize?

	// mu locks the following maps for concurrent access.
	mu sync.RWMutex
	// Flat and cumulative values as well as labels by the node's ProfileTreeValueNodeKey.
	flatValues       map[ProfileTreeValueNodeKey][]chunkenc.Chunk
	cumulativeValues map[ProfileTreeValueNodeKey][]chunkenc.Chunk
	labels           map[ProfileTreeValueNodeKey]map[string][]string
	numLabels        map[ProfileTreeValueNodeKey]map[string][]int64
	numUnits         map[ProfileTreeValueNodeKey]map[string][]string

	seriesTree *MemSeriesTree
	numSamples uint16

	samplesAppended prometheus.Counter
}

func NewMemSeries(lset labels.Labels, id uint64) *MemSeries {
	s := &MemSeries{
		lset: lset,
		id:   id,
		timestamps: timestampChunks{{
			minTime: math.MaxInt64,
			maxTime: math.MinInt64,
			chunk:   chunkenc.NewDeltaChunk(),
		}},
		durations: []chunkenc.Chunk{chunkenc.NewRLEChunk()},
		periods:   []chunkenc.Chunk{chunkenc.NewRLEChunk()},

		flatValues:       make(map[ProfileTreeValueNodeKey][]chunkenc.Chunk),
		cumulativeValues: make(map[ProfileTreeValueNodeKey][]chunkenc.Chunk),
		labels:           make(map[ProfileTreeValueNodeKey]map[string][]string),
		numLabels:        make(map[ProfileTreeValueNodeKey]map[string][]int64),
		numUnits:         make(map[ProfileTreeValueNodeKey]map[string][]string),
	}
	s.seriesTree = &MemSeriesTree{s: s}

	return s
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
}

type mapInfo struct {
	m      *profile.Mapping
	offset int64
}

func (s *MemSeries) Appender() (*MemSeriesAppender, error) {
	timestamps, err := s.timestamps[len(s.timestamps)-1].chunk.Appender()
	if err != nil {
		return nil, err
	}
	durations, err := s.durations[len(s.timestamps)-1].Appender()
	if err != nil {
		return nil, err
	}
	periods, err := s.periods[len(s.timestamps)-1].Appender()
	if err != nil {
		return nil, err
	}

	return &MemSeriesAppender{
		s:          s,
		timestamps: timestamps,
		duration:   durations,
		periods:    periods,
	}, nil
}

type MemSeriesAppender struct {
	s          *MemSeries
	timestamps chunkenc.Appender
	duration   chunkenc.Appender
	periods    chunkenc.Appender
}

const samplesPerChunk = 120

func (a *MemSeriesAppender) Append(p *Profile) error {
	if a.s.numSamples == 0 {
		a.s.periodType = p.Meta.PeriodType
		a.s.sampleType = p.Meta.SampleType
	}

	if !equalValueType(a.s.periodType, p.Meta.PeriodType) {
		return ErrPeriodTypeMismatch
	}

	if !equalValueType(a.s.sampleType, p.Meta.SampleType) {
		return ErrSampleTypeMismatch
	}

	timestamp := p.Meta.Timestamp

	if timestamp <= a.s.maxTime {
		return ErrOutOfOrderSample
	}

	newChunks := false
	a.s.mu.Lock()
	if a.s.timestamps[len(a.s.timestamps)-1].chunk.NumSamples() >= samplesPerChunk {
		newChunks = true
	}
	a.s.mu.Unlock()

	if newChunks {
		a.s.mu.Lock()

		// If we need new chunks then range over all existing cumulativeValues and flatValues
		// appending new chunks.

		// TODO: We need to somehow add non-existing chunks to the proper index...

		for k := range a.s.cumulativeValues {
			a.s.cumulativeValues[k] = append(a.s.cumulativeValues[k], chunkenc.NewXORChunk())
		}
		for k := range a.s.flatValues {
			a.s.flatValues[k] = append(a.s.flatValues[k], chunkenc.NewXORChunk())
		}

		a.s.timestamps = append(a.s.timestamps, timestampChunk{
			maxTime: timestamp,
			minTime: timestamp,
			chunk:   chunkenc.NewDeltaChunk(),
		})
		app, err := a.s.timestamps[len(a.s.timestamps)-1].chunk.Appender()
		if err != nil {
			a.s.mu.Unlock()
			return fmt.Errorf("failed to add the next timestamp chunk: %w", err)
		}
		a.timestamps = app

		a.s.durations = append(a.s.durations, chunkenc.NewRLEChunk())
		app, err = a.s.durations[len(a.s.durations)-1].Appender()
		if err != nil {
			a.s.mu.Unlock()
			return fmt.Errorf("failed to add the next durations chunk: %w", err)
		}
		a.duration = app

		a.s.periods = append(a.s.periods, chunkenc.NewRLEChunk())
		app, err = a.s.periods[len(a.s.periods)-1].Appender()
		if err != nil {
			a.s.mu.Unlock()
			return fmt.Errorf("failed to add the next periods chunk: %w", err)
		}
		a.periods = app
		a.s.mu.Unlock()
	}

	// appendTree locks the maps itself.
	if err := a.s.appendTree(p.Tree); err != nil {
		return err
	}

	a.timestamps.AppendAt(a.s.numSamples%samplesPerChunk, timestamp)
	a.duration.AppendAt(a.s.numSamples%samplesPerChunk, p.Meta.Duration)
	a.periods.AppendAt(a.s.numSamples%samplesPerChunk, p.Meta.Period)

	a.s.mu.Lock()
	if a.s.timestamps[len(a.s.timestamps)-1].minTime > timestamp {
		a.s.timestamps[len(a.s.timestamps)-1].minTime = timestamp
	}
	if a.s.timestamps[len(a.s.timestamps)-1].maxTime < timestamp {
		a.s.timestamps[len(a.s.timestamps)-1].maxTime = timestamp
	}
	a.s.mu.Unlock()

	// Set the timestamp as minTime if timestamp != 0
	if a.s.minTime == 0 && timestamp != 0 {
		a.s.minTime = timestamp
	}

	a.s.maxTime = timestamp
	a.s.numSamples++

	if a.s.samplesAppended != nil {
		a.s.samplesAppended.Inc()
	}
	return nil
}

func (s *MemSeries) appendTree(profileTree *ProfileTree) error {
	if s.seriesTree == nil {
		s.seriesTree = &MemSeriesTree{s: s}
	}

	return s.seriesTree.Insert(s.numSamples%samplesPerChunk, profileTree)
}

func (s *MemSeries) Labels() labels.Labels {
	return s.lset
}

type MemSeriesStats struct {
	samples     uint16
	Cumulatives []MemSeriesValueStats
	Flat        []MemSeriesValueStats
}

type MemSeriesValueStats struct {
	samples int
	bytes   int
}

func (s *MemSeries) stats() MemSeriesStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flat := make([]MemSeriesValueStats, 0, len(s.flatValues))
	cumulative := make([]MemSeriesValueStats, 0, len(s.cumulativeValues))

	for _, chunks := range s.flatValues {
		for _, c := range chunks {
			flat = append(flat, MemSeriesValueStats{
				samples: c.NumSamples(),
				bytes:   len(c.Bytes()),
			})
		}
	}

	for _, chunks := range s.cumulativeValues {
		for _, c := range chunks {
			cumulative = append(cumulative, MemSeriesValueStats{
				samples: c.NumSamples(),
				bytes:   len(c.Bytes()),
			})
		}
	}

	return MemSeriesStats{
		samples:     s.numSamples,
		Cumulatives: cumulative,
		Flat:        flat,
	}
}

type MemSeriesIteratorTree struct {
	Roots *MemSeriesIteratorTreeNode
}

type MemSeriesIteratorTreeValueNode struct {
	Values   chunkenc.Iterator
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type MemSeriesIteratorTreeNode struct {
	locationID       uint64
	flatValues       []*MemSeriesIteratorTreeValueNode
	cumulativeValues []*MemSeriesIteratorTreeValueNode
	Children         []*MemSeriesIteratorTreeNode
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

func (n *MemSeriesIteratorTreeNode) CumulativeDiffValue() int64                    { return 0 }
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
	chunkStart, chunkEnd := ms.s.timestamps.indexRange(ms.mint, ms.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range ms.s.timestamps[chunkStart:chunkEnd] {
		timestamps = append(timestamps, t.chunk)
	}
	ms.s.mu.RUnlock()

	sl := &SliceProfileSeriesIterator{i: -1}

	start, end, err := getIndexRange(NewMultiChunkIterator(timestamps), ms.mint, ms.maxt)
	if err != nil {
		sl.err = err
		return sl
	}

	it := NewMultiChunkIterator(timestamps)
	it.Seek(uint16(start))
	it.Next()
	minTimestamp := it.At()

	ms.s.mu.RLock()
	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.durations[chunkStart:chunkEnd])
	ms.s.mu.RUnlock()

	duration, err := iteratorRangeSum(it, start, end)
	if err != nil {
		sl.err = err
		return sl
	}

	ms.s.mu.RLock()
	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.periods[chunkStart:chunkEnd])
	ms.s.mu.RUnlock()

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

	ms.s.mu.RLock()
	// reuse NewMultiChunkIterator with new chunks.
	it.Reset(ms.s.cumulativeValues[rootKey][chunkStart:chunkEnd])
	ms.s.mu.RUnlock()

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

			ms.s.mu.Lock()
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
			ms.s.mu.Unlock()

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

func getIndexRange(it chunkenc.Iterator, mint, maxt int64) (uint64, uint64, error) {
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

func iteratorRangeMax(it chunkenc.Iterator, start, end uint64) (int64, error) {
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

func iteratorRangeSum(it chunkenc.Iterator, start, end uint64) (int64, error) {
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
	chunkStart, chunkEnd := rs.s.timestamps.indexRange(rs.mint, rs.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range rs.s.timestamps[chunkStart:chunkEnd] {
		timestamps = append(timestamps, t.chunk)
	}
	rs.s.mu.RUnlock()

	start, end, err := getIndexRange(NewMultiChunkIterator(timestamps), rs.mint, rs.maxt)
	if err != nil {
		// TODO
	}

	root := &MemSeriesIteratorTreeNode{}
	rootKey := ProfileTreeValueNodeKey{location: "0"}

	rs.s.mu.Lock()
	rootIt := NewMultiChunkIterator(rs.s.cumulativeValues[rootKey][chunkStart:chunkEnd])
	if start != 0 {
		rootIt.Seek(uint16(start))
	}
	root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
		Values:   rootIt,
		Label:    rs.s.labels[rootKey],
		NumLabel: rs.s.numLabels[rootKey],
		NumUnit:  rs.s.numUnits[rootKey],
	})
	rs.s.mu.Unlock()

	memItStack := MemSeriesIteratorTreeStack{{
		node:  root,
		child: 0,
	}}

	it := rs.s.seriesTree.Iterator()

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()

			n := &MemSeriesIteratorTreeNode{
				locationID: child.LocationID,
				Children:   make([]*MemSeriesIteratorTreeNode, 0, len(child.Children)),
			}

			rs.s.mu.RLock()
			for _, key := range child.keys {
				if chunks, ok := rs.s.flatValues[key]; ok {
					it := NewMultiChunkIterator(chunks[chunkStart:chunkEnd])
					if start != 0 {
						it.Seek(uint16(start))
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
						it.Seek(uint16(start)) // We might need another interface with Seek(index uint64) for multi chunks.
					}
					n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
						Values:   it,
						Label:    rs.s.labels[key],
						NumLabel: rs.s.numLabels[key],
						NumUnit:  rs.s.numUnits[key],
					})
				}
			}
			rs.s.mu.RUnlock()

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

	timestampIterator := NewMultiChunkIterator(timestamps)
	durationsIterator := NewMultiChunkIterator(rs.s.durations[chunkStart:chunkEnd])
	periodsIterator := NewMultiChunkIterator(rs.s.periods[chunkStart:chunkEnd])

	if start != 0 {
		timestampIterator.Seek(uint16(start))
		durationsIterator.Seek(uint16(start))
		periodsIterator.Seek(uint16(start))
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
	timestampsIterator chunkenc.Iterator
	durationsIterator  chunkenc.Iterator
	periodsIterator    chunkenc.Iterator

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

type MemSeriesIterator struct {
	tree               *MemSeriesIteratorTree
	timestampsIterator chunkenc.Iterator
	durationsIterator  chunkenc.Iterator
	periodsIterator    chunkenc.Iterator

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

type profileNormalizer struct {
	// Memoization tables within a profile.
	locationsByID map[uint64]*profile.Location
	functionsByID map[uint64]*profile.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	samples   map[stacktraceKey]*profile.Sample
	metaStore ProfileMetaStore
}

// Returns the mapped sample and whether it is new or a known sample.
func (pn *profileNormalizer) mapSample(src *profile.Sample, sampleIndex int) (*profile.Sample, bool) {
	s := &profile.Sample{
		Location: make([]*profile.Location, len(src.Location)),
		Label:    make(map[string][]string, len(src.Label)),
		NumLabel: make(map[string][]int64, len(src.NumLabel)),
		NumUnit:  make(map[string][]string, len(src.NumLabel)),
	}
	for i, l := range src.Location {
		s.Location[i] = pn.mapLocation(l)
	}
	for k, v := range src.Label {
		vv := make([]string, len(v))
		copy(vv, v)
		s.Label[k] = vv
	}
	for k, v := range src.NumLabel {
		u := src.NumUnit[k]
		vv := make([]int64, len(v))
		uu := make([]string, len(u))
		copy(vv, v)
		copy(uu, u)
		s.NumLabel[k] = vv
		s.NumUnit[k] = uu
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping. Add current values to the
	// existing sample.
	k := makeStacktraceKey(s)
	sa, found := pn.samples[k]
	if found {
		sa.Value[0] += src.Value[sampleIndex]
		return sa, false
	}

	s.Value = []int64{src.Value[sampleIndex]}
	pn.samples[k] = s
	return s, true
}

func (pn *profileNormalizer) mapLocation(src *profile.Location) *profile.Location {
	if src == nil {
		return nil
	}

	if l, ok := pn.locationsByID[src.ID]; ok {
		return l
	}

	mi := pn.mapMapping(src.Mapping)
	l := &profile.Location{
		Mapping:  mi.m,
		Address:  uint64(int64(src.Address) + mi.offset),
		Line:     make([]profile.Line, len(src.Line)),
		IsFolded: src.IsFolded,
	}
	for i, ln := range src.Line {
		l.Line[i] = pn.mapLine(ln)
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping ID.
	k := MakeLocationKey(l)
	ll, err := pn.metaStore.GetLocationByKey(k)
	if err != ErrLocationNotFound {
		pn.locationsByID[src.ID] = ll
		return ll
	}
	pn.locationsByID[src.ID] = l
	pn.metaStore.CreateLocation(l)
	return l
}

func (pn *profileNormalizer) mapMapping(src *profile.Mapping) mapInfo {
	if src == nil {
		return mapInfo{}
	}

	if mi, ok := pn.mappingsByID[src.ID]; ok {
		return mi
	}

	// Check memoization tables.
	mk := MakeMappingKey(src)
	m, err := pn.metaStore.GetMappingByKey(mk)
	if err != ErrMappingNotFound {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi
	}
	m = &profile.Mapping{
		Start:           src.Start,
		Limit:           src.Limit,
		Offset:          src.Offset,
		File:            src.File,
		BuildID:         src.BuildID,
		HasFunctions:    src.HasFunctions,
		HasFilenames:    src.HasFilenames,
		HasLineNumbers:  src.HasLineNumbers,
		HasInlineFrames: src.HasInlineFrames,
	}

	// Update memoization tables.
	pn.metaStore.CreateMapping(m)
	mi := mapInfo{m, 0}
	pn.mappingsByID[src.ID] = mi
	return mi
}

func (pn *profileNormalizer) mapLine(src profile.Line) profile.Line {
	ln := profile.Line{
		Function: pn.mapFunction(src.Function),
		Line:     src.Line,
	}
	return ln
}

func (pn *profileNormalizer) mapFunction(src *profile.Function) *profile.Function {
	if src == nil {
		return nil
	}
	if f, ok := pn.functionsByID[src.ID]; ok {
		return f
	}
	k := MakeFunctionKey(src)
	f, err := pn.metaStore.GetFunctionByKey(k)
	if err != ErrFunctionNotFound {
		pn.functionsByID[src.ID] = f
		return f
	}
	f = &profile.Function{
		Name:       src.Name,
		SystemName: src.SystemName,
		Filename:   src.Filename,
		StartLine:  src.StartLine,
	}
	pn.metaStore.CreateFunction(f)
	pn.functionsByID[src.ID] = f
	return f
}

// key generates stacktraceKey to be used as a key for maps.
func makeStacktraceKey(sample *profile.Sample) stacktraceKey {
	ids := make([]string, len(sample.Location))
	for i, l := range sample.Location {
		ids[i] = strconv.FormatUint(l.ID, 16)
	}

	labels := make([]string, 0, len(sample.Label))
	for k, v := range sample.Label {
		labels = append(labels, fmt.Sprintf("%q%q", k, v))
	}
	sort.Strings(labels)

	numlabels := make([]string, 0, len(sample.NumLabel))
	for k, v := range sample.NumLabel {
		numlabels = append(numlabels, fmt.Sprintf("%q%x%x", k, v, sample.NumUnit[k]))
	}
	sort.Strings(numlabels)

	return stacktraceKey{
		locations: strings.Join(ids, "|"),
		labels:    strings.Join(labels, ""),
		numlabels: strings.Join(numlabels, ""),
	}
}

func isZeroSample(s *profile.Sample) bool {
	for _, v := range s.Value {
		if v != 0 {
			return false
		}
	}
	return true
}
