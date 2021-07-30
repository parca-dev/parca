package server

import (
	"net/http"
	"os"
	"strings"

	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_kit "github.com/grpc-ecosystem/go-grpc-middleware/logging/kit"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/parca-dev/api/proto/api"
	"golang.org/x/net/context"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// ListenAndServe starts the http grpc gateway server
func ListenAndServe(ctx context.Context, port string, s api.APIServer) error {
	serverStr := figure.NewColorFigure("Parca API", "roman", "cyan", true)
	serverStr.Print()

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
	level.Info(logger).Log("msg", "Serving API", "port", port)

	// Start grpc server with API server registered
	srv := grpc.NewServer(
		grpc.StreamInterceptor(
			grpc_middleware.ChainStreamServer(
				grpc_kit.StreamServerInterceptor(logger),
			)),
		grpc.UnaryInterceptor(
			grpc_middleware.ChainUnaryServer(
				grpc_kit.UnaryServerInterceptor(logger),
			),
		),
	)
	api.RegisterAPIServer(srv, s)
	reflection.Register(srv)

	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithInsecure()}
	if err := api.RegisterAPIHandlerFromEndpoint(ctx, mux, port, opts); err != nil {
		return err
	}

	return http.ListenAndServe(port, grpcHandlerFunc(srv, mux))
}

func grpcHandlerFunc(grpcServer *grpc.Server, otherHandler http.Handler) http.Handler {
	return h2c.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ProtoMajor == 2 && strings.Contains(r.Header.Get("Content-Type"), "application/grpc") {
			grpcServer.ServeHTTP(w, r)
		} else {
			otherHandler.ServeHTTP(w, r)
		}
	}), &http2.Server{})
}
