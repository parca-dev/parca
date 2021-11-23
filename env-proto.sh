#! /usr/bin/env bash
set -euo pipefail

BIN_DIR=${BIN_DIR:-/usr/local/bin}
INCLUDE_DIR=${INCLUDE_DIR:-/usr/local/include}
PROTOC_VERSION=${PROTOC_VERSION:-3.19.1}

mkdir -p ./tmp
PROTOC_VERSION="${PROTOC_VERSION}" BUILD_DIR="./tmp" scripts/download-protoc.sh
sudo mv -v -- "./tmp/protoc/bin/protoc" "${BIN_DIR}/protoc"
sudo cp -vR ./tmp/protoc/include/* "${INCLUDE_DIR}"

go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@v2.5.0
go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@v2.5.0
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1.0
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.27.1

# Substitute VERSION for the current released version.
# Substitute BINARY_NAME for "buf", "protoc-gen-buf-breaking", or "protoc-gen-buf-lint".
VERSION="1.0.0-rc8" && \
BINARY_NAME="buf" && \
  curl -sSL \
    "https://github.com/bufbuild/buf/releases/download/v${VERSION}/${BINARY_NAME}-$(uname -s)-$(uname -m)" \
    -o "${BIN_DIR}/${BINARY_NAME}" && \
  chmod +x "${BIN_DIR}/${BINARY_NAME}"
