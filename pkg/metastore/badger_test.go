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

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

func TestBadgerStoreMappingStore(t *testing.T) {
	db := NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		db.Close()
	})

	mappingStoreTest(t, db)
}

func TestBadgerStoreFunctionStore(t *testing.T) {
	db := NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		db.Close()
	})

	functionStoreTest(t, db)
}

func TestBadgerStoreLocationLinesStore(t *testing.T) {
	db := NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		db.Close()
	})

	LocationLinesStoreTest(t, db)
}

func TestBadgerStoreLocationStore(t *testing.T) {
	db := NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		db.Close()
	})

	LocationStoreTest(t, db)
}
