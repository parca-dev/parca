assets:
	@echo ">> writing assets"
	cd web && GO111MODULE=on go generate -x -v
	@gofmt -w ./web
