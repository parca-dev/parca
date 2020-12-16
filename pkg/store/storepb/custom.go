// Copyright 2020 The conprof Authors
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

// The content of this file is originally copied from
// github.com/thanos-io/thanos/pkg/store/storepb/custom.go.

package storepb

import (
	"bytes"

	"github.com/conprof/db/storage"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
)

func NewWarnSeriesResponse(err error) *SeriesResponse {
	return &SeriesResponse{
		Result: &SeriesResponse_Warning{
			Warning: err.Error(),
		},
	}
}

func NewSeriesResponse(series *RawProfileSeries) *SeriesResponse {
	return &SeriesResponse{
		Result: &SeriesResponse_Series{
			Series: series,
		},
	}
}

type emptySeriesSet struct{}

func (emptySeriesSet) Next() bool                       { return false }
func (emptySeriesSet) At() (labels.Labels, []AggrChunk) { return nil, nil }
func (emptySeriesSet) Err() error                       { return nil }

// EmptySeriesSet returns a new series set that contains no series.
func EmptySeriesSet() SeriesSet {
	return emptySeriesSet{}
}

// MergeSeriesSets takes all series sets and returns as a union single series set.
// It assumes series are sorted by labels within single SeriesSet, similar to remote read guarantees.
// However, they can be partial: in such case, if the single SeriesSet returns the same series within many iterations,
// MergeSeriesSets will merge those into one.
//
// It also assumes in a "best effort" way that chunks are sorted by min time. It's done as an optimization only, so if input
// series' chunks are NOT sorted, the only consequence is that the duplicates might be not correctly removed. This is double checked
// which on just-before PromQL level as well, so the only consequence is increased network bandwidth.
// If all chunks were sorted, MergeSeriesSet ALSO returns sorted chunks by min time.
//
// Chunks within the same series can also overlap (within all SeriesSet
// as well as single SeriesSet alone). If the chunk ranges overlap, the *exact* chunk duplicates will be removed
// (except one), and any other overlaps will be appended into on chunks slice.
func MergeSeriesSets(all ...SeriesSet) SeriesSet {
	switch len(all) {
	case 0:
		return emptySeriesSet{}
	case 1:
		return newUniqueSeriesSet(all[0])
	}
	h := len(all) / 2

	return newMergedSeriesSet(
		MergeSeriesSets(all[:h]...),
		MergeSeriesSets(all[h:]...),
	)
}

// SeriesSet is a set of series and their corresponding chunks.
// The set is sorted by the label sets. Chunks may be overlapping or expected of order.
type SeriesSet interface {
	Next() bool
	At() (labels.Labels, []AggrChunk)
	Err() error
}

// mergedSeriesSet takes two series sets as a single series set.
type mergedSeriesSet struct {
	a, b SeriesSet

	lset         labels.Labels
	chunks       []AggrChunk
	adone, bdone bool
}

func newMergedSeriesSet(a, b SeriesSet) *mergedSeriesSet {
	s := &mergedSeriesSet{a: a, b: b}
	// Initialize first elements of both sets as Next() needs
	// one element look-ahead.
	s.adone = !s.a.Next()
	s.bdone = !s.b.Next()

	return s
}

func (s *mergedSeriesSet) At() (labels.Labels, []AggrChunk) {
	return s.lset, s.chunks
}

func (s *mergedSeriesSet) Err() error {
	if s.a.Err() != nil {
		return s.a.Err()
	}
	return s.b.Err()
}

func (s *mergedSeriesSet) compare() int {
	if s.adone {
		return 1
	}
	if s.bdone {
		return -1
	}
	lsetA, _ := s.a.At()
	lsetB, _ := s.b.At()
	return labels.Compare(lsetA, lsetB)
}

