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

	"github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/pprofui"
	"github.com/conprof/conprof/web"
	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerWeb registers a web command.
func registerWeb(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}) {
	cmd := app.Command(name, "Run a web interface to view profiles from a storage.")

	storeAddress := cmd.Flag("store", "Address of statically configured store.").
		Default("127.0.0.1:10000").String()

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		conn, err := grpc.Dial(*storeAddress, grpc.WithInsecure())
		if err != nil {
			return err
		}
		c := storepb.NewProfileStoreClient(conn)
		return runWeb(mux, logger, store.NewGRPCQueryable(c), reloadCh)
	}
}

func runWeb(mux *http.ServeMux, logger log.Logger, db storage.Queryable, reloadCh chan struct{}) error {
	ui := pprofui.New(log.With(logger, "component", "pprofui"), db)

	router := httprouter.New()
	router.RedirectTrailingSlash = false

	router.GET("/pprof/*remainder", ui.PprofView)
	router.GET("/download/*remainder", ui.PprofDownload)

	api := api.New(log.With(logger, "component", "pprofui"), db, reloadCh)

	router.GET("/-/reload", api.Reload)

	// TODO(bwplotka): Consider replacing this with remote read https://github.com/conprof/conprof/issues/80.
	router.GET("/api/v1/query_range", api.QueryRange)
	router.GET("/api/v1/series", api.Series)
	router.GET("/api/v1/labels", api.LabelNames)
	router.GET("/api/v1/label/:label_name/values", api.LabelValues)

	router.NotFound = http.FileServer(web.Assets)

	mux.Handle("/", router)

	return nil
}
