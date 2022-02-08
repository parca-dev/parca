// Copyright 2021 The Parca Authors
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

package parca

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	scrapepb "github.com/parca-dev/parca/gen/proto/go/parca/scrape/v1alpha1"
	"github.com/parca-dev/parca/pkg/config"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/scrape"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/parca-dev/parca/pkg/storage"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/parca-dev/parca/pkg/symbolizer"
)

const symbolizationInterval = 10 * time.Second

type Flags struct {
	ConfigPath         string   `default:"parca.yaml" help:"Path to config file."`
	LogLevel           string   `default:"info" enum:"error,warn,info,debug" help:"log level."`
	Port               string   `default:":7070" help:"Port string for server"`
	CORSAllowedOrigins []string `help:"Allowed CORS origins."`
	OTLPAddress        string   `help:"OpenTelemetry collector address to send traces to."`
	Version            bool     `help:"Show application version."`

	StorageTSDBRetentionTime    time.Duration `default:"6h" help:"How long to retain samples in storage."`
	StorageTSDBExpensiveMetrics bool          `default:"false" help:"Enable really heavy metrics. Only do this for debugging as the metrics are slowing Parca down by a lot." hidden:"true"`

	SymbolizerDemangleMode  string `default:"simple" help:"Mode to demangle C++ symbols. Default mode is simplified: no parameters, no templates, no return type" enum:"simple,full,none,templates"`
	SymbolizerNumberOfTries int    `default:"3" help:"Number of tries to attempt to symbolize an unsybolized location"`

	Metastore string `default:"badgerinmemory" help:"Which metastore implementation to use" enum:"sqliteinmemory,badgerinmemory"`

	UpstreamDebuginfodServer string `default:"https://debuginfod.systemtap.org" help:"Upstream private/public server for debuginfod files. Defaults to https://debuginfod.systemtap.org
	."`
	//TODO: change this to a url type
}

