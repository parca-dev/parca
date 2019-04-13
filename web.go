package main

import (
	"net/http"

	"github.com/Go-SIP/conprof/pprofui"
	"github.com/Go-SIP/conprof/storage/tsdb"
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

	storagePath := cmd.Flag("storage.path", "Directory to read storage from.").
		Default("./data").String()

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		db, err := tsdb.Open(*storagePath, logger, prometheus.DefaultRegisterer, tsdb.DefaultOptions)
		if err != nil {
			return err
		}
		return runWeb(mux, logger, db)
	}
}

func runWeb(mux *http.ServeMux, logger log.Logger, db *tsdb.DB) error {
	ui := pprofui.New(logger, db)

	router := httprouter.New()

	router.GET("/", func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		http.Redirect(w, r, "/pprof", http.StatusMovedPermanently)
		return
	})
	router.GET("/pprof", ui.QueryView)
	router.GET("/pprof/:series/:timestamp/*remainder", ui.PprofView)

	mux.Handle("/", router)

	return nil
}
