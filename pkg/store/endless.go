package store

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// EndlessProfileStore is a no-op writable store and returns infinite amounts
// of series when reading. This is meant for testing timeout issues.
type EndlessProfileStore struct{}

func NewEndlessProfileStore() *EndlessProfileStore {
	return &EndlessProfileStore{}
}

func (s *EndlessProfileStore) Write(ctx context.Context, r *storepb.WriteRequest) (*storepb.WriteResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) Series(r *storepb.SeriesRequest, srv storepb.ReadableProfileStore_SeriesServer) error {
	ctx := srv.Context()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c := chunkenc.NewBytesChunk()
	app, err := c.Appender()
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile("./testdata/alloc_objects.pb.gz")
	if err != nil {
		return err
	}

	app.Append(1, b)
	app.Append(5, b)

	cbytes, err := c.Bytes()
	if err != nil {
		return err
	}

	i := 0
	for {
		if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{
			Labels: []labelpb.Label{
				{
					Name:  "__name__",
					Value: "allocs",
				},
				{
					Name:  "count",
					Value: fmt.Sprintf("%d", i),
				},
			},
			Chunks: []storepb.AggrChunk{
				{
					MinTime: 0,
					MaxTime: 10,
					Raw: &storepb.Chunk{
						Type: 1,
						Data: cbytes,
					},
				},
			},
		})); err != nil {
			return grpcstatus.Error(codes.Aborted, err.Error())
		}
		i++
	}
}

func (s *EndlessProfileStore) Profile(ctx context.Context, r *storepb.ProfileRequest) (*storepb.ProfileResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) LabelNames(ctx context.Context, r *storepb.LabelNamesRequest) (*storepb.LabelNamesResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) LabelValues(ctx context.Context, r *storepb.LabelValuesRequest) (*storepb.LabelValuesResponse, error) {
	return nil, nil
}
