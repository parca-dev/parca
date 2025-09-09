// Copyright 2022-2025 The Parca Authors
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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/version"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/util/pool"
	"golang.org/x/net/context/ctxhttp"

	profilepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/config"
)

// scrapePool manages scrapes for sets of targets.
type scrapePool struct {
	store   profilepb.ProfileStoreServiceServer
	logger  log.Logger
	metrics *scrapePoolMetrics

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

type scrapePoolMetrics struct {
	targetIntervalLength          *prometheus.SummaryVec
	targetReloadIntervalLength    *prometheus.SummaryVec
	targetSyncIntervalLength      *prometheus.SummaryVec
	targetScrapePoolSyncsCounter  *prometheus.CounterVec
	targetScrapeSampleLimit       prometheus.Counter
	targetScrapeSampleDuplicate   prometheus.Counter
	targetScrapeSampleOutOfOrder  prometheus.Counter
	targetScrapeSampleOutOfBounds prometheus.Counter
}

func newScrapePool(
	cfg *config.ScrapeConfig,
	store profilepb.ProfileStoreServiceServer,
	logger log.Logger,
	externalLabels labels.Labels,
	metrics *scrapePoolMetrics,
) *scrapePool {
	if logger == nil {
		logger = log.NewNopLogger()
	}

	client, err := commonconfig.NewClientFromConfig(cfg.HTTPClientConfig, cfg.JobName)
	if err != nil {
		// Any errors that could occur here should be caught during config validation.
		level.Error(logger).Log("msg", "Error creating HTTP client", "err", err)
	}

	buffers := pool.New(1e3, 100e6, 3, func(sz int) interface{} { return make([]byte, 0, sz) })

	ctx, cancel := context.WithCancel(context.Background())
	sp := &scrapePool{
		cancel:        cancel,
		store:         store,
		config:        cfg,
		client:        client,
		activeTargets: map[uint64]*Target{},
		loops:         map[uint64]loop{},
		logger:        logger,
		metrics:       metrics,
	}
	sp.newLoop = func(t *Target, s scraper) loop {
		return newScrapeLoop(
			ctx,
			t,
			s,
			log.With(logger, "target", t),
			externalLabels,
			sp.metrics.targetIntervalLength,
			buffers,
			store,
			cfg.NormalizedAddresses,
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

	client, err := commonconfig.NewClientFromConfig(cfg.HTTPClientConfig, cfg.JobName)
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
			s       = &targetScraper{Target: t, logger: sp.logger, client: sp.client, timeout: timeout}
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
	sp.metrics.targetReloadIntervalLength.WithLabelValues(interval.String()).Observe(
		time.Since(start).Seconds(),
	)
}

// Sync converts target groups into actual scrape targets and synchronizes
// the currently running scraper with the resulting set and returns all scraped and dropped targets.
func (sp *scrapePool) Sync(tgs []*targetgroup.Group) {
	start := time.Now()

	var all []*Target
	var targets []*Target
	lb := labels.NewBuilder(labels.EmptyLabels())
	sp.mtx.Lock()
	sp.droppedTargets = []*Target{}
	for _, tg := range tgs {
		targets, err := targetsFromGroup(tg, sp.config, targets, lb)
		if err != nil {
			level.Error(sp.logger).Log("msg", "creating targets failed", "err", err)
			continue
		}

		for _, t := range targets {
			// Replicate .Labels().IsEmpty() with a loop here to avoid generating garbage.
			nonEmpty := false
			t.LabelsRange(func(l labels.Label) { nonEmpty = true })
			if nonEmpty {
				all = append(all, t)
			} else if !t.discoveredLabels.IsEmpty() {
				sp.droppedTargets = append(sp.droppedTargets, t)
			}
		}
	}
	sp.mtx.Unlock()
	sp.sync(all)

	sp.metrics.targetSyncIntervalLength.WithLabelValues(sp.config.JobName).Observe(
		time.Since(start).Seconds(),
	)
	sp.metrics.targetScrapePoolSyncsCounter.WithLabelValues(sp.config.JobName).Inc()
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
		if resp.StatusCode == http.StatusPermanentRedirect {
			return fmt.Errorf("server is being redirected with HTTP status %s, please add the destination path", resp.Status)
		}

		return fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	switch profileType {
	case ProfileTraceType:
		return fmt.Errorf("unimplemented")
	default:
		b, err := io.ReadAll(io.TeeReader(resp.Body, w))
		if err != nil {
			return fmt.Errorf("failed to read body: %w", err)
		}

		if len(b) == 0 {
			return fmt.Errorf("empty %s profile from %s", profileType, s.req.URL.String())
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
	intervalLength *prometheus.SummaryVec
	lastScrapeSize int
	externalLabels labels.Labels

	normalizedAddresses bool

	buffers *pool.Pool

	store     profilepb.ProfileStoreServiceServer
	ctx       context.Context
	scrapeCtx context.Context
	cancel    func()
	stopped   chan struct{}
}

func newScrapeLoop(ctx context.Context,
	t *Target,
	sc scraper,
	l log.Logger,
	externalLabels labels.Labels,
	targetIntervalLength *prometheus.SummaryVec,
	buffers *pool.Pool,
	store profilepb.ProfileStoreServiceServer,
	normalizedAddresses bool,
) *scrapeLoop {
	if l == nil {
		l = log.NewNopLogger()
	}
	if buffers == nil {
		buffers = pool.New(1e3, 1e6, 3, func(sz int) interface{} { return make([]byte, 0, sz) })
	}
	sl := &scrapeLoop{
		target:              t,
		scraper:             sc,
		buffers:             buffers,
		store:               store,
		stopped:             make(chan struct{}),
		l:                   l,
		externalLabels:      externalLabels,
		intervalLength:      targetIntervalLength,
		ctx:                 ctx,
		normalizedAddresses: normalizedAddresses,
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
			sl.intervalLength.WithLabelValues(interval.String()).Observe(
				time.Since(last).Seconds(),
			)
		}

		b := sl.buffers.Get(sl.lastScrapeSize).([]byte)
		buf := bytes.NewBuffer(b)

		profileType := sl.target.labels.Get(ProfileName)

		scrapeCtx, cancel := context.WithTimeout(sl.ctx, timeout)
		scrapeErr := sl.scraper.scrape(scrapeCtx, buf, profileType)
		cancel()

		if scrapeErr == nil {
			err := processScrapeResp(buf, sl, profileType)
			if err != nil {
				if errc != nil {
					errc <- err
				}
				sl.target.health = HealthBad
				sl.target.lastError = err
			} else {
				sl.target.health = HealthGood
			}

			sl.target.lastScrapeDuration = time.Since(start)
		} else {
			level.Debug(sl.l).Log("msg", "Scrape failed", "err", scrapeErr.Error())
			if errc != nil {
				errc <- scrapeErr
			}

			sl.target.health = HealthBad
			sl.target.lastScrapeDuration = time.Since(start)
			sl.target.lastError = scrapeErr
		}

		last = start
		sl.target.lastScrape = last

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

func processScrapeResp(buf *bytes.Buffer, sl *scrapeLoop, profileType string) error {
	b := buf.Bytes()
	defer sl.buffers.Put(b)
	// NOTE: There were issues with misbehaving clients in the past
	// that occasionally returned empty results. We don't want those
	// to falsely reset our buffer size.
	if len(b) > 0 {
		sl.lastScrapeSize = len(b)
	}

	tl := labels.NewBuilder(sl.target.Labels())
	tl.Set("__name__", profileType)
	sl.externalLabels.Range(func(l labels.Label) {
		tl.Set(l.Name, l.Value)
	})

	protolbls := &profilepb.LabelSet{
		Labels: []*profilepb.Label{},
	}
	tl.Range(func(l labels.Label) {
		protolbls.Labels = append(protolbls.Labels, &profilepb.Label{
			Name:  l.Name,
			Value: l.Value,
		})
	})

	byt := buf.Bytes()
	p, err := profile.ParseData(byt)
	if err != nil {
		level.Error(sl.l).Log("msg", "failed to parse profile data", "err", err)
		return err
	}

	var executableInfo []*profilepb.ExecutableInfo
	for _, comment := range p.Comments {
		if strings.HasPrefix(comment, "executableInfo=") {
			ei, err := parseExecutableInfo(comment)
			if err != nil {
				level.Error(sl.l).Log("msg", "failed to parse executableInfo", "err", err)
				continue
			}

			executableInfo = append(executableInfo, ei)
		}
	}

	ks := sl.target.KeepSet()
	if len(ks) > 0 {
		keepIndexes := []int{}
		newTypes := []*profile.ValueType{}
		for i, st := range p.SampleType {
			if _, ok := ks[config.SampleType{Type: st.Type, Unit: st.Unit}]; ok {
				keepIndexes = append(keepIndexes, i)
				newTypes = append(newTypes, st)
			}
		}
		p.SampleType = newTypes
		for _, s := range p.Sample {
			newValues := []int64{}
			for _, i := range keepIndexes {
				newValues = append(newValues, s.Value[i])
			}
			s.Value = newValues
		}
		p = p.Compact()
		sl.buffers.Put(b)
		b = sl.buffers.Get(sl.lastScrapeSize).([]byte)
		newBuf := bytes.NewBuffer(b)
		if err := p.Write(newBuf); err != nil {
			level.Error(sl.l).Log("msg", "failed to write profile data", "err", err)
			return err
		}
		byt = newBuf.Bytes()
	}

	_, err = sl.store.WriteRaw(sl.ctx, &profilepb.WriteRawRequest{
		Normalized: sl.normalizedAddresses,
		Series: []*profilepb.RawProfileSeries{
			{
				Labels: protolbls,
				Samples: []*profilepb.RawSample{
					{
						RawProfile:     byt,
						ExecutableInfo: executableInfo,
					},
				},
			},
		},
	})
	return err
}

// parseExecutableInfo parses the executableInfo string from the comment. It is in the format of: "executableInfo=elfType;offset;vaddr".
func parseExecutableInfo(comment string) (*profilepb.ExecutableInfo, error) {
	eiString := strings.TrimPrefix(comment, "executableInfo=")
	eiParts := strings.Split(eiString, ";")
	if len(eiParts) == 0 {
		return nil, errors.New("executableInfo string is empty")
	}

	var (
		res = &profilepb.ExecutableInfo{}
		err error
	)

	if len(eiParts) >= 1 {
		elfType, err := strconv.ParseUint(strings.TrimPrefix(eiParts[0], "0x"), 16, 32)
		if err != nil {
			return nil, fmt.Errorf("parse elfType: %w", err)
		}

		res.ElfType = uint32(elfType)
	}

	if len(eiParts) == 3 {
		res.LoadSegment = &profilepb.LoadSegment{}
		res.LoadSegment.Offset, err = strconv.ParseUint(strings.TrimPrefix(eiParts[1], "0x"), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parse load segment offset: %w", err)
		}

		res.LoadSegment.Vaddr, err = strconv.ParseUint(strings.TrimPrefix(eiParts[2], "0x"), 16, 64)
		if err != nil {
			return nil, fmt.Errorf("parse load segment vaddr: %w", err)
		}
	}

	return res, nil
}

// Stop the scraping. May still write data and stale markers after it has
// returned. Cancel the context to stop all writes.
func (sl *scrapeLoop) stop() {
	sl.cancel()
	<-sl.stopped
}
