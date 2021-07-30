package server

import (
	"net/http"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/parca-dev/api/proto/api"
	"golang.org/x/net/context"
)

// ListenAndServe starts the http grpc gateway server
func ListenAndServe(ctx context.Context, port string, s api.APIServer) error {
	serverStr := figure.NewColorFigure("Parca API", "roman", "cyan", true)
	serverStr.Print()

	logger := log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
	level.Info(logger).Log("msg", "Serving API", "port", port)

	mux := runtime.NewServeMux()
	api.RegisterAPIHandlerServer(ctx, mux, s)
	return http.ListenAndServe(port, mux)
}
