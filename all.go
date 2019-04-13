package main

import (
	"net/http"

	"github.com/Go-SIP/conprof/storage/tsdb"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		return runAll(g, mux, logger, *storagePath, *configFile)
	}
}

func runAll(g *run.Group, mux *http.ServeMux, logger log.Logger, storagePath, configFile string) error {
	db, err := tsdb.Open(storagePath, logger, prometheus.DefaultRegisterer, tsdb.DefaultOptions)
	if err != nil {
		return err
	}

	err = runSampler(g, logger, db, configFile)
	if err != nil {
		return err
	}

	err = runWeb(mux, logger, db)
	if err != nil {
		return err
	}

	return nil
}
