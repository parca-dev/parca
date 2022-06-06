CMD_DOCKER ?= docker
CMD_GIT ?= git
SHELL := /bin/bash
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
ifeq ($(GITHUB_BRANCH_NAME),)
	BRANCH := $(shell git rev-parse --abbrev-ref HEAD)-
else
	BRANCH := $(GITHUB_BRANCH_NAME)-
endif
ifeq ($(GITHUB_SHA),)
	COMMIT := $(shell git describe --no-match --dirty --always --abbrev=8)
else
	COMMIT := $(shell echo $(GITHUB_SHA) | cut -c1-8)
endif
VERSION ?= $(if $(RELEASE_TAG),$(RELEASE_TAG),$(shell $(CMD_GIT) describe --tags 2>/dev/null || echo '$(BRANCH)$(COMMIT)'))
OUT_DOCKER ?= ghcr.io/parca-dev/parca
HAVE_AVX2 := $(shell grep avx2 /proc/cpuinfo)
ifneq ($(.SHELLSTATUS),0)
	GO_BUILD_TAGS := -tags purego
endif

.PHONY: build
build: ui/build go/bin

.PHONY: clean
clean:
	rm -rf bin
	rm -rf ui/packages/app/web/build

.PHONY: go/deps
go/deps:
	go mod tidy

.PHONY: go/bin
go/bin: go/deps
	mkdir -p ./bin
	go build $(GO_BUILD_TAGS) -o bin/ ./cmd/parca

.PHONY: format
format: go/fmt proto/format check-license

gofumpt:
ifeq (, $(shell which gofumpt))
	go install mvdan.cc/gofumpt@v0.3.0
GOFUMPT=$(GOBIN)/gofumpt
else
GOFUMPT=$(shell which gofumpt)
endif

# Rather than running this over and over we recommend running gofumpt on save with your editor.
# Check https://github.com/mvdan/gofumpt#installation for instructions.
.PHONY: go/fmt
go/fmt: gofumpt
	$(GOFUMPT) -l -w $(shell go list -f {{.Dir}} ./... | grep -v gen/proto)

go/lint: check-license
	golangci-lint run

.PHONY: check-license
check-license:
	./scripts/check-license.sh

.PHONY: go/test
go/test:
	go test -v `go list ./...`
	mkdir -pm 777 tmp/
	go test -run=. -bench=. -benchtime=1x `go list ./...` # run benchmark with one iteration to make sure they work

VCR_FILES ?= $(shell find ./pkg/*/testdata -name "fixtures.yaml")

.PHONY: go/test-clean
go/test-clean:
	rm -f $(VCR_FILES)


UI_FILES ?= $(shell find ./ui -name "*" -not -path "./ui/lib/node_modules/*" -not -path "./ui/node_modules/*" -not -path "./ui/packages/app/template/node_modules/*" -not -path "./ui/packages/app/web/node_modules/*" -not -path "./ui/packages/app/web/build/*")

.PHONY: ui/build
ui/build: $(UI_FILES)
	cd ui && yarn install && yarn workspace @parca/web build

.PHONY: proto/all
proto/all: proto/vendor proto/format proto/lint proto/generate

.PHONY: proto/lint
proto/lint:
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf lint
	buf lint

.PHONY: proto/format
proto/format:
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf format
	buf format -w

.PHONY: proto/generate
proto/generate: proto/vendor
	# Generate just the annotations and http protos.
	buf generate buf.build/googleapis/googleapis --path google/api/annotations.proto --path google/api/http.proto
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
	podman build --timestamp 0 --layers --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(OUT_DOCKER):$(VERSION) .

.PHONY: container
container:
	./scripts/make-containers.sh $(VERSION) $(COMMIT) $(OUT_DOCKER):$(VERSION)

.PHONY: push-container
push-container:
	podman manifest push --all $(OUT_DOCKER):$(VERSION) docker://$(OUT_DOCKER):$(VERSION)

.PHONY: sign-container
sign-container:
	crane digest $(OUT_DOCKER):$(VERSION)
	cosign sign --force -a GIT_HASH=$(COMMIT) -a GIT_VERSION=$(VERSION) $(OUT_DOCKER)@$(shell crane digest $(OUT_DOCKER):$(VERSION))

.PHONY: push-quay-container
push-quay-container:
	podman manifest push --all $(OUT_DOCKER):$(VERSION) docker://quay.io/parca/parca:$(VERSION)

.PHONY: push-signed-quay-container
push-signed-quay-container:
	cosign copy $(OUT_DOCKER):$(VERSION) quay.io/parca/parca:$(VERSION)

.PHONY: deploy/manifests
deploy/manifests:
	cd deploy && make manifests

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

tmp/help.txt: go/bin
	mkdir -p tmp
	bin/parca --help > $@

embedmd:
ifeq (, $(shell which embedmd))
	go install github.com/campoy/embedmd@latest
EMBEDMD=$(GOBIN)/embedmd
else
EMBEDMD=$(shell which embedmd)
endif

README.md: embedmd tmp/help.txt
	$(EMBEDMD) -w README.md
