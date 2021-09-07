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
	Close() error
	Ping() error
}

type LocationStore interface {
	GetLocationByKey(ctx context.Context, k LocationKey) (*profile.Location, error)
	GetLocationsByIDs(ctx context.Context, id ...uint64) (map[uint64]*profile.Location, error)
	CreateLocation(ctx context.Context, l *profile.Location) (uint64, error)
	UpdateLocation(ctx context.Context, location *profile.Location) error
	GetUnsymbolizedLocations(ctx context.Context) ([]*profile.Location, error)
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
	GetFunctionByKey(ctx context.Context, key FunctionKey) (*profile.Function, error)
	CreateFunction(ctx context.Context, f *profile.Function) (uint64, error)
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
	GetMappingByKey(ctx context.Context, key MappingKey) (*profile.Mapping, error)
	CreateMapping(ctx context.Context, m *profile.Mapping) (uint64, error)
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
