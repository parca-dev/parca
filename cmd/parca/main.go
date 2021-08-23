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

func main() {
	ctx := context.Background()
	flags := &parca.Flags{}
	kong.Parse(flags)

	serverStr := figure.NewColorFigure("Parca", "roman", "cyan", true)
	serverStr.Print()

	logger := log.NewJSONLogger(log.NewSyncWriter(os.Stdout))

	err := parca.Run(ctx, logger, flags)
	if err != nil {
		level.Error(logger).Log("msg", "Program exited with error", "err", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "exited")
}
