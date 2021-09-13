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

func (s *InMemoryProfileMetaStore) GetLocationByID(ctx context.Context, id uint64) (*profile.Location, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if uint64(len(s.locations)) <= id-1 {
		return nil, ErrLocationNotFound
	}

	return s.locations[id-1], nil
}

func (s *InMemoryProfileMetaStore) GetLocationsByIDs(ctx context.Context, ids ...uint64) (map[uint64]*profile.Location, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res := map[uint64]*profile.Location{}

	for _, id := range ids {
		if uint64(len(s.locations)) <= id-1 {
			return nil, ErrLocationNotFound
		}
		res[id] = s.locations[id-1]
	}

	return res, nil
}

func (s *InMemoryProfileMetaStore) GetLocationByKey(ctx context.Context, key LocationKey) (*profile.Location, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	i, found := s.locationsByKey[key]
	if !found {
		return nil, ErrLocationNotFound
	}

	return s.locations[i-1], nil
}

func (s *InMemoryProfileMetaStore) GetLocations(ctx context.Context) ([]*profile.Location, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*profile.Location, len(s.locations))
	copy(res, s.locations)

	return res, nil
}

func (s *InMemoryProfileMetaStore) GetSymbolizableLocations(ctx context.Context) ([]*profile.Location, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

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

func (s *InMemoryProfileMetaStore) CreateLocation(ctx context.Context, l *profile.Location) (uint64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	key := MakeLocationKey(l)
	id := uint64(len(s.locations)) + 1
	l.ID = id
	s.locations = append(s.locations, l)
	s.locationsByKey[key] = id
	return id, nil
}

func (s *InMemoryProfileMetaStore) Symbolize(ctx context.Context, l *profile.Location) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	loc, err := s.GetLocationByID(ctx, l.ID)
	if err != nil {
		return err
	}

	s.mu.Lock()
	loc.Line = l.Line
	loc.Address = l.Address
	loc.Mapping = l.Mapping
	s.mu.Unlock()

	for _, ln := range l.Line {
		_, err := s.CreateFunction(ctx, ln.Function)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *InMemoryProfileMetaStore) GetMappingByKey(ctx context.Context, key MappingKey) (*profile.Mapping, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	i, found := s.mappingsByKey[key]
	if !found {
		return nil, ErrMappingNotFound
	}

	return s.mappings[i-1], nil
}

func (s *InMemoryProfileMetaStore) CreateMapping(ctx context.Context, m *profile.Mapping) (uint64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := MakeMappingKey(m)
	id := uint64(len(s.mappings)) + 1
	m.ID = id
	s.mappings = append(s.mappings, m)
	s.mappingsByKey[key] = id
	return id, nil
}

func (s *InMemoryProfileMetaStore) GetFunctionByKey(ctx context.Context, key FunctionKey) (*profile.Function, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	i, found := s.functionsByKey[key]
	if !found {
		return nil, ErrFunctionNotFound
	}

	return s.functions[i-1], nil
}

func (s *InMemoryProfileMetaStore) GetFunctions(ctx context.Context) ([]*profile.Function, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	res := make([]*profile.Function, len(s.functions))
	copy(res, s.functions)

	return res, nil
}

func (s *InMemoryProfileMetaStore) CreateFunction(ctx context.Context, f *profile.Function) (uint64, error) {
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := MakeFunctionKey(f)
	id := uint64(len(s.functions)) + 1
	f.ID = id
	s.functions = append(s.functions, f)
	s.functionsByKey[key] = id
	return id, nil
}

func (s *InMemoryProfileMetaStore) Close() error {
	return nil
}

func (s *InMemoryProfileMetaStore) Ping() error {
	return nil
}
