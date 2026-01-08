// Copyright 2024-2026 The Parca Authors
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

package scrape

import (
	"sort"
	"testing"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/config"
)

func TestTargetsFromGroup(t *testing.T) {
	testCases := []struct {
		name     string
		tg       *targetgroup.Group
		cfg      config.ScrapeConfig
		targets  Targets
		lb       *labels.Builder
		expected Targets
		err      error
	}{
		{
			name: "default-scrape-config",
			tg: &targetgroup.Group{
				Targets: []model.LabelSet{
					{"__address__": "localhost:9090"},
				},
				Labels: model.LabelSet{},
			},
			cfg: config.DefaultScrapeConfig(),
			lb:  labels.NewBuilder(labels.EmptyLabels()),
			expected: Targets{
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/allocs",
					),
				},
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/mutex",
					),
				},
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/block",
					),
				},
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/goroutine",
					),
				},
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/profile",
						model.ParamLabelPrefix+"seconds", "10",
					),
				},
			},
			err: nil,
		},
		{
			name: "custom-scrape-config-with-pprof-seconds",
			tg: &targetgroup.Group{
				Targets: []model.LabelSet{
					{"__address__": "localhost:9090"},
				},
				Labels: model.LabelSet{},
			},
			cfg: config.ScrapeConfig{
				ScrapeInterval: model.Duration(time.Minute * 10),
				Scheme:         "http",
				ProfilingConfig: &config.ProfilingConfig{
					PprofConfig: config.PprofConfig{
						"pprofMemory": &config.PprofProfilingConfig{
							Enabled: trueValue(),
							Delta:   true,
							Path:    "/debug/pprof/allocs",
							Seconds: 10,
						},
						"pprofProcessCPU": &config.PprofProfilingConfig{
							Enabled: trueValue(),
							Delta:   true,
							Path:    "/debug/pprof/profile",
							Seconds: 30,
						},
					},
				},
			},
			lb: labels.NewBuilder(labels.EmptyLabels()),
			expected: Targets{
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/allocs",
						model.ParamLabelPrefix+"seconds", "10",
					),
				},
				{
					labels: labels.FromStrings(
						model.AddressLabel, "localhost:9090",
						model.SchemeLabel, "http",
						ProfilePath, "/debug/pprof/profile",
						model.ParamLabelPrefix+"seconds", "30",
					),
				},
			},
			err: nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			a, err := targetsFromGroup(tc.tg, &tc.cfg, tc.targets, tc.lb)
			actual := Targets(a) // convert to slice type for convenience
			if tc.err != nil && (err == nil || err.Error() != tc.err.Error()) {
				t.Fatalf("unexpected error: %v", err)
				return
			}
			if tc.err != nil && err != nil && err.Error() == tc.err.Error() {
				return
			}
			require.Equal(t, len(tc.expected), len(actual), "unexpected number of targets")
			sort.Sort(tc.expected)
			sort.Sort(actual)
			for i := range actual {
				require.Equal(t, tc.expected[i].URL(), actual[i].URL())
			}
		})
	}
}

func trueValue() *bool {
	a := true
	return &a
}
