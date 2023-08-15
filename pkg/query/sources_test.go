package query

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
)

func TestSourcesOnlyRequest(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	f, err := os.Open("testdata/source.tar.zstd")
	require.NoError(t, err)
	defer f.Close()

	bucket := objstore.NewInMemBucket()
	require.NoError(t, bucket.Upload(ctx, "test/sources", f))

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
			memory.DefaultAllocator,
		),
		memory.DefaultAllocator,
		parcacol.NewArrowToProfileConverter(tracer, metastore.NewKeyMaker()),
		NewBucketSourceFinder(bucket),
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
			SourceOnly: true,
			BuildId:    "test",
			Filename:   "file",
		},
	})
	require.ErrorContains(t, err, "rpc error: code = NotFound desc = Source file not found. Either profiling metadata is wrong, or the referenced file was not included in the uploaded sources.")

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
	require.ErrorContains(t, err, "rpc error: code = NotFound desc = No sources for this build id have been uploaded.")

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

	require.Equal(t, 5045, len(resp.GetSource().Source))
}
