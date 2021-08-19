package main

import (
	"context"
	"os"

	"github.com/alecthomas/kong"
	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/parca-dev/parca/pkg/parca"
)

type flags struct {
	ConfigPath string `kong:"help='Path to config file.',default='parca.yaml'"`
	LogLevel   string `kong:"enum='error,warn,info,debug',help='Log level.',default='info'"`
	Port       string `kong:"help='Port string for server',default=':7070'"`
}

func main() {
	ctx := context.Background()
	flags := &flags{}
	kong.Parse(flags)

	serverStr := figure.NewColorFigure("Parca", "roman", "cyan", true)
	serverStr.Print()

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

	err := parca.Run(ctx, logger, flags.ConfigPath, flags.Port)
	if err != nil {
		level.Error(logger).Log("msg", "Program exited with error", "err", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "exited")
}
