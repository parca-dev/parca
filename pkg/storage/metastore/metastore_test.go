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
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

type TestProfileMetaStore interface {
	TestLocationStore
	TestFunctionStore
	MappingStore
	Close() error
	Ping() error
}

type TestLocationStore interface {
	LocationStore
	GetLocations() ([]*profile.Location, error)
}

type TestFunctionStore interface {
	FunctionStore
	GetFunctions() ([]*profile.Function, error)
}

func TestInMemoryLocationStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore()
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	largeLoc := -1
	l := &profile.Location{
		ID:      uint64(largeLoc),
		Address: uint64(42),
	}
	_, err = s.CreateLocation(l)
	require.NoError(t, err)

	l1 := &profile.Location{
		ID:      uint64(18),
		Address: uint64(421),
	}
	_, err = s.CreateLocation(l1)
	require.NoError(t, err)

	locs, err := s.GetLocations()
	require.NoError(t, err)
	require.Equal(t, locs[0], l)
	require.Equal(t, locs[1], l1)

	locByID, err := s.GetLocationByID(l.ID)
	require.NoError(t, err)
	require.Equal(t, uint64(1), locByID.ID)

	locByKey, err := s.GetLocationByKey(MakeLocationKey(l))
	require.NoError(t, err)
	require.Equal(t, uint64(1), locByKey.ID)

	f := &profile.Function{
		ID:         8,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	l1.Line = []profile.Line{
		{Line: 1, Function: f},
		{Line: 5, Function: f},
	}

	err = s.UpdateLocation(l1)
	require.NoError(t, err)

	locByID, err = s.GetLocationByID(l1.ID)
	require.NoError(t, err)
	require.Equal(t, l1, locByID)
}

func TestInMemoryFunctionStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore()
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	largeLoc := -1
	f := &profile.Function{
		ID:         uint64(largeLoc),
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	_, err = s.CreateFunction(f)
	require.NoError(t, err)

	f1 := &profile.Function{
		ID:         18,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  42,
	}
	_, err = s.CreateFunction(f1)
	require.NoError(t, err)

	funcByID, err := s.GetFunctionByKey(MakeFunctionKey(f))
	require.NoError(t, err)
	require.Equal(t, uint64(1), funcByID.ID)
	require.Equal(t, f.Name, funcByID.Name)
	require.Equal(t, f.SystemName, funcByID.SystemName)
	require.Equal(t, f.Filename, funcByID.Filename)
	require.Equal(t, f.StartLine, funcByID.StartLine)

	funcs, err := s.GetFunctions()
	require.NoError(t, err)
	require.Equal(t, funcs[0], f)
	require.Equal(t, funcs[1], f1)
}

func TestInMemoryMappingStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore()
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	largeLoc := -1
	m := &profile.Mapping{
		ID:              uint64((largeLoc)),
		Start:           1,
		Limit:           10,
		Offset:          5,
		File:            "file",
		BuildID:         "buildID0",
		HasFunctions:    false,
		HasFilenames:    false,
		HasLineNumbers:  false,
		HasInlineFrames: false,
	}
	_, err = s.CreateMapping(m)
	require.NoError(t, err)

	m1 := &profile.Mapping{
		ID:              18,
		Start:           12,
		Limit:           110,
		Offset:          51,
		File:            "file1",
		BuildID:         "buildID1",
		HasFunctions:    true,
		HasFilenames:    true,
		HasLineNumbers:  false,
		HasInlineFrames: true,
	}
	_, err = s.CreateMapping(m1)
	require.NoError(t, err)

	mapByKey, err := s.GetMappingByKey(MakeMappingKey(m))
	require.NoError(t, err)
	require.Equal(t, uint64(1), mapByKey.ID)
	require.Equal(t, m.Start, mapByKey.Start)
	require.Equal(t, m.Limit, mapByKey.Limit)
	require.Equal(t, m.Offset, mapByKey.Offset)
	require.Equal(t, m.File, mapByKey.File)
	require.Equal(t, m.BuildID, mapByKey.BuildID)
	require.Equal(t, m.HasFunctions, mapByKey.HasFunctions)
	require.Equal(t, m.HasFilenames, mapByKey.HasFilenames)
	require.Equal(t, m.HasLineNumbers, mapByKey.HasLineNumbers)
	require.Equal(t, m.HasInlineFrames, mapByKey.HasInlineFrames)
}

func TestInMemoryMetaStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore()
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	largeLoc := -1
	m := &profile.Mapping{
		ID:              uint64(largeLoc),
		Start:           1,
		Limit:           10,
		Offset:          5,
		File:            "file",
		BuildID:         "buildID0",
		HasFunctions:    false,
		HasFilenames:    false,
		HasLineNumbers:  false,
		HasInlineFrames: false,
	}
	_, err = s.CreateMapping(m)
	require.NoError(t, err)

	l := &profile.Location{
		ID:      uint64(8),
		Address: uint64(42),
		Mapping: m,
	}
	_, err = s.CreateLocation(l)
	require.NoError(t, err)

	m1 := &profile.Mapping{
		ID:              18,
		Start:           12,
		Limit:           110,
		Offset:          51,
		File:            "file1",
		BuildID:         "buildID1",
		HasFunctions:    true,
		HasFilenames:    true,
		HasLineNumbers:  false,
		HasInlineFrames: true,
	}
	_, err = s.CreateMapping(m1)
	require.NoError(t, err)

	f := &profile.Function{
		ID:         8,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	_, err = s.CreateFunction(f)
	require.NoError(t, err)

	l1 := &profile.Location{
		ID:      uint64(18),
		Address: uint64(421),
		Mapping: m1,
		Line: []profile.Line{
			{Line: 1, Function: f},
			{Line: 5, Function: f},
		},
	}
	_, err = s.CreateLocation(l1)
	require.NoError(t, err)

	locs, err := s.GetLocations()
	require.NoError(t, err)
	require.Equal(t, 2, len(locs))
	require.Equal(t, l, locs[0])
	require.Equal(t, l1, locs[1])

	unsymlocs, err := s.GetUnsymbolizedLocations()
	require.NoError(t, err)
	require.Equal(t, 1, len(unsymlocs))
	require.Equal(t, l, unsymlocs[0])
}
