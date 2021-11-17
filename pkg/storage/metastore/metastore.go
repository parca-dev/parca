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
	"encoding/binary"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
)

var (
	ErrLocationNotFound = errors.New("location not found")
	ErrMappingNotFound  = errors.New("mapping not found")
	ErrFunctionNotFound = errors.New("function not found")
)

type ProfileMetaStore interface {
	LocationStore
	LocationLineStore
	FunctionStore
	MappingStore
	Close() error
	Ping() error
}

type LocationStore interface {
	GetLocations(ctx context.Context) ([]SerializedLocation, []uuid.UUID, error)
	GetLocationByKey(ctx context.Context, k LocationKey) (SerializedLocation, error)
	GetLocationsByIDs(ctx context.Context, id ...uuid.UUID) (map[uuid.UUID]SerializedLocation, []uuid.UUID, error)
	CreateLocation(ctx context.Context, l *Location) (uuid.UUID, error)
	Symbolize(ctx context.Context, location *Location) error
	GetSymbolizableLocations(ctx context.Context) ([]SerializedLocation, []uuid.UUID, error)
}

type LocationLineStore interface {
	CreateLocationLines(ctx context.Context, locID uuid.UUID, lines []LocationLine) error
	GetLinesByLocationIDs(ctx context.Context, id ...uuid.UUID) (map[uuid.UUID][]Line, []uuid.UUID, error)
}

type Location struct {
	ID       uuid.UUID
	Address  uint64
	Mapping  *Mapping
	Lines    []LocationLine
	IsFolded bool
}

type LocationLine struct {
	Line     int64
	Function *Function
}

type SerializedLocation struct {
	ID                uuid.UUID
	Address           uint64
	NormalizedAddress uint64
	MappingID         uuid.UUID
	IsFolded          bool
}

type Line struct {
	FunctionID uuid.UUID
	Line       int64
}

type LocationKey struct {
	NormalizedAddress uint64
	MappingID         uuid.UUID
	Lines             string
	IsFolded          bool
}

func (l *Location) Key() LocationKey {
	return MakeLocationKey(l)
}

const locationsKeyPrefix = "locations/by-key/"

func (k LocationKey) Bytes() []byte {
	buf := make([]byte, len(locationsKeyPrefix)+8+16+8+len(k.Lines))
	copy(buf, locationsKeyPrefix)
	copy(buf[len(locationsKeyPrefix):], k.MappingID[:])
	binary.BigEndian.PutUint64(buf[len(locationsKeyPrefix)+16:], k.NormalizedAddress)
	if k.IsFolded {
		// If IsFolded is false this means automatically that these 8 bytes are 0.
		binary.BigEndian.PutUint64(buf[len(locationsKeyPrefix)+8+16:], 1)
	}
	copy(buf[len(locationsKeyPrefix)+8+16+8:], k.Lines)
	return buf
}

func MakeLocationKey(l *Location) LocationKey {
	key := LocationKey{
		NormalizedAddress: l.Address,
		IsFolded:          l.IsFolded,
	}
	if l.Mapping != nil {
		// Normalizes address to handle address space randomization.
		key.NormalizedAddress -= l.Mapping.Start
		key.MappingID = l.Mapping.ID
	}

	// If the normalized address is 0, then the functions attached to the
	// location are not from a native binary, but instead from a dynamic
	// runtime/language eg. ruby or python. In those cases we have no better
	// uniqueness factor than the actual functions, and since there is no
	// address there is no potential for asynchronously symbolizing.
	if key.NormalizedAddress == 0 {
		lines := make([]string, len(l.Lines)*2)
		for i, line := range l.Lines {
			if line.Function != nil {
				lines[i*2] = line.Function.ID.String()
			}
			lines[i*2+1] = strconv.FormatInt(line.Line, 16)
		}
		key.Lines = strings.Join(lines, "|")
	}
	return key
}

