#!/usr/bin/env bash
# Copyright 2023-2025 The Parca Authors
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

# renovate: datasource=go depName=github.com/brancz/gojsontoyaml
GOJSONTOYAML_VERSION='v0.1.0'
go install "github.com/brancz/gojsontoyaml@${GOJSONTOYAML_VERSION}"

# renovate: datasource=go depName=github.com/google/go-jsonnet
JSONNET_VERSION='v0.21.0'
go install "github.com/google/go-jsonnet/cmd/jsonnet@${JSONNET_VERSION}"
go install "github.com/google/go-jsonnet/cmd/jsonnetfmt@${JSONNET_VERSION}"

# renovate: datasource=go depName=github.com/jsonnet-bundler/jsonnet-bundler
JB_VERSION='v0.6.0'
go install github.com/jsonnet-bundler/jsonnet-bundler/cmd/jb@${JB_VERSION}
