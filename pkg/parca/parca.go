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
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	goruntime "runtime"
	"strings"
	"syscall"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/labels"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	scrapepb "github.com/parca-dev/parca/gen/proto/go/parca/scrape/v1alpha1"
	"github.com/parca-dev/parca/pkg/config"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profilestore"
	queryservice "github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/scrape"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/parca-dev/parca/pkg/symbolizer"
)

const (
	symbolizationInterval = 10 * time.Second
	flagModeScraperOnly   = "scraper-only"
)

type Flags struct {
	ConfigPath         string   `default:"parca.yaml" help:"Path to config file."`
	Mode               string   `default:"all" enum:"all,scraper-only" help:"Scraper only runs a scraper that sends to a remote gRPC endpoint. All runs all components."`
	LogLevel           string   `default:"info" enum:"error,warn,info,debug" help:"log level."`
	Port               string   `default:":7070" help:"Port string for server"`
	CORSAllowedOrigins []string `help:"Allowed CORS origins."`
	OTLPAddress        string   `help:"OpenTelemetry collector address to send traces to."`
	Version            bool     `help:"Show application version."`
	PathPrefix         string   `default:"" help:"Path prefix for the UI"`

	MutexProfileFraction int `default:"0" help:"Fraction of mutex profile samples to collect."`
	BlockProfileRate     int `default:"0" help:"Sample rate for block profile."`

	StorageTSDBRetentionTime    time.Duration `default:"6h" help:"How long to retain samples in storage."`
	StorageTSDBExpensiveMetrics bool          `default:"false" help:"Enable really heavy metrics. Only do this for debugging as the metrics are slowing Parca down by a lot." hidden:"true"`

	Storage              string `default:"tsdb" enum:"columnstore,tsdb" help:"Storage type to use."`
	StorageDebugValueLog bool   `default:"false" help:"Log every value written to the database into a separate file. This is only for debugging purposes to produce data to replay situations in tests."`
	StorageGranuleSize   int    `default:"8196" help:"Granule size for storage."`
	StorageActiveMemory  int64  `default:"536870912" help:"Amount of memory to use for active storage. Defaults to 512MB."`

	SymbolizerDemangleMode  string `default:"simple" help:"Mode to demangle C++ symbols. Default mode is simplified: no parameters, no templates, no return type" enum:"simple,full,none,templates"`
	SymbolizerNumberOfTries int    `default:"3" help:"Number of tries to attempt to symbolize an unsybolized location"`

	Metastore string `default:"badgerinmemory" help:"Which metastore implementation to use" enum:"badgerinmemory"`

	DebugInfodUpstreamServers    []string      `default:"https://debuginfod.elfutils.org" help:"Upstream debuginfod servers. Defaults to https://debuginfod.elfutils.org. It is an ordered list of servers to try. Learn more at https://sourceware.org/elfutils/Debuginfod.html"`
	DebugInfodHTTPRequestTimeout time.Duration `default:"5m" help:"Timeout duration for HTTP request to upstream debuginfod server. Defaults to 5m"`

	StoreAddress       string            `kong:"help='gRPC address to send profiles and symbols to.'"`
	BearerToken        string            `kong:"help='Bearer token to authenticate with store.'"`
	BearerTokenFile    string            `kong:"help='File to read bearer token from to authenticate with store.'"`
	Insecure           bool              `kong:"help='Send gRPC requests via plaintext instead of TLS.'"`
	InsecureSkipVerify bool              `kong:"help='Skip TLS certificate verification.'"`
	ExternalLabel      map[string]string `kong:"help='Label(s) to attach to all profiles in scraper-only mode.'"`
}