// Run the parca server.
func Run(ctx context.Context, logger log.Logger, reg *prometheus.Registry, flags *Flags, version string) error {
	tracerProvider := trace.NewNoopTracerProvider()
	if flags.OTLPAddress != "" {
		var closer func()
		var err error
		tracerProvider, closer, err = initTracer(logger, flags.OTLPAddress)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize tracing", "err", err)
			return err
		}
		defer closer()
	}

	cfgContent, err := ioutil.ReadFile(flags.ConfigPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to read config", "path", flags.ConfigPath)
		return err
	}

	cfg := config.Config{}
	if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
		level.Error(logger).Log("msg", "failed to parse config", "err", err, "path", flags.ConfigPath)
		return err
	}

	if err := cfg.Validate(); err != nil {
		level.Error(logger).Log("msg", "parsed config invalid", "err", err, "path", flags.ConfigPath)
		return err
	}

	var mStr metastore.ProfileMetaStore
	switch flags.Metastore {
	case "sqliteinmemory":
		mStr, err = metastore.NewInMemorySQLiteProfileMetaStore(
			reg,
			// Produces high cardinality traces - uncomment locally if needed.
			// tracerProvider.Tracer("inmemory-sqlite"),
			trace.NewNoopTracerProvider().Tracer("inmemory-sqlite"),
		)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize metadata store", "err", err)
			return err
		}
		defer mStr.Close()
	case "badgerinmemory":
		mStr = metastore.NewBadgerMetastore(
			logger,
			reg,
			tracerProvider.Tracer("badgerinmemory"),
			metastore.NewRandomUUIDGenerator(),
		)
	default:
		level.Error(logger).Log("msg", "unknown metastore implementation", "chosen", flags.Metastore)
		return err
	}

	db := storage.OpenDB(
		reg,
		tracerProvider.Tracer("db"),
		&storage.DBOptions{
			Retention:            flags.StorageTSDBRetentionTime,
			HeadExpensiveMetrics: flags.StorageTSDBExpensiveMetrics,
		},
	)
	s := profilestore.NewProfileStore(
		logger,
		tracerProvider.Tracer("profilestore"),
		db,
		mStr,
	)
	q := query.New(
		logger,
		tracerProvider.Tracer("query-service"),
		db,
		mStr,
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	discoveryManager := discovery.NewManager(ctx, logger)
	if err := discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs)); err != nil {
		level.Error(logger).Log("msg", "failed to apply discovery configs", "err", err)
		return err
	}

	m := scrape.NewManager(logger, reg, s, cfg.ScrapeConfigs)
	if err := m.ApplyConfig(cfg.ScrapeConfigs); err != nil {
		level.Error(logger).Log("msg", "failed to apply scrape configs", "err", err)
		return err
	}

	sym, err := symbol.NewSymbolizer(logger,
		symbol.WithDemangleMode(flags.SymbolizerDemangleMode),
		symbol.WithAttemptThreshold(flags.SymbolizerNumberOfTries),
		symbol.WithCacheItemTTL(symbolizationInterval*3),
	)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize symbolizer", "err", err)
		return err
	}

	dbgInfo, err := debuginfo.NewStore(logger, sym, cfg.DebugInfo, flags.UpstreamDebuginfodServer)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize debug info store", "err", err)
		return err
	}

	var gr run.Group
	gr.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM))
	{
		s := symbolizer.New(logger, mStr, dbgInfo)
		ctx, cancel := context.WithCancel(ctx)
		gr.Add(
			func() error {
				return s.Run(ctx, symbolizationInterval)
			},
			func(_ error) {
				level.Debug(logger).Log("msg", "symbolizer server shutting down")
				cancel()
				sym.Close()
			})
	}
	{
		ctx, cancel := context.WithCancel(ctx)
		gr.Add(func() error {
			return db.Run(ctx)
		}, func(err error) {
			level.Debug(logger).Log("msg", "db exiting")
			cancel()
		})
	}
	gr.Add(
		func() error {
			return discoveryManager.Run()
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "discovery manager exiting")
			cancel()
		},
	)
	gr.Add(
		func() error {
			return m.Run(discoveryManager.SyncCh())
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "scrape manager exiting")
			m.Stop()
		},
	)
	parcaserver := server.NewServer(reg, version)
	gr.Add(
		func() error {
			return parcaserver.ListenAndServe(
				ctx,
				logger,
				flags.Port,
				flags.CORSAllowedOrigins,
				server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
					debuginfopb.RegisterDebugInfoServiceServer(srv, dbgInfo)
					profilestorepb.RegisterProfileStoreServiceServer(srv, s)
					querypb.RegisterQueryServiceServer(srv, q)
					scrapepb.RegisterScrapeServiceServer(srv, m)

					if err := debuginfopb.RegisterDebugInfoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					if err := profilestorepb.RegisterProfileStoreServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					if err := querypb.RegisterQueryServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					if err := scrapepb.RegisterScrapeServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
						return err
					}

					return nil
				}),
			)
		},
		func(_ error) {
			ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // TODO make this a graceful shutdown config setting
			defer cancel()

			level.Debug(logger).Log("msg", "server shutting down")
			err := parcaserver.Shutdown(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				level.Error(logger).Log("msg", "error shuttiing down server", "err", err)
			}
		},
	)
	if err := gr.Run(); err != nil {
		if _, ok := err.(run.SignalError); ok {
			return nil
		}
		return err
	}

	return nil
}

func getDiscoveryConfigs(cfgs []*config.ScrapeConfig) map[string]discovery.Configs {
	c := make(map[string]discovery.Configs)
	for _, v := range cfgs {
		c[v.JobName] = v.ServiceDiscoveryConfigs
	}
	return c
}

func initTracer(logger log.Logger, otlpAddress string) (trace.TracerProvider, func(), error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("parca"),
		),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Set up a trace exporter
	exporter, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(otlpAddress),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(exporter)
	provider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(provider)

	return provider, func() {
		err := exporter.Shutdown(context.Background())
		if err != nil {
			level.Error(logger).Log("msg", "failed to stop exporter", "err", err)
		}
	}, nil
}
