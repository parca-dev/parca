// Copyright 2023-2026 The Parca Authors
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
	"os"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/parcacol"
)

func TestSourcesOnlyRequest(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := noop.NewTracerProvider().Tracer("")

	f, err := os.Open("testdata/source.tar.zstd")
	require.NoError(t, err)
	defer f.Close()

	bucket := objstore.NewInMemBucket()
	require.NoError(t, bucket.Upload(ctx, "test/sources", f))

	allocator := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer allocator.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		nil,
		parcacol.NewQuerier(
			logger,
			tracer,
			nil,
			"stacktraces",
			nil,
			allocator,
		),
		allocator,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		NewBucketSourceFinder(bucket, &debuginfo.NopDebuginfodClients{}),
	)

	resp, err := api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test:samples:count:cpu:nanoseconds",
				Start: timestamppb.New(time.Now()),
				End:   timestamppb.New(time.Now().Add(time.Hour)),
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_SOURCE,
		SourceReference: &pb.SourceReference{
			SourceOnly: true,
			BuildId:    "test",
			Filename:   "file",
		},
	})
	require.NoError(t, err)
	require.Equal(t, "", resp.GetSource().Source)

	_, err = api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test:samples:count:cpu:nanoseconds",
				Start: timestamppb.New(time.Now()),
				End:   timestamppb.New(time.Now().Add(time.Hour)),
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_SOURCE,
		SourceReference: &pb.SourceReference{
			SourceOnly: true,
			BuildId:    "test1",
			Filename:   "file",
		},
	})
	require.ErrorContains(t, err, "rpc error: code = NotFound desc = no sources for this build id have been uploaded")

	resp, err = api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test:samples:count:cpu:nanoseconds",
				Start: timestamppb.New(time.Now()),
				End:   timestamppb.New(time.Now().Add(time.Hour)),
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_SOURCE,
		SourceReference: &pb.SourceReference{
			SourceOnly: true,
			BuildId:    "test",
			Filename:   "metadata.go",
		},
	})
	require.NoError(t, err)

	require.Equal(t, 0, len(resp.GetSource().Source)) // Source only only checks if any sources exist it doesn't retrieve them.
}

func TestSourceReportArrowSchema(t *testing.T) {
	allocator := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer allocator.AssertSize(t, 0)

	builder := newSourceReportBuilder(allocator, &pb.SourceReference{
		BuildId:  "test-build-id",
		Filename: "test.go",
	})

	builder.lineData[10] = &lineMetrics{cumulative: 100, flat: 50}
	builder.lineData[25] = &lineMetrics{cumulative: 200, flat: 75}
	builder.lineData[5] = &lineMetrics{cumulative: 50, flat: 25}

	record, cumulative := builder.finish()
	defer record.Release()

	require.Equal(t, int64(0), cumulative)

	schema := record.Schema()
	require.Equal(t, 3, schema.NumFields())

	require.Equal(t, "line_number", schema.Field(0).Name)
	require.Equal(t, "cumulative", schema.Field(1).Name)
	require.Equal(t, "flat", schema.Field(2).Name)

	require.Equal(t, int64(3), record.NumRows())

	lineNumbers := record.Column(0)
	require.Equal(t, int64(5), lineNumbers.(*array.Int64).Value(0))
	require.Equal(t, int64(10), lineNumbers.(*array.Int64).Value(1))
	require.Equal(t, int64(25), lineNumbers.(*array.Int64).Value(2))

	cumulativeCol := record.Column(1)
	require.Equal(t, int64(50), cumulativeCol.(*array.Int64).Value(0))
	require.Equal(t, int64(100), cumulativeCol.(*array.Int64).Value(1))
	require.Equal(t, int64(200), cumulativeCol.(*array.Int64).Value(2))

	flatCol := record.Column(2)
	require.Equal(t, int64(25), flatCol.(*array.Int64).Value(0))
	require.Equal(t, int64(50), flatCol.(*array.Int64).Value(1))
	require.Equal(t, int64(75), flatCol.(*array.Int64).Value(2))
}
