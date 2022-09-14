// Copyright 2022 The Parca Authors
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

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/grpc"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
)

// GRPCForwarder forward profiles via gRPC to another Parca instance
// instead of storing the profiles locally in a database.
type GRPCForwarder struct {
	logger log.Logger
	client profilestorepb.ProfileStoreServiceClient

	profilestorepb.UnimplementedProfileStoreServiceServer
}

func NewGRPCForwarder(conn grpc.ClientConnInterface, logger log.Logger) *GRPCForwarder {
	return &GRPCForwarder{
		client: profilestorepb.NewProfileStoreServiceClient(conn),
		logger: logger,
	}
}

func (s *GRPCForwarder) WriteRaw(ctx context.Context, req *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	// TODO: Batch writes to only send a request every now and then.
	// See https://github.com/parca-dev/parca-agent/blob/main/pkg/agent/write_client.go#L28
	resp, err := s.client.WriteRaw(ctx, req)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to forward profiles", "err", err)
	}
	return resp, err
}
