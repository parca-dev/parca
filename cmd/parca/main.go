// Copyright 2021 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/alecthomas/kong"
	"github.com/common-nighthawk/go-figure"
	"github.com/go-kit/log/level"
	"github.com/parca-dev/parca/pkg/parca"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	version string = "dev"
	commit  string = "dev"
)

func main() {
	ctx := context.Background()
	flags := &parca.Flags{}

	kong.Parse(flags)

	if flags.Version {
		fmt.Printf("parca, version %s (commit: %s)\n", version, commit)
		return
	}

	serverStr := figure.NewColorFigure("Parca", "roman", "cyan", true)
	serverStr.Print()

	logger := parca.NewLogger(flags.LogLevel, parca.LogFormatLogfmt, "parca")
	level.Debug(logger).Log("msg", "parca initialized",
		"version", version,
		"commit", commit,
		"config", fmt.Sprint(flags),
	)

	registry := prometheus.NewRegistry()

	err := parca.Run(ctx, logger, registry, flags, version)

	if err != nil {
		level.Error(logger).Log("msg", "Program exited with error", "err", err)
		os.Exit(1)
	}

	level.Info(logger).Log("msg", "exited")
}
