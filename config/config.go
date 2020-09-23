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
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	commonconfig "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"github.com/prometheus/prometheus/pkg/relabel"
	"gopkg.in/yaml.v2"
)

func trueValue() *bool {
	a := true
	return &a
}

func DefaultScrapeConfig() ScrapeConfig {
	return ScrapeConfig{
		ScrapeInterval: model.Duration(time.Minute),
		ScrapeTimeout:  model.Duration(time.Minute),
		Scheme:         "http",
		ProfilingConfig: &ProfilingConfig{
			PprofConfig: &PprofConfig{
				Allocs: &PprofAllocsConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/allocs",
					},
				},
				Block: &PprofBlockConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/block",
					},
				},
				Goroutine: &PprofGoroutineConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/goroutine",
					},
				},
				Heap: &PprofHeapConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/heap",
					},
				},
				Mutex: &PprofMutexConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/mutex",
					},
				},
				Profile: &PprofProfileConfig{
					PprofProfilingConfig: PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/profile",
					},
					Seconds: 30, // By default Go collects 30s profile.
				},
				Threadcreate: &PprofThreadcreateConfig{
					PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/threadcreate",
					},
				},
				Trace: &PprofTraceConfig{
					PprofProfilingConfig: PprofProfilingConfig{
						Enabled: trueValue(),
						Path:    "/debug/pprof/trace",
					},
					Seconds: 1, // By default Go collects 1s trace.
				},
			},
		},
	}
}

// Config is the top-level configuration for conprof's config files.
type Config struct {
	ScrapeConfigs []*ScrapeConfig `yaml:"scrape_configs,omitempty"`
}

// SetDirectory joins any relative file paths with dir.
func (c *Config) SetDirectory(dir string) {
	for _, c := range c.ScrapeConfigs {
		c.SetDirectory(dir)
	}
}

// Load parses the YAML input s into a Config.
func Load(s string) (*Config, error) {
	cfg := &Config{}

	err := yaml.UnmarshalStrict([]byte(s), cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
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
	cfg.SetDirectory(filepath.Dir(filename))
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

	RelabelConfigs []*relabel.Config `yaml:"relabel_configs,omitempty"`
	// We cannot do proper Go type embedding below as the parser will then parse
	// values arbitrarily into the overflow maps of further-down types.
	ServiceDiscoveryConfigs discovery.Configs             `yaml:"-"`
	HTTPClientConfig        commonconfig.HTTPClientConfig `yaml:",inline"`
}

// SetDirectory joins any relative file paths with dir.
func (c *ScrapeConfig) SetDirectory(dir string) {
	c.ServiceDiscoveryConfigs.SetDirectory(dir)
	c.HTTPClientConfig.SetDirectory(dir)
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
	Goroutine    *PprofGoroutineConfig    `yaml:"goroutine,omitempty"`
	Heap         *PprofHeapConfig         `yaml:"heap,omitempty"`
	Mutex        *PprofMutexConfig        `yaml:"mutex,omitempty"`
	Profile      *PprofProfileConfig      `yaml:"profile,omitempty"`
	Threadcreate *PprofThreadcreateConfig `yaml:"threadcreate,omitempty"`
	Trace        *PprofTraceConfig        `yaml:"trace,omitempty"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *ScrapeConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultScrapeConfig()
	if err := discovery.UnmarshalYAMLWithInlineConfigs(c, unmarshal); err != nil {
		return err
	}

	if len(c.JobName) == 0 {
		return errors.New("job_name is empty")
	}

	// The UnmarshalYAML method of HTTPClientConfig is not being called because it's not a pointer.
	// We cannot make it a pointer as the parser panics for inlined pointer structs.
	// Thus we just do its validation here.
	if err := c.HTTPClientConfig.Validate(); err != nil {
		return err
	}

	// Check for users putting URLs in target groups.
	if len(c.RelabelConfigs) == 0 {
		if err := checkStaticTargets(c.ServiceDiscoveryConfigs); err != nil {
			return err
		}
	}

	for _, rlcfg := range c.RelabelConfigs {
		if rlcfg == nil {
			return errors.New("empty or null target relabeling rule in scrape config")
		}
	}

	return nil
}

func checkStaticTargets(configs discovery.Configs) error {
	for _, cfg := range configs {
		sc, ok := cfg.(discovery.StaticConfig)
		if !ok {
			continue
		}
		for _, tg := range sc {
			for _, t := range tg.Targets {
				if err := CheckTargetAddress(t[model.AddressLabel]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

type PprofAllocsConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofBlockConfig struct {
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
	Seconds              int `yaml:"seconds"`
}

type PprofThreadcreateConfig struct {
	PprofProfilingConfig `yaml:",inline"`
}

type PprofTraceConfig struct {
	PprofProfilingConfig `yaml:",inline"`
	Seconds              int `yaml:"seconds"`
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
