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
	FunctionStore
	MappingStore
	Close() error
	Ping() error
}

type LocationStore interface {
	GetLocationByKey(ctx context.Context, k LocationKey) (*Location, error)
	GetLocationsByIDs(ctx context.Context, id ...uuid.UUID) (map[uuid.UUID]*Location, error)
	CreateLocation(ctx context.Context, l *Location) (uuid.UUID, error)
	Symbolize(ctx context.Context, location *Location) error
	GetSymbolizableLocations(ctx context.Context) ([]*Location, error)
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

var unsetUUID = uuid.UUID{}

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
}

type Function struct {
	ID uuid.UUID
	FunctionKey
}

type FunctionKey struct {
	StartLine                  int64
	Name, SystemName, Filename string
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

type MappingKey struct {
	Size, Offset  uint64
	BuildIDOrFile string
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
