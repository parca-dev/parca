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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/cortexproject/cortex/integration/e2e"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/tsdb/testutil"

	"github.com/conprof/conprof/api"
	"github.com/conprof/conprof/test/e2e/e2econprof"
)

func TestAll(t *testing.T) {
	t.Parallel()

	t.Run("append-restart-append-read", func(t *testing.T) {
		t.Parallel()

		start := time.Now()

		//ctx := context.Background()
		s, err := e2e.NewScenario("e2e_test_append_restart_append_read")
		testutil.Ok(t, err)
		t.Cleanup(e2econprof.CleanScenario(t, s))

		all1, err := e2econprof.NewAll(s.SharedDir(), s.NetworkName(), "1", "test", e2econprof.DefaultScrapeConfig())
		testutil.Ok(t, err)
		testutil.Ok(t, s.StartAndWaitReady(all1))

		// Let it scrape some samples.
		time.Sleep(10 * time.Second)

		res1 := queryRange(t, all1.HTTPEndpoint(), timestamp.FromTime(start), timestamp.FromTime(time.Now()))
		testutil.Equals(t, 1, len(res1.Data), "Unexpected amount of series")

		err = all1.Stop()
		testutil.Ok(t, err)

		all2, err := e2econprof.NewAll(s.SharedDir(), s.NetworkName(), "2", "test", e2econprof.DefaultScrapeConfig())
		testutil.Ok(t, err)
		testutil.Ok(t, s.StartAndWaitReady(all2))

		// Let it scrape some new samples after the restart.
		time.Sleep(10 * time.Second)

		res2 := queryRange(t, all2.HTTPEndpoint(), timestamp.FromTime(start), timestamp.FromTime(time.Now()))
		testutil.Equals(t, 1, len(res2.Data), "Unexpected amount of series: %#+v", res2.Data)
	})
}

type queryRangeResult struct {
	Status string       `json:"status"`
	Data   []api.Series `json:"data"`
}

func queryRange(t *testing.T, hostPort string, from int64, to int64) *queryRangeResult {
	u := url.URL{
		Scheme:   "http",
		Host:     hostPort,
		Path:     "api/v1/query_range",
		RawQuery: fmt.Sprintf("query=heap&from=%d&to=%d", from, to),
	}
	resp, err := http.Get(u.String())
	testutil.Ok(t, err)
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	testutil.Ok(t, err)

	testutil.Equals(t, http.StatusOK, resp.StatusCode, string(body))

	res := queryRangeResult{}
	err = json.Unmarshal(body, &res)
	testutil.Ok(t, err)

	return &res
}
