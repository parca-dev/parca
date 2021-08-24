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

.PHONY: clean
clean:
	rm -r bin
	rm -r ui/dist
	rm -r ui/.next

.PHONY: build
build: ui go/bin

.PHONY: go/bin
go/bin:
	mkdir -p ./bin
	go build -o bin/ ./cmd/parca
	cp parca.yaml bin/

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
