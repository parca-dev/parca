// Copyright 2016 The conprof Authors
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

package scrape

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/conprof/conprof/config"
	"github.com/conprof/conprof/internal/trace"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/pool"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"golang.org/x/net/context/ctxhttp"
)

var (
	targetIntervalLength = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "prometheus_target_interval_length_seconds",
			Help:       "Actual intervals between scrapes.",
			Objectives: map[float64]float64{0.01: 0.001, 0.05: 0.005, 0.5: 0.05, 0.90: 0.01, 0.99: 0.001},
		},
		[]string{"interval"},
	)
	targetReloadIntervalLength = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "prometheus_target_reload_length_seconds",
			Help:       "Actual interval to reload the scrape pool with a given configuration.",
			Objectives: map[float64]float64{0.01: 0.001, 0.05: 0.005, 0.5: 0.05, 0.90: 0.01, 0.99: 0.001},
		},
		[]string{"interval"},
	)
	targetSyncIntervalLength = prometheus.NewSummaryVec(
		prometheus.SummaryOpts{
			Name:       "prometheus_target_sync_length_seconds",
			Help:       "Actual interval to sync the scrape pool.",
			Objectives: map[float64]float64{0.01: 0.001, 0.05: 0.005, 0.5: 0.05, 0.90: 0.01, 0.99: 0.001},
		},
		[]string{"scrape_job"},
	)
	targetScrapePoolSyncsCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "prometheus_target_scrape_pool_sync_total",
			Help: "Total number of syncs that were executed on a scrape pool.",
		},
		[]string{"scrape_job"},
	)
	targetScrapeSampleLimit = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_target_scrapes_exceeded_sample_limit_total",
			Help: "Total number of scrapes that hit the sample limit and were rejected.",
		},
	)
	targetScrapeSampleDuplicate = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_target_scrapes_sample_duplicate_timestamp_total",
			Help: "Total number of samples rejected due to duplicate timestamps but different values",
		},
	)
	targetScrapeSampleOutOfOrder = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_target_scrapes_sample_out_of_order_total",
			Help: "Total number of samples rejected due to not being out of the expected order",
		},
	)
	targetScrapeSampleOutOfBounds = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "prometheus_target_scrapes_sample_out_of_bounds_total",
			Help: "Total number of samples rejected due to timestamp falling outside of the time bounds",
		},
	)
)

func init() {
	prometheus.MustRegister(targetIntervalLength)
	prometheus.MustRegister(targetReloadIntervalLength)
	prometheus.MustRegister(targetSyncIntervalLength)
	prometheus.MustRegister(targetScrapePoolSyncsCounter)
	prometheus.MustRegister(targetScrapeSampleLimit)
	prometheus.MustRegister(targetScrapeSampleDuplicate)
	prometheus.MustRegister(targetScrapeSampleOutOfOrder)
	prometheus.MustRegister(targetScrapeSampleOutOfBounds)
}

// scrapePool manages scrapes for sets of targets.
type scrapePool struct {
	appendable Appendable
	logger     log.Logger

	mtx    sync.RWMutex
	config *config.ScrapeConfig
	client *http.Client
	// Targets and loops must always be synchronized to have the same
	// set of hashes.
	activeTargets  map[uint64]*Target
	droppedTargets []*Target
	loops          map[uint64]loop
	cancel         context.CancelFunc

	// Constructor for new scrape loops. This is settable for testing convenience.
	newLoop func(*Target, scraper) loop
}

func newScrapePool(cfg *config.ScrapeConfig, app Appendable, logger log.Logger) *scrapePool {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	client, err := commonconfig.NewClientFromConfig(cfg.HTTPClientConfig, cfg.JobName, false, false)
	if err != nil {
		// Any errors that could occur here should be caught during config validation.
		level.Error(logger).Log("msg", "Error creating HTTP client", "err", err)
	}

	buffers := pool.New(1e3, 100e6, 3, func(sz int) interface{} { return make([]byte, 0, sz) })

	ctx, cancel := context.WithCancel(context.Background())
	sp := &scrapePool{
		cancel:        cancel,
		appendable:    app,
		config:        cfg,
		client:        client,
		activeTargets: map[uint64]*Target{},
		loops:         map[uint64]loop{},
		logger:        logger,
	}
	sp.newLoop = func(t *Target, s scraper) loop {
		return newScrapeLoop(
			ctx,
			t,
			s,
			log.With(logger, "target", t),
			buffers,
			app,
		)
	}

	return sp
}

func (sp *scrapePool) ActiveTargets() []*Target {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()

	var tActive []*Target
	for _, t := range sp.activeTargets {
		tActive = append(tActive, t)
	}
	return tActive
}

func (sp *scrapePool) DroppedTargets() []*Target {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()
	return sp.droppedTargets
}

// stop terminates all scrape loops and returns after they all terminated.
func (sp *scrapePool) stop() {
	sp.cancel()
	var wg sync.WaitGroup

	sp.mtx.Lock()
	defer sp.mtx.Unlock()

	for fp, l := range sp.loops {
		wg.Add(1)

		go func(l loop) {
			l.stop()
			wg.Done()
		}(l)

		delete(sp.loops, fp)
		delete(sp.activeTargets, fp)
	}
	wg.Wait()
}

