CMD_DOCKER ?= docker
CMD_GIT ?= git
SHELL := /bin/bash
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
ALL_ARCH ?= amd64 arm arm64
OUT_DOCKER ?= ghcr.io/parca-dev/parca

.PHONY: build
build: ui go/bin

.PHONY: clean
clean:
	rm -rf bin
	rm -rf ui/packages/app/web/dist
	rm -rf ui/packages/app/web/.next

.PHONY: go/deps
go/deps:
	go mod tidy

.PHONY: go/bin
go/bin: go/deps
	mkdir -p ./bin
	go build -o bin/ ./cmd/parca

.PHONY: format
format: go/fmt check-license

.PHONY: go/fmt
go/fmt:
	go fmt `go list ./...`

go/lint:
	golangci-lint run

.PHONY: check-license
check-license:
	./scripts/check-license.sh

.PHONY: go/test
go/test:
	 go test -v `go list ./...`

UI_FILES ?= $(shell find ./ui -name "*" -not -path "./ui/lib/node_modules/*" -not -path "./ui/node_modules/*" -not -path "./ui/packages/app/web/node_modules/*" -not -path "./ui/packages/app/web/dist/*" -not -path "./ui/packages/app/web/.next/*")
ui/packages/app/web/dist: $(UI_FILES)
	cd ui && yarn install && yarn workspace @parca/web build

ui: ui/packages/app/web/dist

.PHONY: proto/lint
proto/lint:
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf lint
	buf lint

.PHONY: proto/generate
proto/generate:
	yarn install
	# Generate just the annotations and http protos.
	buf generate buf.build/googleapis/googleapis --path google/api/annotations.proto --path google/api/http.proto
	buf generate

.PHONY: proto/vendor
proto/vendor:
	buf mod update
	mkdir -p proto/google/pprof
	curl https://raw.githubusercontent.com/google/pprof/master/proto/profile.proto > proto/google/pprof/profile.proto

.PHONY: container-dev
container-dev:
       buildah build-using-dockerfile --timestamp 0 --layers --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) -t $(OUT_DOCKER):$(VERSION) .

.PHONY: container
container:
	for arch in $(ALL_ARCH); do \
	buildah build-using-dockerfile --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg ARCH=$$arch --arch $$arch --timestamp 0 --manifest $(OUT_DOCKER):$(VERSION); \
	done

.PHONY: push-container
push-container:
	buildah manifest push --all $(OUT_DOCKER):$(VERSION)

.PHONY: push-quay-container
push-quay-container:
	buildah manifest push --all $(OUT_DOCKER):$(VERSION) quay.io/parca/parca:$(VERSION)

.PHONY: deploy/manifests
deploy/manifests:
	cd deploy && make manifests

.PHONY: dev/setup
dev/setup:
	./env.sh
	./env-jsonnet.sh

.PHONY: dev/up
dev/up: deploy/manifests
	source ./scripts/local-dev.sh && up

.PHONY: dev/down
dev/down:
	source ./scripts/local-dev.sh && down