// Run the parca server.
func Run(ctx context.Context, logger log.Logger, reg *prometheus.Registry, flags *Flags, version string) error {
	goruntime.SetBlockProfileRate(flags.BlockProfileRate)
	goruntime.SetMutexProfileFraction(flags.MutexProfileFraction)

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

	if flags.Mode == flagModeScraperOnly {
		return runScraper(ctx, logger, reg, tracerProvider, flags, version, cfg)
	}

	var mStr metastore.ProfileMetaStore
	switch flags.Metastore {
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

	col := arcticdb.New(
		reg,
		flags.StorageGranuleSize,
		flags.StorageActiveMemory,
	)
	colDB, err := col.DB("parca")
	if err != nil {
		level.Error(logger).Log("msg", "failed to load database", "err", err)
		return err
	}
	table, err := colDB.Table("stacktraces", arcticdb.NewTableConfig(
		parcacol.Schema(),
	), logger)
	if err != nil {
		level.Error(logger).Log("msg", "create table", "err", err)
		return err
	}

	s := profilestore.NewProfileColumnStore(
		logger,
		tracerProvider.Tracer("profilestore"),
		mStr,
		table,
		flags.StorageDebugValueLog,
	)
	q := queryservice.NewColumnQueryAPI(
		logger,
		tracerProvider.Tracer("query-service"),
		mStr,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	discoveryManager := discovery.NewManager(ctx, logger)
	if err := discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs)); err != nil {
		level.Error(logger).Log("msg", "failed to apply discovery configs", "err", err)
		return err
	}

	m := scrape.NewManager(logger, reg, s, cfg.ScrapeConfigs, labels.Labels{})
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

	var debugInfodClient debuginfo.DebugInfodClient = debuginfo.NopDebugInfodClient{}
	if len(flags.DebugInfodUpstreamServers) > 0 {
		httpDebugInfoClient, err := debuginfo.NewHTTPDebugInfodClient(logger, flags.DebugInfodUpstreamServers, flags.DebugInfodHTTPRequestTimeout)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize debuginfod http client", "err", err)
			return err
		}

		debugInfodClient, err = debuginfo.NewDebugInfodClientWithObjectStorageCache(logger, cfg.DebugInfo, httpDebugInfoClient)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize debuginfod client cache", "err", err)
			return err
		}
	}

	dbgInfo, err := debuginfo.NewStore(logger, sym, cfg.DebugInfo, debugInfodClient)
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
				flags.PathPrefix,
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
				level.Error(logger).Log("msg", "error shutting down server", "err", err)
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

func runScraper(
	ctx context.Context,
	logger log.Logger,
	reg *prometheus.Registry,
	tracer trace.TracerProvider,
	flags *Flags,
	version string,
	cfg config.Config,
) error {
	if flags.StoreAddress == "" {
		return fmt.Errorf("parca scraper mode needs to have a --store-address")
	}

	metrics := grpc_prometheus.NewClientMetrics()
	metrics.EnableClientHandlingTimeHistogram()
	reg.MustRegister(metrics)

	opts := []grpc.DialOption{
		grpc.WithUnaryInterceptor(
			metrics.UnaryClientInterceptor(),
		),
	}
	if flags.Insecure {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			InsecureSkipVerify: flags.InsecureSkipVerify,
		})))
	}

	if flags.BearerToken != "" {
		opts = append(opts, grpc.WithPerRPCCredentials(&perRequestBearerToken{
			token:    flags.BearerToken,
			insecure: flags.Insecure,
		}))
	}

	if flags.BearerTokenFile != "" {
		b, err := ioutil.ReadFile(flags.BearerTokenFile)
		if err != nil {
			return fmt.Errorf("failed to read bearer token from file: %w", err)
		}
		opts = append(opts, grpc.WithPerRPCCredentials(&perRequestBearerToken{
			token:    strings.TrimSpace(string(b)),
			insecure: flags.Insecure,
		}))
	}

	conn, err := grpc.Dial(flags.StoreAddress, opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection: %w", err)
	}

	store := profilestore.NewGRPCForwarder(conn, logger)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	discoveryManager := discovery.NewManager(ctx, logger)
	if err := discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs)); err != nil {
		level.Error(logger).Log("msg", "failed to apply discovery configs", "err", err)
		return err
	}

	externalLabels := labels.Labels{}
	for name, value := range flags.ExternalLabel {
		externalLabels = append(externalLabels, labels.Label{Name: name, Value: value})
	}

	m := scrape.NewManager(logger, reg, store, cfg.ScrapeConfigs, externalLabels)
	if err := m.ApplyConfig(cfg.ScrapeConfigs); err != nil {
		level.Error(logger).Log("msg", "failed to apply scrape configs", "err", err)
		return err
	}

	var gr run.Group
	gr.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM))
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
				flags.PathPrefix,
				server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
					scrapepb.RegisterScrapeServiceServer(srv, m)
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
				level.Error(logger).Log("msg", "error shutting down server", "err", err)
			}
		},
	)

	level.Info(logger).Log("msg", "running Parca in scrape mode", "version", version)
	if err := gr.Run(); err != nil {
		if _, ok := err.(run.SignalError); ok {
			return nil
		}
		return err
	}
	return nil
}

type perRequestBearerToken struct {
	token    string
	insecure bool
}

func (t *perRequestBearerToken) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.token,
	}, nil
}

func (t *perRequestBearerToken) RequireTransportSecurity() bool {
	return !t.insecure
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
