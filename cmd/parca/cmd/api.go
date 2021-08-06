package cmd

import (
	"context"
	"os"

	"github.com/go-kit/log"
	"github.com/parca-dev/parca/pkg/api"
	"github.com/parca-dev/parca/pkg/server"
	apipb "github.com/parca-dev/parca/proto/api"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var port string

func init() {
	apiCmd.Flags().StringVarP(&port, "port", "p", ":9090", "port string for server")
	rootCmd.AddCommand(apiCmd)
}

var apiCmd = &cobra.Command{
	Use:   "api",
	Short: "Run the API to query profiles from a storage",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
		return server.ListenAndServe(context.Background(), logger, port, server.RegisterableFunc(func(srv *grpc.Server) {
			s := &api.API{}
			apipb.RegisterAPIServer(srv, s)
		}))
	},
}
