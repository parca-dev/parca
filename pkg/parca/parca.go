// Copyright 2022-2023 The Parca Authors
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
	"net/http"
	"os"
	"path/filepath"
	goruntime "runtime"
	"runtime/pprof"
	"strings"
	"syscall"
	"time"

	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/dgraph-io/badger/v4"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/oklog/run"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	promconfig "github.com/prometheus/common/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	objstoretracing "github.com/thanos-io/objstore/tracing/opentelemetry"
	tracing "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v3"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	scrapepb "github.com/parca-dev/parca/gen/proto/go/parca/scrape/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/config"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/profilestore"
	queryservice "github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/scrape"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/parca-dev/parca/pkg/signedrequests"
	"github.com/parca-dev/parca/pkg/symbolizer"
	"github.com/parca-dev/parca/pkg/tracer"
)

const (
	symbolizationInterval = 10 * time.Second
	flagModeScraperOnly   = "scraper-only"
	metaStoreBadger       = "badger"
)

type Flags struct {
	ConfigPath       string        `default:"parca.yaml" help:"Path to config file."`
	Mode             string        `default:"all" enum:"all,scraper-only" help:"Scraper only runs a scraper that sends to a remote gRPC endpoint. All runs all components."`
	HTTPAddress      string        `default:":7070" help:"Address to bind HTTP server to."`
	HTTPReadTimeout  time.Duration `default:"5s" help:"Timeout duration for HTTP server to read request body. Defaults to 5s"`
	HTTPWriteTimeout time.Duration `default:"1m" help:"Timeout duration for HTTP server to write response body. Defaults to 1m"`
	Port             string        `default:"" help:"(DEPRECATED) Use http-address instead."`

	Logs FlagsLogs `embed:"" prefix:"log-"`
	OTLP FlagsOTLP `embed:"" prefix:"otlp-"`

	CORSAllowedOrigins []string `help:"Allowed CORS origins."`
	Version            bool     `help:"Show application version."`
	PathPrefix         string   `default:"" help:"Path prefix for the UI"`

	MutexProfileFraction int `default:"0" help:"Fraction of mutex profile samples to collect."`
	BlockProfileRate     int `default:"0" help:"Sample rate for block profile."`

	EnablePersistence bool `default:"false" help:"Turn on persistent storage for the metastore and profile storage."`

	Storage FlagsStorage `embed:"" prefix:"storage-"`

	Symbolizer FlagsSymbolizer `embed:"" prefix:"symbolizer-"`

	Debuginfo  FlagsDebuginfo  `embed:"" prefix:"debuginfo-"`
	Debuginfod FlagsDebuginfod `embed:"" prefix:"debuginfod-"`

	Metastore string `default:"badger" help:"Which metastore implementation to use" enum:"badger"`

	ProfileShareServer string `default:"api.pprof.me:443" help:"gRPC address to send share profile requests to."`

	StoreAddress       string            `kong:"help='gRPC address to send profiles and symbols to.'"`
	BearerToken        string            `kong:"help='Bearer token to authenticate with store.'"`
	BearerTokenFile    string            `kong:"help='File to read bearer token from to authenticate with store.'"`
	Insecure           bool              `kong:"help='Send gRPC requests via plaintext instead of TLS.'"`
	InsecureSkipVerify bool              `kong:"help='Skip TLS certificate verification.'"`
	ExternalLabel      map[string]string `kong:"help='Label(s) to attach to all profiles in scraper-only mode.'"`

	ExperimentalArrow bool `default:"false" help:"EXPERIMENTAL: Enables Arrow ingestion, this will reduce CPU usage but will increase memory usage."`

	Hidden FlagsHidden `embed:"" prefix:""`
}

type FlagsLogs struct {
	Level  string `enum:"error,warn,info,debug" default:"info" help:"Log level."`
	Format string `enum:"logfmt,json" default:"logfmt" help:"Configure if structured logging as JSON or as logfmt"`
}

