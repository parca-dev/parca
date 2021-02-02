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
	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"

	//"github.com/julienschmidt/httprouter"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/component"
	extpromhttp "github.com/thanos-io/thanos/pkg/extprom/http"
	"github.com/thanos-io/thanos/pkg/prober"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/conprof/db/storage"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
)

// registerApi registers a API command.
func registerApi(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "Run an API to query profiles from a storage.")

	storeAddress := cmd.Flag("store", "Address of statically configured store.").
		Default("127.0.0.1:10901").String()
	maxMergeBatchSize := cmd.Flag("max-merge-batch-size", "Bytes loaded in one batch for merging. This is to limit the amount of memory a merge query can use.").
		Default("64MB").Bytes()

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
		)
	}
}

func runApi(
	mux httpMux,
	probe prober.Probe,
	reg prometheus.Registerer,
	logger log.Logger,
	db storage.Queryable,
	maxMergeBatchSize int64,
) error {
	logger = log.With(logger, "component", "api")

	router := httprouter.New()
	router.RedirectTrailingSlash = false
	ins := extpromhttp.NewInstrumentationMiddleware(reg)
	instr := conprofapi.Instr(logger, ins)

	api := conprofapi.New(logger, db, nil, maxMergeBatchSize)

	router.GET("/api/v1/query_range", instr("query_range", api.QueryRange))
	router.GET("/api/v1/query", instr("query", api.Query))
	router.GET("/api/v1/series", instr("series", api.Series))
	router.GET("/api/v1/labels", instr("label_names", api.LabelNames))
	router.GET("/api/v1/label/:name/values", instr("label_values", api.LabelValues))

	mux.Handle("/", router)
	probe.Ready()

	return nil
}
