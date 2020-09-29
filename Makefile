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
GOLANGCI_LINT_OPTS = --skip-dirs internal

include Makefile.common
include .bingo/Variables.mk

DOCKER_IMAGE_NAME       ?= conprof

GO111MODULE       ?= on
export GO111MODULE
GOPROXY           ?= https://proxy.golang.org
export GOPROXY

GOBIN          ?= $(firstword $(subst :, ,${GOPATH}))/bin
TMP_GOPATH     ?= /tmp/thanos-go
PROTOC         ?= $(GOBIN)/protoc-$(PROTOC_VERSION)
PROTOC_VERSION ?= 3.4.0
GIT            ?= $(shell which git)

.PHONY: test
test:
	go test ./...

.PHONY: assets
assets:
	@echo ">> writing assets"
	cd $(PREFIX)/web && GO111MODULE=$(GO111MODULE) $(GO) generate -x -v $(GOOPTS)
	@$(GOFMT) -w ./web

.PHONY: check_assets
check_assets: assets
	@echo ">> checking that assets are up-to-date"
	@if ! (cd $(PREFIX)/web/ui && git diff --exit-code); then \
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

# docker-push pushes docker image build under `${DOCKER_IMAGE_NAME}` to quay.io/thanos/"$(DOCKER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)"
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


.PHONY: check-git
check-git:
ifneq ($(GIT),)
	@test -x $(GIT) || (echo >&2 "No git executable binary found at $(GIT)."; exit 1)
else
	@echo >&2 "No git binary found."; exit 1
endif
