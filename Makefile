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
	# docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf generate --path=./proto/api
	buf generate --path=./proto/query
	buf generate --path=./proto/profilestore
	buf generate --path=./proto/debuginfo
	buf generate --path=./proto/google/api
	buf generate --path=./proto/google/pprof

.PHONY: proto/vendor
proto/vendor:
	mkdir -p proto/google/api
	mkdir -p proto/protoc-gen-openapiv2/options
	mkdir -p proto/google/pprof
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto                               > proto/google/api/annotations.proto
	curl https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/timestamp.proto                     > proto/google/api/timestamp.proto
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto                                      > proto/google/api/http.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/annotations.proto       > proto/protoc-gen-openapiv2/options/annotations.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/openapiv2.proto         > proto/protoc-gen-openapiv2/options/openapiv2.proto
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
