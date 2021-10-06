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
	"math"
	"sync"
	"time"

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/parca-dev/parca/pkg/storage/index"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
)

type Head struct {
	minTime, maxTime atomic.Int64 // Current min and max of the samples included in the head.
	lastSeriesID     atomic.Uint64
	numSeries        atomic.Uint64

	// stripeSeries store series by id and hash in maps that make them quickly accessible.
	series *stripeSeries
	// postings are mappings from label name and value to series IDs.
	// Merging and intersecting the resulting IDs we can look up
	// just the series we need from series by their IDs.
	postings *index.MemPostings

	chunkPool ChunkPool

	tracer              trace.Tracer
	minTimeGauge        *prometheus.Desc
	maxTimeGauge        *prometheus.Desc
	seriesCounter       *prometheus.Desc
	seriesValues        *prometheus.SummaryVec
	seriesChunksSize    *prometheus.SummaryVec
	seriesChunksSamples *prometheus.SummaryVec
	profilesAppended    prometheus.Counter
	truncateDuration    prometheus.Summary
	truncatedChunks     prometheus.Counter
}

// ChunkPool stores a set of temporary chunks that may be individually saved and retrieved.
type ChunkPool interface {
	Put(chunkenc.Chunk) error
	GetXOR() chunkenc.Chunk
	GetDelta() chunkenc.Chunk
	GetRLE() chunkenc.Chunk
	GetTimestamp() *timestampChunk
}

type HeadOptions struct {
	ChunkPool        ChunkPool
	ExpensiveMetrics bool
}

func NewHead(r prometheus.Registerer, tracer trace.Tracer, opts *HeadOptions) *Head {
	if opts == nil {
		opts = &HeadOptions{}
	}
	if opts.ChunkPool == nil {
		opts.ChunkPool = newHeadChunkPool()
	}

	h := &Head{
		postings:  index.NewMemPostings(),
		chunkPool: opts.ChunkPool,

		tracer: tracer,
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
		seriesValues: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:       "parca_tsdb_head_series_values",
			Help:       "Total number of series created in the head.",
			Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"values"}),
		seriesChunksSize: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:       "parca_tsdb_head_series_chunk_bytes",
			Help:       "The chunks size of the series.",
			Objectives: map[float64]float64{0.1: 0.05, 0.2: 0.05, 0.3: 0.05, 0.4: 0.05, 0.5: 0.05, 0.6: 0.05, 0.7: 0.05, 0.8: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"values"}),
		seriesChunksSamples: prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:       "parca_tsdb_head_series_chunk_samples",
			Help:       "The amount of samples in the cumulative and flat chunks.",
			Objectives: map[float64]float64{0.1: 0.05, 0.2: 0.05, 0.3: 0.05, 0.4: 0.05, 0.5: 0.05, 0.6: 0.05, 0.7: 0.05, 0.8: 0.05, 0.9: 0.01, 0.99: 0.001},
		}, []string{"values"}),
		profilesAppended: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "parca_tsdb_head_profiles_appended_total",
			Help: "Total number of appended profiles.",
		}),
		truncateDuration: prometheus.NewSummary(prometheus.SummaryOpts{
			Name: "parca_tsdb_head_truncate_duration_seconds",
			Help: "Runtime of truncating old chunks in the head block.",
		}),
		truncatedChunks: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "parca_tsdb_head_truncated_chunks_total",
			Help: "The total amount of truncated chunks over time.",
		}),
	}

	h.series = newStripeSeries(DefaultStripeSize, h.updateMaxTime)

	r.MustRegister(h,
		h.seriesValues,
		h.seriesChunksSize,
		h.seriesChunksSamples,
		h.profilesAppended,
		h.truncateDuration,
		h.truncatedChunks,
	)

	h.minTime.Store(math.MaxInt64)
	h.maxTime.Store(math.MinInt64)

	if opts.ExpensiveMetrics {
		// TODO: Actually do use the cancel function.
		ctx := context.Background()
		go func() {
			t := time.NewTicker(time.Minute)
			for {
				select {
				case <-ctx.Done():
					return
				case <-t.C:
					h.stats()
				}
			}
		}()
	}

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

func (h *Head) updateMaxTime(t int64) {
	for {
		ht := h.MaxTime()
		if t <= ht {
			break
		}
		if h.maxTime.CAS(ht, t) {
			break
		}
	}
}

