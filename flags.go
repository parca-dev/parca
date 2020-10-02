package main

import (
	"github.com/prometheus/common/model"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"gopkg.in/alecthomas/kingpin.v2"
)

func modelDuration(flags *kingpin.FlagClause) *model.Duration {
	value := new(model.Duration)
	flags.SetValue(value)

	return value
}

func regGRPCFlags(cmd extkingpin.FlagClause) (
	grpcBindAddr *string,
	grpcGracePeriod *model.Duration,
	grpcTLSSrvCert *string,
	grpcTLSSrvKey *string,
	grpcTLSSrvClientCA *string,
) {
	grpcBindAddr = cmd.Flag("grpc-address", "Listen ip:port address for gRPC endpoints (StoreAPI). Make sure this address is routable from other components.").
		Default("0.0.0.0:10901").String()
	grpcGracePeriod = modelDuration(cmd.Flag("grpc-grace-period", "Time to wait after an interrupt received for GRPC Server.").Default("2m")) // by default it's the same as query.timeout.

	grpcTLSSrvCert = cmd.Flag("grpc-server-tls-cert", "TLS Certificate for gRPC server, leave blank to disable TLS").Default("").String()
	grpcTLSSrvKey = cmd.Flag("grpc-server-tls-key", "TLS Key for the gRPC server, leave blank to disable TLS").Default("").String()
	grpcTLSSrvClientCA = cmd.Flag("grpc-server-tls-client-ca", "TLS CA to verify clients against. If no client CA is specified, there is no client verification on server side. (tls.NoClientCert)").Default("").String()

	return grpcBindAddr,
		grpcGracePeriod,
		grpcTLSSrvCert,
		grpcTLSSrvKey,
		grpcTLSSrvClientCA
}

func regHTTPFlags(cmd extkingpin.FlagClause) (httpBindAddr *string, httpGracePeriod *model.Duration) {
	httpBindAddr = cmd.Flag("http-address", "Listen host:port for HTTP endpoints.").Default("0.0.0.0:10902").String()
	httpGracePeriod = modelDuration(cmd.Flag("http-grace-period", "Time to wait after an interrupt received for HTTP Server.").Default("2m")) // by default it's the same as query.timeout.

	return httpBindAddr, httpGracePeriod
}
