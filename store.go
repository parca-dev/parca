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

package main

import (
	"net/http"
	"time"

	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/prober"
	grpcserver "github.com/thanos-io/thanos/pkg/server/grpc"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

type component string

func (c component) String() string {
	return string(c)
}

// registerStorage registers a sampler command.
func registerStorage(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}) {
	cmd := app.Command(name, "Run a sampler, that appends profiles to a configured storage.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	grpcBindAddr := ":10000"
	grpcGracePeriod := time.Second * 30
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
		return runStorage(g, reg, logger, db, tracer, grpcBindAddr, grpcGracePeriod)
	}
}

func runStorage(g *run.Group, reg *prometheus.Registry, logger log.Logger, db *tsdb.DB, tracer opentracing.Tracer, grpcBindAddr string, grpcGracePeriod time.Duration) error {
	grpcProbe := prober.NewGRPC()
	maxBytesPerFrame := 1024 * 1024 * 32 // 32 Mb default, might need to be tuned later on.
	s := store.NewProfileStore(logger, db, maxBytesPerFrame)

	srv := grpcserver.New(logger, reg, tracer, component("storage"), grpcProbe,
		grpcserver.WithServer(store.RegisterStoreServer(s)),
		grpcserver.WithListen(grpcBindAddr),
		grpcserver.WithGracePeriod(grpcGracePeriod),
	)

	g.Add(func() error {
		grpcProbe.Ready()
		return srv.ListenAndServe()
	}, func(err error) {
		grpcProbe.NotReady(err)
		srv.Shutdown(err)
	})

	return nil
}
