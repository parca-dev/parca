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
	"context"
	"fmt"
	"net/http"

	"github.com/conprof/conprof/config"
	"github.com/conprof/conprof/scrape"
	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	_ "github.com/prometheus/prometheus/discovery/install" // Register service discovery implementations.
)

// registerSampler registers a sampler command.
func registerSampler(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}) {
	cmd := app.Command(name, "Run a sampler, that appends profiles to a configured storage.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := modelDuration(cmd.Flag("storage.tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		db, err := tsdb.Open(
			*storagePath,
			logger,
			prometheus.DefaultRegisterer,
			&tsdb.Options{
				RetentionDuration:      int64(*retention),
				WALSegmentSize:         wal.DefaultSegmentSize,
				MinBlockDuration:       tsdb.DefaultBlockDuration,
				MaxBlockDuration:       tsdb.DefaultBlockDuration,
				NoLockfile:             true,
				AllowOverlappingBlocks: false,
				WALCompression:         true,
				StripeSize:             tsdb.DefaultStripeSize,
			},
		)
		if err != nil {
			return err
		}
		return runSampler(g, logger, db, *configFile, reloadCh)
	}
}

func getScrapeConfigs(cfg *config.Config) map[string]discovery.Configs {
	c := make(map[string]discovery.Configs)
	for _, v := range cfg.ScrapeConfigs {
		c[v.JobName] = v.ServiceDiscoveryConfigs
	}
	return c
}

func managerReloader(logger log.Logger, reloadCh chan struct{}, d *discovery.Manager, s *scrape.Manager, configFile string) {
	for {
		<-reloadCh
		level.Info(logger).Log("msg", "Reloading configuration")
		cfg, err := config.LoadFile(configFile)
		if err != nil {
			level.Error(logger).Log("could not load config to reload: %v", err)
		}

		err = d.ApplyConfig(getScrapeConfigs(cfg))
		if err != nil {
			level.Error(logger).Log("could not reload scrape configs: %v", err)
		}

		err = s.ApplyConfig(cfg)
		if err != nil {
			level.Error(logger).Log("could not reload config: %v", err)
		}
	}
}

func runSampler(g *run.Group, logger log.Logger, db *tsdb.DB, configFile string, reloadCh chan struct{}) error {
	scrapeManager := scrape.NewManager(log.With(logger, "component", "scrape-manager"), db)
	cfg, err := config.LoadFile(configFile)
	if err != nil {
		return fmt.Errorf("could not load config: %v", err)
	}

	discoveryManagerScrape := discovery.NewManager(context.Background(), log.With(logger, "component", "discovery manager scrape"), discovery.Name("scrape"))

	ctxScrape, cancelScrape := context.WithCancel(context.Background())
	discoveryManagerScrape = discovery.NewManager(ctxScrape, log.With(logger, "component", "discovery manager scrape"), discovery.Name("scrape"))

	go managerReloader(logger, reloadCh, discoveryManagerScrape, scrapeManager, configFile)
	{
		err := discoveryManagerScrape.ApplyConfig(getScrapeConfigs(cfg))
		if err != nil {
			level.Error(logger).Log("msg", err)
			cancelScrape()
			return err
		}
		// Scrape discovery manager.
		g.Add(
			func() error {
				err := discoveryManagerScrape.Run()
				level.Info(logger).Log("msg", "Scrape discovery manager stopped")
				return err
			},
			func(err error) {
				level.Info(logger).Log("msg", "Stopping scrape discovery manager...")
				cancelScrape()
			},
		)
	}
	{
		_, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			err = scrapeManager.ApplyConfig(cfg)
			if err != nil {
				return fmt.Errorf("could not apply config: %v", err)
			}
			return scrapeManager.Run(discoveryManagerScrape.SyncCh())
		}, func(error) {
			level.Debug(logger).Log("msg", "shutting down scrape manager")
			scrapeManager.Stop()
			cancel()
		})
	}
	return nil
}
