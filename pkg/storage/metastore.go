package storage

import (
	"errors"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
)

var (
	ErrLocationNotFound = errors.New("location not found")
	ErrMappingNotFound  = errors.New("mapping not found")
	ErrFunctionNotFound = errors.New("function not found")
)

type ProfileMetaStore interface {
	LocationStore
	FunctionStore
	MappingStore
}

type LocationStore interface {
	GetLocationByKey(k LocationKey) (*profile.Location, error)
	GetLocationByID(id uint64) (*profile.Location, error)
	CreateLocation(l *profile.Location)
}

type LocationKey struct {
	Addr, MappingID uint64
	Lines           string
	IsFolded        bool
}

func MakeLocationKey(l *profile.Location) LocationKey {
	key := LocationKey{
		Addr:     l.Address,
		IsFolded: l.IsFolded,
	}
	if l.Mapping != nil {
		// Normalizes address to handle address space randomization.
		key.Addr -= l.Mapping.Start
		key.MappingID = l.Mapping.ID
	}
	lines := make([]string, len(l.Line)*2)
	for i, line := range l.Line {
		if line.Function != nil {
			lines[i*2] = strconv.FormatUint(line.Function.ID, 16)
		}
		lines[i*2+1] = strconv.FormatInt(line.Line, 16)
	}
	key.Lines = strings.Join(lines, "|")
	return key
}

type FunctionStore interface {
	GetFunctionByKey(key FunctionKey) (*profile.Function, error)
	CreateFunction(f *profile.Function)
}

type FunctionKey struct {
	StartLine                  int64
	Name, SystemName, FileName string
}

func MakeFunctionKey(f *profile.Function) FunctionKey {
	return FunctionKey{
		f.StartLine,
		f.Name,
		f.SystemName,
		f.Filename,
	}
}

type MappingStore interface {
	GetMappingByKey(key MappingKey) (*profile.Mapping, error)
	CreateMapping(m *profile.Mapping)
}

type MappingKey struct {
	Size, Offset  uint64
	BuildIDOrFile string
}

func MakeMappingKey(m *profile.Mapping) MappingKey {
	// Normalize addresses to handle address space randomization.
	// Round up to next 4K boundary to avoid minor discrepancies.
	const mapsizeRounding = 0x1000

	size := m.Limit - m.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)
	key := MappingKey{
		Size:   size,
		Offset: m.Offset,
	}

	switch {
	case m.BuildID != "":
		key.BuildIDOrFile = m.BuildID
	case m.File != "":
		key.BuildIDOrFile = m.File
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}
	return key
}

type InMemoryProfileMetaStore struct {
	locationsByKey map[LocationKey]uint64
	locations      []*profile.Location
	mappingsByKey  map[MappingKey]uint64
	mappings       []*profile.Mapping
	functionsByKey map[FunctionKey]uint64
	functions      []*profile.Function
}

func NewInMemoryProfileMetaStore() *InMemoryProfileMetaStore {
	return &InMemoryProfileMetaStore{
		locationsByKey: map[LocationKey]uint64{},
		functionsByKey: map[FunctionKey]uint64{},
		mappingsByKey:  map[MappingKey]uint64{},
	}
}

func (s *InMemoryProfileMetaStore) GetLocationByID(id uint64) (*profile.Location, error) {
	if uint64(len(s.locations)) <= id-1 {
		return nil, ErrLocationNotFound
	}

	return s.locations[id-1], nil
}

func (s *InMemoryProfileMetaStore) GetLocationByKey(key LocationKey) (*profile.Location, error) {
	i, found := s.locationsByKey[key]
	if !found {
		return nil, ErrLocationNotFound
	}

	return s.locations[i-1], nil
}

func (s *InMemoryProfileMetaStore) CreateLocation(l *profile.Location) {
	key := MakeLocationKey(l)
	id := uint64(len(s.locations)) + 1
	l.ID = uint64(id)
	s.locations = append(s.locations, l)
	s.locationsByKey[key] = id
}

func (s *InMemoryProfileMetaStore) GetMappingByKey(key MappingKey) (*profile.Mapping, error) {
	i, found := s.mappingsByKey[key]
	if !found {
		return nil, ErrMappingNotFound
	}

	return s.mappings[i-1], nil
}

func (s *InMemoryProfileMetaStore) CreateMapping(m *profile.Mapping) {
	key := MakeMappingKey(m)
	id := uint64(len(s.mappings)) + 1
	m.ID = uint64(id)
	s.mappings = append(s.mappings, m)
	s.mappingsByKey[key] = id
}

func (s *InMemoryProfileMetaStore) GetFunctionByKey(key FunctionKey) (*profile.Function, error) {
	i, found := s.functionsByKey[key]
	if !found {
		return nil, ErrFunctionNotFound
	}

	return s.functions[i-1], nil
}

func (s *InMemoryProfileMetaStore) CreateFunction(f *profile.Function) {
	key := MakeFunctionKey(f)
	id := uint64(len(s.functions)) + 1
	f.ID = uint64(id)
	s.functions = append(s.functions, f)
	s.functionsByKey[key] = id
}
