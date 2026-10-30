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

func TestOTLPExampleConfigs(t *testing.T) {
	for _, tc := range []struct {
		name     string
		target   model.LabelSet
		expected map[string]string
	}{
		{
			name: "static",
			target: model.LabelSet{
				"__address__": "checkout:6060", "namespace": "shop", "environment": "development",
			},
			expected: map[string]string{
				"service.name": "checkout", "service.namespace": "shop",
				"service.instance.id": "checkout:6060", "deployment.environment.name": "development",
			},
		},
		{
			name: "kubernetes",
			target: model.LabelSet{
				"__address__": "10.0.0.1:6060", "__meta_kubernetes_pod_phase": "Running",
				"__meta_kubernetes_pod_container_port_name":          "pprof",
				"__meta_kubernetes_pod_label_app_kubernetes_io_name": "checkout",
				"__meta_kubernetes_namespace":                        "shop", "__meta_kubernetes_pod_name": "checkout-abc",
				"__meta_kubernetes_pod_uid": "pod-uid", "__meta_kubernetes_pod_node_name": "worker-3",
				"__meta_kubernetes_pod_container_name": "app",
			},
			expected: map[string]string{
				"service.name": "checkout", "service.namespace": "shop", "service.instance.id": "pod-uid/app",
				"k8s.namespace.name": "shop", "k8s.pod.name": "checkout-abc", "k8s.pod.uid": "pod-uid",
				"k8s.node.name": "worker-3", "k8s.container.name": "app",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.LoadFile("../../deploy/otlp/" + tc.name + ".yaml")
			require.NoError(t, err)
			require.NoError(t, cfg.Validate())
			targets, err := targetsFromGroup(&targetgroup.Group{Targets: []model.LabelSet{tc.target}},
				cfg.ScrapeConfigs[0], nil, labels.NewBuilder(labels.EmptyLabels()))
			require.NoError(t, err)
			require.NotEmpty(t, targets)
			for _, target := range targets {
				for key, value := range tc.expected {
					require.Equal(t, value, target.Labels().Get(key), key)
				}
				require.False(t, target.Labels().Has("namespace"))
				require.False(t, target.Labels().Has("environment"))
			}
		})
	}
}
