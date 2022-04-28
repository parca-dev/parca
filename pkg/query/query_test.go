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
	"os"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
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
			m := metastore.NewBadgerMetastore(
				logger,
				reg,
				tracer,
				metastore.NewRandomUUIDGenerator(),
			)

			f, err := os.Open("../query/testdata/alloc_objects.pb.gz")
			require.NoError(b, err)
			p1, err := profile.Parse(f)
			require.NoError(b, err)
			require.NoError(b, f.Close())

			for _, s := range p1.Sample {
				s.Label = nil
				s.NumLabel = nil
				s.NumUnit = nil
			}

			p1 = p1.Compact()

			p, err := parcaprofile.FromPprof(ctx, log.NewNopLogger(), m, p1, 0, false)
			require.NoError(b, err)

			for j := 0; j < n; j++ {
				p.Meta.Timestamp = int64(j + 1)
				_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{}, p)
				require.NoError(b, err)
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
							Query: `{__name__="alloc_objects_count"}`,
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
