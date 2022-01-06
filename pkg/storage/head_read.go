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
	"math"
	"sort"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/model/labels"
)

var (
	ErrNotFound = errors.New("not found")
)

// IndexReader provides reading access of serialized index data.
type IndexReader interface {
	// Postings returns the postings sroar.Bitmap.
	Postings(name string, values ...string) (*sroar.Bitmap, error)

	// LabelValues returns possible label values which may not be sorted.
	LabelValues(name string, matchers ...*labels.Matcher) ([]string, error)

	// LabelValueFor returns label value for the given label name in the series referred to by ID.
	// If the series couldn't be found or the series doesn't have the requested label a
	// ErrNotFound is returned as error.
	LabelValueFor(id uint64, label string) (string, error)

	// LabelNames returns possible label names which may not be sorted.
	LabelNames(matchers ...*labels.Matcher) ([]string, error)

	// LabelNamesFor returns label names for the given series referred to by IDs.
	LabelNamesFor(ids ...uint64) ([]string, error)

	// Close releases the underlying resource of the reader.
	Close() error
}

// Index returns an IndexReader against the block.
func (h *Head) Index() (IndexReader, error) {
	return h.indexRange(math.MinInt64, math.MaxInt64), nil
}

func (h *Head) indexRange(mint, maxt int64) *headIndexReader {
	if hmin := h.MinTime(); hmin > mint {
		mint = hmin
	}
	return &headIndexReader{head: h, mint: mint, maxt: maxt}
}

type headIndexReader struct {
	head       *Head
	mint, maxt int64
}

func (h *headIndexReader) Close() error {
	return nil
}

// Postings returns the postings list iterator for the label pairs.
func (h *headIndexReader) Postings(name string, values ...string) (*sroar.Bitmap, error) {
	if len(values) == 1 {
		return h.head.postings.Get(name, values[0]), nil
	}

	b := sroar.NewBitmap()
	for _, value := range values {
		// Or/merge/union the postings for all values
		b.Or(h.head.postings.Get(name, value))
	}

	if b.GetCardinality() == 0 {
		b.Set(math.MaxUint64) // This is an errPostings bitmap
	}

	return b, nil
}

// LabelValues returns label values present in the head for the
// specific label name that are within the time range mint to maxt.
// If matchers are specified the returned result set is reduced
// to label values of metrics matching the matchers.
func (h *headIndexReader) LabelValues(name string, matchers ...*labels.Matcher) ([]string, error) {
	if h.maxt < h.head.MinTime() || h.mint > h.head.MaxTime() {
		return []string{}, nil
	}

	if len(matchers) == 0 {
		return h.head.postings.LabelValues(name), nil
	}

	return labelValuesWithMatchers(h, name, matchers...)
}

func (h *headIndexReader) LabelValueFor(id uint64, label string) (string, error) {
	series := h.head.series.getByID(id)
	if series == nil {
		return "", ErrNotFound
	}
	value := series.lset.Get(label)
	if value == "" {
		return "", ErrNotFound
	}
	return value, nil
}

// LabelNamesFor returns all the label names for the series referred to by IDs.
// The names returned are sorted.
func (h *headIndexReader) LabelNamesFor(ids ...uint64) ([]string, error) {
	namesMap := make(map[string]struct{})
	for _, id := range ids {
		memSeries := h.head.series.getByID(id)
		if memSeries == nil {
			return nil, ErrNotFound
		}
		for _, lbl := range memSeries.lset {
			namesMap[lbl.Name] = struct{}{}
		}
	}
	names := make([]string, 0, len(namesMap))
	for name := range namesMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, nil
}

// LabelNames returns all the unique label names present in the head
// that are within the time range mint to maxt.
func (h *headIndexReader) LabelNames(matchers ...*labels.Matcher) ([]string, error) {
	if h.maxt < h.head.MinTime() || h.mint > h.head.MaxTime() {
		return []string{}, nil
	}

	if len(matchers) == 0 {
		labelNames := h.head.postings.LabelNames()

		sort.Strings(labelNames)
		return labelNames, nil
	}

	return labelNamesWithMatchers(h, matchers...)
}
