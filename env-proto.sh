#!/usr/bin/env bash
# Copyright 2023 The Parca Authors
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

BIN_DIR=${BIN_DIR:-/usr/local/bin}

# renovate: datasource=github-releases depName=bufbuild/buf
BUF_VERSION='v1.21.0'

# Substitute BINARY_NAME for "buf", "protoc-gen-buf-breaking", or "protoc-gen-buf-lint".
BINARY_NAME="buf"

curl -fsSL \
    "https://github.com/bufbuild/buf/releases/download/${BUF_VERSION}/${BINARY_NAME}-$(uname -s)-$(uname -m)" \
    -o "${BIN_DIR}/${BINARY_NAME}"

chmod +x "${BIN_DIR}/${BINARY_NAME}"
