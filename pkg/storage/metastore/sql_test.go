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

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

type TestProfileMetaStore interface {
	TestLocationStore
	TestFunctionStore
	MappingStore
	LocationLineStore
	Close() error
	Ping() error
}

type TestLocationStore interface {
	LocationStore
	GetLocations(ctx context.Context) ([]SerializedLocation, []uuid.UUID, error)
}

type TestFunctionStore interface {
	FunctionStore
	GetFunctions(ctx context.Context) ([]*Function, error)
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

	loc1 := uuid.New()
	l := &Location{
		ID:      loc1,
		Address: uint64(42),
	}
	_, err := s.CreateLocation(ctx, l)
	require.NoError(t, err)

	loc2 := uuid.New()
	l1 := &Location{
		ID:      loc2,
		Address: uint64(421),
	}
	_, err = s.CreateLocation(ctx, l1)
	require.NoError(t, err)

	locs, err := GetLocations(context.Background(), s)
	require.NoError(t, err)

	if locs[0].Address == 42 {
		require.Equal(t, locs[0].Address, l.Address)
		require.Equal(t, locs[1].Address, l1.Address)
	} else {
		require.Equal(t, locs[1].Address, l.Address)
		require.Equal(t, locs[0].Address, l1.Address)
	}

	l1, err = GetLocationByKey(ctx, s, MakeLocationKey(l1))
	require.NoError(t, err)

	locByID, err := GetLocationsByIDs(ctx, s, l1.ID)
	require.NoError(t, err)

	require.Equal(t, l1, locByID[l1.ID])

	f := &Function{
		FunctionKey: FunctionKey{
			Name:       "name",
			SystemName: "systemName",
			Filename:   "filename",
			StartLine:  22,
		},
	}
	l1.Lines = []LocationLine{
		{Line: 1, Function: f},
		{Line: 5, Function: f},
	}

	err = s.Symbolize(ctx, l1)
	require.NoError(t, err)

	locByID, err = GetLocationsByIDs(ctx, s, l1.ID)
	require.NoError(t, err)
	require.Equal(t, l1, locByID[l1.ID])
}

func LocationLinesStoreTest(t *testing.T, s LocationLineStore) {
	ctx := context.Background()

	locID := uuid.New()
	f1ID := uuid.New()
	ll := []LocationLine{{
		Line: 2,
		Function: &Function{
			ID: f1ID,
			FunctionKey: FunctionKey{
				Name: "f1",
			},
		},
	}}
	err := s.CreateLocationLines(ctx, locID, ll)
	require.NoError(t, err)

	llRetrieved, functionIDs, err := s.GetLinesByLocationIDs(ctx, locID)
	require.NoError(t, err)
	require.Equal(t, []uuid.UUID{f1ID}, functionIDs)
	require.Equal(t, map[uuid.UUID][]Line{
		locID: {
			{
				Line:       2,
				FunctionID: f1ID,
			},
		},
	}, llRetrieved)
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
	var err error

	f := &Function{
		FunctionKey: FunctionKey{
			Name:       "name",
			SystemName: "systemName",
			Filename:   "filename",
			StartLine:  22,
		},
	}
	f.ID, err = s.CreateFunction(ctx, f)
	require.NoError(t, err)

	f1 := &Function{
		FunctionKey: FunctionKey{
			Name:       "name",
			SystemName: "systemName",
			Filename:   "filename",
			StartLine:  42,
		},
	}
	f1.ID, err = s.CreateFunction(ctx, f1)
	require.NoError(t, err)

	funcByID, err := s.GetFunctionByKey(ctx, MakeFunctionKey(f))
	require.NoError(t, err)
	require.Equal(t, f.ID, funcByID.ID)
	require.Equal(t, f.Name, funcByID.Name)
	require.Equal(t, f.SystemName, funcByID.SystemName)
	require.Equal(t, f.Filename, funcByID.Filename)
	require.Equal(t, f.StartLine, funcByID.StartLine)

	funcs, err := s.GetFunctions(context.Background())
	require.NoError(t, err)

	// Order is not guaranteed, so make sure it's one of the two possibilities.

	if funcs[0].StartLine == 22 {
		require.Equal(t, funcs[0], f)
		require.Equal(t, funcs[1], f1)
	}

	if funcs[0].StartLine == 42 {
		require.Equal(t, funcs[0], f1)
		require.Equal(t, funcs[1], f)
	}
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
	var err error

	m := &Mapping{
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
	m.ID, err = s.CreateMapping(ctx, m)
	require.NoError(t, err)

	m1 := &Mapping{
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
	m1.ID, err = s.CreateMapping(ctx, m1)
	require.NoError(t, err)

	mapByKey, err := s.GetMappingByKey(ctx, MakeMappingKey(m))
	require.NoError(t, err)
	require.Equal(t, m.ID, mapByKey.ID)
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
	var err error

	m := &Mapping{
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
	m.ID, err = s.CreateMapping(ctx, m)
	require.NoError(t, err)

	l := &Location{
		Address: uint64(42),
		Mapping: m,
	}
	l.ID, err = s.CreateLocation(ctx, l)
	require.NoError(t, err)

	m1 := &Mapping{
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
	m1.ID, err = s.CreateMapping(ctx, m1)
	require.NoError(t, err)

	f := &Function{
		FunctionKey: FunctionKey{
			Name:       "name",
			SystemName: "systemName",
			Filename:   "filename",
			StartLine:  22,
		},
	}
	f.ID, err = s.CreateFunction(ctx, f)
	require.NoError(t, err)

	l1 := &Location{
		Address: uint64(421),
		Mapping: m1,
		Lines: []LocationLine{
			{Line: 1, Function: f},
			{Line: 5, Function: f},
		},
	}
	l1.ID, err = s.CreateLocation(ctx, l1)
	require.NoError(t, err)

	locs, err := GetLocations(ctx, s)
	require.NoError(t, err)
	require.Equal(t, 2, len(locs))

	if locs[0].Address == uint64(42) {
		require.Equal(t, l, locs[0])
		require.Equal(t, l1, locs[1])
	} else {
		require.Equal(t, l, locs[1])
		require.Equal(t, l1, locs[0])
	}

	unsymlocs, err := GetSymbolizableLocations(ctx, s)
	require.NoError(t, err)
	require.Equal(t, 1, len(unsymlocs))
	require.Equal(t, l, unsymlocs[0])
}

func TestBuildLinesByLocationIDsQuery(t *testing.T) {
	q := buildLinesByLocationIDsQuery([]uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	})

	require.Equal(
		t,
		`SELECT "location_id", "line", "function_id" FROM "lines" WHERE location_id IN ('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000002')`,
		q,
	)
}
