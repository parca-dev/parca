// Copyright (c) 2022 The Parca Authors
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
//

package metastoretest

import (
	"github.com/dgraph-io/badger/v3"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
)

type Testing interface {
	require.TestingT
	Helper()
	Name() string
}

func NewTestMetastore(
	t Testing,
	logger log.Logger,
	reg prometheus.Registerer,
	tracer trace.Tracer,
) pb.MetastoreServiceServer {
	t.Helper()

	db, err := badger.Open(
		badger.DefaultOptions("").
			WithInMemory(true).
			WithLogger(&metastore.BadgerLogger{Logger: logger}),
	)
	require.NoError(t, err)

	return metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		db,
	)
}
