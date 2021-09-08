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

package metastore

import (
	"context"
	"sync"

	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
)

type metaStoreCache struct {
	metrics *metrics

	mappingsMtx   *sync.RWMutex
	mappingsByID  map[uint64]profile.Mapping
	mappingsByKey map[MappingKey]uint64

	functionsMtx   *sync.RWMutex
	functionsByID  map[uint64]profile.Function
	functionsByKey map[FunctionKey]uint64

	locationLinesMtx  *sync.RWMutex
	locationLinesByID map[uint64][]locationLine
}

type metrics struct {
	idHits    *prometheus.CounterVec
	idMisses  *prometheus.CounterVec
	keyHits   *prometheus.CounterVec
	keyMisses *prometheus.CounterVec
}

func newMetaStoreCacheMetrics(reg prometheus.Registerer) *metrics {
	m := &metrics{
		idHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parca_metastore_cache_id_hits_total",
				Help: "Number of cache hits for id lookups.",
			},
			[]string{"item_type"},
		),
		idMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parca_metastore_cache_id_misses_total",
				Help: "Number of cache misses for id lookups.",
			},
			[]string{"item_type"},
		),
		keyHits: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parca_metastore_cache_key_hits_total",
				Help: "Number of cache hits for key lookups.",
			},
			[]string{"item_type"},
		),
		keyMisses: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "parca_metastore_cache_key_misses_total",
				Help: "Number of cache misses for key lookups.",
			},
			[]string{"item_type"},
		),
	}

	if reg != nil {
		reg.MustRegister(m.idHits)
		reg.MustRegister(m.idMisses)
		reg.MustRegister(m.keyHits)
		reg.MustRegister(m.keyMisses)
	}

	return m
}

func newMetaStoreCache(reg prometheus.Registerer) *metaStoreCache {
	return &metaStoreCache{
		metrics: newMetaStoreCacheMetrics(reg),

		mappingsMtx:   &sync.RWMutex{},
		mappingsByID:  map[uint64]profile.Mapping{},
		mappingsByKey: map[MappingKey]uint64{},

		functionsMtx:   &sync.RWMutex{},
		functionsByID:  map[uint64]profile.Function{},
		functionsByKey: map[FunctionKey]uint64{},

		locationLinesMtx:  &sync.RWMutex{},
		locationLinesByID: map[uint64][]locationLine{},
	}
}

func (c *metaStoreCache) getMappingByKey(ctx context.Context, k MappingKey) (profile.Mapping, bool, error) {
	select {
	case <-ctx.Done():
		return profile.Mapping{}, false, ctx.Err()
	default:
	}

	c.mappingsMtx.RLock()
	defer c.mappingsMtx.RUnlock()

	id, found := c.mappingsByKey[k]
	if !found {
		c.metrics.keyMisses.WithLabelValues("mapping").Inc()
		return profile.Mapping{}, false, nil
	}

	m, found := c.mappingsByID[id]
	if !found {
		c.metrics.keyMisses.WithLabelValues("mapping").Inc()
		return profile.Mapping{}, false, nil
	}

	c.metrics.keyHits.WithLabelValues("mapping").Inc()
	return m, found, nil
}

func (c *metaStoreCache) getMappingByID(ctx context.Context, id uint64) (profile.Mapping, bool, error) {
	select {
	case <-ctx.Done():
		return profile.Mapping{}, false, ctx.Err()
	default:
	}

	c.mappingsMtx.RLock()
	defer c.mappingsMtx.RUnlock()

	m, found := c.mappingsByID[id]
	if !found {
		c.metrics.idHits.WithLabelValues("mapping").Inc()
		return profile.Mapping{}, false, nil
	}

	c.metrics.idHits.WithLabelValues("mapping").Inc()
	return m, found, nil
}

func (c *metaStoreCache) setMappingByKey(ctx context.Context, k MappingKey, m profile.Mapping) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mappingsMtx.Lock()
	defer c.mappingsMtx.Unlock()

	c.mappingsByID[m.ID] = m
	c.mappingsByKey[k] = m.ID

	return nil
}

func (c *metaStoreCache) setMappingByID(ctx context.Context, m profile.Mapping) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.mappingsMtx.Lock()
	defer c.mappingsMtx.Unlock()

	c.mappingsByID[m.ID] = m

	return nil
}

func (c *metaStoreCache) getFunctionByKey(ctx context.Context, k FunctionKey) (profile.Function, bool, error) {
	select {
	case <-ctx.Done():
		return profile.Function{}, false, ctx.Err()
	default:
	}

	c.functionsMtx.RLock()
	defer c.functionsMtx.RUnlock()

	id, found := c.functionsByKey[k]
	if !found {
		c.metrics.keyMisses.WithLabelValues("function").Inc()
		return profile.Function{}, false, nil
	}

	fn, found := c.functionsByID[id]
	if !found {
		c.metrics.keyMisses.WithLabelValues("function").Inc()
		return profile.Function{}, false, nil
	}

	c.metrics.keyHits.WithLabelValues("function").Inc()
	return fn, found, nil
}

func (c *metaStoreCache) setFunctionByKey(ctx context.Context, k FunctionKey, f profile.Function) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.functionsMtx.Lock()
	defer c.functionsMtx.Unlock()

	c.functionsByID[f.ID] = f
	c.functionsByKey[k] = f.ID

	return nil
}

func (c *metaStoreCache) getFunctionByID(ctx context.Context, functionID uint64) (profile.Function, bool, error) {
	select {
	case <-ctx.Done():
		return profile.Function{}, false, ctx.Err()
	default:
	}

	c.functionsMtx.RLock()
	defer c.functionsMtx.RUnlock()

	f, found := c.functionsByID[functionID]
	if !found {
		c.metrics.idMisses.WithLabelValues("function").Inc()
		return profile.Function{}, false, nil
	}

	c.metrics.idHits.WithLabelValues("function").Inc()
	return f, found, nil
}

func (c *metaStoreCache) setFunctionByID(ctx context.Context, f profile.Function) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.functionsMtx.Lock()
	defer c.functionsMtx.Unlock()

	c.functionsByID[f.ID] = f
	return nil
}

func (c *metaStoreCache) setLocationLinesByID(ctx context.Context, locationID uint64, ll []locationLine) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	v := make([]locationLine, len(ll))
	copy(v, ll)

	c.locationLinesMtx.Lock()
	defer c.locationLinesMtx.Unlock()

	c.locationLinesByID[locationID] = v

	return nil
}

func (c *metaStoreCache) getLocationLinesByID(ctx context.Context, locationID uint64) ([]locationLine, bool, error) {
	select {
	case <-ctx.Done():
		return nil, false, ctx.Err()
	default:
	}

	c.locationLinesMtx.RLock()
	defer c.locationLinesMtx.RUnlock()

	ll, found := c.locationLinesByID[locationID]
	if !found {
		c.metrics.idMisses.WithLabelValues("location_lines").Inc()
		return nil, false, nil
	}

	v := make([]locationLine, len(ll))
	copy(v, ll)

	c.metrics.idHits.WithLabelValues("location_lines").Inc()
	return v, true, nil
}
