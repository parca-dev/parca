package storage

import (
	"context"
	"math"
	"sync"

	"github.com/parca-dev/parca/pkg/storage/index"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.uber.org/atomic"
)

type Head struct {
	minTime, maxTime atomic.Int64 // Current min and max of the samples included in the head.
	lastSeriesID     atomic.Uint64
	numSeries        atomic.Uint64
	postings         *index.MemPostings

	seriesMtx *sync.RWMutex
	series    map[string]*MemSeries
}

func NewHead() *Head {
	h := &Head{
		seriesMtx: &sync.RWMutex{},
		series:    map[string]*MemSeries{},
		postings:  index.NewMemPostings(),
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

	h.postings.Add(s.id, lset)

	return s
}

func (h Head) getByID(id uint64) *MemSeries {
	// TODO: Improve with stripeSeries []map[uint64]*MemSeries like Prometheus?
	var s *MemSeries
	h.seriesMtx.RLock()
	for _, series := range h.series {
		if series.id == id {
			s = series
		}
	}
	h.seriesMtx.RUnlock()
	return s
}

// Appender returns a new Appender on the database.
func (h *Head) Appender(_ context.Context, lset labels.Labels) (Appender, error) {
	// The head cache might not have a starting point yet. The init appender
	// picks up the first appended timestamp as the base.
	if h.MinTime() == math.MaxInt64 {
		return &initAppender{
			lset: lset,
			head: h,
		}, nil
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

	var err error
	a.app, err = a.head.appender(a.lset)
	if err != nil {
		return err
	}

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

func (h *Head) appender(lset labels.Labels) (Appender, error) {
	s := h.getOrCreate(lset)
	return s.Appender()
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

	ir, err := q.head.Index()
	if err != nil {
		return nil
	}

	postings, err := PostingsForMatchers(ir, ms...)
	if err != nil {
		return nil
	}

	ss := make([]Series, 0, postings.GetCardinality())
	it := postings.NewIterator()
	for it.HasNext() {
		ss = append(ss, q.head.getByID(it.Next()))
	}

	return &SliceSeriesSet{
		series: ss,
		i:      -1,
	}
}
