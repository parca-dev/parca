package cmd

import (
	"context"
	"os"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
	"github.com/thanos-io/thanos/pkg/objstore/filesystem"
	"google.golang.org/grpc"

	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/server"
	debuginfopb "github.com/parca-dev/parca/proto/debuginfo"
	profilestorepb "github.com/parca-dev/parca/proto/profilestore"
	"github.com/parca-dev/parca/storage"
)

func init() {
	allCmd.Flags().StringVarP(&port, "port", "p", ":9090", "port string for server")
	rootCmd.AddCommand(allCmd)
}

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all parca services from a single binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))
		db := storage.OpenDB()
		metaStore := storage.NewInMemoryProfileMetaStore()
		bucket, err := filesystem.NewBucket("/tmp")
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize filesystem bucket")
		}

		d := debuginfo.NewStore(logger, bucket)
		s := profilestore.NewProfileStore(db, metaStore)
		return server.ListenAndServe(context.Background(), logger, port, server.RegisterableFunc(func(srv *grpc.Server) {
			debuginfopb.RegisterDebugInfoServer(srv, d)
			profilestorepb.RegisterProfileStoreServer(srv, s)
		}))
	},
}
