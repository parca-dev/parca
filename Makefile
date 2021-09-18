CMD_DOCKER ?= docker
CMD_GIT ?= git
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

.PHONY: build
build: ui go/bin

.PHONY: clean
clean:
	rm -rf bin
	rm -rf ui/dist
	rm -rf ui/.next

.PHONY: go/deps
go/deps: internal/pprof
	go mod tidy

.PHONY: go/bin
go/bin: go/deps
	mkdir -p ./bin
	go build -o bin/ ./cmd/parca

.PHONY: format
format: go/fmt check-license

.PHONY: go/fmt
go/fmt:
	go fmt `go list ./... | grep -v ./internal/pprof`

.PHONY: check-license
check-license:
	./scripts/check-license.sh

.PHONY: go/test
go/test:
	 go test -v `go list ./... | grep -v ./internal/pprof`

.PHONY: ui
ui:
	cd ui && yarn install && yarn workspace @parca/web build

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
	curl https://raw.githubusercontent.com/google/pprof/master/proto/profile.proto                                                 > proto/google/pprof/profile.proto

.PHONY: container
container:
	buildah build-using-dockerfile --build-arg TOKEN --timestamp 0 --layers -t $(OUT_DOCKER):$(VERSION)

.PHONY: push-container
push-container:
	buildah push $(OUT_DOCKER):$(VERSION)

.PHONY: push-quay-container
push-quay-container:
	buildah push $(OUT_DOCKER):$(VERSION) quay.io/parca/parca:$(VERSION)

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

internal/pprof:
	rm -rf internal/pprof
	rm -rf tmp/pprof
	git clone https://github.com/google/pprof tmp/pprof
	cp -r tmp/pprof/internal internal/pprof
	find internal/pprof -type f -exec sed -i 's/github.com\/google\/pprof\/internal/github.com\/parca-dev\/parca\/internal\/pprof/g' {} +
	rm -rf tmp/pprof
