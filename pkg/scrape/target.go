// Copyright 2022 The Parca Authors
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
	"errors"
	"fmt"
	"hash/fnv"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/relabel"

	"github.com/parca-dev/parca/pkg/config"
)

// TargetHealth describes the health state of a target.
type TargetHealth string

// The possible health states of a target based on the last performed scrape.
const (
	HealthUnknown TargetHealth = "unknown"
	HealthGood    TargetHealth = "up"
	HealthBad     TargetHealth = "down"
)

// Target refers to a singular HTTP or HTTPS endpoint.
type Target struct {
	// Labels before any processing.
	discoveredLabels labels.Labels
	// Any labels that are added to this target and its metrics.
	labels labels.Labels
	// Additional URL parmeters that are part of the target URL.
	params url.Values

	mtx                sync.RWMutex
	lastError          error
	lastScrape         time.Time
	lastScrapeDuration time.Duration
	health             TargetHealth
}

// NewTarget creates a reasonably configured target for querying.
func NewTarget(labels, discoveredLabels labels.Labels, params url.Values) *Target {
	return &Target{
		labels:           labels,
		discoveredLabels: discoveredLabels,
		params:           params,
		health:           HealthUnknown,
	}
}

func (t *Target) String() string {
	return t.URL().String()
}

// hash returns an identifying hash for the target.
func (t *Target) hash() uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(fmt.Sprintf("%016d", t.labels.Hash())))
	_, _ = h.Write([]byte(t.URL().String()))
	return h.Sum64()
}

// offset returns the time until the next scrape cycle for the target.
func (t *Target) offset(interval time.Duration) time.Duration {
	now := time.Now().UnixNano()

	var (
		base   = now % int64(interval)
		offset = t.hash() % uint64(interval)
		next   = base + int64(offset)
	)

	if next > int64(interval) {
		next -= int64(interval)
	}
	return time.Duration(next)
}

// Params returns a copy of the set of all public params of the target.
func (t *Target) Params() url.Values {
	q := make(url.Values, len(t.params))
	for k, values := range t.params {
		q[k] = make([]string, len(values))
		copy(q[k], values)
	}
	return q
}

// Labels returns a copy of the set of all public labels of the target.
func (t *Target) Labels() labels.Labels {
	b := labels.NewScratchBuilder(t.labels.Len())
	t.labels.Range(func(l labels.Label) {
		if !strings.HasPrefix(l.Name, model.ReservedLabelPrefix) {
			b.Add(l.Name, l.Value)
		}
	})
	return b.Labels()
}

// DiscoveredLabels returns a copy of the target's labels before any processing.
func (t *Target) DiscoveredLabels() labels.Labels {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	return t.discoveredLabels.Copy()
}

// Clone returns a clone of the target.
func (t *Target) Clone() *Target {
	return NewTarget(
		t.Labels(),
		t.DiscoveredLabels(),
		t.Params(),
	)
}

// SetDiscoveredLabels sets new DiscoveredLabels.
func (t *Target) SetDiscoveredLabels(l labels.Labels) {
	t.mtx.Lock()
	defer t.mtx.Unlock()
	t.discoveredLabels = l
}

// URL returns a copy of the target's URL.
func (t *Target) URL() *url.URL {
	params := url.Values{}

	for k, v := range t.params {
		params[k] = make([]string, len(v))
		copy(params[k], v)
	}
	t.labels.Range(func(l labels.Label) {
		if !strings.HasPrefix(l.Name, model.ParamLabelPrefix) {
			return
		}
		ks := l.Name[len(model.ParamLabelPrefix):]

		if len(params[ks]) > 0 {
			params[ks][0] = l.Value
		} else {
			params[ks] = []string{l.Value}
		}
	})

	return &url.URL{
		Scheme:   t.labels.Get(model.SchemeLabel),
		Host:     t.labels.Get(model.AddressLabel),
		Path:     t.labels.Get(ProfilePath),
		RawQuery: params.Encode(),
	}
}

// LastError returns the error encountered during the last scrape.
func (t *Target) LastError() error {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.lastError
}

// LastScrape returns the time of the last scrape.
func (t *Target) LastScrape() time.Time {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.lastScrape
}

// LastScrapeDuration returns how long the last scrape of the target took.
func (t *Target) LastScrapeDuration() time.Duration {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.lastScrapeDuration
}

// Health returns the last known health state of the target.
func (t *Target) Health() TargetHealth {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	return t.health
}

// LabelsByProfiles returns the labels for a given ProfilingConfig.
func LabelsByProfiles(lset labels.Labels, c *config.ProfilingConfig) []labels.Labels {
	res := []labels.Labels{}

	if len(c.PprofConfig) > 0 {
		b := labels.NewScratchBuilder(lset.Len() + 2)
		for profilingType, profilingConfig := range c.PprofConfig {
			if *profilingConfig.Enabled {
				b.Assign(lset)
				b.Add(ProfilePath, profilingConfig.Path)
				b.Add(ProfileName, profilingType)
				res = append(res, b.Labels())
			}
		}
	}

	return res
}

// Targets is a sortable list of targets.
type Targets []*Target

func (ts Targets) Len() int           { return len(ts) }
func (ts Targets) Less(i, j int) bool { return ts[i].URL().String() < ts[j].URL().String() }
func (ts Targets) Swap(i, j int)      { ts[i], ts[j] = ts[j], ts[i] }

