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
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
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
	GetLocations(ctx context.Context) ([]*profile.Location, error)
}

type TestFunctionStore interface {
	FunctionStore
	GetFunctions(ctx context.Context) ([]*profile.Function, error)
}

func TestNewInMemorySQLiteMetaStore(t *testing.T) {
	str, err := NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"metastoreconnection",
	)
	t.Cleanup(func() {
		str.Close()
	})
	require.NoError(t, err)
	require.NoError(t, str.Ping())
}

func TestDiskMetaStoreConnection(t *testing.T) {
	str, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)
	require.NoError(t, err)
	require.NoError(t, str.Ping())
}

func TestInMemorySQLiteLocationStore(t *testing.T) {
	s, err := NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"location",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	LocationStoreTest(t, s)
}

func TestDiskLocationStore(t *testing.T) {
	dbPath := "./parca_location_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		dbPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	LocationStoreTest(t, s)
}

func LocationStoreTest(t *testing.T, s TestProfileMetaStore) {
	ctx := context.Background()

	largeLoc := -1
	l := &profile.Location{
		ID:      uint64(largeLoc),
		Address: uint64(42),
	}
	_, err := s.CreateLocation(ctx, l)
	require.NoError(t, err)

	l1 := &profile.Location{
		ID:      uint64(18),
		Address: uint64(421),
	}
	_, err = s.CreateLocation(ctx, l1)
	require.NoError(t, err)

	locs, err := s.GetLocations(context.Background())
	require.NoError(t, err)
	require.Equal(t, locs[0].Address, l.Address)
	require.Equal(t, locs[1].Address, l1.Address)

	l1, err = s.GetLocationByKey(ctx, MakeLocationKey(l1))
	require.NoError(t, err)

	locByID, err := s.GetLocationsByIDs(ctx, l1.ID)
	require.NoError(t, err)

	require.Equal(t, l1, locByID[l1.ID])

	f := &profile.Function{
		ID:         1,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	l1.Line = []profile.Line{
		{Line: 1, Function: f},
		{Line: 5, Function: f},
	}

	err = s.Symbolize(ctx, l1)
	require.NoError(t, err)

	locByID, err = s.GetLocationsByIDs(ctx, l1.ID)
	require.NoError(t, err)
	require.Equal(t, l1, locByID[l1.ID])
}

func TestInMemorySQLiteFunctionStore(t *testing.T) {
	s, err := NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"function",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	functionStoreTest(t, s)
}

func TestDiskFunctionStore(t *testing.T) {
	dbPath := "./parca_function_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		dbPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	functionStoreTest(t, s)
}

func functionStoreTest(t *testing.T, s TestFunctionStore) {
	ctx := context.Background()

	f := &profile.Function{
		ID:         1,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	_, err := s.CreateFunction(ctx, f)
	require.NoError(t, err)

	f1 := &profile.Function{
		ID:         2,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  42,
	}
	_, err = s.CreateFunction(ctx, f1)
	require.NoError(t, err)

	funcByID, err := s.GetFunctionByKey(ctx, MakeFunctionKey(f))
	require.NoError(t, err)
	require.Equal(t, uint64(1), funcByID.ID)
	require.Equal(t, f.Name, funcByID.Name)
	require.Equal(t, f.SystemName, funcByID.SystemName)
	require.Equal(t, f.Filename, funcByID.Filename)
	require.Equal(t, f.StartLine, funcByID.StartLine)

	funcs, err := s.GetFunctions(context.Background())
	require.NoError(t, err)
	require.Equal(t, funcs[0], f)
	require.Equal(t, funcs[1], f1)
}

func TestInMemorySQLiteMappingStore(t *testing.T) {
	s, err := NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"mapping",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	mappingStoreTest(t, s)
}

func TestDiskMappingStore(t *testing.T) {
	dbPath := "./parca_mapping_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		dbPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	mappingStoreTest(t, s)
}

func mappingStoreTest(t *testing.T, s MappingStore) {
	ctx := context.Background()

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
	_, err := s.CreateMapping(ctx, m)
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
	_, err = s.CreateMapping(ctx, m1)
	require.NoError(t, err)

	mapByKey, err := s.GetMappingByKey(ctx, MakeMappingKey(m))
	require.NoError(t, err)
	require.Equal(t, uint64(largeLoc), mapByKey.ID)
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

func TestInMemorySQLiteMetaStore(t *testing.T) {
	s, err := NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"metastore",
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	metaStoreTest(t, s)
}

func TestDiskMetaStore(t *testing.T) {
	dbPath := "./parca_meta_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		dbPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	metaStoreTest(t, s)
}

func metaStoreTest(t *testing.T, s TestProfileMetaStore) {
	ctx := context.Background()

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
	_, err := s.CreateMapping(ctx, m)
	require.NoError(t, err)

	l := &profile.Location{
		ID:      uint64(8),
		Address: uint64(42),
		Mapping: m,
	}
	_, err = s.CreateLocation(ctx, l)
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
	_, err = s.CreateMapping(ctx, m1)
	require.NoError(t, err)

	f := &profile.Function{
		ID:         1,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	_, err = s.CreateFunction(ctx, f)
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
	_, err = s.CreateLocation(ctx, l1)
	require.NoError(t, err)

	locs, err := s.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, len(locs))
	l.ID = locs[0].ID
	require.Equal(t, l, locs[0])
	l1.ID = locs[1].ID
	require.Equal(t, l1, locs[1])

	unsymlocs, err := s.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(unsymlocs))
	require.Equal(t, l, unsymlocs[0])
}
