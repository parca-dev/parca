module github.com/parca-dev/parca

go 1.16

require (
	github.com/alecthomas/kong v0.3.0
	github.com/cenkalti/backoff/v4 v4.1.2
	github.com/cespare/xxhash/v2 v2.1.2
	github.com/common-nighthawk/go-figure v0.0.0-20210622060536-734e95fb86be
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/dgraph-io/sroar v0.0.0-20211209113350-3e3f1b382a64
	github.com/gin-gonic/gin v1.7.7 // indirect
	github.com/go-chi/cors v1.2.0
	github.com/go-delve/delve v1.8.0
	github.com/go-kit/log v0.2.0
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0
	github.com/goburrow/cache v0.1.4
	github.com/google/pprof v0.0.0-20211214055906-6f57359322fd
	github.com/google/uuid v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/grpc-ecosystem/go-grpc-middleware/providers/kit/v2 v2.0.0-20201002093600-73cf2ae9d891
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.0.0-rc.2.0.20201207153454-9f6bf00c00a7
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.7.2
	github.com/hashicorp/go-multierror v1.1.1
	github.com/ianlancetaylor/demangle v0.0.0-20211126204342-3ad08eb09c01
	github.com/improbable-eng/grpc-web v0.15.0
	github.com/oklog/run v1.1.0
	github.com/prometheus/client_golang v1.11.0
	github.com/prometheus/common v0.32.1
	github.com/prometheus/prometheus v1.8.2-0.20211217191541-41f1a8125e66
	github.com/stretchr/testify v1.7.0
	github.com/thanos-io/thanos v0.24.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.28.0
	go.opentelemetry.io/otel v1.3.0
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.3.0
	go.opentelemetry.io/otel/sdk v1.3.0
	go.opentelemetry.io/otel/trace v1.3.0
	go.uber.org/atomic v1.9.0
	golang.org/x/net v0.0.0-20220105145211-5b0dc2dfae98
	golang.org/x/sync v0.0.0-20210220032951-036812b2e83c
	golang.org/x/sys v0.0.0-20211216021012-1d35b9e2eb4e // indirect
	google.golang.org/genproto v0.0.0-20220106162220-2482ccee2e38
	google.golang.org/grpc v1.43.0
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.2.0
	google.golang.org/protobuf v1.27.1
	gopkg.in/ini.v1 v1.66.2 // indirect
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.40.1 // indirect
	modernc.org/sqlite v1.14.5
	nhooyr.io/websocket v1.8.7 // indirect
)

// Prometheus v2.32.1
replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20211217191541-41f1a8125e66

replace go.opentelemetry.io/otel/metric => go.opentelemetry.io/otel/metric v0.24.0
