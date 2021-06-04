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
	"time"

	"github.com/go-kit/kit/log"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/tags"
	"github.com/oklog/run"
	"github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extflag"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/extprom"
	"github.com/thanos-io/thanos/pkg/logging"
	objstore "github.com/thanos-io/thanos/pkg/objstore/client"
	"github.com/thanos-io/thanos/pkg/prober"
	grpcserver "github.com/thanos-io/thanos/pkg/server/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/symbol"
)

// registerSymbol registers a symbol command.
func registerSymbol(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "Run a symbol management server that allows checking for existence and uploading symbols.")

	grpcBindAddr, grpcGracePeriod, grpcCert, grpcKey, grpcClientCA := extkingpin.RegisterGRPCFlags(cmd)
	objStoreConfig := *extkingpin.RegisterCommonObjStoreFlags(cmd, "", false)
	symbolCache := cmd.Flag("symbol-cache", "Directory to use to cache symbol data from object storage.").
		Default("/tmp").String()
	reqLogConfig := extkingpin.RegisterRequestLoggingFlags(cmd)

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, debugLogging bool) (prober.Probe, error) {
		tagOpts, grpcLogOpts, err := logging.ParsegRPCOptions("", reqLogConfig)
		if err != nil {
			return probe, errors.Wrap(err, "error while parsing config for request logging")
		}
		return runSymbol(
			comp,
			g,
			probe,
			reg,
			logger,
			grpcLogOpts,
			tagOpts,
			*symbolCache,
			objStoreConfig,
			*grpcBindAddr,
			time.Duration(*grpcGracePeriod),
			*grpcCert,
			*grpcKey,
			*grpcClientCA,
		)
	}
}

func runSymbol(
	comp component.Component,
	g *run.Group,
	probe prober.Probe,
	reg *prometheus.Registry,
	logger log.Logger,
	grpcLogOpts []grpc_logging.Option,
	tagOpts []tags.Option,
	symbolCache string,
	objStoreConfig extflag.PathOrContent,
	grpcBindAddr string,
	grpcGracePeriod time.Duration,
	grpcCert string,
	grpcKey string,
	grpcClientCA string,
) (prober.Probe, error) {
	grpcProbe := prober.NewGRPC()
	statusProber := prober.Combine(
		probe,
		grpcProbe,
		prober.NewInstrumentation(comp, logger, extprom.WrapRegistererWithPrefix("conprof_", reg)),
	)

	confContentYaml, err := objStoreConfig.Content()
	if err != nil {
		return nil, err
	}

	bkt, err := objstore.NewBucket(logger, confContentYaml, reg, comp.String())
	if err != nil {
		return nil, errors.Wrap(err, "create object store bucket client")
	}
	sym := symbol.NewSymbolStore(logger, bkt, symbolCache)

	srv := grpcserver.New(logger, reg, &opentracing.NoopTracer{}, grpcLogOpts, tagOpts, comp, grpcProbe,
		grpcserver.WithServer(store.RegisterSymbolStore(sym)),
		grpcserver.WithListen(grpcBindAddr),
		grpcserver.WithGracePeriod(grpcGracePeriod),
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
		return srv.ListenAndServe()
	}, func(err error) {
		grpcProbe.NotReady(err)
		srv.Shutdown(err)
	})

	return statusProber, nil
}
