#!/usr/bin/env bash
set -euo pipefail

# renovate: datasource=go depName=github.com/campoy/embedmd
EMBEDMD_VERSION='v2.0.0'
go install "github.com/campoy/embedmd/v2@${EMBEDMD_VERSION}"

# renovate: datasource=go depName=mvdan.cc/gofumpt
GOFUMPT_VERSION='v0.4.0'
go install "mvdan.cc/gofumpt@${GOFUMPT_VERSION}"

# renovate: datasource=go depName=github.com/golangci/golangci-lint
GOLANGCI_LINT_VERSION='v1.49.0'
go install "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
