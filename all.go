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
	"net/http"
	"time"

	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/wal"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extflag"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/extprom"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"github.com/thanos-io/thanos/pkg/prober"
	grpcserver "github.com/thanos-io/thanos/pkg/server/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/scrape"
	"github.com/conprof/conprof/symbol"
)

type grpcSettings struct {
	grpcBindAddr    string
	grpcGracePeriod time.Duration
	grpcCert        string
	grpcKey         string
	grpcClientCA    string
}

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}, reloaders *configReloaders) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	symbolServerURL := cmd.Flag("symbol-server-url", "Symbol server to request to symbolize native stacktraces.").String()
	retention := extkingpin.ModelDuration(cmd.Flag("storage.tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))
	maxMergeBatchSize := cmd.Flag("max-merge-batch-size", "Bytes loaded in one batch for merging. This is to limit the amount of memory a merge query can use.").
		Default("64MB").Bytes()
	grpcBindAddr, grpcGracePeriod, grpcCert, grpcKey, grpcClientCA := extkingpin.RegisterGRPCFlags(cmd)
	queryTimeout := extkingpin.ModelDuration(cmd.Flag("query.timeout", "Maximum time to process query by query node.").
		Default("10s"))
	objStoreConfig := *extkingpin.RegisterCommonObjStoreFlags(cmd, "", false)

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
			*queryTimeout,
			*symbolServerURL,
			objStoreConfig,
			&grpcSettings{
				grpcBindAddr:    *grpcBindAddr,
				grpcGracePeriod: time.Duration(*grpcGracePeriod),
				grpcCert:        *grpcCert,
				grpcKey:         *grpcKey,
				grpcClientCA:    *grpcClientCA,
			},
		)
	}
}

func runAll(
	comp component.Component,
	g *run.Group,
	mux httpMux,
	probe prober.Probe,
	reg *prometheus.Registry,
	logger log.Logger,
	storagePath,
	configFile string,
	retention time.Duration,
	reloadCh chan struct{},
	reloaders *configReloaders,
	maxMergeBatchSize int64,
	queryTimeout model.Duration,
	symbolServerURL string,
	objStoreConfig extflag.PathOrContent,
	srv *grpcSettings,
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

	sampler, err := NewSampler(db, reloaders,
		SamplerScraper(scrapeManager),
		SamplerConfig(configFile),
	)
	if err != nil {
		return nil, err
	}
	if err := sampler.Run(context.TODO(), g, reloadCh); err != nil {
		return nil, err
	}

	var sym *symbol.Symbolizer
	if symbolServerURL != "" {
		level.Debug(logger).Log("msg", "configuring symbol server", "url", symbolServerURL)
		c := symbol.NewSymbolServerClient(http.DefaultClient, symbolServerURL)
		sym = symbol.NewSymbolizer(logger, c)
	}

	w := NewWeb(mux, db, maxMergeBatchSize, queryTimeout,
		WebLogger(logger),
		WebRegistry(reg),
		WebReloaders(reloaders),
		WebTargets(func(ctx context.Context) conprofapi.TargetRetriever {
			return scrapeManager
		}),
		WebSymbolizer(sym),
	)
	if err = w.Run(context.TODO(), reloadCh); err != nil {
		return nil, err
	}

	grpcProbe := prober.NewGRPC()
	statusProber := prober.Combine(
		probe,
		grpcProbe,
		prober.NewInstrumentation(comp, logger, extprom.WrapRegistererWithPrefix("conprof_", reg)),
	)
	maxBytesPerFrame := 1024 * 1024 * 2 // 2 Mb default, might need to be tuned later on.
	s := store.NewProfileStore(logger, db, maxBytesPerFrame)

	confContentYaml, err := objStoreConfig.Content()
	if err != nil {
		return nil, err
	}

	var symStore storepb.SymbolStoreServer
	if len(confContentYaml) > 0 {
		bkt, err := client.NewBucket(logger, confContentYaml, reg, comp.String())
		if err != nil {
			return nil, errors.Wrap(err, "create bucket client")
		}
		symStore = symbol.NewSymbolStore(logger, bkt)
	}

	gsrv := grpcserver.New(logger, reg, &opentracing.NoopTracer{}, nil, nil, comp, grpcProbe,
		grpcserver.WithServer(store.RegisterReadableStoreServer(s)),
		grpcserver.WithServer(store.RegisterWritableStoreServer(s)),
		grpcserver.WithServer(store.RegisterSymbolStore(symStore)),
		grpcserver.WithListen(srv.grpcBindAddr),
		grpcserver.WithGracePeriod(srv.grpcGracePeriod),
		grpcserver.WithGRPCServerOption(
			grpc.ChainUnaryInterceptor(
				otelgrpc.UnaryServerInterceptor(),
			),
		),
		grpcserver.WithGRPCServerOption(
			grpc.ChainStreamInterceptor(
				otelgrpc.StreamServerInterceptor(),
			),
		),
	)

	g.Add(func() error {
		statusProber.Ready()
		return gsrv.ListenAndServe()
	}, func(err error) {
		grpcProbe.NotReady(err)
		gsrv.Shutdown(err)
	})

	return statusProber, nil
}
