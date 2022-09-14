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

package query

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb"
	columnstore "github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
)

func Benchmark_Query_Merge(b *testing.B) {
	for k := 0.; k <= 7; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			ctx := context.Background()
			logger := log.NewNopLogger()
			reg := prometheus.NewRegistry()
			tracer := trace.NewNoopTracerProvider().Tracer("")
			col, err := columnstore.New()
			require.NoError(b, err)
			colDB, err := col.DB(context.Background(), "parca")
			require.NoError(b, err)

			schema, err := parcacol.Schema()
			require.NoError(b, err)

			table, err := colDB.Table(
				"stacktraces",
				columnstore.NewTableConfig(schema),
			)
			require.NoError(b, err)
			m := metastore.NewInProcessClient(metastoretest.NewTestMetastore(
				b,
				logger,
				reg,
				tracer,
			))

			fileContent := MustReadAllGzip(b, "../query/testdata/alloc_objects.pb.gz")
			require.NoError(b, err)
			p := &pprofpb.Profile{}
			require.NoError(b, p.UnmarshalVT(fileContent))

			for _, s := range p.Sample {
				s.Label = nil
			}

			normalizer := parcacol.NewNormalizer(m)
			ingester := parcacol.NewIngester(logger, normalizer, table, schema)

			profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]struct{}{}, p, false)
			require.NoError(b, err)

			for j := 0; j < n; j++ {
				for _, profile := range profiles {
					profile.Meta.Timestamp = int64(j + 1)
					err = ingester.IngestProfile(ctx, nil, profile)
					require.NoError(b, err)
				}
			}

			table.Sync()

			api := NewColumnQueryAPI(
				logger,
				tracer,
				getShareServerConn(b),
				parcacol.NewQuerier(
					tracer,
					query.NewEngine(
						memory.DefaultAllocator,
						colDB.TableProvider(),
					),
					"stacktraces",
					m,
				),
			)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err = api.Query(ctx, &pb.QueryRequest{
					Mode: pb.QueryRequest_MODE_MERGE,
					Options: &pb.QueryRequest_Merge{
						Merge: &pb.MergeProfile{
							Query: `{__name__="memory:alloc_objects:count:space:bytes"}`,
							Start: timestamppb.New(time.Unix(0, 0)),
							End:   timestamppb.New(time.Unix(0, int64(time.Millisecond)*int64(n+1))),
						},
					},
					ReportType: pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED,
				})
				require.NoError(b, err)
			}
		})
	}
}

func Benchmark_ProfileTypes(b *testing.B) {
	// This benchmark is skipped by default as it requires a write-ahead log to
	// be present for the "stacktraces" table in the "parca" database in
	// "../../data".
	b.Skip()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New(
		frostdb.WithWAL(),
		frostdb.WithStoragePath("../../data"),
	)
	require.NoError(b, err)

	require.NoError(b, col.ReplayWALs(ctx))

	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(b, err)

	table, err := colDB.GetTable("stacktraces")
	require.NoError(b, err)
	table.Sync()

	require.NoError(b, err)
	m := metastore.NewInProcessClient(metastoretest.NewTestMetastore(
		b,
		logger,
		reg,
		tracer,
	))

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(b),
		parcacol.NewQuerier(
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			m,
		),
	)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err = api.ProfileTypes(ctx, &pb.ProfileTypesRequest{})
		require.NoError(b, err)
	}
}
