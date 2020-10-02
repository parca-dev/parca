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
	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/prober"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := extkingpin.ModelDuration(cmd.Flag("storage.tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) (prober.Probe, error) {
		return runAll(comp, g, mux, probe, reg, logger, *storagePath, *configFile, *retention, reloadCh)
	}
}

func runAll(comp component.Component, g *run.Group, mux httpMux, p prober.Probe, reg prometheus.Registerer, logger log.Logger, storagePath, configFile string, retention model.Duration, reloadCh chan struct{}) (prober.Probe, error) {
	db, err := tsdb.Open(
		storagePath,
		logger,
		prometheus.DefaultRegisterer,
		&tsdb.Options{
			RetentionDuration:      int64(retention),
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
		return nil, err
	}

	err = runSampler(g, p, logger, db, configFile, reloadCh)
	if err != nil {
		return nil, err
	}

	err = runWeb(mux, p, reg, logger, db, reloadCh)
	if err != nil {
		return nil, err
	}

	return p, nil
}
