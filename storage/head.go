package storage

import (
	"context"
	"math"
	"sync"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.uber.org/atomic"
)

type Head struct {
	minTime, maxTime atomic.Int64 // Current min and max of the samples included in the head.
	lastSeriesID     atomic.Uint64
	numSeries        atomic.Uint64
	postings         map[string]map[string]*sroar.Bitmap

	seriesMtx *sync.RWMutex
	series    map[string]*MemSeries
}

func NewHead() *Head {
	h := &Head{
		seriesMtx: &sync.RWMutex{},
		series:    map[string]*MemSeries{},
	}
	h.minTime.Store(math.MaxInt64)
	h.maxTime.Store(math.MinInt64)
	return h
}

func (h *Head) getOrCreate(lset labels.Labels) *MemSeries {
	labelString := lset.String()
	h.seriesMtx.RLock()
	s, found := h.series[labelString]
	h.seriesMtx.RUnlock()
	if found {
		return s
	}

	// Optimistically assume that we are the first one to create the series.
	id := h.lastSeriesID.Inc()

	h.seriesMtx.Lock()
	defer h.seriesMtx.Unlock()

	s, found = h.series[labelString]
	if found {
		return s
	}

	s, err := NewMemSeries(lset, id)
	if err != nil {
		panic(err) // TODO: NewMemSeries should not error
	}
	h.series[labelString] = s
	h.numSeries.Inc()

	if h.postings == nil {
		h.postings = map[string]map[string]*sroar.Bitmap{}
	}
	for _, l := range lset {
		if h.postings[l.Name] == nil {
			h.postings[l.Name] = map[string]*sroar.Bitmap{}
		}
		if h.postings[l.Name][l.Value] == nil {
			h.postings[l.Name][l.Value] = sroar.NewBitmap()
		}
		h.postings[l.Name][l.Value].Set(s.id)
	}

	return s
}

// Appender returns a new Appender on the database.
func (h *Head) Appender(_ context.Context, lset labels.Labels) Appender {
	// The head cache might not have a starting point yet. The init appender
	// picks up the first appended timestamp as the base.
	if h.MinTime() == math.MaxInt64 {
		return &initAppender{
			lset: lset,
			head: h,
		}
	}
	return h.appender(lset)
}

// initAppender is a helper to initialize the time bounds of the head
// upon the first sample it receives.
type initAppender struct {
	lset labels.Labels
	app  Appender
	head *Head
}

func (a *initAppender) Append(p *Profile) error {
	if a.app != nil {
		return a.app.Append(p)
	}

	a.head.initTime(p.Meta.Timestamp)
	a.app = a.head.appender(a.lset)
	return a.app.Append(p)
}

// MinTime returns the lowest time bound on visible data in the head.
func (h *Head) MinTime() int64 {
	return h.minTime.Load()
}

// MaxTime returns the highest timestamp seen in data of the head.
func (h *Head) MaxTime() int64 {
	return h.maxTime.Load()
}

// initTime initializes a head with the first timestamp. This only needs to be called
// for a completely fresh head with an empty WAL.
func (h *Head) initTime(t int64) {
	if !h.minTime.CAS(math.MaxInt64, t) {
		return
	}
	// Ensure that max time is initialized to at least the min time we just set.
	// Concurrent appenders may already have set it to a higher value.
	h.maxTime.CAS(math.MinInt64, t)
}

func (h *Head) appender(lset labels.Labels) Appender {
	return h.getOrCreate(lset)
}

func (h *Head) Querier(ctx context.Context, mint, maxt int64) Querier {
	return &HeadQuerier{
		head: h,
		ctx:  ctx,
		mint: mint,
		maxt: maxt,
	}
}

type HeadQuerier struct {
	head       *Head
	ctx        context.Context
	mint, maxt int64
}

func (q *HeadQuerier) Select(hints *SelectHints, ms ...*labels.Matcher) SeriesSet {
	q.head.seriesMtx.RLock()
	defer q.head.seriesMtx.RUnlock()

	ids := map[uint64]struct{}{}
	for _, m := range ms {
		if q.head.postings == nil || q.head.postings[m.Name] == nil || q.head.postings[m.Name][m.Value] == nil {
			continue
		}

		it := q.head.postings[m.Name][m.Value].NewIterator()
		for it.HasNext() {
			ids[it.Next()] = struct{}{}
		}
	}

	// TODO: Improve not looping over all ids and within over all series...
	ss := make([]Series, 0, len(ids))
	for id := range ids {
		for _, series := range q.head.series {
			if series.id == id {
				ss = append(ss, series)
			}
		}
	}

	return &SliceSeriesSet{
		series: ss,
		i:      -1,
	}
}
