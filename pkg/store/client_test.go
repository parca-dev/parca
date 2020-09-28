package store

import (
	"context"
	"net"
	"testing"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/gogo/status"
	"github.com/prometheus/prometheus/pkg/labels"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type fakeProfileStore struct{}

func (s *fakeProfileStore) Write(ctx context.Context, r *storepb.WriteRequest) (*storepb.WriteResponse, error) {
	return nil, nil
}

func (s *fakeProfileStore) Series(r *storepb.SeriesRequest, srv storepb.ProfileStore_SeriesServer) error {
	c := chunkenc.NewBytesChunk()
	app, err := c.Appender()
	if err != nil {
		return err
	}
	app.Append(1, []byte{})
	app.Append(5, []byte{})

	if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{
		Labels: []storepb.Label{
			{
				Name:  "x",
				Value: "y",
			},
		},
		Chunks: []storepb.Chunk{
			{
				MinTime: 0,
				MaxTime: 10,
				Type:    1,
				Data:    c.Bytes(),
			},
		},
	})); err != nil {
		return status.Error(codes.Aborted, err.Error())
	}
	return nil
}

func (s *fakeProfileStore) Profile(ctx context.Context, r *storepb.ProfileRequest) (*storepb.ProfileResponse, error) {
	return nil, nil
}

func TestAPIQueryRangeGRPCCall(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	storepb.RegisterProfileStoreServer(grpcServer, &fakeProfileStore{})
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewProfileStoreClient(conn)
	q := NewGRPCQueryable(c)

	qr, err := q.Querier(context.Background(), 0, 10)
	if err != nil {
		t.Fatal(err)
	}

	ss := qr.Select(false, nil, labels.MustNewMatcher(labels.MatchEqual, "__name__", "allocs"))

	if !ss.Next() {
		if ss.Err() != nil {
			t.Fatal(ss.Err())
		}
		t.Fatal("Expected a next series, but didn't get any")
	}
}
