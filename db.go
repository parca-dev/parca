package storage

import (
	"context"

	"github.com/prometheus/prometheus/pkg/labels"
)

type Appendable interface {
	Appender(ctx context.Context, lset labels.Labels) Appender
}

type SelectHints struct {
	Start int64 // Start time in milliseconds for this select.
	End   int64 // End time in milliseconds for this select.

	Merge    bool // Is the query result a merge of all samples?
	Metadata bool // Is the database just being queried for metadata like label-names/label-values.
}

type Querier interface {
	Select(hints *SelectHints, ms ...*labels.Matcher) SeriesSet
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

func OpenDB() *DB {
	return &DB{
		head: NewHead(),
	}
}

func (db *DB) Appender(ctx context.Context, lset labels.Labels) Appender {
	return db.head.Appender(ctx, lset)
}

func (db *DB) Querier(ctx context.Context, mint, maxt int64) Querier {
	return db.head.Querier(ctx, mint, maxt)
}
