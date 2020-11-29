// Copyright 2020 The conprof Authors
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

package e2econprof

import (
	"github.com/cortexproject/cortex/integration/e2e"
)

type Service struct {
	*e2e.HTTPService

	grpc int
}

func NewService(
	name string,
	image string,
	command *e2e.Command,
	readiness *e2e.HTTPReadinessProbe,
	http, grpc int,
	otherPorts ...int,
) *Service {
	return &Service{
		HTTPService: e2e.NewHTTPService(name, image, command, readiness, http, append(otherPorts, grpc)...),
		grpc:        grpc,
	}
}

func (s *Service) GRPCEndpoint() string { return s.Endpoint(s.grpc) }

func (s *Service) GRPCNetworkEndpoint() string {
	return s.NetworkEndpoint(s.grpc)
}

func (s *Service) GRPCNetworkEndpointFor(networkName string) string {
	return s.NetworkEndpointFor(networkName, s.grpc)
}
