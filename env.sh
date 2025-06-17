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

ROOT_DIR="$(cd -P "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# renovate: datasource=go depName=github.com/campoy/embedmd
EMBEDMD_VERSION='v2.0.0'
go install "github.com/campoy/embedmd/v2@${EMBEDMD_VERSION}"

# renovate: datasource=go depName=mvdan.cc/gofumpt
GOFUMPT_VERSION='v0.7.0'
go install "mvdan.cc/gofumpt@${GOFUMPT_VERSION}"

# renovate: datasource=go depName=github.com/golangci/golangci-lint
GOLANGCI_LINT_VERSION='v1.64.8'
go install "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"

# renovate: datasource=go depName=golang.org/x/vuln
GOVULNCHECK_VERSION='v1.1.4'
go install "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"

"${ROOT_DIR}/env-proto.sh"