func (s *mergedSeriesSet) Next() bool {
	if s.adone && s.bdone || s.Err() != nil {
		return false
	}

	d := s.compare()
	if d > 0 {
		s.lset, s.chunks = s.b.At()
		s.bdone = !s.b.Next()
		return true
	}
	if d < 0 {
		s.lset, s.chunks = s.a.At()
		s.adone = !s.a.Next()
		return true
	}

	// Both a and b contains the same series. Go through all chunks, remove duplicates and concatenate chunks from both
	// series sets. We best effortly assume chunks are sorted by min time. If not, we will not detect all deduplicate which will
	// be account on select layer anyway. We do it still for early optimization.
	lset, chksA := s.a.At()
	_, chksB := s.b.At()
	s.lset = lset

	// Slice reuse is not generally safe with nested merge iterators.
	// We err on the safe side an create a new slice.
	s.chunks = make([]AggrChunk, 0, len(chksA)+len(chksB))

	b := 0
Outer:
	for a := range chksA {
		for {
			if b >= len(chksB) {
				// No more b chunks.
				s.chunks = append(s.chunks, chksA[a:]...)
				break Outer
			}

			cmp := chksA[a].Compare(chksB[b])
			if cmp > 0 {
				s.chunks = append(s.chunks, chksA[a])
				break
			}
			if cmp < 0 {
				s.chunks = append(s.chunks, chksB[b])
				b++
				continue
			}

			// Exact duplicated chunks, discard one from b.
			b++
		}
	}

	if b < len(chksB) {
		s.chunks = append(s.chunks, chksB[b:]...)
	}

	s.adone = !s.a.Next()
	s.bdone = !s.b.Next()
	return true
}

// uniqueSeriesSet takes one series set and ensures each iteration contains single, full series.
type uniqueSeriesSet struct {
	SeriesSet
	done bool

	peek *RawProfileSeries

	lset   labels.Labels
	chunks []AggrChunk
}

func newUniqueSeriesSet(wrapped SeriesSet) *uniqueSeriesSet {
	return &uniqueSeriesSet{SeriesSet: wrapped}
}

func (s *uniqueSeriesSet) At() (labels.Labels, []AggrChunk) {
	return s.lset, s.chunks
}

func (s *uniqueSeriesSet) Next() bool {
	if s.Err() != nil {
		return false
	}

	for !s.done {
		if s.done = !s.SeriesSet.Next(); s.done {
			break
		}
		lset, chks := s.SeriesSet.At()
		if s.peek == nil {
			s.peek = &RawProfileSeries{Labels: labelpb.LabelsFromPromLabels(lset), Chunks: chks}
			continue
		}

		if labels.Compare(lset, s.peek.PromLabels()) != 0 {
			s.lset, s.chunks = s.peek.PromLabels(), s.peek.Chunks
			s.peek = &RawProfileSeries{Labels: labelpb.LabelsFromPromLabels(lset), Chunks: chks}
			return true
		}

		// We assume non-overlapping, sorted chunks. This is best effort only, if it's otherwise it
		// will just be duplicated, but well handled by StoreAPI consumers.
		s.peek.Chunks = append(s.peek.Chunks, chks...)
	}

	if s.peek == nil {
		return false
	}

	s.lset, s.chunks = s.peek.PromLabels(), s.peek.Chunks
	s.peek = nil
	return true
}

// Compare returns positive 1 if chunk is smaller -1 if larger than b by min time, then max time.
// It returns 0 if chunks are exactly the same.
func (m AggrChunk) Compare(b AggrChunk) int {
	if m.MinTime < b.MinTime {
		return 1
	}
	if m.MinTime > b.MinTime {
		return -1
	}

	// Same min time.
	if m.MaxTime < b.MaxTime {
		return 1
	}
	if m.MaxTime > b.MaxTime {
		return -1
	}

	// We could use proto.Equal, but we need ordering as well.
	for _, cmp := range []func() int{
		func() int { return m.Raw.Compare(b.Raw) },
	} {
		if c := cmp(); c == 0 {
			continue
		} else {
			return c
		}
	}
	return 0
}

// Compare returns positive 1 if chunk is smaller -1 if larger.
// It returns 0 if chunks are exactly the same.
func (m *Chunk) Compare(b *Chunk) int {
	if m == nil && b == nil {
		return 0
	}
	if b == nil {
		return 1
	}
	if m == nil {
		return -1
	}

	if m.Type < b.Type {
		return 1
	}
	if m.Type > b.Type {
		return -1
	}
	return bytes.Compare(m.Data, b.Data)
}

// PromLabels return Prometheus labels.Labels without extra allocation.
func (m *RawProfileSeries) PromLabels() labels.Labels {
	return labelpb.LabelsToPromLabels(m.Labels)
}

func TsdbSelectHints(h *SelectHints) *storage.SelectHints {
	if h == nil {
		return nil
	}
	return &storage.SelectHints{
		Start: h.Start,
		End:   h.End,
		Func:  h.Func,
	}
}

func PbSelectHints(h *storage.SelectHints) *SelectHints {
	if h == nil {
		return nil
	}
	return &SelectHints{
		Start: h.Start,
		End:   h.End,
		Func:  h.Func,
	}
}
