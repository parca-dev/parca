// Copyright 2022-2023 The Parca Authors
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
	"os"
	"testing"
	"time"

	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb"
	columnstore "github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profilestore"
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
				columnstore.NewTableConfig(parcacol.SchemaDefinition()),
			)
			require.NoError(b, err)
			metastore := metastore.NewInProcessClient(metastoretest.NewTestMetastore(
				b,
				logger,
				reg,
				tracer,
			))

			fileContent, err := os.ReadFile("../query/testdata/alloc_objects.pb.gz")
			require.NoError(b, err)

			store := profilestore.NewProfileColumnStore(
				logger,
				tracer,
				metastore,
				table,
				schema,
				true,
			)

			for j := 0; j < n; j++ {
				_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
					Series: []*profilestorepb.RawProfileSeries{{
						Labels: &profilestorepb.LabelSet{
							Labels: []*profilestorepb.Label{
								{
									Name:  "__name__",
									Value: "memory",
								},
								{
									Name:  "job",
									Value: "default",
								},
							},
						},
						Samples: []*profilestorepb.RawSample{{
							RawProfile: fileContent,
						}},
					}},
				})
				require.NoError(b, err)
			}

			require.NoError(b, table.EnsureCompaction())

			api := NewColumnQueryAPI(
				logger,
				tracer,
				getShareServerConn(b),
				parcacol.NewQuerier(
					logger,
					tracer,
					query.NewEngine(
						memory.DefaultAllocator,
						colDB.TableProvider(),
					),
					"stacktraces",
					metastore,
				),
			)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err = api.Query(ctx, &pb.QueryRequest{
					Mode: pb.QueryRequest_MODE_MERGE,
					Options: &pb.QueryRequest_Merge{
						Merge: &pb.MergeProfile{
							Query: `{__name__="memory:alloc_objects:count:space:bytes"}`,
							Start: timestamppb.New(time.Unix(0, math.MinInt64)),
							End:   timestamppb.New(time.Unix(0, math.MaxInt64)),
						},
					},
					//nolint:staticcheck // SA1019: Fow now we want to support these APIs
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

	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(b, err)

	table, err := colDB.GetTable("stacktraces")
	require.NoError(b, err)
	require.NoError(b, table.EnsureCompaction())

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
			logger,
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
