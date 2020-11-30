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

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"net/url"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/test/e2e/e2econprof"
	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb/tsdbutil"
	"github.com/cortexproject/cortex/integration/e2e"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/tsdb/testutil"
	"google.golang.org/grpc"
)

type testSample struct {
	timestamp int64
	value     []byte
}

func (s *testSample) T() int64 {
	return s.timestamp
}

func (s *testSample) V() []byte {
	return s.value
}

func TestStorage(t *testing.T) {
	t.Parallel()

	t.Run("append-restart-append-read", func(t *testing.T) {
		t.Parallel()

		ctx := context.Background()

		s, err := e2e.NewScenario("e2e_test_receive_append_restart_append_read")
		testutil.Ok(t, err)
		t.Cleanup(e2econprof.CleanScenario(t, s))

		d := s.SharedDir()

		st1, err := e2econprof.NewStorage(d, s.NetworkName(), "1", "test")
		testutil.Ok(t, err)
		testutil.Ok(t, s.StartAndWaitReady(st1))

		grpcAddress := st1.GRPCEndpoint()
		conn, err := grpc.Dial(grpcAddress, grpc.WithInsecure())
		if err != nil {
			t.Fatal(err)
		}
		c := storepb.NewWritableProfileStoreClient(conn)
		db := store.NewGRPCAppendable(log.NewNopLogger(), c)

		firstSampleSet := []*testSample{}

		for i := 0; i < 50; i++ {
			app := db.Appender(ctx)

			b := bytes.NewBuffer(nil)
			pprof.WriteHeapProfile(b)
			byt := b.Bytes()
			sample := &testSample{
				timestamp: timestamp.FromTime(time.Now()),
				value:     byt,
			}
			firstSampleSet = append(firstSampleSet, sample)

			_, err = app.Add(labels.FromStrings("__name__", "heap"), sample.timestamp, sample.value)
			testutil.Ok(t, err)

			err = app.Commit()
			testutil.Ok(t, err)

			time.Sleep(100 * time.Millisecond)
		}

		err = conn.Close()
		testutil.Ok(t, err)
		err = st1.Stop()
		testutil.Ok(t, err)

		st2, err := e2econprof.NewStorage(d, s.NetworkName(), "2", "test")
		testutil.Ok(t, err)
		testutil.Ok(t, s.StartAndWaitReady(st2))
		defer st2.Stop()

		grpcAddress = st2.GRPCEndpoint()
		conn, err = grpc.Dial(grpcAddress, grpc.WithInsecure())
		if err != nil {
			t.Fatal(err)
		}
		c = storepb.NewWritableProfileStoreClient(conn)
		db = store.NewGRPCAppendable(log.NewNopLogger(), c)

		secondSampleSet := []*testSample{}

		for i := 0; i < 50; i++ {
			app := db.Appender(ctx)

			b := bytes.NewBuffer(nil)
			pprof.WriteHeapProfile(b)
			byt := b.Bytes()
			sample := &testSample{
				timestamp: timestamp.FromTime(time.Now()),
				value:     byt,
			}
			secondSampleSet = append(secondSampleSet, sample)

			_, err = app.Add(labels.FromStrings("__name__", "heap"), sample.timestamp, sample.value)
			testutil.Ok(t, err)

			err = app.Commit()
			testutil.Ok(t, err)

			time.Sleep(100 * time.Millisecond)
		}

		a, err := e2econprof.NewAPI(s.NetworkName(), "1", st2.GRPCNetworkEndpoint())
		testutil.Ok(t, err)
		testutil.Ok(t, s.StartAndWaitReady(a))
		defer a.Stop()

		u := url.URL{
			Scheme:   "http",
			Host:     a.HTTPEndpoint(),
			Path:     "api/v1/query_range",
			RawQuery: fmt.Sprintf("query=heap&from=%d&to=%d", math.MinInt64, math.MaxInt64),
		}
		resp, err := http.Get(u.String())
		testutil.Ok(t, err)
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		testutil.Ok(t, err)

		testutil.Equals(t, http.StatusOK, resp.StatusCode, string(body))

		res := struct {
			Status string       `json:"status"`
			Data   []api.Series `json:"data"`
		}{}

		err = json.Unmarshal(body, &res)
		testutil.Ok(t, err)

		testutil.Equals(t, 1, len(res.Data), "Unexpected amount of series")
		testutil.Equals(t, 100, len(res.Data[0].Timestamps), "Unexpected amount of samples")
	})
}

type sample struct {
	t int64
	v []byte
}

func (s sample) T() int64  { return s.t }
func (s sample) V() []byte { return s.v }

// query runs a matcher query against the querier and fully expands its data.
func query(t testing.TB, q storage.Querier, matchers ...*labels.Matcher) map[string][]tsdbutil.Sample {
	ss := q.Select(false, nil, matchers...)
	defer func() {
		testutil.Ok(t, q.Close())
	}()

	result := map[string][]tsdbutil.Sample{}
	for ss.Next() {
		series := ss.At()

		samples := []tsdbutil.Sample{}
		it := series.Iterator()
		for it.Next() {
			t, v := it.At()
			samples = append(samples, sample{t: t, v: v})
		}
		testutil.Ok(t, it.Err())

		if len(samples) == 0 {
			continue
		}

		name := series.Labels().String()
		result[name] = samples
	}
	testutil.Ok(t, ss.Err())
	testutil.Equals(t, 0, len(ss.Warnings()))

	return result
}
