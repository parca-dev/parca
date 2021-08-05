.PHONY: clean
clean:
	rm parca

.PHONY: go/bin
go/bin:
	go build ./cmd/parca

.PHONY: proto/lint
proto/lint:
	docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf lint

.PHONY: proto/generate
proto/generate:
	#docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf generate --path=./proto/api
	buf generate --path=./proto/api
	buf generate --path=./proto/google/pprof

.PHONY: proto/vendor
proto/vendor:
	mkdir -p proto/google/api
	mkdir -p proto/protoc-gen-openapiv2/options
	mkdir -p proto/pprof
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto                               > proto/google/api/annotations.proto
	curl https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/timestamp.proto                     > proto/google/api/timestamp.proto
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto                                      > proto/google/api/http.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/annotations.proto       > proto/protoc-gen-openapiv2/options/annotations.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/openapiv2.proto         > proto/protoc-gen-openapiv2/options/openapiv2.proto
	curl https://raw.githubusercontent.com/google/pprof/master/proto/profile.proto                                                 > proto/pprof/profile.proto
