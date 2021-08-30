// Copyright 2021 The Parca Authors
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

package parca

import (
	"context"
	"errors"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/config"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/scrape"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/parca-dev/parca/pkg/storage"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/parca-dev/parca/pkg/symbol"
)

type Flags struct {
	ConfigPath         string   `kong:"help='Path to config file.',default='parca.yaml'"`
	LogLevel           string   `kong:"enum='error,warn,info,debug',help='Log level.',default='info'"`
	Port               string   `kong:"help='Port string for server',default=':7070'"`
	CORSAllowedOrigins []string `kong:"help='Allowed CORS origins.'"`
}

// Run the parca server
func Run(ctx context.Context, logger log.Logger, reg *prometheus.Registry, flags *Flags) error {
	cfgContent, err := ioutil.ReadFile(flags.ConfigPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to read config", "path", flags.ConfigPath)
		return err
	}

	cfg := config.Config{}
	if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
		level.Error(logger).Log("msg", "failed to parse config", "err", err, "path", flags.ConfigPath)
		return err
	}

	dbgInfo, err := debuginfo.NewStore(logger, cfg.DebugInfo)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize debug info store", "err", err)
		return err
	}

	mStr, err := metastore.NewInMemoryProfileMetaStore()
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize metadata store", "err", err)
		return err
	}
	defer mStr.Close()

	db := storage.OpenDB(reg)
	s := profilestore.NewProfileStore(logger, db, mStr)
	q := query.New(logger, db, mStr)

	parcaserver := server.NewServer(reg)

	var gr run.Group
	gr.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM))
	gr.Add(
		func() error {
			return parcaserver.ListenAndServe(
				ctx,
				logger,
				flags.Port,
				flags.CORSAllowedOrigins,
				server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
					debuginfopb.RegisterDebugInfoServiceServer(srv, dbgInfo)
					profilestorepb.RegisterProfileStoreServiceServer(srv, s)
					querypb.RegisterQueryServiceServer(srv, q)

					if err := debuginfopb.RegisterDebugInfoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					if err := profilestorepb.RegisterProfileStoreServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					if err := querypb.RegisterQueryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					return nil
				}),
			)
		},
		func(_ error) {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // TODO make this a graceful shutdown config setting
			defer cancel()

			err := parcaserver.Shutdown(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				level.Error(logger).Log("msg", "error shuttiing down server", "err", err)
			}
		},
	)
	{
		sym := symbol.NewSymbolizer(logger, mStr, dbgInfo)
		ctx, cancel := context.WithCancel(ctx)
		gr.Add(
			func() error {
				return sym.Run(ctx, 10*time.Second)
			},
			func(_ error) {
				cancel()
			})
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	discoveryManager := discovery.NewManager(ctx, logger)
	if err := discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs)); err != nil {
		level.Error(logger).Log("msg", "failed to apply discovery configs", "err", err)
		return err
	}

	m := scrape.NewManager(logger, reg, s, cfg.ScrapeConfigs)
	if err := m.ApplyConfig(cfg.ScrapeConfigs); err != nil {
		level.Error(logger).Log("msg", "failed to apply scrape configs", "err", err)
		return err
	}

	gr.Add(
		func() error {
			return discoveryManager.Run()
		},
		func(_ error) {
			cancel()
		},
	)
	gr.Add(
		func() error {
			return m.Run(discoveryManager.SyncCh())
		},
		func(_ error) {
			m.Stop()
		},
	)

	level.Info(logger).Log("msg", "staring Parca...")
	defer level.Info(logger).Log("msg", "Parca stopped!")

	if err := gr.Run(); err != nil {
		if _, ok := err.(run.SignalError); ok {
			return nil
		}
		level.Error(logger).Log("msg", "run group failed", "error", err)
		return err
	}

	return nil
}

func getDiscoveryConfigs(cfgs []*config.ScrapeConfig) map[string]discovery.Configs {
	c := make(map[string]discovery.Configs)
	for _, v := range cfgs {
		c[v.JobName] = v.ServiceDiscoveryConfigs
	}
	return c
}
