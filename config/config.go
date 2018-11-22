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
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	yaml "gopkg.in/yaml.v2"
)

var (
	trueValue           = true
	DefaultScrapeConfig = ScrapeConfig{
		ScrapeInterval: model.Duration(time.Minute),
		ScrapeTimeout:  model.Duration(time.Minute),
		Scheme:         "http",
		ProfilingConfig: &ProfilingConfig{
			PprofConfig: &PprofConfig{
				Allocs: &PprofAllocsConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/allocs",
					},
				},
				Block: &PprofBlockConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/block",
					},
				},
				Cmdline: &PprofCmdlineConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/cmdline",
					},
				},
				Goroutine: &PprofGoroutineConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/goroutine",
					},
				},
				Heap: &PprofHeapConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/heap",
					},
				},
				Mutex: &PprofMutexConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/mutex",
					},
				},
				Profile: &PprofProfileConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/profile",
					},
				},
				Threadcreate: &PprofThreadcreateConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/threadcreate",
					},
				},
				Trace: &PprofTraceConfig{
					PprofProfilingConfig{
						Enabled: &trueValue,
						Path:    "/debug/pprof/trace",
					},
				},
			},
		},
	}
)

// Config is the top-level configuration for conprof's config files.
type Config struct {
	ScrapeConfigs []*ScrapeConfig `yaml:"scrape_configs,omitempty"`

	// original is the input from which the config was parsed.
	original string
}

// Load parses the YAML input s into a Config.
func Load(s string) (*Config, error) {
	cfg := Config{}
	err := yaml.UnmarshalStrict([]byte(s), &cfg)
	if err != nil {
		return nil, err
	}
	cfg.original = s
	return &cfg, nil
}

// LoadFile parses the given YAML file into a Config.
func LoadFile(filename string) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(string(content))
	if err != nil {
		return nil, fmt.Errorf("parsing YAML file %s: %v", filename, err)
	}
	return cfg, nil
}

// ScrapeConfig configures a scraping unit for conprof.
type ScrapeConfig struct {
	// Name of the section in the config
	JobName string `yaml:"job_name,omitempty"`
	// A set of query parameters with which the target is scraped.
	Params url.Values `yaml:"params,omitempty"`
	// How frequently to scrape the targets of this scrape config.
	ScrapeInterval model.Duration `yaml:"scrape_interval,omitempty"`
	// The timeout for scraping targets of this config.
	ScrapeTimeout model.Duration `yaml:"scrape_timeout,omitempty"`
	// The URL scheme with which to fetch metrics from targets.
	Scheme string `yaml:"scheme,omitempty"`

	ProfilingConfig *ProfilingConfig `yaml:"profiling_config,omitempty"`

	// We cannot do proper Go type embedding below as the parser will then parse
	// values arbitrarily into the overflow maps of further-down types.
	ServiceDiscoveryConfig ServiceDiscoveryConfig `yaml:",inline"`
	HTTPClientConfig       HTTPClientConfig       `yaml:",inline"`
}

// ServiceDiscoveryConfig configures lists of different service discovery mechanisms.
type ServiceDiscoveryConfig struct {
	// List of labeled target groups for this job.
	StaticConfigs []*targetgroup.Group `yaml:"static_configs,omitempty"`
}

type ProfilingConfig struct {
	PprofConfig *PprofConfig `yaml:"pprof_config,omitempty"`
}

type PprofConfig struct {
	Allocs       *PprofAllocsConfig       `yaml:"allocs,omitempty"`
	Block        *PprofBlockConfig        `yaml:"block,omitempty"`
	Cmdline      *PprofCmdlineConfig      `yaml:"cmdline,omitempty"`
	Goroutine    *PprofGoroutineConfig    `yaml:"goroutine,omitempty"`
	Heap         *PprofHeapConfig         `yaml:"heap,omitempty"`
	Mutex        *PprofMutexConfig        `yaml:"mutex,omitempty"`
	Profile      *PprofProfileConfig      `yaml:"profile,omitempty"`
	Threadcreate *PprofThreadcreateConfig `yaml:"threadcreate,omitempty"`
	Trace        *PprofTraceConfig        `yaml:"trace,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *ScrapeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultScrapeConfig
	type plain ScrapeConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}

	return nil
}

type PprofAllocsConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofBlockConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofCmdlineConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofGoroutineConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofHeapConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofMutexConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofProfileConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofThreadcreateConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofTraceConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofProfilingConfig struct {
	Enabled *bool  `yaml:"enabled,omitempty"`
	Path    string `yaml:"path,omitempty"`
}

// CheckTargetAddress checks if target address is valid.
func CheckTargetAddress(address model.LabelValue) error {
	// For now check for a URL, we may want to expand this later.
	if strings.Contains(string(address), "/") {
		return fmt.Errorf("%q is not a valid hostname", address)
	}
	return nil
}
