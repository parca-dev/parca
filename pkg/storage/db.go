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
	"time"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"go.opentelemetry.io/otel/trace"
)

type Appendable interface {
	Appender(ctx context.Context, lset labels.Labels) (Appender, error)
}

type SelectHints struct {
	Start int64 // Start time in milliseconds for this select.
	End   int64 // End time in milliseconds for this select.

	Merge    bool // Is the query result a merge of all samples?
	Root     bool // Is the query result supposed to only contain the root's cumulative values?
	Metadata bool // Is the database just being queried for metadata like label-names/label-values.
}

type Queryable interface {
	Querier(ctx context.Context, mint, maxt int64) Querier
}

type Querier interface {
	LabelQuerier
	Select(hints *SelectHints, ms ...*labels.Matcher) SeriesSet
}

// LabelQuerier provides querying access over labels.
type LabelQuerier interface {
	// LabelValues returns all potential values for a label name.
	// It is not safe to use the strings beyond the lifetime of the querier.
	// If matchers are specified the returned result set is reduced
	// to label values of metrics matching the matchers.
	LabelValues(name string, matchers ...*labels.Matcher) ([]string, Warnings, error)

	// LabelNames returns all the unique label names present in the block in sorted order.
	// If matchers are specified the returned result set is reduced
	// to label names of metrics matching the matchers.
	LabelNames(matchers ...*labels.Matcher) ([]string, Warnings, error)
}

// SeriesSet contains a set of series.
type SeriesSet interface {
	Next() bool
	// At returns full series. Returned series should be iterable even after Next is called.
	At() Series
	// The error that iteration as failed with.
	// When an error occurs, set cannot continue to iterate.
	Err() error
	// A collection of warnings for the whole set.
	// Warnings could be return even iteration has not failed with error.
	Warnings() Warnings
}

type Warnings []error

func (w Warnings) ToStrings() []string {
	res := make([]string, 0, len(w))
	for _, warn := range w {
		res = append(res, warn.Error())
	}
	return res
}

type Series interface {
	Labels
	ProfileSeries
}

// Labels represents an item that has labels e.g. time series.
type Labels interface {
	// Labels returns the complete set of labels. For series it means all labels identifying the series.
	Labels() labels.Labels
}

type Appender interface {
	AppendFlat(ctx context.Context, p *profile.FlatProfile) error
}

type SliceSeriesSet struct {
	series []Series
	i      int
}

func (s *SliceSeriesSet) Next() bool {
	s.i++
	return s.i < len(s.series)
}

func (s *SliceSeriesSet) At() Series         { return s.series[s.i] }
func (s *SliceSeriesSet) Err() error         { return nil }
func (s *SliceSeriesSet) Warnings() Warnings { return nil }

type DB struct {
	options *DBOptions

	head *Head
}

type DBOptions struct {
	Retention time.Duration

	HeadExpensiveMetrics bool
}

func OpenDB(r prometheus.Registerer, tracer trace.Tracer, opts *DBOptions) *DB {
	if opts == nil {
		opts = &DBOptions{
			HeadExpensiveMetrics: false,
		}
	}

	return &DB{
		options: opts,
		head: NewHead(r, tracer, &HeadOptions{
			ExpensiveMetrics: opts.HeadExpensiveMetrics,
		}),
	}
}

func (db *DB) Appender(ctx context.Context, lset labels.Labels) (Appender, error) {
	return db.head.Appender(ctx, lset)
}

func (db *DB) Querier(ctx context.Context, mint, maxt int64) Querier {
	return db.head.Querier(ctx, mint, maxt)
}

func (db *DB) Run(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			retention := 6 * time.Hour // default is 6h if nothing is passed.
			if db.options != nil && db.options.Retention != 0 {
				retention = db.options.Retention
			}

			mint := timestamp.FromTime(time.Now().Add(-1 * retention))
			if err := db.head.Truncate(mint); err != nil {
				return err
			}
		}
	}
}
