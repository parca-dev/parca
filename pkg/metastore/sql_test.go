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
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

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
	dbPath := "./parca_meta_store_connection_test.sqlite"
	str, err := NewDiskProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		dbPath,
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		str.Close()
		os.Remove(dbPath)
	})

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
