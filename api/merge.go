package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/google/pprof/profile"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
)

var (
	DefaultMergeBatchSize = int64(1024 * 1024 * 64) // 64Mb
)

type MergeTimeoutError struct {
	mergedSamplesCount int
}

func NewMergeTimeoutError(count int) *MergeTimeoutError {
	return &MergeTimeoutError{mergedSamplesCount: count}
}

func (e *MergeTimeoutError) Error() string {
	return fmt.Sprintf("merge timeout exceeded, used partial merge of %d samples", e.mergedSamplesCount)
}

type batchIterator struct {
	set          storage.SeriesSet
	curIterator  chunkenc.Iterator
	maxBatchSize int64
	err          error

	batch [][]byte
}

func newBatchIterator(set storage.SeriesSet, maxBatchSize int64) *batchIterator {
	return &batchIterator{
		set:          set,
		curIterator:  nil,
		maxBatchSize: maxBatchSize,
		batch:        [][]byte{},
		err:          nil,
	}
}

func (i *batchIterator) Next() bool {
	batchSize := int64(0)
	i.batch = i.batch[:0]

	// Finish previsous iterator if unfinished.
	if i.curIterator != nil {
		for i.curIterator.Next() {
			_, b := i.curIterator.At()
			if err := i.curIterator.Err(); err != nil {
				i.err = i.curIterator.Err()
				return false
			}
			i.batch = append(i.batch, b)
			batchSize += int64(len(b))
			if batchSize >= i.maxBatchSize {
				return true
			}
		}
	}
	for i.set.Next() {
		series := i.set.At()
		i.curIterator = series.Iterator()
		for i.curIterator.Next() {
			_, b := i.curIterator.At()
			if err := i.curIterator.Err(); err != nil {
				i.err = i.curIterator.Err()
				return false
			}
			i.batch = append(i.batch, b)
			batchSize += int64(len(b))
			if batchSize >= i.maxBatchSize {
				return true
			}
		}
	}
	if err := i.set.Err(); err != nil {
		i.err = i.set.Err()
		return false
	}

	// As long as we're returning data we're gonna go on.
	return len(i.batch) > 0
}

func (i *batchIterator) Batch() [][]byte {
	return i.batch
}

func (i *batchIterator) Err() error {
	return i.err
}

func (a *API) mergeProfiles(ctx context.Context, from, to time.Time, sel []*labels.Matcher) (*profile.Profile, storage.Warnings, *ApiError) {
	q, err := a.db.Querier(ctx, timestamp.FromTime(from), timestamp.FromTime(to))
	if err != nil {
		return nil, nil, &ApiError{Typ: ErrorExec, Err: err}
	}

	set := q.Select(false, nil, sel...)
	mergedProfile, count, err := mergeSeriesSet(ctx, set, a.maxMergeBatchSize)
	if err != nil && err != context.DeadlineExceeded {
		return nil, nil, &ApiError{Typ: ErrorInternal, Err: err}
	}
	var warnings storage.Warnings = nil
	if err != nil && err == context.DeadlineExceeded {
		warnings = append(warnings, NewMergeTimeoutError(count))
	}
	a.mergeSizeHist.Observe(float64(count))

	return mergedProfile, warnings, nil
}

func mergeSeriesSet(ctx context.Context, set storage.SeriesSet, maxMergeBatchSize int64) (*profile.Profile, int, error) {
	bi := newBatchIterator(set, maxMergeBatchSize)
	profiles := []*profile.Profile{}
	var acc *profile.Profile = nil
	count := 0
	for bi.Next() {
		profiles = profiles[:0]
		batch := bi.Batch()

		for _, b := range batch {
			p, err := profile.ParseData(b)
			if err != nil {
				return nil, 0, err
			}
			profiles = append(profiles, p)
		}

		if acc == nil && len(profiles) > 0 {
			acc = profiles[0]
			profiles = profiles[1:]
		}

		var err error
		acc, err = profile.Merge(append([]*profile.Profile{acc}, profiles...))
		if err != nil {
			return nil, 0, err
		}

		count += len(profiles)
	}
	if err := bi.Err(); err != nil {
		return nil, 0, bi.Err()
	}

	return acc, count, ctx.Err()
}

func (a *API) MergeProfiles(r *http.Request) (*profile.Profile, storage.Warnings, *ApiError) {
	ctx := r.Context()

	return a.profileByParameters(
		ctx,
		"merge",
		"",
		r.URL.Query().Get("query"),
		r.URL.Query().Get("from"),
		r.URL.Query().Get("to"),
	)
}
