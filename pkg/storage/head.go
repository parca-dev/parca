package storage

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/parca-dev/parca/pkg/storage/index"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.uber.org/atomic"
)

type Head struct {
	reg prometheus.Registerer

	minTime, maxTime atomic.Int64 // Current min and max of the samples included in the head.
	lastSeriesID     atomic.Uint64
	numSeries        atomic.Uint64

	// stripeSeries store series by id and hash in maps that make them quickly accessible.
	series *stripeSeries
	// postings are mappings from label name and value to series IDs.
	// Merging and intersecting the resulting IDs we can look up
	// just the series we need from series by their IDs.
	postings *index.MemPostings

	minTimeGauge               *prometheus.Desc
	maxTimeGauge               *prometheus.Desc
	seriesCounter              *prometheus.Desc
}

func NewHead(r prometheus.Registerer) *Head {
	h := &Head{
		series:   newStripeSeries(DefaultStripeSize),
		postings: index.NewMemPostings(),
		reg:      r,

		minTimeGauge: prometheus.NewDesc(
			"parca_tsdb_head_min_time",
			"Minimum time bound of the head block. The unit is decided by the library consumer.",
			nil, nil,
		),
		maxTimeGauge: prometheus.NewDesc(
			"parca_tsdb_head_max_time",
			"Maximum timestamp of the head block. The unit is decided by the library consumer.",
			nil, nil,
		),
		seriesCounter: prometheus.NewDesc(
			"parca_tsdb_head_series_created_total",
			"Total number of series created in the head.",
			nil, nil,
		),
	}
	h.minTime.Store(math.MaxInt64)
	h.maxTime.Store(math.MinInt64)
	return h
}

func (h *Head) Describe(descs chan<- *prometheus.Desc) {
	descs <- h.seriesCounter
	descs <- h.minTimeGauge
	descs <- h.maxTimeGauge
}

func (h *Head) Collect(metrics chan<- prometheus.Metric) {
	metrics <- prometheus.MustNewConstMetric(h.seriesCounter, prometheus.CounterValue, float64(h.numSeries.Load()))
	metrics <- prometheus.MustNewConstMetric(h.minTimeGauge, prometheus.GaugeValue, float64(h.MinTime()/1000))
	metrics <- prometheus.MustNewConstMetric(h.maxTimeGauge, prometheus.GaugeValue, float64(h.MaxTime()/1000))
}

func (h *Head) getOrCreate(lset labels.Labels) *MemSeries {
	s := h.series.getByHash(lset.Hash(), lset)
	if s != nil {
		return s
	}

	// Optimistically assume that we are the first one to create the series.
	id := h.lastSeriesID.Inc()

	h.numSeries.Inc()

	s, _ = h.series.getOrCreateWithID(id, lset.Hash(), lset)

	h.postings.Add(s.id, lset)

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
	ir, err := q.head.Index()
	if err != nil {
		return nil
	}

	postings, err := PostingsForMatchers(ir, ms...)
	if err != nil {
		return nil
	}

	mint := q.mint
	maxt := q.maxt
	if hints != nil {
		mint = hints.Start
		maxt = hints.End
	}

	ss := make([]Series, 0, postings.GetCardinality())
	it := postings.NewIterator()
	for it.HasNext() {
		s := q.head.series.getByID(it.Next())
		s.mu.RLock()
		seriesMaxTime := s.maxTime
		seriesMinTime := s.minTime
		s.mu.RUnlock()
		if seriesMaxTime < mint {
			continue
		}
		if seriesMinTime > maxt {
			continue
		}
		ss = append(ss, s)
	}

	return &SliceSeriesSet{
		series: ss,
		i:      -1,
	}
}

const (
	// DefaultStripeSize is the default number of entries to allocate in the stripeSeries hash map.
	DefaultStripeSize = 1 << 14
)

