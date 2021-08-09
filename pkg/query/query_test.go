package query

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/proto/gen/go/profilestore"
	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/parca-dev/parca/storage"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
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
