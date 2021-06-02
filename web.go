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

	"github.com/conprof/db/storage"
	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/logging"
	"github.com/thanos-io/thanos/pkg/prober"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"

	conprofapi "github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pkg/store"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/conprof/pprofui"
	"github.com/conprof/conprof/symbol"
	"github.com/conprof/conprof/web"
)

// registerWeb registers a web command.
func registerWeb(m map[string]setupFunc, app *kingpin.Application, name string, reloadCh chan struct{}, reloaders *configReloaders) {
	cmd := app.Command(name, "Run a web interface to view profiles from a storage.")

	storeAddress := cmd.Flag("store", "Address of statically configured store.").
		Default("127.0.0.1:10901").String()
	symbolServerURL := cmd.Flag("symbol-server-url", "Symbol server to request to symbolize native stacktraces.").String()
	maxMergeBatchSize := cmd.Flag("max-merge-batch-size", "Bytes loaded in one batch for merging. This is to limit the amount of memory a merge query can use.").
		Default("64MB").Bytes()
	queryTimeout := extkingpin.ModelDuration(cmd.Flag("query.timeout", "Maximum time to process query by query node.").
		Default("10s"))

	m[name] = func(comp component.Component, g *run.Group, mux httpMux, probe prober.Probe, logger log.Logger, reg *prometheus.Registry, debugLogging bool) (prober.Probe, error) {
		conn, err := grpc.Dial(*storeAddress, grpc.WithInsecure())
		if err != nil {
			return probe, err
		}
		c := storepb.NewReadableProfileStoreClient(conn)

		var s *symbol.Symbolizer = nil
		if *symbolServerURL != "" {
			c := symbol.NewSymbolServerClient(http.DefaultClient, *symbolServerURL)
			s = symbol.NewSymbolizer(logger, c)
		}

		w := NewWeb(
			mux,
			store.NewGRPCQueryable(c),
			int64(*maxMergeBatchSize),
			*queryTimeout,
			WebLogger(logger),
			WebRegistry(reg),
			WebSymbolizer(s),
		)
		err = w.Run(context.Background(), reloadCh)
		if err != nil {
			return probe, err
		}

		probe.Ready()

		return probe, nil
	}
}

type Web struct {
	mux               httpMux
	logger            log.Logger
	registry          *prometheus.Registry
	db                storage.Queryable
	reloaders         *configReloaders
	maxMergeBatchSize int64
	queryTimeout      model.Duration
	targets           func(context.Context) conprofapi.TargetRetriever
	symbolizer        *symbol.Symbolizer
	httpLogOpts       []logging.Option
}

func NewWeb(
	mux httpMux,
	db storage.Queryable,
	maxMergeBatchSize int64,
	queryTimeout model.Duration,
	opts ...WebOption,
) *Web {
	w := &Web{
		mux:               mux,
		logger:            log.NewNopLogger(),
		registry:          prometheus.NewRegistry(),
		db:                db,
		reloaders:         nil,
		maxMergeBatchSize: maxMergeBatchSize,
		queryTimeout:      queryTimeout,
		targets: func(ctx context.Context) conprofapi.TargetRetriever {
			return nil
		},
	}

	for _, opt := range opts {
		opt(w)
	}

	return w
}

type WebOption func(w *Web)

func WebLogger(logger log.Logger) WebOption {
	return func(w *Web) {
		w.logger = logger
	}
}

func WebLogOpts(httpLogOpts ...logging.Option) WebOption {
	return func(w *Web) {
		w.httpLogOpts = httpLogOpts
	}
}

func WebSymbolizer(s *symbol.Symbolizer) WebOption {
	return func(w *Web) {
		w.symbolizer = s
	}
}

func WebRegistry(registry *prometheus.Registry) WebOption {
	return func(w *Web) {
		w.registry = registry
	}
}

func WebReloaders(reloaders *configReloaders) WebOption {
	return func(w *Web) {
		w.reloaders = reloaders
	}
}

func WebTargets(targets func(context.Context) conprofapi.TargetRetriever) WebOption {
	return func(w *Web) {
		w.targets = targets
	}
}

func (w *Web) Run(_ context.Context, reloadCh chan struct{}) error {
	ui := pprofui.New(log.With(w.logger, "component", "pprofui"), w.db, w.symbolizer)

	logMiddleware := logging.NewHTTPServerMiddleware(w.logger, w.httpLogOpts...)

	const apiPrefix = "/api/v1/"
	api := conprofapi.New(log.With(w.logger, "component", "api"), w.registry,
		conprofapi.WithDB(w.db),
		conprofapi.WithMaxMergeBatchSize(w.maxMergeBatchSize),
		conprofapi.WithReloadChannel(reloadCh),
		conprofapi.WithTargets(w.targets),
		conprofapi.WithPrefix(apiPrefix),
		conprofapi.WithQueryTimeout(time.Duration(w.queryTimeout)),
		conprofapi.WithSymbolizer(w.symbolizer),
	)
	w.mux.Handle(apiPrefix, logMiddleware.HTTPMiddleware("api", api.Routes()))

	if w.reloaders != nil {
		w.reloaders.Register(api.ApplyConfig)
	}

	router := httprouter.New()
	router.RedirectTrailingSlash = false

	router.GET("/-/reload", api.Reload)
	router.GET("/pprof/*remainder", ui.PprofView)
	router.NotFound = http.FileServer(web.Assets)

	w.mux.Handle("/", router)

	return nil
}
