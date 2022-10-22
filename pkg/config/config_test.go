// Copyright 2022 The Parca Authors
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
	"github.com/thanos-io/objstore/client"
)

func TestLoad(t *testing.T) {
	t.Parallel()

	_, err := Load(`scrape_configs:
- job_name: 'test'
  static_configs:
  - targets: ['localhost:8080']`)
	if err != nil {
		t.Fatal(err)
	}
}

func TestLoadComplex(t *testing.T) {
	t.Parallel()

	// TODO: Make even more complex if necessary
	complexYAML := `
scrape_configs:
  - job_name: 'parca'
    scrape_interval: 10s
    static_configs:
      - targets: [ 'localhost:10902' ]
    profiling_config:
      pprof_config:
        memory:
          enabled: true
          path: /parca/debug/pprof/allocs
        fgprof:
          enabled: true
          path: /debug/fgprof
  - job_name: 'empty-profiling-config'
    profiling_config: {}
  - job_name: 'path-prefix'
    profiling_config:
      path_prefix: /test/prefix
      pprof_config:
        memory:
          enabled: true
          path: /parca/debug/pprof/allocs
        fgprof:
          enabled: true
          path: /debug/fgprof
  - job_name: 'path-prefix-with-defaults'
    profiling_config:
      path_prefix: /test/prefix
`

	expected := &Config{
		ScrapeConfigs: []*ScrapeConfig{
			{
				JobName:        "parca",
				ScrapeInterval: model.Duration(10 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				Scheme:         "http",
				ProfilingConfig: &ProfilingConfig{
					PprofConfig: PprofConfig{
						"memory": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/parca/debug/pprof/allocs",
						},
						"block": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/debug/pprof/block",
						},
						"goroutine": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/debug/pprof/goroutine",
						},
						"mutex": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/debug/pprof/mutex",
						},
						"process_cpu": &PprofProfilingConfig{
							Enabled: trueValue(),
							Delta:   true,
							Path:    "/debug/pprof/profile",
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
			},
			{
				JobName:         "empty-profiling-config",
				ScrapeInterval:  model.Duration(10 * time.Second),
				ScrapeTimeout:   model.Duration(10 * time.Second),
				Scheme:          "http",
				ProfilingConfig: DefaultScrapeConfig().ProfilingConfig,
			},
			{
				JobName:        "path-prefix",
				ScrapeInterval: model.Duration(10 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				Scheme:         "http",
				ProfilingConfig: &ProfilingConfig{
					PprofPrefix: "/test/prefix",
					PprofConfig: PprofConfig{
						"memory": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/parca/debug/pprof/allocs",
						},
						"block": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/block",
						},
						"goroutine": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/goroutine",
						},
						"mutex": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/mutex",
						},
						"process_cpu": &PprofProfilingConfig{
							Enabled: trueValue(),
							Delta:   true,
							Path:    "/test/prefix/debug/pprof/profile",
						},
						"fgprof": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/fgprof",
						},
					},
				},
			},
			{
				JobName:        "path-prefix-with-defaults",
				ScrapeInterval: model.Duration(10 * time.Second),
				ScrapeTimeout:  model.Duration(10 * time.Second),
				Scheme:         "http",
				ProfilingConfig: &ProfilingConfig{
					PprofPrefix: "/test/prefix",
					PprofConfig: PprofConfig{
						"memory": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/allocs",
						},
						"block": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/block",
						},
						"goroutine": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/goroutine",
						},
						"mutex": &PprofProfilingConfig{
							Enabled: trueValue(),
							Path:    "/test/prefix/debug/pprof/mutex",
						},
						"process_cpu": &PprofProfilingConfig{
							Enabled: trueValue(),
							Delta:   true,
							Path:    "/test/prefix/debug/pprof/profile",
						},
					},
				},
			},
		},
	}
	c, err := Load(complexYAML)
	require.NoError(t, err)
	require.Len(t, c.ScrapeConfigs, 4)
	require.Equal(t, expected, c)
}

func Test_Config_Validation(t *testing.T) {
	t.Parallel()

	tests := map[string]Config{
		"nilObjectStorage": {
			ObjectStorage: nil,
		},
		"nilBucket": {
			ObjectStorage: &ObjectStorage{
				Bucket: nil,
			},
		},
		"emptyType": {
			ObjectStorage: &ObjectStorage{
				Bucket: &client.BucketConfig{
					Config: struct {
						Directory string
					}{
						Directory: "./tmp",
					},
				},
			},
		},
		"emptyConfig": {
			ObjectStorage: &ObjectStorage{
				Bucket: &client.BucketConfig{
					Type: client.FILESYSTEM,
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			require.Error(t, test.Validate())
		})
	}
}
