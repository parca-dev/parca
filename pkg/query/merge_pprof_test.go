// Copyright 2022-2026 The Parca Authors
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
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	columnstore "github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/ingester"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/profilestore"
)

// A merged pprof report carries the query start as its time, not a value
// that overflowed when a nanosecond timestamp was scaled as milliseconds.
func TestMergePprofTimeIsQueryStart(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := noop.NewTracerProvider().Tracer("")

	col, err := columnstore.New()
	require.NoError(t, err)
	t.Cleanup(func() { col.Close() })
	colDB, err := col.DB(ctx, "parca")
	require.NoError(t, err)
	schema, err := profile.Schema()
	require.NoError(t, err)
	table, err := colDB.Table("stacktraces", columnstore.NewTableConfig(profile.SchemaDefinition()))
	require.NoError(t, err)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	store := profilestore.NewProfileColumnStore(
		prometheus.NewRegistry(),
		logger,
		tracer,
		ingester.NewIngester(logger, table),
		schema,
		memory.DefaultAllocator,
	)
	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{Labels: []*profilestorepb.Label{
				{Name: "__name__", Value: "memory"},
				{Name: "job", Value: "default"},
			}},
			Samples: []*profilestorepb.RawSample{{RawProfile: fileContent}},
		}},
	})
	require.NoError(t, err)
	require.NoError(t, table.EnsureCompaction())

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(logger, tracer, query.NewEngine(mem, colDB.TableProvider()), "stacktraces", nil, nil, mem),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
	)

	start := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	resp, err := api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{Merge: &pb.MergeProfile{
			Query: `{__name__="memory:alloc_objects:count:space:bytes"}`,
			Start: timestamppb.New(start),
			End:   timestamppb.New(start.AddDate(10, 0, 0)),
		}},
		ReportType: pb.QueryRequest_REPORT_TYPE_PPROF,
	})
	require.NoError(t, err)

	p, err := pprofprofile.Parse(bytes.NewReader(resp.GetPprof()))
	require.NoError(t, err)
	require.NotEmpty(t, p.Sample)
	require.Equal(t, start.UnixNano(), p.TimeNanos)
}
