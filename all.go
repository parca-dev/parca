package main

import (
	"net/http"
	"time"

	"github.com/conprof/conprof/filestorage"
	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// registerAll registers the all command.
func registerAll(m map[string]setupFunc, app *kingpin.Application, name string) {
	cmd := app.Command(name, "All in one command.")

	storagePath := cmd.Flag("storage.fs.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := modelDuration(cmd.Flag("storage.fs.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))

	m[name] = func(g *run.Group, mux *http.ServeMux, logger log.Logger, reg *prometheus.Registry, tracer opentracing.Tracer, debugLogging bool) error {
		return runAll(g, mux, logger, *storagePath, *configFile, *retention)
	}
}

func modelDuration(flags *kingpin.FlagClause) *model.Duration {
	var value = new(model.Duration)
	flags.SetValue(value)

	return value
}

func runAll(g *run.Group, mux *http.ServeMux, logger log.Logger, storagePath, configFile string, retention model.Duration) error {
	storage := filestorage.NewFileStorage(storagePath, time.Duration(retention), logger)

	err := runSampler(g, logger, storage, configFile)
	if err != nil {
		return err
	}

	err = runWeb(mux, logger, storage)
	if err != nil {
		return err
	}

	return nil
}
