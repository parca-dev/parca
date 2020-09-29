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

package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/go-kit/kit/log"
	"github.com/gogo/status"
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
				Name:  "__name__",
				Value: "allocs",
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
	q := store.NewGRPCQueryable(c)
	api := New(log.NewNopLogger(), q, make(chan struct{}))

	req := httptest.NewRequest("GET", "http://example.com/query_range?from=0&to=10&query=allocs", nil)
	w := httptest.NewRecorder()
	api.QueryRange(w, req, nil)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		t.Fatalf("Unexpected status code, expected 200, got %d", resp.StatusCode)
	}

	expectedContentType := "application/json"
	gotContentType := resp.Header.Get("Content-Type")
	if gotContentType != expectedContentType {
		t.Fatalf("Unexpected Content-Type, expected %s, got %s", expectedContentType, gotContentType)
	}

	queryResult := QueryResult{}
	err = json.Unmarshal(body, &queryResult)
	if err != nil {
		t.Fatalf("Failed to unmarshal query result")
	}

	queryResultLen := len(queryResult.Series)
	if queryResultLen != 1 {
		t.Fatalf("Unexpected series in query result. Expected 1, got %d", queryResultLen)
	}

	series := queryResult.Series[0]

	expectedLabels := map[string]string{"__name__": "allocs"}
	if !reflect.DeepEqual(series.Labels, expectedLabels) {
		t.Fatalf("Unexpected labels, expected %s, got %s", fmt.Sprintf("%#+v", expectedLabels), fmt.Sprintf("%#+v", series.Labels))
	}

	expectedTimestamps := []int64{1, 5}
	if !reflect.DeepEqual(series.Timestamps, expectedTimestamps) {
		t.Fatalf("Unexpected timestamps, expected %s, got %s", fmt.Sprintf("%#+v", expectedTimestamps), fmt.Sprintf("%#+v", series.Timestamps))
	}
}
