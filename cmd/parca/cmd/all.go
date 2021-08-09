package cmd

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"

	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/server"
	debuginfopb "github.com/parca-dev/parca/proto/debuginfo"
	profilestorepb "github.com/parca-dev/parca/proto/profilestore"
	querypb "github.com/parca-dev/parca/proto/query"
	"github.com/parca-dev/parca/storage"
)

type Config struct {
	DebugInfo *debuginfo.Config `yaml:"debug_info"`
}

var port string

func init() {
	allCmd.Flags().StringVarP(&port, "port", "p", ":9090", "port string for server")
	rootCmd.AddCommand(allCmd)
}

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Run all parca services from a single binary",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverStr := figure.NewColorFigure("Parca", "roman", "cyan", true)
		serverStr.Print()

		logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

		cfgContent, err := ioutil.ReadFile(configPath)
		if err != nil {
			level.Error(logger).Log("msg", "failed to read config", "path", configPath)
			os.Exit(1)
		}

		cfg := Config{}
		if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
			level.Error(logger).Log("msg", "failed to parse config", "err", err, "path", configPath)
			os.Exit(1)
		}

		d, err := debuginfo.NewStore(logger, cfg.DebugInfo)
		if err != nil {
			level.Error(logger).Log("msg", "failed to initialize debug info store", "err", err)
			os.Exit(1)
		}

		db := storage.OpenDB()
		metaStore := storage.NewInMemoryProfileMetaStore()
		s := profilestore.NewProfileStore(logger, db, metaStore)
		q := query.New(db, metaStore)
		return server.ListenAndServe(
			context.Background(),
			logger,
			port,
			server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
				debuginfopb.RegisterDebugInfoServer(srv, d)
				profilestorepb.RegisterProfileStoreServer(srv, s)
				querypb.RegisterQueryServer(srv, q)

				if err := debuginfopb.RegisterDebugInfoHandlerFromEndpoint(ctx, mux, port, opts); err != nil {
					return err
				}

				if err := profilestorepb.RegisterProfileStoreHandlerFromEndpoint(ctx, mux, port, opts); err != nil {
					return err
				}

				if err := querypb.RegisterQueryHandlerFromEndpoint(ctx, mux, port, opts); err != nil {
					return err
				}

				return nil
			}))
	},
}