// reload the scrape pool with the given scrape configuration. The target state is preserved
// but all scrape loops are restarted with the new scrape configuration.
// This method returns after all scrape loops that were stopped have stopped scraping.
func (sp *scrapePool) reload(cfg *config.ScrapeConfig) {
	start := time.Now()

	sp.mtx.Lock()
	defer sp.mtx.Unlock()

	client, err := commonconfig.NewClientFromConfig(cfg.HTTPClientConfig, cfg.JobName, false, false)
	if err != nil {
		// Any errors that could occur here should be caught during config validation.
		level.Error(sp.logger).Log("msg", "Error creating HTTP client", "err", err)
	}
	sp.config = cfg
	sp.client = client

	var (
		wg       sync.WaitGroup
		interval = time.Duration(sp.config.ScrapeInterval)
		timeout  = time.Duration(sp.config.ScrapeTimeout)
	)

	for fp, oldLoop := range sp.loops {
		var (
			t       = sp.activeTargets[fp]
			s       = &targetScraper{Target: t, client: sp.client, timeout: timeout}
			newLoop = sp.newLoop(t, s)
		)
		wg.Add(1)

		go func(oldLoop, newLoop loop) {
			oldLoop.stop()
			wg.Done()

			go newLoop.run(interval, timeout, nil)
		}(oldLoop, newLoop)

		sp.loops[fp] = newLoop
	}

	wg.Wait()
	targetReloadIntervalLength.WithLabelValues(interval.String()).Observe(
		time.Since(start).Seconds(),
	)
}

// Sync converts target groups into actual scrape targets and synchronizes
// the currently running scraper with the resulting set and returns all scraped and dropped targets.
func (sp *scrapePool) Sync(tgs []*targetgroup.Group) {
	start := time.Now()

	var all []*Target
	sp.mtx.Lock()
	sp.droppedTargets = []*Target{}
	for _, tg := range tgs {
		targets, err := targetsFromGroup(tg, sp.config)
		if err != nil {
			level.Error(sp.logger).Log("msg", "creating targets failed", "err", err)
			continue
		}

		for _, t := range targets {
			if t.Labels().Len() > 0 {
				all = append(all, t)
			} else if t.DiscoveredLabels().Len() > 0 {
				sp.droppedTargets = append(sp.droppedTargets, t)
			}
		}
	}
	sp.mtx.Unlock()
	sp.sync(all)

	targetSyncIntervalLength.WithLabelValues(sp.config.JobName).Observe(
		time.Since(start).Seconds(),
	)
	targetScrapePoolSyncsCounter.WithLabelValues(sp.config.JobName).Inc()
}

// sync takes a list of potentially duplicated targets, deduplicates them, starts
// scrape loops for new targets, and stops scrape loops for disappeared targets.
// It returns after all stopped scrape loops terminated.
func (sp *scrapePool) sync(targets []*Target) {
	sp.mtx.Lock()
	defer sp.mtx.Unlock()

	var (
		uniqueTargets = map[uint64]struct{}{}
		interval      = time.Duration(sp.config.ScrapeInterval)
		timeout       = time.Duration(sp.config.ScrapeTimeout)
	)

	for _, t := range targets {
		t := t
		hash := t.hash()
		uniqueTargets[hash] = struct{}{}

		if _, ok := sp.activeTargets[hash]; !ok {
			s := &targetScraper{Target: t, client: sp.client, timeout: timeout, logger: sp.logger}
			l := sp.newLoop(t, s)

			sp.activeTargets[hash] = t
			sp.loops[hash] = l

			go l.run(interval, timeout, nil)
		} else {
			// Need to keep the most updated labels information
			// for displaying it in the Service Discovery web page.
			sp.activeTargets[hash].SetDiscoveredLabels(t.DiscoveredLabels())
		}
	}

	var wg sync.WaitGroup

	// Stop and remove old targets and scraper loops.
	for hash := range sp.activeTargets {
		if _, ok := uniqueTargets[hash]; !ok {
			wg.Add(1)
			go func(l loop) {
				l.stop()

				wg.Done()
			}(sp.loops[hash])

			delete(sp.loops, hash)
			delete(sp.activeTargets, hash)
		}
	}

	// Wait for all potentially stopped scrapers to terminate.
	// This covers the case of flapping targets. If the server is under high load, a new scraper
	// may be active and tries to insert. The old scraper that didn't terminate yet could still
	// be inserting a previous sample set.
	wg.Wait()
}

// A scraper retrieves samples and accepts a status report at the end.
type scraper interface {
	scrape(ctx context.Context, w io.Writer, profileType string) error
	offset(interval time.Duration) time.Duration
}

// targetScraper implements the scraper interface for a target.
type targetScraper struct {
	*Target

	logger  log.Logger
	client  *http.Client
	req     *http.Request
	timeout time.Duration
}