type FunctionStore interface {
	GetFunctionByKey(ctx context.Context, key FunctionKey) (*Function, error)
	CreateFunction(ctx context.Context, f *Function) (uuid.UUID, error)
	GetFunctionsByIDs(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]*Function, error)
	GetFunctions(ctx context.Context) ([]*Function, error)
}

type Function struct {
	ID uuid.UUID
	FunctionKey
}

func (f Function) Key() FunctionKey {
	return f.FunctionKey
}

type FunctionKey struct {
	StartLine                  int64
	Name, SystemName, Filename string
}

const functionKeyPrefix = "functions/by-key/"

func (f FunctionKey) Bytes() []byte {
	buf := make([]byte, len(functionKeyPrefix)+len(f.Name)+len(f.SystemName)+len(f.Filename)+8)
	copy(buf, functionKeyPrefix)
	binary.BigEndian.PutUint64(buf[len(functionKeyPrefix):], uint64(f.StartLine))
	copy(buf[len(functionKeyPrefix)+8:], f.Name)
	copy(buf[len(functionKeyPrefix)+8+len(f.Name):], f.SystemName)
	copy(buf[len(functionKeyPrefix)+8+len(f.Name)+len(f.SystemName):], f.Filename)

	return buf
}

func MakeFunctionKey(f *Function) FunctionKey {
	return FunctionKey{
		f.StartLine,
		f.Name,
		f.SystemName,
		f.Filename,
	}
}

type MappingStore interface {
	GetMappingByKey(ctx context.Context, key MappingKey) (*Mapping, error)
	CreateMapping(ctx context.Context, m *Mapping) (uuid.UUID, error)
	GetMappingsByIDs(ctx context.Context, ids ...uuid.UUID) (map[uuid.UUID]*Mapping, error)
}

type Mapping struct {
	ID              uuid.UUID
	Start           uint64
	Limit           uint64
	Offset          uint64
	File            string
	BuildID         string
	HasFunctions    bool
	HasFilenames    bool
	HasLineNumbers  bool
	HasInlineFrames bool
}

// Unsymbolizable returns true if a mapping points to a binary for which
// locations can't be symbolized in principle, at least now. Examples are
// "[vdso]", [vsyscall]" and some others, see the code.
func (m *Mapping) Unsymbolizable() bool {
	name := filepath.Base(m.File)
	return strings.HasPrefix(name, "[") || strings.HasPrefix(name, "linux-vdso") || strings.HasPrefix(m.File, "/dev/dri/")
}

// Key returns a key for the mapping.
func (m *Mapping) Key() MappingKey {
	return MakeMappingKey(m)
}

type MappingKey struct {
	Size, Offset  uint64
	BuildIDOrFile string
}

const mappingKeyPrefix = "mappings/by-key/"

func (k MappingKey) Bytes() []byte {
	buf := make([]byte, len(mappingKeyPrefix)+len(k.BuildIDOrFile)+16)
	copy(buf, mappingKeyPrefix)
	binary.BigEndian.PutUint64(buf[len(mappingKeyPrefix):], k.Size)
	binary.BigEndian.PutUint64(buf[len(mappingKeyPrefix)+8:], k.Offset)
	copy(buf[len(mappingKeyPrefix)+16:], k.BuildIDOrFile)

	return buf
}

