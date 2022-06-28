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

package query

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
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
			col := columnstore.New(
				reg,
				8196,
				64*1024*1024,
			)
			colDB, err := col.DB("parca")
			require.NoError(b, err)
			table, err := colDB.Table(
				"stacktraces",
				columnstore.NewTableConfig(
					parcacol.Schema(),
				),
				logger,
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
			ingester := parcacol.NewIngester(logger, normalizer, table)

			profiles, err := normalizer.NormalizePprof(ctx, "memory", p, false)
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
				m,
				query.NewEngine(
					memory.NewGoAllocator(),
					colDB.TableProvider(),
				),
				"stacktraces",
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
