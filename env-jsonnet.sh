#! /usr/bin/env bash
set -euo pipefail

# renovate: datasource=go depName=github.com/brancz/gojsontoyaml
GOJSONTOYAML_VERSION='v0.1.0'
go install "github.com/brancz/gojsontoyaml@${GOJSONTOYAML_VERSION}"

# renovate: datasource=go depName=github.com/google/go-jsonnet
JSONNET_VERSION='v0.18.0'
go install "github.com/google/go-jsonnet/cmd/jsonnet@${JSONNET_VERSION}"
go install "github.com/google/go-jsonnet/cmd/jsonnetfmt@${JSONNET_VERSION}"

# renovate: datasource=go depName=github.com/jsonnet-bundler/jsonnet-bundler
JB_VERSION='v0.4.0'
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@${JB_VERSION}
