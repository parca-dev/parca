// Copyright 2018 The conprof Authors
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

package config

import (
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/stretchr/testify/require"
)

func TestLoad(t *testing.T) {
	_, err := Load(`scrape_configs:
- job_name: 'test'
  static_configs:
  - targets: ['localhost:8080']`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadComplex(t *testing.T) {
	// TODO: Make even more complex if necessary
	complexYAML := `
scrape_configs:
  - job_name: 'conprof'
    scrape_interval: 10s
    static_configs:
      - targets: [ 'localhost:10902' ]
    profiling_config:
      pprof_config:
        allocs:
          enabled: true
          path: /conprof/debug/pprof/allocs
        fgprof:
          enabled: true
          path: /debug/fgprof
`

	expected := &Config{
		ScrapeConfigs: []*ScrapeConfig{{
			JobName:        "conprof",
			ScrapeInterval: model.Duration(10 * time.Second),
			ScrapeTimeout:  model.Duration(time.Minute),
			Scheme:         "http",
			ProfilingConfig: &ProfilingConfig{
				PprofConfig: PprofConfig{
					"allocs": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/conprof/debug/pprof/allocs",
					},
					"block": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/block",
					},
					"goroutine": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/goroutine",
					},
					"heap": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/heap",
					},
					"mutex": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/mutex",
					},
					"profile": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/profile",
						Seconds: 30, // By default Go collects 30s profile.
					},
					"threadcreate": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/threadcreate",
					},
					"fgprof": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/fgprof",
					},
				},
			},
			ServiceDiscoveryConfigs: discovery.Configs{discovery.StaticConfig{{
				Targets: []model.LabelSet{{"__address__": "localhost:10902"}},
				Labels:  nil,
				Source:  "0",
			}}},
		}},
	}

	c, err := Load(complexYAML)
	require.NoError(t, err)
	require.Len(t, c.ScrapeConfigs, 1)
	require.Equal(t, expected, c)
}
