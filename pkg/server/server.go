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

	"github.com/go-chi/cors"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	"github.com/grpc-ecosystem/go-grpc-middleware/providers/kit/v2"
	grpc_logging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/thanos-io/thanos/pkg/prober"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	grpc_health "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/parca-dev/parca/ui"
)

type Registerable interface {
	Register(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error
}

type RegisterableFunc func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error

func (f RegisterableFunc) Register(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
	return f(ctx, srv, mux, endpoint, opts)
}

// MapAllowedLevels allows to map a given level to a list of allowed level.
// Convention taken from go-kit/level v0.10.0 https://godoc.org/github.com/go-kit/kit/log/level#AllowAll.
var MapAllowedLevels = map[string][]string{
	"DEBUG": {"INFO", "DEBUG", "WARN", "ERROR"},
	"ERROR": {"ERROR"},
	"INFO":  {"INFO", "WARN", "ERROR"},
	"WARN":  {"WARN", "ERROR"},
}

// Server is a wrapper around the http.Server
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

// ListenAndServe starts the http grpc gateway server
func (s *Server) ListenAndServe(ctx context.Context, logger log.Logger, port string, allowedCORSOrigins []string, registerables ...Registerable) error {
	level.Info(logger).Log("msg", "starting server", "addr", port)
	logLevel := "ERROR"

	logOpts := []grpc_logging.Option{
		grpc_logging.WithDecider(func(_ string, err error) grpc_logging.Decision {

			runtimeLevel := grpc_logging.DefaultServerCodeToLevel(status.Code(err))
			for _, lvl := range MapAllowedLevels[logLevel] {
				if string(runtimeLevel) == strings.ToLower(lvl) {
					return grpc_logging.LogFinishCall
				}
			}
			return grpc_logging.NoLogCall
		}),
		grpc_logging.WithLevels(DefaultCodeToLevelGRPC),
	}

	met := grpc_prometheus.NewServerMetrics()
	met.EnableHandlingTimeHistogram(
		grpc_prometheus.WithHistogramBuckets([]float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120}),
	)

	// Start grpc server with API server registered
	srv := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				otelgrpc.StreamServerInterceptor(),
				met.StreamServerInterceptor(),
				grpc_logging.StreamServerInterceptor(kit.InterceptorLogger(logger), logOpts...),
			)),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				otelgrpc.UnaryServerInterceptor(),
				met.UnaryServerInterceptor(),
				grpc_logging.UnaryServerInterceptor(kit.InterceptorLogger(logger), logOpts...),
			),
		),
	)

	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	mux := runtime.NewServeMux()
	for _, r := range registerables {
		if err := r.Register(ctx, srv, mux, port, opts); err != nil {
			return err
		}
	}
	reflection.Register(srv)
	grpc_health.RegisterHealthServer(srv, s.grpcProbe.HealthServer())

	err := mux.HandlePath(http.MethodGet, "/metrics", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		promhttp.HandlerFor(s.reg, promhttp.HandlerOpts{}).ServeHTTP(w, r)
	})
	if err != nil {
		return fmt.Errorf("failed to register metrics handler: %w", err)
	}

	// Add the pprof handler to profile Parca
	err = mux.HandlePath(http.MethodGet, "/debug/pprof/*", func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		if r.URL.Path == "/debug/pprof/profile" {
			pprof.Profile(w, r)
			return
		}
		pprof.Index(w, r)
	})
	if err != nil {
		return fmt.Errorf("failed to register pprof handlers: %w", err)
	}

	// Strip the subpath
	uiFS, err := fs.Sub(ui.FS, "packages/app/web/dist")
	if err != nil {
		return fmt.Errorf("failed to initialize UI filesystem: %w", err)
	}

	uiHandler, err := s.uiHandler(uiFS)

	if err != nil {
		return fmt.Errorf("failed to walk ui filesystem: %w", err)
	}

	s.Server = http.Server{
		Addr: port,
		Handler: grpcHandlerFunc(
			srv,
			fallbackNotFound(mux, uiHandler),
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

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.grpcProbe.NotReady(nil)
	return s.Server.Shutdown(ctx)
}

// uiHandler initialize a http.ServerMux with the UI files.
//
// There is currently no way to go between `http.FileServer(http.FS(uiFS))` and execute
// templates. Taking an FS registering paths and executing templates seems to be the best option
// for now.
func (s *Server) uiHandler(uiFS fs.FS) (*http.ServeMux, error) {
	uiHandler := http.ServeMux{}

	err := fs.WalkDir(uiFS, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(uiFS, path)

		if err != nil {
			return fmt.Errorf("failed to read ui file %s: %w", path, err)
		}

		tmpl, err := template.New(path).Parse(string(b))

		if err != nil {
			return fmt.Errorf("failed to parse ui file %s: %w", path, err)
		}

		var outputBuffer bytes.Buffer

		err = tmpl.Execute(&outputBuffer, struct {
			Version string
		}{
			s.version,
		})

		if err != nil {
			return fmt.Errorf("failed to execute ui file %s: %w", path, err)
		}

		fi, err := d.Info()

		if err != nil {
			return fmt.Errorf("failed to receive file info %s: %w", path, err)
		}

		outputBytes := outputBuffer.Bytes()

		paths := []string{fmt.Sprintf("/%s", path)}

		if paths[0] == "/index.html" {
			paths = append(paths, "/")
		}

		for _, path := range paths {
			uiHandler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				http.ServeContent(w, r, d.Name(), fi.ModTime(), bytes.NewReader(outputBytes))
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
	wrappedGrpc := grpcweb.WrapServer(grpcServer, grpcweb.WithOriginFunc(func(origin string) bool {
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
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			if wrappedGrpc.IsGrpcWebRequest(r) {
				wrappedGrpc.ServeHTTP(w, r)
				return
			}

			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{}))
}

// DefaultCodeToLevelGRPC is the helper mapper that maps gRPC Response codes to log levels.
func DefaultCodeToLevelGRPC(c codes.Code) grpc_logging.Level {
	switch c {
	case codes.Unknown, codes.Unimplemented, codes.Internal, codes.DataLoss:
		return grpc_logging.ERROR
	default:
		return grpc_logging.DEBUG
	}
}
