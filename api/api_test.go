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
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/prometheus/common/route"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"

	"github.com/conprof/conprof/config"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/pkg/testutil"
	"github.com/conprof/db/tsdb/chunkenc"
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
				Name:  "__name__",
				Value: "allocs",
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
		return grpcstatus.Error(codes.Aborted, err.Error())
	}

	if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{
		Labels: []labelpb.Label{
			{
				Name:  "__name__",
				Value: "heap",
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
		return grpcstatus.Error(codes.Aborted, err.Error())
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

type endpointTestCase struct {
	endpoint ApiFunc
	params   map[string]string
	query    url.Values
	response interface{}
	warn     []error
	errType  ErrorType
}

func testEndpoint(t *testing.T, test endpointTestCase, name string) bool {
	return t.Run(name, func(t *testing.T) {
		// Build a context with the correct request params.
		ctx := context.Background()
		for p, v := range test.params {
			ctx = route.WithParam(ctx, p, v)
		}

		reqURL := "http://example.com"
		params := test.query.Encode()

		var body io.Reader
		reqURL += "?" + params

		req, err := http.NewRequest(http.MethodGet, reqURL, body)
		if err != nil {
			t.Fatal(err)
		}

		resp, warn, apiErr := test.endpoint(req.WithContext(ctx))
		if apiErr != nil {
			if test.errType == ErrorNone {
				t.Fatalf("Unexpected error: %s", apiErr)
			}
			if test.errType != apiErr.Typ {
				t.Fatalf("Expected error of type %q but got type %q", test.errType, apiErr.Typ)
			}
			return
		}
		if test.errType != ErrorNone {
			t.Fatalf("Expected error of type %q but got none", test.errType)
		}
		if !reflect.DeepEqual(warn, test.warn) {
			t.Fatalf("Warnings do not match, expected:\n%+v\ngot:\n%+v", test.warn, warn)
		}

		if !reflect.DeepEqual(resp, test.response) {
			t.Fatalf("Response does not match, expected:\n%+v\ngot:\n%+v", test.response, resp)
		}
	})
}

func TestAPIQueryRangeGRPCCall(t *testing.T) {
	api, closer := createFakeGRPCAPI(t)
	defer closer.Close()
	var tests = []endpointTestCase{
		{
			endpoint: api.QueryRange,
			query: url.Values{
				"query": []string{"allocs"},
				"from":  []string{"0"},
				"to":    []string{"10"},
			},
			response: []Series{
				{
					Labels:     map[string]string{"__name__": "allocs"},
					Timestamps: []int64{1, 5},
				},
				{
					Labels:     map[string]string{"__name__": "heap"},
					Timestamps: []int64{1, 5},
				},
			},
		},
		// limit to 1 series
		{
			endpoint: api.QueryRange,
			query: url.Values{
				"query": []string{"allocs"},
				"from":  []string{"0"},
				"to":    []string{"10"},
				"limit": []string{"1"},
			},
			warn: []error{fmt.Errorf("retrieved %d series, more available", 1)},
			response: []Series{
				{
					Labels:     map[string]string{"__name__": "allocs"},
					Timestamps: []int64{1, 5},
				},
			},
		},
		// from and to not set.
		{
			endpoint: api.QueryRange,
			query:    url.Values{"query": []string{"allocs"}},
			errType:  ErrorBadData,
		},
		// Invalid format.
		{
			endpoint: api.QueryRange,
			query:    url.Values{"query": []string{"allocs"}, "from": []string{"aaa"}, "to": []string{"10"}},
			errType:  ErrorBadData,
		},
		// to time before from time
		{
			endpoint: api.QueryRange,
			query:    url.Values{"query": []string{"allocs"}, "from": []string{"9"}, "to": []string{"1"}},
			errType:  ErrorBadData,
		},
	}

	for i, test := range tests {
		if ok := testEndpoint(t, test, fmt.Sprintf("#%d %s", i, test.query.Encode())); !ok {
			return
		}
	}
}

func TestAPILabelNames(t *testing.T) {
	lbls := []labels.Labels{
		{
			labels.Label{Name: "__name__", Value: "allocs"},
			labels.Label{Name: "foo", Value: "bar"},
		},
		{
			labels.Label{Name: "__name__", Value: "goroutine"},
			labels.Label{Name: "foo", Value: "boo"},
			labels.Label{Name: "baz", Value: "faz"},
		},
	}

	db, err := testutil.NewTSDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
	}()

	app := db.Appender(context.Background())
	for _, lbl := range lbls {
		for i := int64(0); i < 10; i++ {
			_, err := app.Add(lbl, timestamp.FromTime(time.Now()), []byte{byte(i)})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := app.Commit(); err != nil {
		t.Fatal(err)
	}

	api := API{log.NewNopLogger(), db, make(chan struct{}), DefaultMergeBatchSize, sync.RWMutex{}, &config.Config{}}
	var tests = []endpointTestCase{
		{
			endpoint: api.LabelNames,
			query:    url.Values{},
			response: []string{"__name__", "baz", "foo"},
		},
		{
			endpoint: api.LabelNames,
			query:    url.Values{"match[]": []string{"allocs"}},
			response: []string{"__name__", "foo"},
		},
		// Invalid format.
		{
			endpoint: api.LabelNames,
			query:    url.Values{"start": []string{"aaa"}, "end": []string{"10"}},
			errType:  ErrorBadData,
		},
		// to time before from time
		{
			endpoint: api.LabelNames,
			query:    url.Values{"start": []string{"9"}, "end": []string{"1"}},
			errType:  ErrorBadData,
		},
	}

	for i, test := range tests {
		if ok := testEndpoint(t, test, fmt.Sprintf("#%d %s", i, test.query.Encode())); !ok {
			return
		}
	}
}

func TestAPILabelValues(t *testing.T) {
	lbls := []labels.Labels{
		{
			labels.Label{Name: "__name__", Value: "allocs"},
			labels.Label{Name: "foo", Value: "bar"},
		},
		{
			labels.Label{Name: "__name__", Value: "goroutine"},
			labels.Label{Name: "foo", Value: "boo"},
		},
	}

	db, err := testutil.NewTSDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
	}()

	app := db.Appender(context.Background())
	for _, lbl := range lbls {
		for i := int64(0); i < 10; i++ {
			_, err := app.Add(lbl, timestamp.FromTime(time.Now()), []byte{byte(i)})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := app.Commit(); err != nil {
		t.Fatal(err)
	}

	api := API{log.NewNopLogger(), db, make(chan struct{}), DefaultMergeBatchSize, sync.RWMutex{}, &config.Config{}}
	var tests = []endpointTestCase{
		{
			endpoint: api.LabelValues,
			params: map[string]string{
				"name": "__name__",
			},
			response: []string{"allocs", "goroutine"},
		},
		{
			endpoint: api.LabelValues,
			params: map[string]string{
				"name": "__name__",
			},
			query:    url.Values{"match[]": []string{"{foo=\"bar\"}"}},
			response: []string{"allocs"},
		},
		// Invalid format.
		{
			endpoint: api.LabelValues,
			query:    url.Values{"start": []string{"aaa"}, "end": []string{"10"}},
			errType:  ErrorBadData,
		},
		// to time before from time
		{
			endpoint: api.LabelValues,
			query:    url.Values{"start": []string{"9"}, "end": []string{"1"}},
			errType:  ErrorBadData,
		},
	}

	for i, test := range tests {
		if ok := testEndpoint(t, test, fmt.Sprintf("#%d %s", i, test.query.Encode())); !ok {
			return
		}
	}
}

func TestAPISeries(t *testing.T) {
	lbls := []labels.Labels{
		{
			labels.Label{Name: "__name__", Value: "allocs"},
			labels.Label{Name: "foo", Value: "bar"},
		},
		{
			labels.Label{Name: "__name__", Value: "goroutine"},
			labels.Label{Name: "foo", Value: "boo"},
		},
	}

	db, err := testutil.NewTSDB()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		db.Close()
	}()

	app := db.Appender(context.Background())
	for _, lbl := range lbls {
		for i := int64(0); i < 10; i++ {
			_, err := app.Add(lbl, timestamp.FromTime(time.Now()), []byte{byte(i)})
			if err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := app.Commit(); err != nil {
		t.Fatal(err)
	}

	api := API{log.NewNopLogger(), db, make(chan struct{}), DefaultMergeBatchSize, sync.RWMutex{}, &config.Config{}}
	var tests = []endpointTestCase{
		{
			endpoint: api.Series,
			errType:  ErrorBadData,
		},
		{
			endpoint: api.Series,
			query: url.Values{
				"match[]": []string{`allocs`},
			},
			response: []labels.Labels{
				labels.FromStrings("__name__", "allocs", "foo", "bar"),
			},
		},
		// Invalid format.
		{
			endpoint: api.Series,
			query:    url.Values{"start": []string{"aaa"}, "end": []string{"10"}},
			errType:  ErrorBadData,
		},
		// to time before from time
		{
			endpoint: api.Series,
			query:    url.Values{"start": []string{"9"}, "end": []string{"1"}},
			errType:  ErrorBadData,
		},
	}

	for i, test := range tests {
		if ok := testEndpoint(t, test, fmt.Sprintf("#%d %s", i, test.query.Encode())); !ok {
			return
		}
	}
}

func createFakeGRPCAPI(t *testing.T) (*API, io.Closer) {
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	storepb.RegisterReadableProfileStoreServer(grpcServer, &fakeProfileStore{})
	storepb.RegisterWritableProfileStoreServer(grpcServer, &fakeProfileStore{})
	go grpcServer.Serve(lis)

	storeAddress := lis.Addr().String()

	conn, err := grpc.Dial(storeAddress, grpc.WithInsecure())
	if err != nil {
		t.Fatal(err)
	}

	c := storepb.NewReadableProfileStoreClient(conn)
	q := store.NewGRPCQueryable(c)
	return New(log.NewNopLogger(), q, make(chan struct{}), DefaultMergeBatchSize), lis
}
