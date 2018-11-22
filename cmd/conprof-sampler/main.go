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


package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Go-SIP/conprof/config"
	"github.com/Go-SIP/conprof/scrape"
	"github.com/Go-SIP/conprof/storage"
	"github.com/Go-SIP/conprof/version"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	yaml "gopkg.in/yaml.v2"
)

const (
	logLevelAll   = "all"
	logLevelDebug = "debug"
	logLevelInfo  = "info"
	logLevelWarn  = "warn"
	logLevelError = "error"
	logLevelNone  = "none"
)

var (
	availableLogLevels = []string{
		logLevelAll,
		logLevelDebug,
		logLevelInfo,
		logLevelWarn,
		logLevelError,
		logLevelNone,
	}
)

func main() {
	os.Exit(Main())
}

func Main() int {
	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stdout))
	logger = log.With(logger, "ts", log.DefaultTimestampUTC)
	logger = log.With(logger, "caller", log.DefaultCaller)

	logger.Log("msg", fmt.Sprintf("Starting conprof version '%v'.", version.Version))

	logLevel := ""
	flagset := flag.NewFlagSet(os.Args[0], flag.ExitOnError)
	flagset.StringVar(&logLevel, "log.level", logLevelDebug, fmt.Sprintf("The log level to print. (Available are: %s)", strings.Join(availableLogLevels, ", ")))
	flagset.Parse(os.Args[1:])

	switch logLevel {
	case logLevelAll:
		logger = level.NewFilter(logger, level.AllowAll())
	case logLevelDebug:
		logger = level.NewFilter(logger, level.AllowDebug())
	case logLevelInfo:
		logger = level.NewFilter(logger, level.AllowInfo())
	case logLevelWarn:
		logger = level.NewFilter(logger, level.AllowWarn())
	case logLevelError:
		logger = level.NewFilter(logger, level.AllowError())
	case logLevelNone:
		logger = level.NewFilter(logger, level.AllowNone())
	default:
		fmt.Fprintf(os.Stderr, "log level %v unknown, %v are possible values", logLevel, availableLogLevels)
		return 1
	}

	storage := storage.NewDiskStorage(log.With(logger, "component", "storage"), "data/")
	scrapeManager := scrape.NewManager(log.With(logger, "component", "scrape-manager"), storage)
	c, err := config.Load(`scrape_configs:
- job_name: 'test'
  scrape_interval: 1m
  scrape_timeout: 1m
  static_configs:
  - targets: ['localhost:9090']`)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not load config: %v", err)
		return 1
	}

	cfgstring, err := yaml.Marshal(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not marshal: %v", err)
		return 1
	}
	fmt.Println(string(cfgstring))

	err = scrapeManager.ApplyConfig(c)
	if err != nil {
		fmt.Fprintf(os.Stderr, "could not apply config: %v", err)
		return 1
	}
	syncCh := make(chan map[string][]*targetgroup.Group)
	go scrapeManager.Run(syncCh)
	sc := c.ScrapeConfigs[0]
	syncCh <- map[string][]*targetgroup.Group{sc.JobName: sc.ServiceDiscoveryConfig.StaticConfigs}
	time.Sleep(10 * time.Minute)
	return 0
}
