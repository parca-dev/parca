package metastore

import (
	"sync"

	"github.com/google/pprof/profile"
)

type metaStoreCache struct {
	mappingsMtx   *sync.RWMutex
	mappingsByID  map[uint64]profile.Mapping
	mappingsByKey map[MappingKey]uint64

	functionsMtx   *sync.RWMutex
	functionsByID  map[uint64]profile.Function
	functionsByKey map[FunctionKey]uint64

	locationLinesMtx  *sync.RWMutex
	locationLinesByID map[uint64][]locationLine
}

func newMetaStoreCache() *metaStoreCache {
	return &metaStoreCache{
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

func (c *metaStoreCache) getMappingByKey(k MappingKey) (profile.Mapping, bool) {
	c.mappingsMtx.RLock()
	defer c.mappingsMtx.RUnlock()

	id, found := c.mappingsByKey[k]
	if !found {
		return profile.Mapping{}, false
	}

	m, found := c.mappingsByID[id]
	return m, found
}

func (c *metaStoreCache) getMappingByID(id uint64) (profile.Mapping, bool) {
	c.mappingsMtx.RLock()
	defer c.mappingsMtx.RUnlock()

	m, found := c.mappingsByID[id]
	return m, found
}

func (c *metaStoreCache) setMappingByKey(k MappingKey, m profile.Mapping) {
	c.mappingsMtx.Lock()
	defer c.mappingsMtx.Unlock()

	c.mappingsByID[m.ID] = m
	c.mappingsByKey[k] = m.ID
}

func (c *metaStoreCache) setMappingByID(m profile.Mapping) {
	c.mappingsMtx.Lock()
	defer c.mappingsMtx.Unlock()

	c.mappingsByID[m.ID] = m
}

func (c *metaStoreCache) getFunctionByKey(k FunctionKey) (profile.Function, bool) {
	c.functionsMtx.RLock()
	defer c.functionsMtx.RUnlock()

	id, found := c.functionsByKey[k]
	if !found {
		return profile.Function{}, false
	}

	fn, found := c.functionsByID[id]
	return fn, found
}

func (c *metaStoreCache) setFunctionByKey(k FunctionKey, f profile.Function) {
	c.functionsMtx.Lock()
	defer c.functionsMtx.Unlock()

	c.functionsByID[f.ID] = f
	c.functionsByKey[k] = f.ID
}

func (c *metaStoreCache) setLocationLinesByID(locationID uint64, ll []locationLine) {
	v := make([]locationLine, 0, len(ll))
	for _, l := range ll {
		v = append(v, l)
	}

	c.locationLinesMtx.Lock()
	defer c.locationLinesMtx.Unlock()

	c.locationLinesByID[locationID] = ll
}

func (c *metaStoreCache) getLocationLinesByID(locationID uint64) ([]locationLine, bool) {
	c.locationLinesMtx.RLock()
	defer c.locationLinesMtx.RUnlock()

	ll, found := c.locationLinesByID[locationID]
	if !found {
		return nil, false
	}

	v := make([]locationLine, 0, len(ll))
	for _, l := range ll {
		v = append(v, l)
	}

	return v, true
}
