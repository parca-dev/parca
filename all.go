package main

import (
	"net/http"
	"time"

	"github.com/conprof/conprof/storage/tsdb"
	"github.com/conprof/conprof/storage/tsdb/wal"
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

	storagePath := cmd.Flag("storage.tsdb.path", "Directory to read storage from.").
		Default("./data").String()
	configFile := cmd.Flag("config.file", "Config file to use.").
		Default("conprof.yaml").String()
	retention := modelDuration(cmd.Flag("tsdb.retention.time", "How long to retain raw samples on local storage. 0d - disables this retention").Default("15d"))

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
	db, err := tsdb.Open(
		storagePath,
		logger,
		prometheus.DefaultRegisterer,
		&tsdb.Options{
			WALSegmentSize:    wal.DefaultSegmentSize,
			RetentionDuration: uint64(retention),
			BlockRanges:       tsdb.ExponentialBlockRanges(int64(2*time.Hour)/1e6, 3, 5),
			NoLockfile:        true,
		},
	)
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
