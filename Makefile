CMD_GIT ?= git
SHELL := /usr/bin/env bash
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
ifeq ($(GITHUB_BRANCH_NAME),)
	BRANCH := $(shell $(CMD_GIT) rev-parse --abbrev-ref HEAD)-
else
	BRANCH := $(GITHUB_BRANCH_NAME)-
endif
COMMIT_TIMESTAMP := $(shell $(CMD_GIT) show --no-patch --format=%ct)-
ifeq ($(GITHUB_SHA),)
	COMMIT := $(shell $(CMD_GIT) rev-parse --short=8 HEAD)
else
	COMMIT := $(shell echo $(GITHUB_SHA) | cut -c1-8)
endif
VERSION ?= $(if $(RELEASE_TAG),$(RELEASE_TAG),$(shell $(CMD_GIT) describe --tags --match='v*' || echo '$(subst /,-,$(BRANCH))$(COMMIT_TIMESTAMP)$(COMMIT)'))
OUT_DOCKER ?= ghcr.io/parca-dev/parca

ENABLE_RACE := no

ifeq ($(ENABLE_RACE), yes)
	SANITIZERS += -race
endif

.PHONY: build
build: ui/build go/bin

.PHONY: format
format: go/fmt proto/format

.PHONY: lint
lint: check-license go/lint proto/lint ui/lint

.PHONY: test
test: go/test ui/test

.PHONY: clean
clean:
	rm -rf data
	rm -rf tmp
	rm -rf bin
	rm -rf ui/packages/app/web/build
	find . -name 'node_modules' -type d -prune -exec rm -rf '{}' +
	find . -name 'dist' -type d -prune -exec rm -rf '{}' +

.PHONY: go/deps
go/deps:
	go mod tidy

.PHONY: go/bin
go/bin: go/deps
	mkdir -p ./bin
	go build $(SANITIZERS) -gcflags="all=-N -l" -o bin/ ./cmd/parca

# renovate: datasource=go depName=mvdan.cc/gofumpt
GOFUMPT_VERSION := v0.5.0
gofumpt:
ifeq (, $(shell command -v gofumpt >/dev/null))
	go install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
GOFUMPT=$(GOBIN)/gofumpt
else
GOFUMPT=$(shell command -v gofumpt)
endif

# Rather than running this over and over we recommend running gofumpt on save with your editor.
# Check https://github.com/mvdan/gofumpt#installation for instructions.
.PHONY: go/fmt
go/fmt: gofumpt
	$(GOFUMPT) -l -w $(shell go list -f {{.Dir}} ./... | grep -v gen/proto)

.PHONY: go/lint
go/lint:
	golangci-lint run

.PHONY: check-license
check-license:
	./scripts/check-license.sh

.PHONY: go/test
go/test:
	go test $(SANITIZERS) -v `go list ./...`

.PHONY: go/bench
go/bench:
	mkdir -pm 777 tmp/
	go test $(SANITIZERS) -run=. -bench=. -benchtime=1x -v `go list ./...` # run benchmark with one iteration to make sure they work

VCR_FILES ?= $(shell find ./pkg/*/testdata -name "fixtures.yaml")

.PHONY: go/test-clean
go/test-clean:
	rm -f $(VCR_FILES)

UI_FILES ?= $(shell find ./ui -name "*" -not -path "./ui/lib/node_modules/*" -not -path "./ui/node_modules/*" -not -path "./ui/packages/app/template/node_modules/*" -not -path "./ui/packages/app/web/node_modules/*" -not -path "./ui/packages/app/web/build/*")
.PHONY: ui/build
ui/build: $(UI_FILES)
	cd ui && yarn install --frozen-lockfile --prefer-offline && yarn build

.PHONY: ui/test
ui/test:
	cd ui && yarn test

.PHONY: ui/lint
ui/lint:
	cd ui && yarn run lint

.PHONY: proto/all
proto/all: proto/vendor proto/format proto/lint proto/generate

.PHONY: proto/lint
proto/lint:
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf lint
	buf lint
	buf breaking -v --against '.git#branch=main,subdir=proto'

.PHONY: proto/format
proto/format:
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf format
	buf format -w

.PHONY: proto/generate
proto/generate: proto/vendor
	# Generate just the annotations and http protos.
	buf generate buf.build/googleapis/googleapis --path google/api/annotations.proto --path google/api/http.proto
	buf generate buf.build/grpc/grpc --path grpc/health/
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf generate
	buf generate

.PHONY: proto/vendor
proto/vendor: proto/google/pprof/profile.proto
	cd proto && buf mod update

proto/google/pprof/profile.proto:
	mkdir -p proto/google/pprof
	curl https://raw.githubusercontent.com/google/pprof/master/proto/profile.proto > proto/google/pprof/profile.proto

.PHONY: container-dev
container-dev:
	docker build -t parca-dev/parca-agent:dev -t $(OUT_DOCKER):$(VERSION) .
	#podman build --timestamp 0 --layers -t $(OUT_DOCKER):$(VERSION) .

.PHONY: container
container:
	podman build \
		--platform linux/amd64,linux/arm64 \
		--timestamp 0 \
		--manifest $(OUT_DOCKER):$(VERSION) .

.PHONY: push-container
push-container:
	podman manifest push --all $(OUT_DOCKER):$(VERSION) docker://$(OUT_DOCKER):$(VERSION)

.PHONY: sign-container
sign-container:
	crane digest $(OUT_DOCKER):$(VERSION)
	cosign sign --yes -a GIT_HASH=$(COMMIT) -a GIT_VERSION=$(VERSION) $(OUT_DOCKER)@$(shell crane digest $(OUT_DOCKER):$(VERSION))

.PHONY: push-quay-container
push-quay-container:
	podman manifest push --all $(OUT_DOCKER):$(VERSION) docker://quay.io/parca/parca:$(VERSION)

.PHONY: push-signed-quay-container
push-signed-quay-container:
	cosign copy $(OUT_DOCKER):$(VERSION) quay.io/parca/parca:$(VERSION)

.PHONY: deploy/manifests
deploy/manifests:
	$(MAKE) -C deploy SEPARATE_UI=false manifests

.PHONY: dev/setup
dev/setup:
	./env.sh
	./env-local-test.sh
	./env-jsonnet.sh

.PHONY: dev/up
dev/up: deploy/manifests
	source ./scripts/local-dev.sh && up

.PHONY: dev/down
dev/down:
	source ./scripts/local-dev.sh && down

tmp/help.txt: build
	mkdir -p tmp
	bin/parca --help > $@

# renovate: datasource=go depName=github.com/campoy/embedmd
EMBEDMD_VERSION ?= v2.0.0

embedmd:
ifeq (, $(shell command -v embedmd >/dev/null))
	go install github.com/campoy/embedmd/v2@$(EMBEDMD_VERSION)
EMBEDMD=$(GOBIN)/embedmd
else
EMBEDMD=$(shell command -v embedmd)
endif

README.md: embedmd tmp/help.txt
	$(EMBEDMD) -w README.md

.PHONY: release-dry-run
release-dry-run:
	goreleaser release --clean --auto-snapshot --skip-validate --skip-publish --debug

.PHONY: release-build
release-build:
	goreleaser build --clean --skip-validate --snapshot --debug
