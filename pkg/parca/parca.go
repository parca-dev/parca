package parca

import (
	"context"
	"io/ioutil"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/server"
	"github.com/parca-dev/parca/pkg/storage"
	debuginfopb "github.com/parca-dev/parca/proto/gen/go/debuginfo"
	profilestorepb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	querypb "github.com/parca-dev/parca/proto/gen/go/query"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

// Config is the configuration for debug info storage
type Config struct {
	DebugInfo *debuginfo.Config `yaml:"debug_info"`
}

// Run the parca server
func Run(ctx context.Context, logger log.Logger, configPath, port string) error {
	cfgContent, err := ioutil.ReadFile(configPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to read config", "path", configPath)
		return err
	}

	cfg := Config{}
	if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
		level.Error(logger).Log("msg", "failed to parse config", "err", err, "path", configPath)
		return err
	}

	d, err := debuginfo.NewStore(logger, cfg.DebugInfo)
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize debug info store", "err", err)
		return err
	}

	db := storage.OpenDB()
	metaStore := storage.NewInMemoryProfileMetaStore()
	s := profilestore.NewProfileStore(logger, db, metaStore)
	q := query.New(db, metaStore)
	err = server.ListenAndServe(
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
		}),
	)

	return err
}
