GO_PKG=github.com/conprof/conprof
GOLANG_FILES:=$(shell find . -name \*.go -print)

assets:
	@echo ">> writing assets"
	cd web && GO111MODULE=on go generate -x -v
	@gofmt -w ./web

conprof: $(GOLANG_FILES)
	GOOS=linux GO111MODULE=on CGO_ENABLED=0 go build \
	-ldflags "-X $(GO_PKG)/version.Version=$(shell cat VERSION)" \
	-o $@
