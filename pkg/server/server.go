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

package server

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"net/http/pprof"
	"strings"
	"text/template"
	"time"

	"github.com/felixge/fgprof"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"

	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/prober"
	"github.com/parca-dev/parca/ui"
)

type Registerable interface {
	Register(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type RegisterableFunc func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func (f RegisterableFunc) Register(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return f(ctx, srv, mux, endpoint, opts)
}

// Server is a wrapper around the http.Server.
type Server struct {
	http.Server
	grpcProbe *prober.GRPCProbe
	reg       *prometheus.Registry
	version   string
}

func NewServer(reg *prometheus.Registry, version string) *Server {
	return &Server{
		grpcProbe: prober.NewGRPC(),
		reg:       reg,
		version:   version,
	}
}

// ListenAndServe starts the http grpc gateway server.
func (s *Server) ListenAndServe(ctx context.Context, logger log.Logger, addr string, allowedCORSOrigins []string, pathPrefix string, registerables ...Registerable) error {
	level.Info(logger).Log("msg", "starting server", "addr", addr)

	logOpts := []grpc_logging.Option{
		grpc_logging.WithLogOnEvents(grpc_logging.FinishCall),
		grpc_logging.WithLevels(DefaultCodeToLevelGRPC),
	}

	met := grpc_prometheus.NewServerMetrics(
		grpc_prometheus.WithServerHandlingTimeHistogram(
			grpc_prometheus.WithHistogramOpts(&prometheus.HistogramOpts{
				NativeHistogramBucketFactor: 1.1,
				Buckets:                     nil,
			}),
		),
	)

	// Start grpc server with API server registered
	srv := grpc.NewServer(
		// It is increased to 32MB to account for large protobuf messages (debug information uploads and downloads).
		grpc.MaxSendMsgSize(debuginfo.MaxMsgSize),
		grpc.MaxRecvMsgSize(debuginfo.MaxMsgSize),
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				otelgrpc.StreamServerInterceptor(),
				met.StreamServerInterceptor(),
				grpc_logging.StreamServerInterceptor(InterceptorLogger(logger), logOpts...),
			)),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				otelgrpc.UnaryServerInterceptor(),
				met.UnaryServerInterceptor(),
				grpc_logging.UnaryServerInterceptor(InterceptorLogger(logger), logOpts...),
			),
		),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	grpcWebMux := runtime.NewServeMux()
	for _, r := range registerables {
		if err := r.Register(ctx, srv, grpcWebMux, addr, opts); err != nil {
			return err
		}
	}
	reflection.Register(srv)
	grpc_health.RegisterHealthServer(srv, s.grpcProbe.HealthServer())

	internalMux := chi.NewRouter()

	internalMux.Route(pathPrefix+"/", func(r chi.Router) {
		r.Mount("/api", grpcWebMux)

		r.Handle("/metrics", promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{}))

		// Add the pprof handler to profile Parca
		r.Handle("/debug/pprof/*", http.StripPrefix(pathPrefix, http.HandlerFunc(pprof.Index)))
		r.Handle("/debug/pprof/fgprof", fgprof.Handler())
		r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		r.HandleFunc("/debug/pprof/profile", pprof.Profile)
		r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		r.HandleFunc("/debug/pprof/trace", pprof.Trace)
	})

	// Strip the subpath
	uiFS, err := fs.Sub(ui.FS, "packages/app/web/build")
	if err != nil {
		return fmt.Errorf("failed to initialize UI filesystem: %w", err)
	}

	uiHandler, err := s.uiHandler(uiFS, pathPrefix)
	if err != nil {
		return fmt.Errorf("failed to walk ui filesystem: %w", err)
	}

	s.Server = http.Server{
		Addr: addr,
		Handler: grpcHandlerFunc(
			srv,
			fallbackNotFound(internalMux, uiHandler),
			allowedCORSOrigins,
		),
		ReadTimeout:  5 * time.Second, // TODO make config option
		WriteTimeout: time.Minute,     // TODO make config option
	}

	met.InitializeMetrics(srv)
	s.reg.MustRegister(met)

	s.reg.MustRegister(
		collectors.NewBuildInfoCollector(),
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	s.grpcProbe.Ready()
	s.grpcProbe.Healthy()
	return s.Server.ListenAndServe()
}

