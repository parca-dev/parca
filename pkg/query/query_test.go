package query

import (
	"context"
	"testing"
	"time"

	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/parca-dev/parca/storage"
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
