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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"google.golang.org/grpc"
)

type fakeAppender struct {
	storage.Queryable
	storage.ChunkQueryable

	l labels.Labels
	t int64
	v []byte
}

var _ storage.Appendable = &fakeAppender{}

func (a *fakeAppender) Appender(ctx context.Context) storage.Appender {
	return a
}

func (a *fakeAppender) Add(l labels.Labels, t int64, v []byte) (uint64, error) {
	a.l = l
	a.t = t
	a.v = v
	return 0, nil
}

func (a *fakeAppender) AddFast(ref uint64, t int64, v []byte) error {
	return errors.New("not implemented")
}

func (a *fakeAppender) Commit() error {
	return nil
}

func (a *fakeAppender) Rollback() error {
	return errors.New("not implemented")
}

func TestStoreWrite(t *testing.T) {
	a := &fakeAppender{}
	s := NewProfileStore(log.NewNopLogger(), a, 100000)
	_, err := s.Write(context.Background(), &storepb.WriteRequest{
		ProfileSeries: []storepb.ProfileSeries{
			{
				Labels: []storepb.Label{
					{
						Name:  "__name__",
						Value: "allocs",
					},
				},
				Samples: []storepb.Sample{
					{
						Timestamp: 10,
						Value:     []byte("test"),
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	expectedLabels := labels.Labels{labels.Label{Name: "__name__", Value: "allocs"}}
	if !reflect.DeepEqual(expectedLabels, a.l) {
		t.Fatal("unexpected labels written")
	}

	expectedTimestamp := int64(10)
	if expectedTimestamp != a.t {
		t.Fatal("unexpected timestamp written")
	}

	expectedValue := []byte("test")
	if !bytes.Equal(expectedValue, a.v) {
		t.Fatal("unexpected value written")
	}
}

func TestGRPCAppendable(t *testing.T) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	a := &fakeAppender{}
	s := NewProfileStore(log.NewNopLogger(), a, 100000)
	storepb.RegisterProfileStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewProfileStoreClient(conn)
	q := NewGRPCAppendable(c)

	app := q.Appender(context.Background())
	_, err = app.Add(labels.Labels{
		{
			Name:  "__name__",
			Value: "allocs",
		},
	},
		10,
		[]byte("test"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = app.Commit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedLabels := labels.Labels{labels.Label{Name: "__name__", Value: "allocs"}}
	if !reflect.DeepEqual(expectedLabels, a.l) {
		t.Fatalf("unexpected labels written, expected %#+v, got %#+v", expectedLabels, a.l)
	}

	expectedTimestamp := int64(10)
	if expectedTimestamp != a.t {
		t.Fatalf("unexpected timestamp written, expected %d, got %d", expectedTimestamp, a.t)
	}

	expectedValue := []byte("test")
	if !bytes.Equal(expectedValue, a.v) {
		t.Fatalf("unexpected value written, expected %#+v, got %#+v", expectedValue, a.v)
	}
}

func TestStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "conprof-test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // clean up

	db, err := tsdb.Open(
		dir,
		log.NewNopLogger(),
		prometheus.DefaultRegisterer,
		&tsdb.Options{
			RetentionDuration:      int64(15 * 24 * time.Hour / time.Millisecond),
			WALSegmentSize:         wal.DefaultSegmentSize,
			MinBlockDuration:       tsdb.DefaultBlockDuration,
			MaxBlockDuration:       tsdb.DefaultBlockDuration,
			NoLockfile:             true,
			AllowOverlappingBlocks: false,
			WALCompression:         true,
			StripeSize:             tsdb.DefaultStripeSize,
		},
	)
	if err != nil {
		t.Fatalf("failed to open tsdb: %v", err)
	}

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	s := NewProfileStore(log.NewNopLogger(), db, 100000)
	storepb.RegisterProfileStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewProfileStoreClient(conn)
	a := NewGRPCAppendable(c)

	app := a.Appender(context.Background())
	_, err = app.Add(labels.Labels{
		{
			Name:  "__name__",
			Value: "allocs",
		},
	},
		5,
		[]byte("test"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = app.Commit()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q := NewGRPCQueryable(c)

	httpapi := api.New(log.NewNopLogger(), q, make(chan struct{}))

	req := httptest.NewRequest("GET", "http://example.com/query_range?from=0&to=10&query=allocs", nil)
	w := httptest.NewRecorder()
	httpapi.QueryRange(w, req, nil)

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

	queryResult := api.QueryResult{}
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

	expectedTimestamps := []int64{5}
	if !reflect.DeepEqual(series.Timestamps, expectedTimestamps) {
		t.Fatalf("Unexpected timestamps, expected %s, got %s", fmt.Sprintf("%#+v", expectedTimestamps), fmt.Sprintf("%#+v", series.Timestamps))
	}
}
