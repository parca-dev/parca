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

package telemetry

import (
	"context"
	"fmt"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

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
	level.Info(t.logger).Log("msg", "agent panic'ed with", "stderr", req.Stderr, "metadata", fmt.Sprintf("%v", req.Metadata))
	return &pb.ReportPanicResponse{}, nil
}
