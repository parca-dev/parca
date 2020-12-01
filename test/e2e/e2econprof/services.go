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

package e2econprof

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/cortexproject/cortex/integration/e2e"
	"github.com/cortexproject/cortex/pkg/util"
	"github.com/pkg/errors"
)

const logLevel = "info"

// Same as default for now.
var defaultBackoffConfig = util.BackoffConfig{
	MinBackoff: 300 * time.Millisecond,
	MaxBackoff: 600 * time.Millisecond,
	MaxRetries: 50,
}

// DefaultImage returns the local docker image to use to run Thanos.
func DefaultImage() string {
	// Get the Thanos image from the THANOS_IMAGE env variable.
	if os.Getenv("CONPROF_IMAGE") != "" {
		return os.Getenv("CONPROF_IMAGE")
	}

	return "conprof"
}

func NewStorage(sharedDir string, networkName string, name string, dirSuffix string) (*Service, error) {
	dir := filepath.Join(sharedDir, "data", "storage", dirSuffix)
	dataDir := filepath.Join(dir, "data")
	container := filepath.Join(e2e.ContainerSharedDir, "data", "storage", dirSuffix)
	if err := os.MkdirAll(dataDir, 0777); err != nil {
		return nil, errors.Wrap(err, "create storage dir")
	}

	storage := NewService(
		fmt.Sprintf("storage-%v", name),
		DefaultImage(),
		e2e.NewCommand("storage", e2e.BuildArgs(map[string]string{
			"--debug.name":        fmt.Sprintf("storage-%v", name),
			"--grpc-address":      ":9091",
			"--grpc-grace-period": "0s",
			"--http-address":      ":8080",
			"--storage.tsdb.path": filepath.Join(container, "data"),
			"--log.level":         logLevel,
		})...),
		e2e.NewHTTPReadinessProbe(8080, "/-/ready", 200, 200),
		8080,
		9091,
	)
	storage.SetUser(strconv.Itoa(os.Getuid()))
	storage.SetBackoff(defaultBackoffConfig)

	return storage, nil
}

func NewAPI(networkName string, name string, storeAddress string) (*e2e.HTTPService, error) {
	api := e2e.NewHTTPService(
		fmt.Sprintf("api-%v", name),
		DefaultImage(),
		e2e.NewCommand("api", e2e.BuildArgs(map[string]string{
			"--debug.name":   fmt.Sprintf("api-%v", name),
			"--http-address": ":8080",
			"--log.level":    logLevel,
			"--store":        storeAddress,
		})...),
		e2e.NewHTTPReadinessProbe(8080, "/-/ready", 200, 200),
		8080,
	)
	api.SetUser(strconv.Itoa(os.Getuid()))
	api.SetBackoff(defaultBackoffConfig)

	return api, nil
}

func NewAll(sharedDir, networkName, name, dirSuffix, config string) (*e2e.HTTPService, error) {
	dir := filepath.Join(sharedDir, "data", "all", dirSuffix)
	dataDir := filepath.Join(dir, "data")
	container := filepath.Join(e2e.ContainerSharedDir, "data", "all", dirSuffix)
	if err := os.MkdirAll(dataDir, 0777); err != nil {
		return nil, errors.Wrap(err, "create storage dir")
	}

	if err := ioutil.WriteFile(filepath.Join(dir, "conprof.yaml"), []byte(config), 0666); err != nil {
		return nil, errors.Wrap(err, "creating conprof config failed")
	}

	all := e2e.NewHTTPService(
		fmt.Sprintf("all-%v", name),
		DefaultImage(),
		e2e.NewCommand("all", e2e.BuildArgs(map[string]string{
			"--debug.name":        fmt.Sprintf("storage-%v", name),
			"--http-address":      ":8080",
			"--storage.tsdb.path": filepath.Join(container, "data"),
			"--log.level":         logLevel,
			"--config.file":       filepath.Join(container, "conprof.yaml"),
		})...),
		e2e.NewHTTPReadinessProbe(8080, "/-/ready", 200, 200),
		8080,
	)
	all.SetUser(strconv.Itoa(os.Getuid()))
	all.SetBackoff(defaultBackoffConfig)

	return all, nil
}

func DefaultScrapeConfig() string {
	config := fmt.Sprintf(`
scrape_configs:
- job_name: 'conprof'
  # Quick scrapes for test purposes.
  scrape_interval: 1s
  scrape_timeout: 1s
  profiling_config:
    pprof_config:
      allocs:
        enabled: false
      block:
        enabled: false
      goroutine:
        enabled: false
      heap:
        enabled: true
        path: /debug/pprof/heap
      mutex:
        enabled: false
      profile:
        enabled: false
      threadcreate:
        enabled: false
      trace:
        enabled: false
  static_configs:
  - targets: ['localhost:8080']
`)

	return config
}