// FlagsOTLP provides OTLP configuration flags.
type FlagsOTLP struct {
	Address  string `help:"The endpoint to send OTLP traces to."`
	Exporter string `default:"grpc"                              enum:"grpc,http,stdout" help:"The OTLP exporter to use."`
}

type FlagsStorage struct {
	GranuleSize         int64  `default:"26265625" help:"Granule size in bytes for storage."`
	ActiveMemory        int64  `default:"536870912" help:"Amount of memory to use for active storage. Defaults to 512MB."`
	Path                string `default:"data" help:"Path to storage directory."`
	EnableWAL           bool   `default:"false" help:"Enables write ahead log for profile storage."`
	SnapshotTriggerSize int64  `default:"134217728" help:"Number of bytes to trigger a snapshot. Defaults to 1/4 of active memory. This is only used if enable-wal is set."`
	RowGroupSize        int    `default:"8192" help:"Number of rows in each row group during compaction and persistence. Setting to <= 0 results in a single row group per file."`
}

type FlagsSymbolizer struct {
	DemangleMode  string `default:"simple" help:"Mode to demangle C++ symbols. Default mode is simplified: no parameters, no templates, no return type" enum:"simple,full,none,templates"`
	NumberOfTries int    `default:"3" help:"Number of tries to attempt to symbolize an unsybolized location"`
}

// FlagsDebuginfo configures the Parca Debuginfo client.
type FlagsDebuginfo struct {
	CacheDir          string        `default:"/tmp" help:"Path to directory where debuginfo is cached."`
	UploadMaxSize     int64         `default:"1000000000" help:"Maximum size of debuginfo upload in bytes."`
	UploadMaxDuration time.Duration `default:"15m" help:"Maximum duration of debuginfo upload."`
	UploadsSignedURL  bool          `default:"false" help:"Whether to use signed URLs for debuginfo uploads."`
}

// FlagsDebuginfod configures the Parca Debuginfo daemon / server.
type FlagsDebuginfod struct {
	UpstreamServers    []string      `default:"debuginfod.elfutils.org" help:"Upstream debuginfod servers. Defaults to debuginfod.elfutils.org. It is an ordered list of servers to try. Learn more at https://sourceware.org/elfutils/Debuginfod.html"`
	HTTPRequestTimeout time.Duration `default:"5m" help:"Timeout duration for HTTP request to upstream debuginfod server. Defaults to 5m"`
}

// FlagsHidden contains hidden flags intended only for debugging.
type FlagsHidden struct {
	DebugNormalizeAddresses bool `kong:"help='Normalize sampled addresses.',default='true',hidden=''"`
}

