#! /usr/bin/env bash
set -euo pipefail

go install github.com/campoy/embedmd@latest

go install mvdan.cc/gofumpt@latest

go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.45.0
