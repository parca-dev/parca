// Copyright 2022 The Parca Authors
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

package profilestore

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
)

func Test_LabelName_Invalid(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := frostdb.New(
		logger,
		reg,
	)
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		frostdb.NewTableConfig(schema),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	api := NewProfileColumnStore(
		logger,
		tracer,
		metastore.NewInProcessClient(m),
		table,
		schema,
		false,
	)

	req := &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{{
					Name:  "n0:n",
					Value: "v0",
				}},
			},
		}},
	}

	_, err = api.WriteRaw(ctx, req)
	st, _ := status.FromError(err)

	require.Equal(t, st.Code(), codes.InvalidArgument)
}
