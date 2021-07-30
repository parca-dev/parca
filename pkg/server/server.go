package server

import (
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/parca-dev/api/proto/api"
	"golang.org/x/net/context"
)

// ListenAndServe starts the http grpc gateway server
func ListenAndServe(ctx context.Context, port string, s api.APIServer) error {
	mux := runtime.NewServeMux()
	api.RegisterAPIHandlerServer(ctx, mux, s)
	return http.ListenAndServe(port, mux)
}
