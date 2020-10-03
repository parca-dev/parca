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

	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/thanos/pkg/component"
	extpromhttp "github.com/thanos-io/thanos/pkg/extprom/http"
	"github.com/thanos-io/thanos/pkg/prober"
	"google.golang.org/grpc"
	kingpin "gopkg.in/alecthomas/kingpin.v2"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/pprofui"
	"github.com/conprof/conprof/web"
	"github.com/conprof/db/storage"
)

// registerWeb registers a web command.
func registerWeb(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}) {
	cmd := app.Command(name, "Run a web interface to view profiles from a storage.")

	storeAddress := cmd.Flag("store", "Address of statically configured store.").
		Default("127.0.0.1:10901").String()

	corsOrigin := cmd.Flag("cors.access-control-allow-origin", "Cross-origin resource sharing allowed origins.").
		Default("").String()
	corsMethods := cmd.Flag("cors.access-control-allow-methods", "Cross-origin resource sharing allowed methods.").
		Default("").String()

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) (prober.Probe, error) {
		conn, err := grpc.Dial(*storeAddress, grpc.WithInsecure())
		if err != nil {
			return probe, err
		}
		c := storepb.NewProfileStoreClient(conn)
		return probe, runWeb(
			mux,
			probe,
			reg,
			logger,
			store.NewGRPCQueryable(c),
			reloadCh,
			*corsOrigin,
			*corsMethods,
		)
	}
}

func runWeb(
	mux httpMux,
	probe prober.Probe,
	reg prometheus.Registerer,
	logger log.Logger,
	db storage.Queryable,
	reloadCh chan struct{},
	corsOrigin, corsMethods string,
) error {
	logger = log.With(logger, "component", "pprofui")
	ui := pprofui.New(logger, db)

	router := httprouter.New()
	router.RedirectTrailingSlash = false
	ins := extpromhttp.NewInstrumentationMiddleware(reg)
	instr := conprofapi.GetInstr(logger, ins)

	router.GET("/pprof/*remainder", ui.PprofView)
	router.GET("/download/*remainder", ui.PprofDownload)

	api := conprofapi.New(logger, db, reloadCh)

	router.GET("/-/reload", api.Reload)

	router.GET("/api/v1/query_range", instr("query_range", api.QueryRange))
	router.GET("/api/v1/series", instr("series", api.Series))
	router.GET("/api/v1/labels", instr("label_names", api.LabelNames))
	router.GET("/api/v1/label/:name/values", instr("label_values", api.LabelValues))

	router.NotFound = http.FileServer(web.Assets)
	mux.Handle("/", cors(corsOrigin, corsMethods, router))
	probe.Ready()

	return nil
}

func cors(corsOrigin, corsMethods string, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if corsOrigin != "" {
			w.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		}
		if corsMethods != "" {
			w.Header().Set("Access-Control-Allow-Methods", corsMethods)
		}
		h.ServeHTTP(w, r)
	})
}
