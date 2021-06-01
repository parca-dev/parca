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
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/prober"
	httpserver "github.com/thanos-io/thanos/pkg/server/http"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp"
	"go.opentelemetry.io/otel/exporters/otlp/otlpgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv"
	"gopkg.in/alecthomas/kingpin.v2"

	"github.com/conprof/conprof/config"
)

const (
	logFormatLogfmt = "logfmt"
	logFormatJSON   = "json"
)

type httpMux interface {
	Handle(pattern string, handler http.Handler)
}

type setupFunc func(component.Component, *run.Group, httpMux, prober.Probe, log.Logger, *prometheus.Registry, bool) (prober.Probe, error)

type configReloaders struct {
	funcs []func(*config.Config) error
}

func (r *configReloaders) Register(reloader func(*config.Config) error) {
	r.funcs = append(r.funcs, reloader)
}

func main() {
	if os.Getenv("DEBUG") != "" {
		runtime.SetMutexProfileFraction(10)
		runtime.SetBlockProfileRate(10)
	}

	app := kingpin.New(filepath.Base(os.Args[0]), "Continuous profiling - to have a profile when it matters.")
	app.Version(version.Print("conprof"))
	app.HelpFlag.Short('h')

	debugName := app.Flag("debug.name", "Name to add as prefix to log lines.").Hidden().String()

	logLevel := app.Flag("log.level", "Log filtering level.").
		Default("info").Enum("error", "warn", "info", "debug")
	logFormat := app.Flag("log.format", "Log format to use.").
		Default(logFormatLogfmt).Enum(logFormatLogfmt, logFormatJSON)
	otlpAddress := app.Flag("otlp-address", "OpenTelemetry collector address to send traces to.").
		Default("").String()
	corsOrigin := app.Flag("cors.access-control-allow-origin", "Cross-origin resource sharing allowed origins.").
		Default("").String()
	corsMethods := app.Flag("cors.access-control-allow-methods", "Cross-origin resource sharing allowed methods.").
		Default("").String()
	httpBindAddr, httpGracePeriod, _ := extkingpin.RegisterHTTPFlags(app)

	cmds := map[string]setupFunc{}
	reloadCh := make(chan struct{}, 1)

	reloaders := &configReloaders{}

	registerSampler(cmds, app, "sampler", reloadCh, reloaders)
	registerStorage(cmds, app, "storage", reloadCh)
	registerWeb(cmds, app, "web", reloadCh, reloaders)
	registerApi(cmds, app, "api")
	registerAll(cmds, app, "all", reloadCh, reloaders)

	cmd, err := app.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "Error parsing commandline arguments"))
		app.Usage(os.Args[1:])
		os.Exit(2)
	}

	var logger log.Logger
	{
		var lvl level.Option
		switch *logLevel {
		case "error":
			lvl = level.AllowError()
		case "warn":
			lvl = level.AllowWarn()
		case "info":
			lvl = level.AllowInfo()
		case "debug":
			lvl = level.AllowDebug()
		default:
			panic("unexpected log level")
		}
		logger = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		if *logFormat == logFormatJSON {
			logger = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
		}
		logger = level.NewFilter(logger, lvl)

		if *debugName != "" {
			logger = log.With(logger, "name", *debugName)
		}

		logger = log.With(logger, "ts", log.DefaultTimestampUTC, "caller", log.DefaultCaller)
	}

	reg := prometheus.NewRegistry()
	reg.MustRegister(
		version.NewCollector("conprof"),
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	)

	prometheus.DefaultRegisterer = reg

	var g run.Group
	mux := http.NewServeMux()
	httpProbe := prober.NewHTTP()
	comp := componentString(cmd)
	if *otlpAddress != "" {
		initTracer(logger, cmd, *otlpAddress)
	}

	statusProber, err := cmds[cmd](comp, &g, mux, httpProbe, logger, reg, *logLevel == "debug")
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "%s command failed", cmd))
		os.Exit(1)
	}

	{
		srv := httpserver.New(logger, reg, comp, httpProbe,
			httpserver.WithListen(*httpBindAddr),
			httpserver.WithGracePeriod(time.Duration(*httpGracePeriod)),
		)
		srv.Handle("/", cors(*corsOrigin, *corsMethods, mux))
		g.Add(func() error {
			statusProber.Healthy()

			return srv.ListenAndServe()
		}, func(err error) {
			statusProber.NotReady(err)
			defer statusProber.NotHealthy(err)

			srv.Shutdown(err)
		})
	}

	// Listen for termination signals.
	g.Add(run.SignalHandler(context.Background(), syscall.SIGINT, syscall.SIGTERM))

	// Synchronize all reloaders for the first time
	reloadCh <- struct{}{}

	if err := g.Run(); err != nil {
		level.Error(logger).Log("msg", "running command failed", "err", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "exiting")
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

func initTracer(logger log.Logger, serviceName string, otlpAddress string) func() {
	ctx := context.Background()
	driver := otlpgrpc.NewDriver(
		otlpgrpc.WithInsecure(),
		otlpgrpc.WithEndpoint(otlpAddress),
	)
	exporter, err := otlp.NewExporter(ctx, driver)
	handleErr(logger, err, "failed to create exporter")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	handleErr(logger, err, "failed to create resource")

	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tracerProvider)

	return func() {
		handleErr(logger, exporter.Shutdown(context.Background()), "failed to stop exporter")
	}
}

func handleErr(logger log.Logger, err error, message string) {
	if err != nil {
		level.Error(logger).Log("msg", message, "err", err)
		os.Exit(1)
	}
}