const (
	ProfilePath      = "__profile_path__"
	ProfileName      = "__name__"
	ProfileTraceType = "trace"
)

// populateLabels builds a label set from the given label set and scrape configuration.
// It returns a label set before relabeling was applied as the second return value.
// Returns the original discovered label set found before relabelling was applied if the target is dropped during relabeling.
func populateLabels(lset labels.Labels, cfg *config.ScrapeConfig) (res, orig labels.Labels, err error) {
	// Copy labels into the labelset for the target if they are not set already.
	scrapeLabels := labels.Labels{
		{Name: model.JobLabel, Value: cfg.JobName},
		{Name: model.SchemeLabel, Value: cfg.Scheme},
	}
	lb := labels.NewBuilder(lset)

	scrapeLabels.Range(func(l labels.Label) {
		if lv := lset.Get(l.Name); lv == "" {
			lb.Set(l.Name, l.Value)
		}
	})
	// Encode scrape query parameters as labels.
	for k, v := range cfg.Params {
		if len(v) > 0 {
			lb.Set(model.ParamLabelPrefix+k, v[0])
		}
	}

	preRelabelLabels := lb.Labels(labels.EmptyLabels())
	lset, keep := relabel.Process(preRelabelLabels, cfg.RelabelConfigs...)

	// Check if the target was dropped.
	if !keep {
		return labels.EmptyLabels(), preRelabelLabels, nil
	}
	if v := lset.Get(model.AddressLabel); v == "" {
		return labels.EmptyLabels(), labels.EmptyLabels(), errors.New("no address")
	}

	lb = labels.NewBuilder(lset)

	// addPort checks whether we should add a default port to the address.
	// If the address is not valid, we don't append a port either.
	addPort := func(s string) bool {
		// If we can split, a port exists and we don't have to add one.
		if _, _, err := net.SplitHostPort(s); err == nil {
			return false
		}
		// If adding a port makes it valid, the previous error
		// was not due to an invalid address and we can append a port.
		_, _, err := net.SplitHostPort(s + ":1234")
		return err == nil
	}
	addr := lset.Get(model.AddressLabel)
	// If it's an address with no trailing port, infer it based on the used scheme.
	if addPort(addr) {
		// Addresses reaching this point are already wrapped in [] if necessary.
		switch lset.Get(model.SchemeLabel) {
		case "http", "":
			addr = addr + ":80"
		case "https":
			addr = addr + ":443"
		default:
			return labels.EmptyLabels(), labels.EmptyLabels(), fmt.Errorf("invalid scheme: %q", cfg.Scheme)
		}
		lb.Set(model.AddressLabel, addr)
	}

	if err := config.CheckTargetAddress(model.LabelValue(addr)); err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}

	// Meta labels are deleted after relabelling. Other internal labels propagate to
	// the target which decides whether they will be part of their label set.
	lset.Range(func(l labels.Label) {
		if strings.HasPrefix(l.Name, model.MetaLabelPrefix) {
			lb.Del(l.Name)
		}
	})

	// Default the instance label to the target address.
	if v := lset.Get(model.InstanceLabel); v == "" {
		lb.Set(model.InstanceLabel, addr)
	}

	res = lb.Labels(labels.EmptyLabels())
	err = res.Validate(func(l labels.Label) error {
		// Check label values are valid, drop the target if not.
		if !model.LabelValue(l.Value).IsValid() {
			return fmt.Errorf("invalid label value for %q: %q", l.Name, l.Value)
		}
		return nil
	})
	if err != nil {
		return labels.EmptyLabels(), labels.EmptyLabels(), err
	}

	return res, lset, nil
}

// targetsFromGroup builds targets based on the given TargetGroup and config.
func targetsFromGroup(tg *targetgroup.Group, cfg *config.ScrapeConfig) ([]*Target, error) {
	targets := make([]*Target, 0, len(tg.Targets))

	for i, tlset := range tg.Targets {
		b := labels.NewScratchBuilder(len(tlset) + len(tg.Labels))

		for ln, lv := range tlset {
			b.Add(string(ln), string(lv))
		}
		for ln, lv := range tg.Labels {
			if _, ok := tlset[ln]; !ok {
				b.Add(string(ln), string(lv))
			}
		}

		lset := labels.New(b.Labels()...)
		lsets := LabelsByProfiles(lset, cfg.ProfilingConfig)

		for _, lset := range lsets {
			var profType string
			lset.Range(func(l labels.Label) {
				if l.Name == ProfileName {
					profType = l.Value
				}
			})
			lset, origLabels, err := populateLabels(lset, cfg)
			if err != nil {
				return nil, fmt.Errorf("instance %d in group %s: %s", i, tg, err)
			}
			if !lset.IsEmpty() || !origLabels.IsEmpty() {
				params := cfg.Params
				if params == nil {
					params = url.Values{}
				}

				if pcfg, found := cfg.ProfilingConfig.PprofConfig[profType]; found && pcfg.Delta {
					params.Add("seconds", strconv.Itoa(int(time.Duration(cfg.ScrapeInterval)/time.Second)))
				}

				targets = append(targets, NewTarget(lset, origLabels, params))
			}
		}
	}

	return targets, nil
}
