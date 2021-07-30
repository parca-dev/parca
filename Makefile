proto/lint:
	docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf lint

proto/generate:
	#docker run --volume ${PWD}:/workspace --workdir /workspace bufbuild/buf generate --path=./proto/api
	buf generate --path=./proto/api

.PHONY: proto/vendor
proto/vendor:
	mkdir -p proto-vendor/google/api
	mkdir -p proto-vendor/protoc-gen-openapiv2/options
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/annotations.proto                               > proto-vendor/google/api/annotations.proto
	curl https://raw.githubusercontent.com/protocolbuffers/protobuf/master/src/google/protobuf/timestamp.proto                     > proto-vendor/google/api/timestamp.proto
	curl https://raw.githubusercontent.com/googleapis/googleapis/master/google/api/http.proto                                      > proto-vendor/google/api/http.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/annotations.proto       > proto-vendor/protoc-gen-openapiv2/options/annotations.proto
	curl https://raw.githubusercontent.com/grpc-ecosystem/grpc-gateway/master/protoc-gen-openapiv2/options/openapiv2.proto         > proto-vendor/protoc-gen-openapiv2/options/openapiv2.proto
