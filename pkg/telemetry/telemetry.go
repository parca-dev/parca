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

package telemetry

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"google.golang.org/grpc/peer"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/telemetry/v1alpha1"
)

type TelemetryAPI struct {
	pb.UnimplementedTelemetryServiceServer

	logger log.Logger
}

func NewTelemetry(logger log.Logger) *TelemetryAPI {
	return &TelemetryAPI{
		logger: logger,
	}
}

// ReportPanic reports a panic experienced by Agents. Just now it just logs the last
// few KBs of stderr and the metadata.
func (t *TelemetryAPI) ReportPanic(ctx context.Context, req *pb.ReportPanicRequest) (*pb.ReportPanicResponse, error) {
	ip, err := extractIPFromContext(ctx)
	if err != nil {
		ip = "unknown"
	}
	level.Info(t.logger).Log(
		"msg", "agent panic'ed with",
		"stderr", req.Stderr,
		"metadata", fmt.Sprintf("%v", req.Metadata),
		"agent_ip", ip,
	)
	return &pb.ReportPanicResponse{}, nil
}

// extractIPFromContext extracts the IP address of the agent from the context.
func extractIPFromContext(ctx context.Context) (string, error) {
	var ip string
	if p, ok := peer.FromContext(ctx); ok {
		ipPort := p.Addr.String()
		if colon := strings.LastIndex(ipPort, ":"); colon != -1 {
			ip = ipPort[:colon]
		}
		return ip, nil
	}
	return "", fmt.Errorf("failed to extract IP from context")
}
