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
	"net/http"
	"time"

	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/prober"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/symbol"
)

// registerApi registers a API command.
func registerApi(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "Run an API to query profiles from a storage.")

	storeAddress := cmd.Flag("store", "Address of statically configured store.").
		Default("127.0.0.1:10901").String()
	symbolServerURL := cmd.Flag("symbol-server-url", "Symbol server to request to symbolize native stacktraces.").String()
	maxMergeBatchSize := cmd.Flag("max-merge-batch-size", "Bytes loaded in one batch for merging. This is to limit the amount of memory a merge query can use.").
		Default("64MB").Bytes()
	queryTimeout := extkingpin.ModelDuration(cmd.Flag("query.timeout", "Maximum time to process query by query node.").
		Default("10s"))

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, debugLogging bool) (prober.Probe, error) {
		conn, err := grpc.Dial(
			*storeAddress,
			grpc.WithInsecure(),
			grpc.WithUnaryInterceptor(
				otelgrpc.UnaryClientInterceptor(),
			),
			grpc.WithStreamInterceptor(
				otelgrpc.StreamClientInterceptor(),
			),
		)
		if err != nil {
			return probe, err
		}
		c := storepb.NewReadableProfileStoreClient(conn)
		return probe, runApi(
			mux,
			probe,
			reg,
			logger,
			store.NewGRPCQueryable(c),
			int64(*maxMergeBatchSize),
			*queryTimeout,
			*symbolServerURL,
		)
	}
}

func runApi(
	mux httpMux,
	probe prober.Probe,
	reg *prometheus.Registry,
	logger log.Logger,
	db storage.Queryable,
	maxMergeBatchSize int64,
	queryTimeout model.Duration,
	symbolServerURL string,
) error {
	logger = log.With(logger, "component", "api")

	var s *symbol.Symbolizer
	if symbolServerURL != "" {
		c := symbol.NewSymbolServerClient(http.DefaultClient, symbolServerURL)
		s = symbol.NewSymbolizer(logger, c)
	}

	const apiPrefix = "/api/v1/"
	api := conprofapi.New(logger, reg,
		conprofapi.WithDB(db),
		conprofapi.WithMaxMergeBatchSize(maxMergeBatchSize),
		conprofapi.WithPrefix(apiPrefix),
		conprofapi.WithQueryTimeout(time.Duration(queryTimeout)),
		conprofapi.WithSymbolizer(s),
	)
	mux.Handle(apiPrefix, api.Routes())

	probe.Ready()

	return nil
}
