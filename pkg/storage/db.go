package storage

import (
	"context"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
)

type Appendable interface {
	Appender(ctx context.Context, lset labels.Labels) (Appender, error)
}

type SelectHints struct {
	Start int64 // Start time in milliseconds for this select.
	End   int64 // End time in milliseconds for this select.

	Merge    bool // Is the query result a merge of all samples?
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
	Append(p *Profile) error
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
	head *Head
}

func OpenDB(r prometheus.Registerer) *DB {
	return &DB{
		head: NewHead(r),
	}
}

func (db *DB) Appender(ctx context.Context, lset labels.Labels) (Appender, error) {
	return db.head.Appender(ctx, lset)
}

func (db *DB) Querier(ctx context.Context, mint, maxt int64) Querier {
	return db.head.Querier(ctx, mint, maxt)
}
