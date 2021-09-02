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
	"sync"

	"github.com/google/pprof/profile"
)

var _ ProfileMetaStore = &InMemoryProfileMetaStore{}

type InMemoryProfileMetaStore struct {
	mu             sync.RWMutex
	locationsByKey map[LocationKey]uint64
	locations      []*profile.Location
	mappingsByKey  map[MappingKey]uint64
	mappings       []*profile.Mapping
	functionsByKey map[FunctionKey]uint64
	functions      []*profile.Function
}

func NewInMemoryProfileMetaStore() (*InMemoryProfileMetaStore, error) {
	return &InMemoryProfileMetaStore{
		locationsByKey: map[LocationKey]uint64{},
		functionsByKey: map[FunctionKey]uint64{},
		mappingsByKey:  map[MappingKey]uint64{},
	}, nil
}

func (s *InMemoryProfileMetaStore) GetLocationByID(id uint64) (*profile.Location, error) {
	if uint64(len(s.locations)) <= id-1 {
		return nil, ErrLocationNotFound
	}

	return s.locations[id-1], nil
}

func (s *InMemoryProfileMetaStore) GetLocationByKey(key LocationKey) (*profile.Location, error) {
	s.mu.RLock()
	i, found := s.locationsByKey[key]
	s.mu.RUnlock()
	if !found {
		return nil, ErrLocationNotFound
	}

	return s.locations[i-1], nil
}

func (s *InMemoryProfileMetaStore) GetLocations() ([]*profile.Location, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.locations, nil
}

func (s *InMemoryProfileMetaStore) GetUnsymbolizedLocations() ([]*profile.Location, error) {
	locs := []*profile.Location{}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, loc := range s.locations {
		if len(loc.Line) == 0 {
			locs = append(locs, loc)
		}
	}
	return locs, nil
}

func (s *InMemoryProfileMetaStore) CreateLocation(l *profile.Location) (uint64, error) {
	key := MakeLocationKey(l)
	id := uint64(len(s.locations)) + 1
	l.ID = id
	s.locations = append(s.locations, l)
	s.mu.Lock()
	s.locationsByKey[key] = id
	s.mu.Unlock()
	return id, nil
}

func (s *InMemoryProfileMetaStore) UpdateLocation(l *profile.Location) error {
	loc, err := s.GetLocationByID(l.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	loc.Line = l.Line
	loc.Address = l.Address
	loc.Mapping = l.Mapping
	s.mu.Unlock()

	for _, ln := range l.Line {
		_, err := s.CreateFunction(ln.Function)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *InMemoryProfileMetaStore) GetMappingByKey(key MappingKey) (*profile.Mapping, error) {
	s.mu.RLock()
	i, found := s.mappingsByKey[key]
	s.mu.RUnlock()
	if !found {
		return nil, ErrMappingNotFound
	}

	return s.mappings[i-1], nil
}

func (s *InMemoryProfileMetaStore) CreateMapping(m *profile.Mapping) (uint64, error) {
	key := MakeMappingKey(m)
	id := uint64(len(s.mappings)) + 1
	m.ID = id
	s.mappings = append(s.mappings, m)
	s.mu.Lock()
	s.mappingsByKey[key] = id
	s.mu.Unlock()
	return id, nil
}

func (s *InMemoryProfileMetaStore) GetFunctionByKey(key FunctionKey) (*profile.Function, error) {
	s.mu.RLock()
	i, found := s.functionsByKey[key]
	s.mu.RUnlock()
	if !found {
		return nil, ErrFunctionNotFound
	}

	return s.functions[i-1], nil
}

func (s *InMemoryProfileMetaStore) GetFunctions() ([]*profile.Function, error) {
	return s.functions, nil
}

func (s *InMemoryProfileMetaStore) CreateFunction(f *profile.Function) (uint64, error) {
	key := MakeFunctionKey(f)
	id := uint64(len(s.functions)) + 1
	f.ID = id
	s.functions = append(s.functions, f)
	s.mu.Lock()
	s.functionsByKey[key] = id
	s.mu.Unlock()
	return id, nil
}

func (s *InMemoryProfileMetaStore) Close() error {
	return nil
}

func (s *InMemoryProfileMetaStore) Ping() error {
	return nil
}
