package cmd

import (
	"context"

	"github.com/parca-dev/parca/pkg/api"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/spf13/cobra"
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
		return server.ListenAndServe(context.Background(), port, &api.API{})
	},
}
