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
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	opentracing "github.com/opentracing/opentracing-go"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/version"
	"github.com/thanos-io/thanos/pkg/component"
	"github.com/thanos-io/thanos/pkg/prober"
	httpserver "github.com/thanos-io/thanos/pkg/server/http"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

const (
	logFormatLogfmt = "logfmt"
	logFormatJSON   = "json"
)

type httpMux interface {
	Handle(pattern string, handler http.Handler)
}

type setupFunc func(component.Component, *run.Group, httpMux, prober.Probe, log.Logger, *prometheus.Registry, opentracing.Tracer, bool) (prober.Probe, error)

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
	httpBindAddr, httpGracePeriod := regHTTPFlags(app)

	cmds := map[string]setupFunc{}
	reloadCh := make(chan struct{}, 1)
	registerSampler(cmds, app, "sampler", reloadCh)
	registerStorage(cmds, app, "storage", reloadCh)
	registerWeb(cmds, app, "web", reloadCh)
	registerAll(cmds, app, "all", reloadCh)

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
	var tracer opentracing.Tracer
	mux := http.NewServeMux()
	httpProbe := prober.NewHTTP()
	comp := componentString(cmd)

	statusProber, err := cmds[cmd](comp, &g, mux, httpProbe, logger, reg, tracer, *logLevel == "debug")
	if err != nil {
		fmt.Fprintln(os.Stderr, errors.Wrapf(err, "%s command failed", cmd))
		os.Exit(1)
	}

	{
		srv := httpserver.New(logger, reg, comp, httpProbe,
			httpserver.WithListen(*httpBindAddr),
			httpserver.WithGracePeriod(time.Duration(*httpGracePeriod)),
		)
		srv.Handle("/", mux)
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
	{
		cancel := make(chan struct{})
		g.Add(func() error {
			return interrupt(logger, cancel)
		}, func(error) {
			level.Debug(logger).Log("msg", "shutting down interrupt handler")
			close(cancel)
		})
	}

	if err := g.Run(); err != nil {
		level.Error(logger).Log("msg", "running command failed", "err", err)
		os.Exit(1)
	}
	level.Info(logger).Log("msg", "exiting")
}

func interrupt(logger log.Logger, cancel <-chan struct{}) error {
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	select {
	case s := <-c:
		level.Info(logger).Log("msg", "caught signal. Exiting.", "signal", s)
		return nil
	case <-cancel:
		return errors.New("canceled")
	}
}
