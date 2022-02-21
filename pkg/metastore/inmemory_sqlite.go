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
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	_ "modernc.org/sqlite" // We need to import the SQLite driver.

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

var _ ProfileMetaStore = &InMemorySQLiteMetaStore{}

type InMemorySQLiteMetaStore struct {
	*sqlMetaStore
}

func NewInMemorySQLiteProfileMetaStore(
	reg prometheus.Registerer,
	tracer trace.Tracer,
	name ...string,
) (*InMemorySQLiteMetaStore, error) {
	dsn := "file::memory:?cache=shared"
	if len(name) > 0 {
		dsn = fmt.Sprintf("file:%s?mode=memory&cache=shared", name[0])
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		db.Close()
		return nil, err
	}

	sqlite := &sqlMetaStore{
		db:     db,
		tracer: tracer,
		cache:  newMetaStoreCache(reg),
	}
	if err := sqlite.migrate(); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	return &InMemorySQLiteMetaStore{sqlMetaStore: sqlite}, nil
}

func (i InMemorySQLiteMetaStore) GetStacktraceByKey(ctx context.Context, key []byte) (uuid.UUID, error) {
	panic("implement me")
}

func (i InMemorySQLiteMetaStore) GetStacktraceByIDs(ctx context.Context, ids ...[]byte) (map[string]*pb.Sample, error) {
	panic("implement me")
}

func (i InMemorySQLiteMetaStore) CreateStacktrace(ctx context.Context, key []byte, sample *pb.Sample) (uuid.UUID, error) {
	panic("implement me")
}
