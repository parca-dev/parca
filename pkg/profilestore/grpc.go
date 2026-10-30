// Copyright 2022-2026 The Parca Authors
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
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	pprof "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/translator/pprof"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.opentelemetry.io/collector/pdata/pprofile/pprofileotlp"
	otelgrpcprofilingpb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/proto"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
)

// GRPCForwarder forwards profiles via gRPC to a remote store,
// optionally converting raw pprof profiles to OTLP.
type GRPCForwarder struct {
	logger        log.Logger
	client        *Client
	convertToOTLP bool

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

func NewGRPCForwarder(client *Client, logger log.Logger, convertToOTLP bool) *GRPCForwarder {
	return &GRPCForwarder{
		client:        client,
		logger:        logger,
		convertToOTLP: convertToOTLP,
	}
}

func (s *GRPCForwarder) WriteRaw(ctx context.Context, req *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	if s.convertToOTLP {
		return s.writeRawOTLP(ctx, req)
	}
	// TODO: Batch writes to only send a request every now and then.
	// See https://github.com/parca-dev/parca-agent/blob/main/pkg/agent/write_client.go#L28
	resp, err := s.client.WriteRaw(ctx, req)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to forward profiles", "err", err)
	}
	return resp, err
}

func (s *GRPCForwarder) writeRawOTLP(ctx context.Context, req *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	profiles := pprofile.NewProfiles()
	for _, series := range req.GetSeries() {
		for _, sample := range series.GetSamples() {
			p, err := profile.ParseData(sample.GetRawProfile())
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "parse pprof profile: %v", err)
			}
			// A period type is optional in pprof, but the translator requires one.
			if p.PeriodType == nil {
				p.PeriodType = &profile.ValueType{}
			}
			converted, err := pprof.ConvertPprofToProfiles(p)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "convert pprof profile to OTLP: %v", err)
			}
			for _, resource := range converted.ResourceProfiles().All() {
				// Target and external labels describe the resource. Labels inside
				// the pprof data remain sample attributes, as set by the translator.
				for _, label := range series.GetLabels().GetLabels() {
					resource.Resource().Attributes().PutStr(label.GetName(), label.GetValue())
				}
				for _, scope := range resource.ScopeProfiles().All() {
					for _, p := range scope.Profiles().All() {
						p.SetProfileID(pprofile.ProfileID(uuid.New()))
					}
				}
			}
			if err := converted.MergeTo(profiles); err != nil {
				return nil, status.Errorf(codes.Internal, "merge OTLP profiles: %v", err)
			}
		}
	}
	if profiles.ProfileCount() == 0 {
		return &profilestorepb.WriteRawResponse{}, nil
	}

	// Collector pdata uses its own protobuf types. Bridge through the wire
	// format to reuse the existing OTLP client and its connection settings.
	data, err := pprofileotlp.NewExportRequestFromProfiles(profiles).MarshalProto()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "marshal OTLP profiles: %v", err)
	}
	export := &otelgrpcprofilingpb.ExportProfilesServiceRequest{}
	if err := proto.Unmarshal(data, export); err != nil {
		return nil, status.Errorf(codes.Internal, "unmarshal OTLP profiles: %v", err)
	}
	resp, err := s.Export(ctx, export)
	if err != nil {
		return nil, err
	}
	if partial := resp.GetPartialSuccess(); partial.GetRejectedProfiles() > 0 {
		return nil, status.Errorf(codes.DataLoss, "upstream rejected %d profiles: %s", partial.GetRejectedProfiles(), partial.GetErrorMessage())
	}
	return &profilestorepb.WriteRawResponse{}, nil
}

func (s *GRPCForwarder) Write(srv profilestorepb.ProfileStoreService_WriteServer) error {
	if s.convertToOTLP {
		return status.Error(codes.Unimplemented, "OTLP forwarding supports pprof WriteRaw and OTLP Export, not Arrow Write")
	}
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