// stripeSeries locks modulo ranges of IDs and hashes to reduce lock contention.
// The locks are padded to not be on the same cache line. Filling the padded space
// with the maps was profiled to be slower – likely due to the additional pointer
// dereferences.
type stripeSeries struct {
	size   int
	series []map[uint64]*MemSeries
	hashes []seriesHashmap
	locks  []stripeLock
}

func newStripeSeries(size int) *stripeSeries {
	s := &stripeSeries{
		size:   size,
		series: make([]map[uint64]*MemSeries, size),
		hashes: make([]seriesHashmap, size),
		locks:  make([]stripeLock, size),
	}
	for i := range s.series {
		s.series[i] = map[uint64]*MemSeries{}
	}
	for i := range s.hashes {
		s.hashes[i] = seriesHashmap{}
	}
	return s
}

type stripeLock struct {
	sync.RWMutex
	// Padding to avoid multiple locks being on the same cache line.
	_ [40]byte
}

func (s stripeSeries) getOrCreateWithID(id, hash uint64, lset labels.Labels) (*MemSeries, bool) {
	i := hash & uint64(s.size-1)

	s.locks[i].RLock()
	series := s.getByHash(hash, lset)
	s.locks[i].RUnlock()

	if series != nil {
		return series, false
	}

	series = NewMemSeries(lset, id)

	s.locks[i].Lock()
	s.hashes[i].set(hash, series)
	s.locks[i].Unlock()

	// overwrite i for the id based index
	i = id & uint64(s.size-1)

	s.locks[i].Lock()
	s.series[i][id] = series
	s.locks[i].Unlock()

	return series, true
}

func (s *stripeSeries) getByID(id uint64) *MemSeries {
	i := id & uint64(s.size-1)
	s.locks[i].RLock()
	series := s.series[i][id]
	s.locks[i].RUnlock()

	return series
}

func (s stripeSeries) getByHash(hash uint64, lset labels.Labels) *MemSeries {
	i := hash & uint64(s.size-1)

	s.locks[i].RLock()
	series := s.hashes[i].get(hash, lset)
	s.locks[i].RUnlock()

	return series
}

// seriesHashmap is a simple hashmap for memSeries by their label set. It is built
// on top of a regular hashmap and holds a slice of series to resolve hash collisions.
// Its methods require the hash to be submitted with it to avoid re-computations throughout
// the code.
type seriesHashmap map[uint64][]*MemSeries

func (m seriesHashmap) set(hash uint64, s *MemSeries) {
	l := m[hash]
	// Try to find existing series with the same labels.Labels and overwrite it.
	for i, prev := range l {
		if labels.Equal(prev.lset, s.lset) {
			l[i] = s
			return
		}
	}
	// If nothing was found then append the series.
	m[hash] = append(l, s)
}

func (m seriesHashmap) get(hash uint64, lset labels.Labels) *MemSeries {
	for _, s := range m[hash] {
		if labels.Equal(s.lset, lset) {
			return s
		}
	}
	return nil
}

func (m seriesHashmap) del(hash uint64, lset labels.Labels) {
	var rem []*MemSeries
	for _, s := range m[hash] {
		// Append the series that don't match the label set
		// to exclude the one that matches.
		if !labels.Equal(s.lset, lset) {
			rem = append(rem, s)
		}
	}
	if len(rem) == 0 {
		delete(m, hash)
	} else {
		m[hash] = rem
	}
}

func (q *HeadQuerier) LabelValues(name string, ms ...*labels.Matcher) ([]string, Warnings, error) {
	ir, err := q.head.Index()
	if err != nil {
		return nil, nil, err
	}

	values, err := ir.LabelValues(name, ms...)
	return values, nil, err
}

func (q *HeadQuerier) LabelNames(ms ...*labels.Matcher) ([]string, Warnings, error) {
	ir, err := q.head.Index()
	if err != nil {
		return nil, nil, err
	}

	names, err := ir.LabelNames(ms...)
	return names, nil, err
}
