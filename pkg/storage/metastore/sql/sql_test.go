package sql

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
)

type TestProfileMetaStore interface {
	TestLocationStore
	TestFunctionStore
	metastore.MappingStore
	Close() error
	Ping() error
}

type TestLocationStore interface {
	GetLocationByKey(k metastore.LocationKey) (*profile.Location, error)
	GetLocationByID(id uint64) (*profile.Location, error)
	CreateLocation(l *profile.Location) error
	UpdateLocation(location *profile.Location) error
	GetUnsymbolizedLocations() ([]*profile.Location, error)

	GetLocations() ([]*profile.Location, error)
}

type TestFunctionStore interface {
	GetFunctionByKey(key metastore.FunctionKey) (*profile.Function, error)
	CreateFunction(f *profile.Function) error

	GetFunctions() ([]*profile.Function, error)
}

func TestNewInMemoryMetaStore(t *testing.T) {
	str, err := NewInMemoryProfileMetaStore("metastoreconnection")
	t.Cleanup(func() {
		str.Close()
	})
	require.NoError(t, err)
	require.NoError(t, str.Ping())
}

func TestDiskMetaStoreConnection(t *testing.T) {
	str, err := NewDiskProfileMetaStore()
	require.NoError(t, err)
	require.NoError(t, str.Ping())
}

func TestInMemoryLocationStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore("location")
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	LocationStoreTest(t, s)
}

func TestDiskLocationStore(t *testing.T) {
	dbPath := "./parca_location_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	LocationStoreTest(t, s)
}

func LocationStoreTest(t *testing.T, s TestProfileMetaStore) {
	largeLoc := -1
	l := &profile.Location{
		ID:      uint64(largeLoc),
		Address: uint64(42),
	}
	err := s.CreateLocation(l)
	require.NoError(t, err)

	l1 := &profile.Location{
		ID:      uint64(18),
		Address: uint64(421),
	}
	err = s.CreateLocation(l1)
	require.NoError(t, err)

	locs, err := s.GetLocations()
	require.NoError(t, err)
	require.Equal(t, locs[0], l)
	require.Equal(t, locs[1], l1)

	locByID, err := s.GetLocationByID(l.ID)
	require.NoError(t, err)
	require.Equal(t, uint64(largeLoc), locByID.ID)

	locByKey, err := s.GetLocationByKey(metastore.MakeLocationKey(l))
	require.NoError(t, err)
	require.Equal(t, uint64(largeLoc), locByKey.ID)

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
	s, err := NewInMemoryProfileMetaStore("function")
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	functionStoreTest(t, s)
}

func TestDiskFunctionStore(t *testing.T) {
	dbPath := "./parca_function_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	functionStoreTest(t, s)
}

func functionStoreTest(t *testing.T, s TestFunctionStore) {
	largeLoc := -1
	f := &profile.Function{
		ID:         uint64(largeLoc),
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	err := s.CreateFunction(f)
	require.NoError(t, err)

	f1 := &profile.Function{
		ID:         18,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  42,
	}
	err = s.CreateFunction(f1)
	require.NoError(t, err)

	funcByID, err := s.GetFunctionByKey(metastore.MakeFunctionKey(f))
	require.NoError(t, err)
	require.Equal(t, uint64(largeLoc), funcByID.ID)
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
	s, err := NewInMemoryProfileMetaStore("mapping")
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	mappingStoreTest(t, s)
}

func TestDiskMappingStore(t *testing.T) {
	dbPath := "./parca_mapping_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	mappingStoreTest(t, s)
}

func mappingStoreTest(t *testing.T, s metastore.MappingStore) {
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
	err := s.CreateMapping(m)
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
	err = s.CreateMapping(m1)
	require.NoError(t, err)

	mapByKey, err := s.GetMappingByKey(metastore.MakeMappingKey(m))
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

func TestInMemoryMetaStore(t *testing.T) {
	s, err := NewInMemoryProfileMetaStore("metastore")
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
	})

	metaStoreTest(t, s)
}

func TestDiskMetaStore(t *testing.T) {
	dbPath := "./parca_meta_store_test.sqlite"
	s, err := NewDiskProfileMetaStore(dbPath)
	require.NoError(t, err)
	t.Cleanup(func() {
		s.Close()
		os.Remove(dbPath)
	})

	metaStoreTest(t, s)
}

func metaStoreTest(t *testing.T, s TestProfileMetaStore) {
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
	err := s.CreateMapping(m)
	require.NoError(t, err)

	l := &profile.Location{
		ID:      uint64(8),
		Address: uint64(42),
		Mapping: m,
	}
	err = s.CreateLocation(l)
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
	err = s.CreateMapping(m1)
	require.NoError(t, err)

	f := &profile.Function{
		ID:         8,
		Name:       "name",
		SystemName: "systemName",
		Filename:   "filename",
		StartLine:  22,
	}
	err = s.CreateFunction(f)
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
	err = s.CreateLocation(l1)
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
