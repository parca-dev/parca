#!/usr/bin/env bash
#
# The intent of script to install the standard protocol buffer implementation - protoc.
set -eou pipefail

PROTOC_VERSION=${PROTOC_VERSION:-3.17.0}
BUILD_DIR=${BUILD_DIR:-/tmp}
PROTOC_DOWNLOAD_URL="https://github.com/protocolbuffers/protobuf/releases/download/v${PROTOC_VERSION}"

OS=$(go env GOOS)
ARCH=$(go env GOARCH)
PLATFORM="${OS}/${ARCH}"

is_platform_supported() {
  platform=$1
  found=1
  case "$platform" in
    darwin/amd64) found=0 ;;
    darwin/i386) found=0 ;;
    linux/amd64) found=0 ;;
    linux/i386) found=0 ;;
    linux/arm64) found=0 ;;
  esac
  return $found
}

set_os() {
  case ${OS} in
    darwin) OS=osx ;;
  esac
  true
}

set_arch() {
  case ${ARCH} in
    amd64) ARCH=x86_64 ;;
    i386) ARCH=x86_32 ;;
    arm64) ARCH=aarch_64 ;;
  esac
  true
}

mkdir -p ${BUILD_DIR}

is_platform_supported "$PLATFORM"
if [[ $? -eq 1 ]]; then
  echo "platform $PLATFORM is not supported. See https://github.com/protocolbuffers/protobuf/releases for details"
  exit 1
fi

set_os

set_arch

PACKAGE="protoc-${PROTOC_VERSION}-${OS}-${ARCH}.zip"
curl -LSs "${PROTOC_DOWNLOAD_URL}/${PACKAGE}" -o ${BUILD_DIR}/${PACKAGE}
unzip ${BUILD_DIR}/${PACKAGE} -d ${BUILD_DIR}/protoc
