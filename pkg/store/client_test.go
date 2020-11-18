// Copyright 2020 The conprof Authors
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

package store

import (
	"context"
	"net"
	"testing"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/gogo/status"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

type fakeProfileStore struct{}

func (s *fakeProfileStore) Write(ctx context.Context, r *storepb.WriteRequest) (*storepb.WriteResponse, error) {
	return nil, nil
}

func (s *fakeProfileStore) Series(r *storepb.SeriesRequest, srv storepb.ReadableProfileStore_SeriesServer) error {
	c := chunkenc.NewBytesChunk()
	app, err := c.Appender()
	if err != nil {
		return err
	}
	app.Append(1, []byte{})
	app.Append(5, []byte{})

	if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{
		Labels: []labelpb.Label{
			{
				Name:  "x",
				Value: "y",
			},
		},
		Chunks: []storepb.AggrChunk{
			{
				MinTime: 0,
				MaxTime: 10,
				Raw: &storepb.Chunk{
					Type: 1,
					Data: c.Bytes(),
				},
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

func (s *fakeProfileStore) LabelNames(ctx context.Context, r *storepb.LabelNamesRequest) (*storepb.LabelNamesResponse, error) {
	return nil, nil
}

func (s *fakeProfileStore) LabelValues(ctx context.Context, r *storepb.LabelValuesRequest) (*storepb.LabelValuesResponse, error) {
	return nil, nil
}

func TestAPIQueryRangeGRPCCall(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	s := &fakeProfileStore{}
	storepb.RegisterWritableProfileStoreServer(grpcServer, s)
	storepb.RegisterReadableProfileStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewReadableProfileStoreClient(conn)
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