var userAgentHeader = fmt.Sprintf("conprof/%s", version.Version)

func (s *targetScraper) scrape(ctx context.Context, w io.Writer, profileType string) error {
	if s.req == nil {
		req, err := http.NewRequest("GET", s.URL().String(), nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", userAgentHeader)

		s.req = req
	}

	level.Debug(s.logger).Log("msg", "scraping profile", "url", s.req.URL.String())
	resp, err := ctxhttp.Do(ctx, s.client, s.req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	switch profileType {
	case ProfileTraceType:
		// This is needed as trace.Parse fails with obscure error
		// when you pass resp.Body
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "failed to read body")
		}
		_, err = trace.Parse(io.TeeReader(bytes.NewBuffer(b), w), "")
		if err != nil {
			return errors.Wrap(err, "failed to parse target's trace profile")
		}

	default:
		_, err = profile.Parse(io.TeeReader(resp.Body, w))
		if err != nil {
			return errors.Wrap(err, "failed to parse target's pprof profile")
		}
	}

	return nil
}

// A loop can run and be stopped again. It must not be reused after it was stopped.
type loop interface {
	run(interval, timeout time.Duration, errc chan<- error)
	stop()
}

type scrapeLoop struct {
	target         *Target
	scraper        scraper
	l              log.Logger
	lastScrapeSize int
	buffers        *pool.Pool

	appendable Appendable

	ctx       context.Context
	scrapeCtx context.Context
	cancel    func()
	stopped   chan struct{}
}

func newScrapeLoop(ctx context.Context,
	t *Target,
	sc scraper,
	l log.Logger,
	buffers *pool.Pool,
	appendable Appendable,
) *scrapeLoop {
	if l == nil {
		l = log.NewNopLogger()
	}
	if buffers == nil {
		buffers = pool.New(1e3, 1e6, 3, func(sz int) interface{} { return make([]byte, 0, sz) })
	}
	sl := &scrapeLoop{
		target:     t,
		scraper:    sc,
		buffers:    buffers,
		appendable: appendable,
		stopped:    make(chan struct{}),
		l:          l,
		ctx:        ctx,
	}
	sl.scrapeCtx, sl.cancel = context.WithCancel(ctx)

	return sl
}

func (sl *scrapeLoop) run(interval, timeout time.Duration, errc chan<- error) {
	select {
	case <-time.After(sl.scraper.offset(interval)):
		// Continue after a scraping offset.
	case <-sl.scrapeCtx.Done():
		close(sl.stopped)
		return
	}

	var last time.Time

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

mainLoop:
	for {
		select {
		case <-sl.ctx.Done():
			close(sl.stopped)
			return
		case <-sl.scrapeCtx.Done():
			break mainLoop
		default:
		}

		start := time.Now()

		// Only record after the first scrape.
		if !last.IsZero() {
			targetIntervalLength.WithLabelValues(interval.String()).Observe(
				time.Since(last).Seconds(),
			)
		}

		b := sl.buffers.Get(sl.lastScrapeSize).([]byte)
		buf := bytes.NewBuffer(b)

		var profileType string
		for _, l := range sl.target.labels {
			if l.Name == ProfileName {
				profileType = l.Value
				break
			}
		}

		scrapeCtx, cancel := context.WithTimeout(sl.ctx, timeout)
		scrapeErr := sl.scraper.scrape(scrapeCtx, buf, profileType)
		cancel()

		if scrapeErr == nil {
			b = buf.Bytes()
			// NOTE: There were issues with misbehaving clients in the past
			// that occasionally returned empty results. We don't want those
			// to falsely reset our buffer size.
			if len(b) > 0 {
				sl.lastScrapeSize = len(b)
			}

			tl := sl.target.Labels()
			for _, l := range sl.target.labels {
				if l.Name == "__name__" {
					tl = append(tl, labels.Label{Name: l.Name, Value: l.Value})
				}
			}
			level.Debug(sl.l).Log("msg", "appending new sample", "labels", tl.String())

			app := sl.appendable.Appender(sl.ctx)
			_, err := app.Add(tl, timestamp.FromTime(start), buf.Bytes())
			if err != nil && errc != nil {
				level.Debug(sl.l).Log("err", err)
				errc <- err
			}

			err = app.Commit()
			if err != nil && errc != nil {
				level.Debug(sl.l).Log("err", err)
				errc <- err
			}
		} else {
			level.Debug(sl.l).Log("msg", "Scrape failed", "err", scrapeErr.Error())
			if errc != nil {
				errc <- scrapeErr
			}
		}

		sl.buffers.Put(b)
		last = start

		select {
		case <-sl.ctx.Done():
			close(sl.stopped)
			return
		case <-sl.scrapeCtx.Done():
			break mainLoop
		case <-ticker.C:
		}
	}

	close(sl.stopped)
}

// Stop the scraping. May still write data and stale markers after it has
// returned. Cancel the context to stop all writes.
func (sl *scrapeLoop) stop() {
	sl.cancel()
	<-sl.stopped
}
