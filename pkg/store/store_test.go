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
	"github.com/thanos-io/thanos/pkg/store/labelpb"
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
				Labels: []labelpb.Label{
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
	storepb.RegisterWritableProfileStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewWritableProfileStoreClient(conn)
	q := NewGRPCAppendable(log.NewNopLogger(), c)

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
	storepb.RegisterWritableProfileStoreServer(grpcServer, s)
	storepb.RegisterReadableProfileStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}
	c := storepb.NewWritableProfileStoreClient(conn)
	a := NewGRPCAppendable(log.NewNopLogger(), c)

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

	rc := storepb.NewReadableProfileStoreClient(conn)
	q := NewGRPCQueryable(rc)

	httpapi := api.New(log.NewNopLogger(), prometheus.NewRegistry(), q, make(chan struct{}), api.DefaultMergeBatchSize, api.NoTargets)

	req := httptest.NewRequest("GET", "http://example.com/query_range?from=0&to=10&query=allocs", nil)

	result, warnings, apiErr := httpapi.QueryRange(req)
	if apiErr != nil && apiErr.Err != nil {
		t.Fatalf("Unexpected err: %v", apiErr)
	}

	series, ok := result.([]api.Series)
	if !ok {
		t.Fatalf("Unexpected return value")
	}

	if len(warnings) != 0 {
		t.Fatalf("Unexpected warnings length %d", len(warnings))
	}

	queryResultLen := len(series)
	if queryResultLen != 1 {
		t.Fatalf("Unexpected series in query result. Expected 1, got %d", queryResultLen)
	}

	res := series[0]
	expectedLabels := map[string]string{"__name__": "allocs"}
	if !reflect.DeepEqual(res.Labels, expectedLabels) {
		t.Fatalf("Unexpected labels, expected %s, got %s", fmt.Sprintf("%#+v", expectedLabels), fmt.Sprintf("%#+v", res.Labels))
	}

	expectedTimestamps := []int64{5}
	if !reflect.DeepEqual(res.Timestamps, expectedTimestamps) {
		t.Fatalf("Unexpected timestamps, expected %s, got %s", fmt.Sprintf("%#+v", expectedTimestamps), fmt.Sprintf("%#+v", res.Timestamps))
	}
}
