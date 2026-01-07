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

	_, err = api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test_profile",
				Start: timestamppb.New(time.Now()),
				End:   timestamppb.New(time.Now().Add(time.Hour)),
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_SOURCE,
		SourceReference: &pb.SourceReference{
			BuildId:  "test",
			Filename: "file",
		},
	})
	require.ErrorContains(t, err, "rpc error: code = NotFound desc = source file not found; either profiling metadata is wrong, or the referenced file was not included in the uploaded sources")

	_, err = api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test_profile",
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

	resp, err := api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_MERGE,
		Options: &pb.QueryRequest_Merge{
			Merge: &pb.MergeProfile{
				Query: "test_profile",
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