func MakeMappingKey(m *Mapping) MappingKey {
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
		// BuildID has precedence over file as we can rely on it being more
		// unique.
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

func GetLocationByKey(ctx context.Context, s ProfileMetaStore, k LocationKey) (*Location, error) {
	res := Location{}

	l, err := s.GetLocationByKey(ctx, k)
	if err != nil {
		return nil, err
	}

	res.ID = l.ID
	res.Address = l.Address
	res.IsFolded = l.IsFolded

	if k.MappingID != uuid.Nil {
		mappings, err := s.GetMappingsByIDs(ctx, k.MappingID)
		if err != nil {
			return nil, fmt.Errorf("get mapping by ID: %w", err)
		}
		res.Mapping = mappings[k.MappingID]
	}

	linesByLocation, functionIDs, err := s.GetLinesByLocationIDs(ctx, l.ID)
	if err != nil {
		return nil, fmt.Errorf("get lines by location ID: %w", err)
	}

	functions, err := s.GetFunctionsByIDs(ctx, functionIDs...)
	if err != nil {
		return nil, fmt.Errorf("get functions by IDs: %w", err)
	}

	for _, line := range linesByLocation[l.ID] {
		res.Lines = append(res.Lines, LocationLine{
			Line:     line.Line,
			Function: functions[line.FunctionID],
		})
	}

	return &res, nil
}

func GetLocationsByIDs(ctx context.Context, s ProfileMetaStore, ids ...uuid.UUID) (
	map[uuid.UUID]*Location,
	error,
) {
	locs, mappingIDs, err := s.GetLocationsByIDs(ctx, ids...)
	if err != nil {
		return nil, fmt.Errorf("get locations by IDs: %w", err)
	}

	return getLocationsFromSerializedLocations(ctx, s, ids, locs, mappingIDs)
}

// Only used in tests so not as important to be efficient.
func GetLocations(ctx context.Context, s ProfileMetaStore) ([]*Location, error) {
	lArr, mappingIDs, err := s.GetLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("get serialized locations: %w", err)
	}

	l := map[uuid.UUID]SerializedLocation{}
	locIDs := []uuid.UUID{}
	for _, loc := range lArr {
		l[loc.ID] = loc
		locIDs = append(locIDs, loc.ID)
	}

	locs, err := getLocationsFromSerializedLocations(ctx, s, locIDs, l, mappingIDs)
	if err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}

	res := make([]*Location, 0, len(locs))
	for _, loc := range locs {
		res = append(res, loc)
	}

	return res, nil
}

func getLocationsFromSerializedLocations(ctx context.Context, s ProfileMetaStore, ids []uuid.UUID, locs map[uuid.UUID]SerializedLocation, mappingIDs []uuid.UUID) (map[uuid.UUID]*Location, error) {
	mappings, err := s.GetMappingsByIDs(ctx, mappingIDs...)
	if err != nil {
		return nil, fmt.Errorf("get mappings by IDs: %w", err)
	}

	linesByLocation, functionIDs, err := s.GetLinesByLocationIDs(ctx, ids...)
	if err != nil {
		return nil, fmt.Errorf("get lines by location IDs: %w", err)
	}

	functions, err := s.GetFunctionsByIDs(ctx, functionIDs...)
	if err != nil {
		return nil, fmt.Errorf("get functions by ids: %w", err)
	}

	res := make(map[uuid.UUID]*Location, len(locs))
	for locationID, loc := range locs {
		location := &Location{
			ID:       loc.ID,
			Address:  loc.Address,
			IsFolded: loc.IsFolded,
		}
		location.Mapping = mappings[loc.MappingID]
		locationLines := linesByLocation[locationID]
		if len(locationLines) > 0 {
			lines := make([]LocationLine, 0, len(locationLines))
			for _, line := range locationLines {
				function, found := functions[line.FunctionID]
				if found {
					lines = append(lines, LocationLine{
						Line:     line.Line,
						Function: function,
					})
				}
			}
			location.Lines = lines
		}
		res[locationID] = location
	}

	return res, nil
}

func GetSymbolizableLocations(ctx context.Context, s ProfileMetaStore) (
	[]*Location,
	error,
) {
	locs, mappingIDs, err := s.GetSymbolizableLocations(ctx)
	if err != nil {
		return nil, fmt.Errorf("get symbolizable locations: %w", err)
	}

	mappings, err := s.GetMappingsByIDs(ctx, mappingIDs...)
	if err != nil {
		return nil, fmt.Errorf("get mappings by IDs: %w", err)
	}

	res := make([]*Location, 0, len(locs))
	for _, loc := range locs {
		res = append(res, &Location{
			ID:       loc.ID,
			Address:  loc.Address,
			IsFolded: loc.IsFolded,
			Mapping:  mappings[loc.MappingID],
		})
	}

	return res, nil
}
