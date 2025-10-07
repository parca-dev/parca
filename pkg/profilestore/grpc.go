// Copyright 2022-2025 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package profilestore

import (
	"context"
	"io"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/gogo/status"
	"github.com/prometheus/client_golang/prometheus"
	otelgrpcprofilingpb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/grpc/codes"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
)

// GRPCForwarder forward profiles via gRPC to another Parca instance
// instead of storing the profiles locally in a database.
type GRPCForwarder struct {
	logger         log.Logger
	forwardedBytes prometheus.Counter

	client *Client

	profilestorepb.UnimplementedProfileStoreServiceServer
	otelgrpcprofilingpb.UnimplementedProfilesServiceServer
}

type Client struct {
	profilestorepb.ProfileStoreServiceClient
	otelgrpcprofilingpb.ProfilesServiceClient
}

func NewClient(
	profilestoreClient profilestorepb.ProfileStoreServiceClient,
	otelgrpcprofilingClient otelgrpcprofilingpb.ProfilesServiceClient,
) *Client {
	return &Client{
		ProfileStoreServiceClient: profilestoreClient,
		ProfilesServiceClient:     otelgrpcprofilingClient,
	}
}

func NewGRPCForwarder(client *Client, logger log.Logger, reg *prometheus.Registry) *GRPCForwarder {
	forwardedBytes := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "parca_forwarded_bytes_total",
		Help: "The number of profile bytes forwarded to a remote server server.",
	})
	reg.MustRegister(forwardedBytes)
	forwardedBytes.Add(0)

	return &GRPCForwarder{
		client:         client,
		logger:         logger,
		forwardedBytes: forwardedBytes,
	}
}

func (s *GRPCForwarder) WriteRaw(ctx context.Context, req *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	// TODO: Batch writes to only send a request every now and then.
	// See https://github.com/parca-dev/parca-agent/blob/main/pkg/agent/write_client.go#L28
	resp, err := s.client.WriteRaw(ctx, req)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to forward profiles", "err", err)
		return resp, err
	}

	var written int
	for _, series := range req.GetSeries() {
		for _, samples := range series.GetSamples() {
			written += len(samples.RawProfile)
		}
	}

	s.forwardedBytes.Add(float64(written))

	return resp, nil
}

func (s *GRPCForwarder) Write(srv profilestorepb.ProfileStoreService_WriteServer) error {
	c, err := s.client.Write(srv.Context())
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create stream: %v", err)
	}

	for {
		r, err := srv.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}

			return status.Errorf(codes.Internal, "failed to receive profile: %v", err)
		}
		if err := c.Send(r); err != nil {
			return status.Errorf(codes.Internal, "failed to send profile: %v", err)
		}
	}

	if err := c.CloseSend(); err != nil {
		return status.Errorf(codes.Internal, "failed to close send: %v", err)
	}

	return nil
}

func (s *GRPCForwarder) Export(ctx context.Context, req *otelgrpcprofilingpb.ExportProfilesServiceRequest) (*otelgrpcprofilingpb.ExportProfilesServiceResponse, error) {
	// TODO: Batch writes to only send a request every now and then.
	// See https://github.com/parca-dev/parca-agent/blob/main/pkg/agent/write_client.go#L28
	resp, err := s.client.Export(ctx, req)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to forward profiles", "err", err)
	}
	return resp, err
}
