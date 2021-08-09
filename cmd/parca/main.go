package main

import (
	"context"
	"io/ioutil"
	"os"

	"github.com/alecthomas/kong"
	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/query"
	"github.com/parca-dev/parca/pkg/server"
	debuginfopb "github.com/parca-dev/parca/proto/gen/go/debuginfo"
	profilestorepb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	querypb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/parca-dev/parca/storage"
	"google.golang.org/grpc"
	"gopkg.in/yaml.v2"
)

type flags struct {
	ConfigPath string `kong:"help='Path to config file.',default='parca.yaml'"`
	LogLevel   string `kong:"enum='error,warn,info,debug',help='Log level.',default='info'"`
	Port       string `kong:"help='Port string for server',default=':9090'"`
}

// Config is the configuration for debug info storage
type Config struct {
	DebugInfo *debuginfo.Config `yaml:"debug_info"`
}

func main() {
	flags := &flags{}
	kong.Parse(flags)

	serverStr := figure.NewColorFigure("Parca", "roman", "cyan", true)
	serverStr.Print()

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

	cfgContent, err := ioutil.ReadFile(flags.ConfigPath)
	if err != nil {
		level.Error(logger).Log("msg", "failed to read config", "path", flags.ConfigPath)
		os.Exit(1)
	}

	cfg := Config{}
	if err := yaml.Unmarshal(cfgContent, &cfg); err != nil {
		level.Error(logger).Log("msg", "failed to parse config", "err", err, "path", flags.ConfigPath)
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
	err = server.ListenAndServe(
		context.Background(),
		logger,
		flags.Port,
		server.RegisterableFunc(func(ctx context.Context, srv *grpc.Server, mux *runtime.ServeMux, endpoint string, opts []grpc.DialOption) error {
			debuginfopb.RegisterDebugInfoServer(srv, d)
			profilestorepb.RegisterProfileStoreServer(srv, s)
			querypb.RegisterQueryServer(srv, q)

			if err := debuginfopb.RegisterDebugInfoHandlerFromEndpoint(ctx, mux, flags.Port, opts); err != nil {
				return err
			}

			if err := profilestorepb.RegisterProfileStoreHandlerFromEndpoint(ctx, mux, flags.Port, opts); err != nil {
				return err
			}

			if err := querypb.RegisterQueryHandlerFromEndpoint(ctx, mux, flags.Port, opts); err != nil {
				return err
			}

			return nil
		}),
	)

	if err != nil {
		level.Error(logger).Log("msg", "Program exited with error", "err", err)
		os.Exit(1)
	}
}
