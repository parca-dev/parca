package main

import (
	"net/http"

	"github.com/Go-SIP/conprof/pprofui"
	"github.com/go-kit/kit/log"
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
		return runWeb(mux, *storagePath)
	}
}

func runWeb(mux *http.ServeMux, storagePath string) error {
	store := pprofui.NewDiskStorage(storagePath)
	server := pprofui.NewServer(store)

	mux.Handle("/", server)

	return nil
}
