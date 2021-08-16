package query

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage"
	"github.com/parca-dev/parca/proto/gen/go/profilestore"
	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func Test_QueryRange_EmptyStore(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB()
	q := New(db, nil)

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(5 * time.Minute)

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: 10,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Series)
}

func Test_QueryRange_Valid(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB()
	s := storage.NewInMemoryProfileMetaStore()
	q := New(db, s)

	appender := db.Appender(ctx, labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
	})

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	appender.Append(storage.ProfileFromPprof(s, p))

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(5 * time.Minute)

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: 10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Series)
	require.Equal(t, 1, len(resp.Series))
	require.Equal(t, 1, len(resp.Series[0].Samples))
	require.Equal(t, &profilestore.LabelSet{
		Labels: []*profilestore.Label{
			{
				Name:  "__name__",
				Value: "allocs",
			},
		},
	}, resp.Series[0].Labelset)
	require.Equal(t, int64(310797348), resp.Series[0].Samples[0].Value)
}

func Test_QueryRange_Limited(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB()
	s := storage.NewInMemoryProfileMetaStore()
	q := New(db, s)

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	numSeries := 10
	for i := 0; i < numSeries; i++ {
		appender := db.Appender(ctx, labels.Labels{
			labels.Label{
				Name:  "__name__",
				Value: "allocs",
			},
			labels.Label{
				Name:  "meta",
				Value: fmt.Sprintf("series_%v", i),
			},
		})
		appender.Append(storage.ProfileFromPprof(s, p))
	}

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(5 * time.Minute)

	limit := rand.Intn(numSeries)
	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: uint32(limit),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Series)
	require.Equal(t, limit, len(resp.Series))
	for i := 0; i < limit; i++ {
		require.Equal(t, 1, len(resp.Series[i].Samples))
	}
}

func Test_QueryRange_InputValidation(t *testing.T) {
	ctx := context.Background()
	end := time.Now()
	start := end.Add(5 * time.Minute)

	tests := map[string]struct {
		req *pb.QueryRangeRequest
	}{
		"Empty query": {
			req: &pb.QueryRangeRequest{
				Query: "",
				Start: timestamppb.New(start),
				End:   timestamppb.New(end),
			},
		},
		"Empty start": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: nil,
				End:   timestamppb.New(end),
			},
		},
		"Empty End": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: timestamppb.New(start),
				End:   nil,
			},
		},
		"End before start": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: timestamppb.New(end),
				End:   timestamppb.New(start),
			},
		},
	}

	q := New(nil, nil)

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := q.QueryRange(ctx, test.req)
			require.Error(t, err)
			require.Empty(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func Test_Query_InputValidation(t *testing.T) {
	ctx := context.Background()

	invalidMode := pb.QueryRequest_Mode(1000)
	invalidReportType := pb.QueryRequest_ReportType(1000)

	tests := map[string]struct {
		req *pb.QueryRequest
	}{
		"Invalid mode": {
			req: &pb.QueryRequest{
				Mode:       &invalidMode,
				Options:    &pb.QueryRequest_Single_{},
				ReportType: pb.QueryRequest_Flamegraph.Enum(),
			},
		},
		"Invalid report type": {
			req: &pb.QueryRequest{
				Mode:       pb.QueryRequest_SINGLE.Enum(),
				Options:    &pb.QueryRequest_Single_{},
				ReportType: &invalidReportType,
			},
		},
		"option doesn't match mode": {
			req: &pb.QueryRequest{
				Mode:       pb.QueryRequest_SINGLE.Enum(),
				Options:    &pb.QueryRequest_Merge_{},
				ReportType: pb.QueryRequest_Flamegraph.Enum(),
			},
		},
		"option not provided": {
			req: &pb.QueryRequest{
				Mode:       pb.QueryRequest_SINGLE.Enum(),
				Options:    nil,
				ReportType: pb.QueryRequest_Flamegraph.Enum(),
			},
		},
	}

	q := New(nil, nil)

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := q.Query(ctx, test.req)
			require.Error(t, err)
			require.Empty(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}
