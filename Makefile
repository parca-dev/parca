# Copyright 2018 The Prometheus Authors
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

# Needs to be defined before including Makefile.common to auto-generate targets

DOCKER_ARCHS ?= amd64 armv7 arm64
GOLANGCI_LINT_OPTS = --skip-dirs=internal --modules-download-mode=readonly

include Makefile.common
include .bingo/Variables.mk

DOCKER_IMAGE_NAME       ?= conprof

GOPROXY           ?= https://proxy.golang.org
export GOPROXY

GOBIN          ?= $(firstword $(subst :, ,${GOPATH}))/bin
TMP_GOPATH     ?= /tmp/conprof-go
PROTOC         ?= $(GOBIN)/protoc-$(PROTOC_VERSION)
PROTOC_VERSION ?= 3.4.0
GIT            ?= $(shell which git)

.PHONY: assets
assets:
	@echo ">> writing assets"
	cd $(PREFIX)/web && $(GO) generate -x -v $(GOOPTS)
	@$(GOFMT) -w ./web

.PHONY: check_assets
check_assets: assets
	@echo ">> checking that assets are up-to-date"
	@if ! (cd $(PREFIX)/web && git diff --exit-code); then \
		echo "Run 'make assets' and commit the changes to fix the error."; \
		exit 1; \
	fi

.PHONY: sync
sync: sync-trace-pkg

.PHONY: sync-trace-pkg
sync-trace-pkg:
	mkdir tmp && cd tmp && git clone https://github.com/golang/go.git && cd ../
	cp -r tmp/go/src/internal/trace internal/trace
	rm -rf tmp
	echo "#IMPORTANT DO NOT EDIT! This code is synced from go repository. Use make sync to update it." > internal/trace/README.md

# crossbuild builds all binaries for all platforms.
.PHONY: crossbuild
crossbuild: $(PROMU)
	@echo ">> crossbuilding all binaries"
	$(PROMU) crossbuild -v

# docker builds docker with no tag.
.PHONY: docker
docker: common-build
	@echo ">> building docker image '${DOCKER_IMAGE_NAME}'"
	@docker build -t "${DOCKER_IMAGE_NAME}" .

# docker-push pushes docker image build under `${DOCKER_IMAGE_NAME}` to quay.io/conprof/"$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
.PHONY: docker-push
docker-push:
	@echo ">> pushing image"
	@docker tag "${DOCKER_IMAGE_NAME}" quay.io/conprof/"$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
	@docker push quay.io/conprof/"$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"

.PHONY: proto
proto: check-git $(GOIMPORTS) $(PROTOC) $(PROTOC_GEN_GOGOFAST)
		@GOIMPORTS_BIN="$(GOIMPORTS)" PROTOC_BIN="$(PROTOC)" PROTOC_GEN_GOGOFAST_BIN="$(PROTOC_GEN_GOGOFAST)" scripts/genproto.sh

$(PROTOC):
	@mkdir -p $(TMP_GOPATH)
	@echo ">> fetching protoc@${PROTOC_VERSION}"
	@PROTOC_VERSION="$(PROTOC_VERSION)" TMP_GOPATH="$(TMP_GOPATH)" scripts/installprotoc.sh
	@echo ">> installing protoc@${PROTOC_VERSION}"
	@mv -- "$(TMP_GOPATH)/bin/protoc" "$(GOBIN)/protoc-$(PROTOC_VERSION)"
	@echo ">> produced $(GOBIN)/protoc-$(PROTOC_VERSION)"

.PHONY: test-e2e
test-e2e: ## Runs all Conprof e2e docker-based e2e tests from test/e2e. Required access to docker daemon.
test-e2e: docker
	@echo ">> cleaning docker environment."
	@docker system prune -f --volumes
	@echo ">> cleaning e2e test garbage."
	@rm -rf ./test/e2e/e2e_integration_test*
	@echo ">> running /test/e2e tests."
	# NOTE(bwplotka):
	# 	# * If you see errors on CI (timeouts), but not locally, try to add -parallel 1 to limit to single CPU to reproduce small 1CPU machine.
	@go test $(GOTEST_OPTS) ./test/e2e/...

.PHONY: check-git
check-git:
ifneq ($(GIT),)
	@test -x $(GIT) || (echo >&2 "No git executable binary found at $(GIT)."; exit 1)
else
	@echo >&2 "No git binary found."; exit 1
endif

internal/pprof:
	rm -rf internal/pprof
	rm -rf tmp
	git clone https://github.com/google/pprof tmp/pprof
	cp -r tmp/pprof/internal internal/pprof
	find internal/pprof -type f -exec sed -i 's/github.com\/google\/pprof\/internal/github.com\/conprof\/conprof\/internal\/pprof/g' {} +
	rm -rf tmp
