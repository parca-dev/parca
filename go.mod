module github.com/conprof/conprof

go 1.15

require (
	github.com/NYTimes/gziphandler v1.1.1
	github.com/conprof/db v0.0.0-20210317165925-a59fb33c527d
	github.com/cortexproject/cortex v1.9.1-0.20210601081042-d7d87369965a
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.2
	github.com/gogo/status v1.0.3
	github.com/google/pprof v0.0.0-20210504235042-3a04a4d88a10
	github.com/grpc-ecosystem/go-grpc-prometheus v1.2.0
	github.com/ianlancetaylor/demangle v0.0.0-20200824232613-28f6c0f3b639
	github.com/julienschmidt/httprouter v1.3.0
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.10.0
	github.com/prometheus/common v0.23.0
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/prometheus/tsdb v0.10.0
	github.com/shurcooL/vfsgen v0.0.0-20200824052919-0d455de96546
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/thanos-io/thanos v0.21.0-rc.0
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.20.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.20.0
	go.opentelemetry.io/otel v0.20.0
	go.opentelemetry.io/otel/exporters/otlp v0.20.0
	go.opentelemetry.io/otel/sdk v0.20.0
	golang.org/x/net v0.0.0-20210505214959-0714010a04ed
	google.golang.org/grpc v1.37.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.4.0
)

replace gopkg.in/alecthomas/kingpin.v2 => github.com/alecthomas/kingpin v1.3.8-0.20210301060133-17f40c25f497

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20210519120135-d95b0972505f
