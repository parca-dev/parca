package main

import (
	"net/http"
	"time"

	"github.com/conprof/conprof/api"
	"github.com/conprof/conprof/pprofui"
	"github.com/conprof/conprof/storage/tsdb"
	"github.com/conprof/conprof/storage/tsdb/wal"
	"github.com/conprof/conprof/web"
	"github.com/go-kit/kit/log"
	"github.com/julienschmidt/httprouter"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerSampler registers a sampler command.
func registerWeb(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "Run a web interface to view profiles from a storage.")

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	retention := modelDuration(cmd.Flag("tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		db, err := tsdb.Open(
			*storagePath,
			logger,
			prometheus.DefaultRegisterer,
			&tsdb.Options{
				WALSegmentSize:    wal.DefaultSegmentSize,
				RetentionDuration: uint64(*retention),
				BlockRanges:       tsdb.ExponentialBlockRanges(int64(2*time.Hour)/1e6, 3, 5),
				NoLockfile:        true,
			},
		)
		if err != nil {
			return err
		}
		return runWeb(mux, logger, db)
	}
}

func runWeb(mux *http.ServeMux, logger log.Logger, db *tsdb.DB) error {
	ui := pprofui.New(log.With(logger, "component", "pprofui"), db)

	router := httprouter.New()
	router.RedirectTrailingSlash = false

	router.GET("/pprof/*remainder", ui.PprofView)

	api := api.New(log.With(logger, "component", "pprofui"), db)
	router.GET("/api/v1/query_range", api.QueryRange)

	router.NotFound = http.FileServer(web.Assets)

	mux.Handle("/", router)

	return nil
}
