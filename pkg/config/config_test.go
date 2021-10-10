// Copyright 2021 The Parca Authors
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

	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore/client"
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
        memory_total:
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
			ScrapeTimeout:  model.Duration(10 * time.Second),
			Scheme:         "http",
			ProfilingConfig: &ProfilingConfig{
				PprofConfig: PprofConfig{
					"memory_total": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/conprof/debug/pprof/allocs",
					},
					"block_total": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/block",
					},
					"goroutine_total": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/goroutine",
					},
					"mutex_total": &PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/mutex",
					},
					"process_cpu": &PprofProfilingConfig{
						Enabled: trueValue(),
						Delta:   true,
						Path:    "/debug/pprof/profile",
					},
					"threadcreate_total": &PprofProfilingConfig{
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

func Test_Config_Validation(t *testing.T) {

	tests := map[string]struct {
		cfg Config
	}{
		"nil debug": {
			Config{
				DebugInfo: nil,
			},
		},
		"nil bucket": {
			Config{
				DebugInfo: &debuginfo.Config{
					Bucket: nil,
				},
			},
		},
		"empty type": {
			Config{
				DebugInfo: &debuginfo.Config{
					Bucket: &client.BucketConfig{
						Config: struct {
							Directory string
						}{
							Directory: "./tmp",
						},
					},
				},
			},
		},
		"empty config": {
			Config{
				DebugInfo: &debuginfo.Config{
					Bucket: &client.BucketConfig{
						Type: client.FILESYSTEM,
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Error(t, test.cfg.Validate())
		})
	}
}