func (h *Head) getOrCreate(ctx context.Context, lset labels.Labels) *MemSeries {
	ctx, span := h.tracer.Start(ctx, "getOrCreate")
	span.SetAttributes(attribute.String("labels", lset.String()))
	defer span.End()

	s := h.series.getByHash(lset.Hash(), lset)
	if s != nil {
		return s
	}

	// Optimistically assume that we are the first one to create the series.
	id := h.lastSeriesID.Inc()

	h.numSeries.Inc()

	// Trace from the outside to not have to pass tracer into stripeSeries.
	_, span = h.tracer.Start(ctx, "getOrCreateWithID")
	s, _ = h.series.getOrCreateWithID(id, lset.Hash(), lset, h.chunkPool)
	span.End()

	h.postings.Add(s.id, lset)

	return s
}

// Appender returns a new Appender on the database.
func (h *Head) Appender(ctx context.Context, lset labels.Labels) (Appender, error) {
	// The head cache might not have a starting point yet. The init appender
	// picks up the first appended timestamp as the base.
	if h.MinTime() == math.MaxInt64 {
		return &initAppender{
			lset: lset,
			head: h,
		}, nil
	}
	return h.appender(ctx, lset)
}

// initAppender is a helper to initialize the time bounds of the head
// upon the first sample it receives.
type initAppender struct {
	lset labels.Labels
	app  Appender
	head *Head
}

func (a *initAppender) Append(ctx context.Context, p *Profile) error {
	if a.app != nil {
		return a.app.Append(ctx, p)
	}

	a.head.initTime(p.Meta.Timestamp)

	var err error
	a.app, err = a.head.appender(ctx, a.lset)
	if err != nil {
		return err
	}

	return a.app.Append(ctx, p)
}

// MinTime returns the lowest time bound on visible data in the head.
func (h *Head) MinTime() int64 {
	return h.minTime.Load()
}

// MaxTime returns the highest timestamp seen in data of the head.
func (h *Head) MaxTime() int64 {
	return h.maxTime.Load()
}

func (h *Head) stats() {
	for i, series := range h.series.series {
		h.series.locks[i].RLock()
		for _, memSeries := range series {
			stats := memSeries.stats()
			h.seriesValues.WithLabelValues("flat").Observe(float64(len(stats.Flat)))
			h.seriesValues.WithLabelValues("cumulative").Observe(float64(len(stats.Cumulatives)))

			for _, s := range stats.Flat {
				h.seriesChunksSize.WithLabelValues("flat").Observe(float64(s.bytes))
				h.seriesChunksSamples.WithLabelValues("flat").Observe(float64(s.samples))
			}
			for _, s := range stats.Cumulatives {
				h.seriesChunksSize.WithLabelValues("cumulative").Observe(float64(s.bytes))
				h.seriesChunksSamples.WithLabelValues("cumulative").Observe(float64(s.samples))
			}
		}
		h.series.locks[i].RUnlock()
	}
}

// Truncate removes old data before mint from the head and WAL.
func (h *Head) Truncate(mint int64) error {
	return h.truncateMemory(mint)
}

func (h *Head) truncateMemory(mint int64) error {
	if h.MinTime() > mint {
		return nil
	}

	// Ensure that max time is at least as high as min time.
	for h.MaxTime() < mint {
		h.maxTime.CAS(h.MaxTime(), mint)
	}

	start := time.Now()

	_, truncatedChunks, actualMint := h.series.truncate(mint)

	h.truncateDuration.Observe(time.Since(start).Seconds())
	h.truncatedChunks.Add(float64(truncatedChunks))

	h.minTime.Store(actualMint)

	return nil
}