// Run the parca server.
func Run(ctx context.Context, logger log.Logger, reg *prometheus.Registry, flags *Flags, version string) error {
	goruntime.SetBlockProfileRate(flags.BlockProfileRate)
	goruntime.SetMutexProfileFraction(flags.MutexProfileFraction)

	// Initialize tracing.
	var (
		exporter       tracer.Exporter
		tracerProvider = trace.NewNoopTracerProvider()
	)
	if flags.OTLP.Address != "" {
		var err error

		exporter, err = tracer.NewExporter(flags.OTLP.Exporter, flags.OTLP.Address)
		if err != nil {
			level.Error(logger).Log("msg", "failed to create tracing exporter", "err", err)
		}
		// NewExporter always returns a non-nil exporter and non-nil error.
		tracerProvider, err = tracer.NewProvider(ctx, version, exporter)
		if err != nil {
			level.Error(logger).Log("msg", "failed to create tracing provider", "err", err)
		}
	}

	// Enable arrow ingestion
	parcacol.ExperimentalArrow = flags.ExperimentalArrow

	if flags.Port != "" {
		level.Warn(logger).Log("msg", "flag --port is deprecated, use --http-address instead")
		flags.HTTPAddress = flags.Port
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

	bucket, err := client.NewBucket(logger, bucketCfg, "parca")
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize object storage bucket", "err", err)
		return err
	}
	bucket = objstore.WrapWithMetrics(bucket, reg, bucket.Name())
	bucket = objstoretracing.WrapWithTraces(bucket, tracerProvider.Tracer("objstore_bucket"))

	var signedRequestsClient signedrequests.Client
	if flags.Debuginfo.UploadsSignedURL {
		var err error
		signedRequestsClient, err = signedrequests.NewClient(
			context.Background(),
			cfg.ObjectStorage.Bucket,
		)

		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize signed upload client", "err", err)
			return err
		}

		defer signedRequestsClient.Close()
	}

	var mStr metastorepb.MetastoreServiceServer
	switch flags.Metastore {
	case metaStoreBadger:
		var badgerOptions badger.Options
		switch flags.EnablePersistence {
		case true:
			badgerOptions = badger.DefaultOptions(filepath.Join(flags.Storage.Path, "metastore"))
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

	mc := metastore.NewInProcessClient(mStr)

	frostdbOptions := []frostdb.Option{
		frostdb.WithActiveMemorySize(flags.Storage.ActiveMemory),
		frostdb.WithLogger(logger),
		frostdb.WithRegistry(reg),
		frostdb.WithTracer(tracerProvider.Tracer("frostdb")),
		frostdb.WithGranuleSizeBytes(flags.Storage.GranuleSize),
	}

	if flags.EnablePersistence {
		frostdbOptions = append(
			frostdbOptions,
			frostdb.WithReadWriteStorage(
				frostdb.NewDefaultObjstoreBucket(objstore.NewPrefixedBucket(bucket, "blocks")),
			),
		)
	}

	if flags.Storage.EnableWAL {
		frostdbOptions = append(
			frostdbOptions,
			frostdb.WithWAL(),
			frostdb.WithStoragePath(flags.Storage.Path),
			frostdb.WithSnapshotTriggerSize(flags.Storage.SnapshotTriggerSize),
		)
	}

	col, err := frostdb.New(frostdbOptions...)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize storage", "err", err)
		return err
	}

	colDB, err := col.DB(ctx, "parca")
	if err != nil {
		level.Error(logger).Log("msg", "failed to load database", "err", err)
		return err
	}

	def := profile.SchemaDefinition()
	table, err := colDB.Table("stacktraces",
		frostdb.NewTableConfig(
			def,
			frostdb.WithRowGroupSize(flags.Storage.RowGroupSize),
		),
	)
	if err != nil {
		level.Error(logger).Log("msg", "create table", "err", err)
		return err
	}

	schema, err := dynparquet.SchemaFromDefinition(def)
	if err != nil {
		level.Error(logger).Log("msg", "schema from definition", "err", err)
		return err
	}
	s := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracerProvider.Tracer("profilestore"),
		mc,
		table,
		schema,
		flags.Hidden.DebugNormalizeAddresses,
	)

	propagators := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
		grpc.WithChainUnaryInterceptor(
			tracing.UnaryClientInterceptor(
				tracing.WithTracerProvider(tracerProvider),
				tracing.WithPropagators(propagators),
			),
		),
		grpc.WithChainStreamInterceptor(
			tracing.StreamClientInterceptor(
				tracing.WithTracerProvider(tracerProvider),
				tracing.WithPropagators(propagators),
			),
		),
	}
	conn, err := grpc.Dial(flags.ProfileShareServer, opts...)
	if err != nil {
		return fmt.Errorf("failed to create gRPC connection to ProfileShareServer: %s, %w", flags.ProfileShareServer, err)
	}

	debuginfoBucket := objstore.NewPrefixedBucket(bucket, "debuginfo")
	q := queryservice.NewColumnQueryAPI(
		logger,
		tracerProvider.Tracer("query-service"),
		sharepb.NewShareServiceClient(conn),
		parcacol.NewQuerier(
			logger,
			tracerProvider.Tracer("querier"),
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
				query.WithTracer(tracerProvider.Tracer("query-engine")),
			),
			"stacktraces",
			parcacol.NewProfileSymbolizer(
				tracerProvider.Tracer("profile-symbolizer"),
				mc,
			),
			memory.DefaultAllocator,
		),
		memory.DefaultAllocator,
		parcacol.NewArrowToProfileConverter(
			tracerProvider.Tracer("arrow_to_profile_converter"),
			metastore.NewKeyMaker(),
		),
		queryservice.NewBucketSourceFinder(debuginfoBucket),
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

	var debuginfodClients debuginfo.DebuginfodClients = debuginfo.NopDebuginfodClients{}
	if len(flags.Debuginfod.UpstreamServers) > 0 {
		debuginfodClients = debuginfo.NewDebuginfodClients(
			reg,
			tracerProvider,
			flags.Debuginfod.UpstreamServers,
			promconfig.NewUserAgentRoundTripper(fmt.Sprintf("parca.dev/debuginfod-client/%s", version), http.DefaultTransport),
			flags.Debuginfod.HTTPRequestTimeout,
			objstore.NewPrefixedBucket(bucket, "debuginfod-cache"),
		)
	}

	prefixedSignedRequestsClient := signedrequests.NewPrefixedClient(signedRequestsClient, "debuginfo")
	debuginfoMetadata := debuginfo.NewObjectStoreMetadata(logger, debuginfoBucket)
	dbginfo, err := debuginfo.NewStore(
		tracerProvider.Tracer("debuginfo"),
		logger,
		debuginfoMetadata,
		debuginfoBucket,
		debuginfodClients,
		debuginfo.SignedUpload{
			Enabled: flags.Debuginfo.UploadsSignedURL,
			Client:  prefixedSignedRequestsClient,
		},
		flags.Debuginfo.UploadMaxDuration,
		flags.Debuginfo.UploadMaxSize,
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

	// Run group of OTL exporter.
	if exporter != nil {
		logger := log.With(logger, "group", "otlp_exporter")
		ctx, cancel := context.WithCancel(ctx)
		gr.Add(func() error {
			if err := exporter.Start(ctx); err != nil {
				return fmt.Errorf("failed to start exporter: %w", err)
			}
			<-ctx.Done()
			return nil
		}, func(error) {
			level.Debug(logger).Log("msg", "shutting down otlp exporter")
			cancel()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			if err := exporter.Shutdown(ctx); err != nil {
				level.Error(logger).Log("msg", "failed to stop exporter", "err", err)
			}
		})
	}

	{
		s := symbolizer.New(
			logger,
			reg,
			debuginfoMetadata,
			mc,
			debuginfo.NewFetcher(debuginfodClients, debuginfoBucket),
			flags.Debuginfo.CacheDir,
			0,
			symbolizer.WithDemangleMode(flags.Symbolizer.DemangleMode),
			symbolizer.WithAttemptThreshold(flags.Symbolizer.NumberOfTries),
		)
		ctx, cancel := context.WithCancel(ctx)
		gr.Add(
			func() error {
				var err error

				pprof.Do(ctx, pprof.Labels("parca_component", "symbolizer"), func(ctx context.Context) {
					err = s.Run(ctx, symbolizationInterval)
				})

				return err
			},
			func(_ error) {
				level.Debug(logger).Log("msg", "symbolizer server shutting down")
				cancel()
			})
	}
	gr.Add(
		func() error {
			var err error

			pprof.Do(ctx, pprof.Labels("parca_component", "discovery"), func(_ context.Context) {
				err = discoveryManager.Run()
			})

			return err
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "discovery manager exiting")
			cancel()
		},
	)
	gr.Add(
		func() error {
			var err error

			pprof.Do(ctx, pprof.Labels("parca_component", "scraper"), func(_ context.Context) {
				err = m.Run(discoveryManager.SyncCh())
			})

			return err
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "scrape manager exiting")
			m.Stop()
		},
	)
	gr.Add(
		func() error {
			var err error

			pprof.Do(ctx, pprof.Labels("parca_component", "config_reloader"), func(ctx context.Context) {
				err = cfgReloader.Run(ctx)
			})

			return err
		},
		func(_ error) {
			level.Debug(logger).Log("msg", "config file reloader exiting")
			cancel()
		},
	)
	parcaserver := server.NewServer(reg, version)
	gr.Add(
		func() error {
			var err error

			pprof.Do(ctx, pprof.Labels("parca_component", "http_server"), func(ctx context.Context) {
				err = parcaserver.ListenAndServe(
					ctx,
					logger,
					flags.HTTPAddress,
					flags.HTTPReadTimeout,
					flags.HTTPWriteTimeout,
					flags.CORSAllowedOrigins,
					flags.PathPrefix,
					server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
						debuginfopb.RegisterDebuginfoServiceServer(srv, dbginfo)
						profilestorepb.RegisterProfileStoreServiceServer(srv, s)
						profilestorepb.RegisterAgentsServiceServer(srv, s)
						querypb.RegisterQueryServiceServer(srv, q)
						scrapepb.RegisterScrapeServiceServer(srv, m)

						if err := debuginfopb.RegisterDebuginfoServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
							return err
						}

						if err := profilestorepb.RegisterProfileStoreServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
							return err
						}

						if err := profilestorepb.RegisterAgentsServiceHandlerFromEndpoint(ctx, mux, endpoint, opts); err != nil {
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
			})

			return err
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
			level.Info(logger).Log("msg", "terminating", "reason", err)
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

	metrics := grpc_prometheus.NewClientMetrics(
		grpc_prometheus.WithClientHandlingTimeHistogram(
			grpc_prometheus.WithHistogramOpts(&prometheus.HistogramOpts{
				NativeHistogramBucketFactor: 1.1,
			}),
		),
	)
	reg.MustRegister(metrics)

	propagators := propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{})

	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(
			metrics.UnaryClientInterceptor(),
			tracing.UnaryClientInterceptor(
				tracing.WithTracerProvider(tracer),
				tracing.WithPropagators(propagators),
			),
		),
		grpc.WithChainStreamInterceptor(
			metrics.StreamClientInterceptor(),
			tracing.StreamClientInterceptor(
				tracing.WithTracerProvider(tracer),
				tracing.WithPropagators(propagators),
			),
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

	externalLabels := labels.FromMap(flags.ExternalLabel)

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

	{
		parcaserver := server.NewServer(reg, version)
		serveCtx, cancelServe := context.WithCancel(ctx)
		gr.Add(
			func() error {
				return parcaserver.ListenAndServe(
					serveCtx,
					logger,
					flags.HTTPAddress,
					flags.HTTPReadTimeout,
					flags.HTTPWriteTimeout,
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
				level.Debug(logger).Log("msg", "server shutting down")

				// Create a new context for the server shutdown with a deadline.
				shutdownCtx, cancelShutdown := context.WithTimeout(ctx, 30*time.Second) // TODO make this a graceful shutdown config setting
				defer cancelShutdown()

				err := parcaserver.Shutdown(shutdownCtx)
				// Cancels ListenAndServe after shutting down the connections.
				// If the timeout was reached during shutdown, this should
				// cancel any remaining live application code.
				cancelServe()
				if err != nil && !errors.Is(err, context.Canceled) {
					level.Error(logger).Log("msg", "error shutting down server", "err", err)
				}
			},
		)
	}

	level.Info(logger).Log("msg", "running Parca in scrape mode", "version", version)
	if err := gr.Run(); err != nil {
		if _, ok := err.(run.SignalError); ok {
			level.Info(logger).Log("msg", "terminating", "reason", err)
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