// Shutdown the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.grpcProbe.NotReady(nil)
	return s.Server.Shutdown(ctx)
}

// uiHandler initialize a http.ServerMux with the UI files.
//
// There is currently no way to go between `http.FileServer(http.FS(uiFS))` and execute
// templates. Taking an FS registering paths and executing templates seems to be the best option
// for now.
func (s *Server) uiHandler(uiFS fs.FS, pathPrefix string) (*http.ServeMux, error) {
	uiHandler := http.ServeMux{}

	err := fs.WalkDir(uiFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || strings.HasSuffix(d.Name(), ".map") {
			return nil
		}

		b, err := fs.ReadFile(uiFS, path)
		if err != nil {
			return fmt.Errorf("failed to read ui file %s: %w", path, err)
		}

		if strings.HasSuffix(path, ".html") {
			tmpl, err := template.New(path).Parse(strings.Replace(string(b), "/PATH_PREFIX_VAR", "{{.PathPrefix}}", -1))
			if err != nil {
				return fmt.Errorf("failed to parse ui file %s: %w", path, err)
			}

			var outputBuffer bytes.Buffer

			err = tmpl.Execute(&outputBuffer, struct {
				Version    string
				PathPrefix string
			}{
				s.version,
				pathPrefix,
			})

			if err != nil {
				return fmt.Errorf("failed to execute ui file %s: %w", path, err)
			}

			b = outputBuffer.Bytes()
		}

		fi, err := d.Info()
		if err != nil {
			return fmt.Errorf("failed to receive file info %s: %w", path, err)
		}

		paths := []string{fmt.Sprintf("/%s", path)}

		if paths[0] == "/index.html" {
			paths = append(paths, "/", "/*")
		}

		if paths[0] == "/targets/index.html" {
			paths = append(paths, "/targets")
		}

		for _, path := range paths {
			uiHandler.HandleFunc(pathPrefix+path, func(w http.ResponseWriter, r *http.Request) {
				http.ServeContent(w, r, d.Name(), fi.ModTime(), bytes.NewReader(b))
			})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return &uiHandler, nil
}

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler, allowedCORSOrigins []string) http.Handler {
	allowAll := false
	if len(allowedCORSOrigins) == 1 && allowedCORSOrigins[0] == "*" {
		allowAll = true
	}
	origins := map[string]struct{}{}
	for _, o := range allowedCORSOrigins {
		origins[o] = struct{}{}
	}
	wrappedGrpc := grpcweb.WrapServer(grpcServer,
		grpcweb.WithAllowNonRootResource(true),
		grpcweb.WithOriginFunc(func(origin string) bool {
			_, found := origins[origin]
			return found || allowAll
		}))

	corsMiddleware := cors.New(cors.Options{
		AllowOriginFunc: func(r *http.Request, origin string) bool {
			_, found := origins[origin]
			return found || allowAll
		},
		AllowedHeaders: []string{"*"},
		AllowedMethods: []string{
			http.MethodHead,
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
		},
		AllowCredentials: true,
	})

	return corsMiddleware.Handler(h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.Header.Get("Content-Type"), "application/grpc") {
			wrappedGrpc.ServeHTTP(w, r)
			return
		}
		otherHandler.ServeHTTP(w, r)
	}), &http2.Server{}))
}

// InterceptorLogger adapts go-kit logger to interceptor logger.
func InterceptorLogger(l log.Logger) grpc_logging.Logger {
	return grpc_logging.LoggerFunc(func(_ context.Context, lvl grpc_logging.Level, msg string, fields ...any) {
		largs := append([]any{"msg", msg}, fields...)
		switch lvl {
		case grpc_logging.LevelDebug:
			_ = level.Debug(l).Log(largs...)
		case grpc_logging.LevelInfo:
			_ = level.Info(l).Log(largs...)
		case grpc_logging.LevelWarn:
			_ = level.Warn(l).Log(largs...)
		case grpc_logging.LevelError:
			_ = level.Error(l).Log(largs...)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}

// DefaultCodeToLevelGRPC is the helper mapper that maps gRPC Response codes to log levels.
func DefaultCodeToLevelGRPC(c codes.Code) grpc_logging.Level {
	switch c {
	case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
		return grpc_logging.LevelError
	default:
		return grpc_logging.LevelDebug
	}
}