func (h *Head) appender(ctx context.Context, lset labels.Labels) (Appender, error) {
	s := h.getOrCreate(ctx, lset)
	s.tracer = h.tracer
	s.samplesAppended = h.profilesAppended
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

func (q *HeadQuerier) LabelNames(ms ...*labels.Matcher) ([]string, Warnings, error) {
	_, span := q.head.tracer.Start(q.ctx, "LabelNames")
	defer span.End()

	ir, err := q.head.Index()
	if err != nil {
		return nil, nil, err
	}

	names, err := ir.LabelNames(ms...)
	return names, nil, err
}

func (q *HeadQuerier) LabelValues(name string, ms ...*labels.Matcher) ([]string, Warnings, error) {
	_, span := q.head.tracer.Start(q.ctx, "LabelValues")
	defer span.End()

	ir, err := q.head.Index()
	if err != nil {
		return nil, nil, err
	}

	values, err := ir.LabelValues(name, ms...)
	return values, nil, err
}

func (q *HeadQuerier) Select(hints *SelectHints, ms ...*labels.Matcher) SeriesSet {
	ctx, span := q.head.tracer.Start(q.ctx, "Select")
	defer span.End()

	ir, err := q.head.Index()
	if err != nil {
		return &SliceSeriesSet{}
	}

	_, postingSpan := q.head.tracer.Start(ctx, "PostingsForMatchers")
	postings, err := PostingsForMatchers(ir, ms...)
	if err != nil {
		postingSpan.End()
		return &SliceSeriesSet{}
	}
	postingSpan.End()

	mint := q.mint
	maxt := q.maxt
	if hints != nil {
		mint = hints.Start
		maxt = hints.End
	}

	ss := make([]Series, 0, postings.GetCardinality())
	it := postings.NewIterator()

	for {
		id := it.Next()
		if id == 0 {
			break
		}

		s := q.head.series.getByID(id)
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
		if hints != nil && hints.Merge {
			ss = append(ss, &MemMergeSeries{s: s, mint: mint, maxt: maxt})
			continue
		}
		if hints != nil && hints.Root {
			ss = append(ss, &MemRootSeries{s: s, mint: mint, maxt: maxt})
			continue
		}
		ss = append(ss, &MemRangeSeries{s: s, mint: mint, maxt: maxt})
	}

	return &SliceSeriesSet{
		series: ss,
		i:      -1,
	}
}

const (
	// DefaultStripeSize is the default number of entries to allocate in the stripeSeries hash map.
	DefaultStripeSize = 1 << 10
)

// stripeSeries locks modulo ranges of IDs and hashes to reduce lock contention.
// The locks are padded to not be on the same cache line. Filling the padded space
// with the maps was profiled to be slower – likely due to the additional pointer
// dereferences.
type stripeSeries struct {
	size          int
	series        []map[uint64]*MemSeries
	hashes        []seriesHashmap
	locks         []stripeLock
	updateMaxTime func(int64)
}

func newStripeSeries(size int, updateMaxTime func(int64)) *stripeSeries {
	s := &stripeSeries{
		size:          size,
		series:        make([]map[uint64]*MemSeries, size),
		hashes:        make([]seriesHashmap, size),
		locks:         make([]stripeLock, size),
		updateMaxTime: updateMaxTime,
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

func (s stripeSeries) getOrCreateWithID(id, hash uint64, lset labels.Labels, chunkPool ChunkPool) (*MemSeries, bool) {
	i := hash & uint64(s.size-1)

	s.locks[i].RLock()
	series := s.getByHash(hash, lset)
	s.locks[i].RUnlock()

	if series != nil {
		return series, false
	}

	series = NewMemSeries(id, lset, s.updateMaxTime, chunkPool)

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

func (s stripeSeries) truncate(mint int64) (map[uint64]struct{}, int, int64) {
	var (
		deleted               = map[uint64]struct{}{}
		truncatedChunks       = 0
		actualMint      int64 = math.MaxInt64
	)

	for i := 0; i < s.size; i++ {
		if len(s.hashes[i]) == 0 {
			continue
		}
		s.locks[i].Lock()
		for _, all := range s.hashes[i] {
			for _, series := range all {
				truncatedChunks += series.truncateChunksBefore(mint)

				// TODO: Delete series that have no chunks left entirely.
				if series.minTime < actualMint {
					actualMint = series.minTime
				}
			}
		}
		s.locks[i].Unlock()
	}

	if actualMint == math.MaxInt64 {
		actualMint = mint
	}

	return deleted, truncatedChunks, actualMint
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

// HeadChunkPool wraps chunkenc.Pool and adds support for timestampChunk.
type HeadChunkPool struct {
	chunks     chunkenc.Pool
	timestamps *sync.Pool
}

func newHeadChunkPool() *HeadChunkPool {
	return &HeadChunkPool{
		chunks: chunkenc.NewPool(),
		timestamps: &sync.Pool{
			New: func() interface{} {
				// Make sure to GetDelta from the chunks pool and populate it later!
				return &timestampChunk{
					minTime: math.MaxInt64,
					maxTime: math.MinInt64,
				}
			},
		},
	}
}

func (p *HeadChunkPool) Put(c chunkenc.Chunk) error {
	if tc, ok := c.(*timestampChunk); ok {
		tc.maxTime = 0
		tc.minTime = 0
		p.timestamps.Put(tc)
		return p.chunks.Put(tc.chunk)
	}
	return p.chunks.Put(c)
}

func (p *HeadChunkPool) GetXOR() chunkenc.Chunk {
	c, _ := p.chunks.Get(chunkenc.EncXOR, nil)
	return c
}

func (p *HeadChunkPool) GetDelta() chunkenc.Chunk {
	c, _ := p.chunks.Get(chunkenc.EncDelta, nil)
	return c
}

func (p *HeadChunkPool) GetRLE() chunkenc.Chunk {
	c, _ := p.chunks.Get(chunkenc.EncRLE, nil)
	return c
}

func (p *HeadChunkPool) GetTimestamp() *timestampChunk {
	tc := p.timestamps.Get().(*timestampChunk)
	tc.chunk = p.GetDelta()
	tc.minTime = math.MaxInt64
	tc.maxTime = math.MinInt64
	return tc
}
