module github.com/conprof/conprof

go 1.14

require (
	github.com/conprof/db v0.0.0-20200923100233-d202624dc72c
	github.com/go-kit/kit v0.10.0
	github.com/gogo/protobuf v1.3.1
	github.com/gogo/status v1.0.3
	github.com/google/pprof v0.0.0-20200708004538-1a94d8640e99
	github.com/julienschmidt/httprouter v1.3.0
	github.com/oklog/run v1.1.0
	github.com/opentracing/opentracing-go v1.2.0
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.7.1
	github.com/prometheus/common v0.13.0
	github.com/prometheus/prometheus v2.5.0+incompatible
	github.com/shurcooL/vfsgen v0.0.0-20200627165143-92b8a710ab6c
	github.com/spf13/pflag v1.0.5
	github.com/thanos-io/thanos v0.13.1-0.20200929105836-a30254dc9454
	golang.org/x/net v0.0.0-20200822124328-c89045814202
	google.golang.org/grpc v1.30.0
	gopkg.in/alecthomas/kingpin.v2 v2.2.6
	gopkg.in/yaml.v2 v2.3.0
)

replace github.com/prometheus/prometheus => github.com/prometheus/prometheus v1.8.2-0.20200922180708-b0145884d381

replace k8s.io/client-go => k8s.io/client-go v0.18.3
