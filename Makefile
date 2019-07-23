GO_PKG=github.com/conprof/conprof
GOLANG_FILES:=$(shell find . -name \*.go -print)
VERSION:=$(shell cat VERSION)

assets:
	@echo ">> writing assets"
	cd web && GO111MODULE=on go generate -x -v
	@gofmt -w ./web

conprof: $(GOLANG_FILES)
	GOOS=linux GO111MODULE=on CGO_ENABLED=0 go build \
	-ldflags "-X $(GO_PKG)/version.Version=$(VERSION)" \
	-o $@

validate-modules:
	@echo "- Verifying that the dependencies have expected content..."
	GO111MODULE=on go mod verify
	@echo "- Checking for any unused/missing packages in go.mod..."
	GO111MODULE=on go mod tidy
	@echo "- Checking for unused packages in vendor..."
	GO111MODULE=on go mod vendor
	@git diff --exit-code -- go.sum go.mod vendor/


container: conprof
	docker build -t quay.io/conprof/conprof:$(VERSION) .
