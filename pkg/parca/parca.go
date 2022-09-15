// Copyright 2022 The Parca Authors
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
	"os"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"syscall"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/dgraph-io/badger/v3"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
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
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	scrapepb "github.com/parca-dev/parca/gen/proto/go/parca/scrape/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/share"
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
	metaStoreBadger       = "badger"
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

	EnablePersistence bool `default:"false" help:"Turn on persistent storage for the metastore and profile storage."`

	StorageDebugValueLog bool   `default:"false" help:"Log every value written to the database into a separate file. This is only for debugging purposes to produce data to replay situations in tests."`
	StorageGranuleSize   int64  `default:"26265625" help:"Granule size in bytes for storage."`
	StorageActiveMemory  int64  `default:"536870912" help:"Amount of memory to use for active storage. Defaults to 512MB."`
	StoragePath          string `default:"data" help:"Path to storage directory."`
	StorageEnableWAL     bool   `default:"false" help:"Enables write ahead log for profile storage."`

	SymbolizerDemangleMode  string `default:"simple" help:"Mode to demangle C++ symbols. Default mode is simplified: no parameters, no templates, no return type" enum:"simple,full,none,templates"`
	SymbolizerNumberOfTries int    `default:"3" help:"Number of tries to attempt to symbolize an unsybolized location"`

	Metastore string `default:"badger" help:"Which metastore implementation to use" enum:"badger"`

	ProfileShareServer string `default:"api.pprof.me:443" help:"gRPC address to send share profile requests to."`

	DebugInfodUpstreamServers    []string      `default:"https://debuginfod.elfutils.org" help:"Upstream debuginfod servers. Defaults to https://debuginfod.elfutils.org. It is an ordered list of servers to try. Learn more at https://sourceware.org/elfutils/Debuginfod.html"`
	DebugInfodHTTPRequestTimeout time.Duration `default:"5m" help:"Timeout duration for HTTP request to upstream debuginfod server. Defaults to 5m"`
	DebuginfoCacheDir            string        `default:"/tmp" help:"Path to directory where debuginfo is cached."`

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

	cfg, err := config.LoadFile(flags.ConfigPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to read config", "path", flags.ConfigPath)
		return err
	}

	if err := cfg.Validate(); err != nil {
		level.Error(logger).Log("msg", "parsed config invalid", "err", err, "path", flags.ConfigPath)
		return err
	}

	if flags.StoreAddress != "" && flags.Mode != flagModeScraperOnly {
		return fmt.Errorf("the mode should be set as `--mode=scraper-only`, if `StoreAddress` is set")
	}

	if flags.Mode == flagModeScraperOnly {
		return runScraper(ctx, logger, reg, tracerProvider, flags, version, cfg)
	}

	bucketCfg, err := yaml.Marshal(cfg.ObjectStorage.Bucket)
	if err != nil {
		level.Error(logger).Log("msg", "failed to marshal object storage bucket config", "err", err)
		return err
	}

	bucket, err := client.NewBucket(logger, bucketCfg, reg, "parca")
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize object storage bucket", "err", err)
		return err
	}

	var mStr metastorepb.MetastoreServiceServer
	switch flags.Metastore {
	case metaStoreBadger:
		var badgerOptions badger.Options
		switch flags.EnablePersistence {
		case true:
			badgerOptions = badger.DefaultOptions(filepath.Join(flags.StoragePath, "metastore"))
		default:
			badgerOptions = badger.DefaultOptions("").WithInMemory(true)
		}

		badgerOptions = badgerOptions.WithLogger(&metastore.BadgerLogger{Logger: logger})
		db, err := badger.Open(badgerOptions)
		if err != nil {
			level.Error(logger).Log("msg", "failed to open badger database for metastore", "err", err)
			return err
		}

		mStr = metastore.NewBadgerMetastore(
			logger,
			reg,
			tracerProvider.Tracer(metaStoreBadger),
			db,
		)
	default:
		err := fmt.Errorf("unknown metastore implementation: %s", flags.Metastore)
		level.Error(logger).Log("msg", "failed to initialize metastore", "err", err)
		return err
	}

	metastore := metastore.NewInProcessClient(mStr)

	frostdbOptions := []frostdb.Option{
		frostdb.WithActiveMemorySize(flags.StorageActiveMemory),
		frostdb.WithLogger(logger),
		frostdb.WithRegistry(reg),
		frostdb.WithTracer(tracerProvider.Tracer("frostdb")),
		frostdb.WithGranuleSizeBytes(flags.StorageGranuleSize),
	}

	if flags.EnablePersistence {
		frostdbOptions = append(frostdbOptions, frostdb.WithBucketStorage(objstore.NewPrefixedBucket(bucket, "blocks")))
	}

	if flags.StorageEnableWAL {
		frostdbOptions = append(frostdbOptions, frostdb.WithWAL(), frostdb.WithStoragePath(flags.StoragePath))
	}

	col, err := frostdb.New(frostdbOptions...)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize storage", "err", err)
		return err
	}

	if err := col.ReplayWALs(context.Background()); err != nil {
		level.Error(logger).Log("msg", "failed to replay WAL", "err", err)
		return err
	}

	colDB, err := col.DB(ctx, "parca")
	if err != nil {
		level.Error(logger).Log("msg", "failed to load database", "err", err)
		return err
	}

	schema, err := parcacol.Schema()
	if err != nil {
		level.Error(logger).Log("msg", "failed to get schema", "err", err)
		return err
	}

	table, err := colDB.Table("stacktraces", frostdb.NewTableConfig(schema))
	if err != nil {
		level.Error(logger).Log("msg", "create table", "err", err)
		return err
	}

	s := profilestore.NewProfileColumnStore(
		logger,
		tracerProvider.Tracer("profilestore"),
		metastore,
		table,
		schema,
		flags.StorageDebugValueLog,
	)
	conn, err := grpc.Dial(flags.ProfileShareServer, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to ProfileShareServer: %s, %w", flags.ProfileShareServer, err)
	}
	q := queryservice.NewColumnQueryAPI(
		logger,
		tracerProvider.Tracer("query-service"),
		sharepb.NewShareClient(conn),
		parcacol.NewQuerier(
			tracerProvider.Tracer("querier"),
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
				query.WithTracer(tracerProvider.Tracer("query-engine")),
			),
			"stacktraces",
			metastore,
		),
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

		debugInfodClient, err = debuginfo.NewDebugInfodClientWithObjectStorageCache(
			logger,
			objstore.NewPrefixedBucket(bucket, "debuginfod-cache"),
			httpDebugInfoClient,
		)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize debuginfod client cache", "err", err)
			return err
		}
	}

	dbgInfoMetadata := debuginfo.NewObjectStoreMetadata(logger, bucket)
	dbgInfo, err := debuginfo.NewStore(
		tracerProvider.Tracer("debuginfo"),
		logger,
		flags.DebuginfoCacheDir,
		dbgInfoMetadata,
		objstore.NewPrefixedBucket(bucket, "debuginfo"),
		debugInfodClient,
	)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize debug info store", "err", err)
		return err
	}

	reloaders := []config.ComponentReloader{
		{
			Name: "scrape_sd",
			Reloader: func(cfg *config.Config) error {
				return discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs))
			},
		},
		{
			Name: "scrape",
			Reloader: func(cfg *config.Config) error {
				return m.ApplyConfig(cfg.ScrapeConfigs)
			},
		},
	}

	cfgReloader, err := config.NewConfigReloader(logger, reg, flags.ConfigPath, reloaders)
	if err != nil {
		level.Error(logger).Log("msg", "failed to instantiate config reloader", "err", err)
		return err
	}

	var gr run.Group
	gr.Add(run.SignalHandler(ctx, os.Interrupt, syscall.SIGINT, syscall.SIGTERM))
	{
		s := symbolizer.New(
			logger,
			reg,
			metastore,
			dbgInfo,
			sym,
			flags.DebuginfoCacheDir,
			flags.DebuginfoCacheDir,
			0,
		)
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
	gr.Add(
		func() error {
			return cfgReloader.Run(ctx)
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "config file reloader exiting")
			cancel()
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

			// Close the columnstore after the parcaserver has shutdown to ensure no more writes occur against it.
			if err := col.Close(); err != nil {
				level.Error(logger).Log("msg", "error closing columnstore", "err", err)
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
	cfg *config.Config,
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
		b, err := os.ReadFile(flags.BearerTokenFile)
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

	reloaders := []config.ComponentReloader{
		{
			Name: "scrape_sd",
			Reloader: func(cfg *config.Config) error {
				return discoveryManager.ApplyConfig(getDiscoveryConfigs(cfg.ScrapeConfigs))
			},
		},
		{
			Name: "scrape",
			Reloader: func(cfg *config.Config) error {
				return m.ApplyConfig(cfg.ScrapeConfigs)
			},
		},
	}

	cfgReloader, err := config.NewConfigReloader(logger, reg, flags.ConfigPath, reloaders)
	if err != nil {
		level.Error(logger).Log("msg", "failed to instantiate config reloader", "err", err)
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
	gr.Add(
		func() error {
			return cfgReloader.Run(ctx)
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "config file reloader exiting")
			cancel()
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
