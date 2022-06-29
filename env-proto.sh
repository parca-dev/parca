#! /usr/bin/env bash
set -euo pipefail

BIN_DIR=${BIN_DIR:-/usr/local/bin}

# renovate: datasource=github-releases depName=bufbuild/buf
BUF_VERSION='v1.6.0'

# Substitute BINARY_NAME for "buf", "protoc-gen-buf-breaking", or "protoc-gen-buf-lint".
BINARY_NAME="buf"

curl -fsSL \
  "https://github.com/bufbuild/buf/releases/download/${BUF_VERSION}/${BINARY_NAME}-$(uname -s)-$(uname -m)" \
  -o "${BIN_DIR}/${BINARY_NAME}"

chmod +x "${BIN_DIR}/${BINARY_NAME}"
