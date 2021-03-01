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
	"time"

	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/prober"
	"gopkg.in/alecthomas/kingpin.v2"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/scrape"
)

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}, reloaders *configReloaders) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := extkingpin.ModelDuration(cmd.Flag("storage.tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))
	maxMergeBatchSize := cmd.Flag("max-merge-batch-size", "Bytes loaded in one batch for merging. This is to limit the amount of memory a merge query can use.").
		Default("64MB").Bytes()

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, debugLogging bool) (prober.Probe, error) {
		return runAll(
			comp,
			g,
			mux,
			probe,
			reg,
			logger,
			*storagePath,
			*configFile,
			time.Duration(*retention),
			reloadCh,
			reloaders,
			int64(*maxMergeBatchSize),
		)
	}
}

func runAll(
	comp component.Component,
	g *run.Group,
	mux httpMux,
	p prober.Probe,
	reg *prometheus.Registry,
	logger log.Logger,
	storagePath,
	configFile string,
	retention time.Duration,
	reloadCh chan struct{},
	reloaders *configReloaders,
	maxMergeBatchSize int64,
) (prober.Probe, error) {
	db, err := tsdb.Open(
		storagePath,
		logger,
		prometheus.DefaultRegisterer,
		&tsdb.Options{
			RetentionDuration:      retention.Milliseconds(),
			WALSegmentSize:         wal.DefaultSegmentSize,
			MinBlockDuration:       tsdb.DefaultBlockDuration,
			MaxBlockDuration:       retention.Milliseconds() / 10,
			NoLockfile:             true,
			AllowOverlappingBlocks: false,
			WALCompression:         true,
			StripeSize:             tsdb.DefaultStripeSize,
		},
	)
	if err != nil {
		return nil, err
	}

	scrapeManager := scrape.NewManager(log.With(logger, "component", "scrape-manager"), db)

	s, err := NewSampler(db, reloaders,
		SamplerScraper(scrapeManager),
		SamplerConfig(configFile),
	)
	if err != nil {
		return nil, err
	}
	if err := s.Run(context.TODO(), g, reloadCh); err != nil {
		return nil, err
	}

	w := NewWeb(mux, db, maxMergeBatchSize,
		WebLogger(logger),
		WebRegistry(reg),
		WebReloaders(reloaders),
		WebTargets(func(ctx context.Context) conprofapi.TargetRetriever {
			return scrapeManager
		}),
	)
	if err = w.Run(context.TODO(), reloadCh); err != nil {
		return nil, err
	}

	p.Ready()

	return p, nil
}
