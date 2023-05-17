// Copyright 2022-2023 The Parca Authors
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
	"bytes"
	"context"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/dynparquet"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/parcacol"
)

type agent struct {
	nodeName         string
	lastError        error
	lastPush         time.Time
	lastPushDuration time.Duration
}

type ProfileColumnStore struct {
	profilestorepb.UnimplementedProfileStoreServiceServer
	profilestorepb.UnimplementedAgentsServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	metastore metastorepb.MetastoreServiceClient

	table  *frostdb.Table
	schema *dynparquet.Schema
	// isAddrNormEnabled indicates whether the ingester has to
	// normalize sampled addresses for PIC/PIE (position independent code/executable).
	isAddrNormEnabled bool

	mtx sync.Mutex
	// ip as the key
	agents map[string]agent

	bufferPool *sync.Pool
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileColumnStore{}

func NewProfileColumnStore(
	logger log.Logger,
	tracer trace.Tracer,
	metastore metastorepb.MetastoreServiceClient,
	table *frostdb.Table,
	schema *dynparquet.Schema,
	enableAddressNormalization bool,
) *ProfileColumnStore {
	return &ProfileColumnStore{
		logger:            logger,
		tracer:            tracer,
		metastore:         metastore,
		table:             table,
		schema:            schema,
		isAddrNormEnabled: enableAddressNormalization,
		agents:            make(map[string]agent),
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

func (s *ProfileColumnStore) writeSeries(ctx context.Context, req *profilestorepb.WriteRawRequest) error {
	return parcacol.NormalizedIngest(ctx, req, s.logger, s.table, s.schema, s.metastore, s.bufferPool, s.isAddrNormEnabled)
}

func (s *ProfileColumnStore) updateAgents(nodeNameAndIP string, ag agent) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	s.agents[nodeNameAndIP] = ag

	for i, a := range s.agents {
		if a.lastPush.Before(time.Now().Add(-5 * time.Minute)) {
			delete(s.agents, i)
		}
	}
}

func nodeNameFromLabels(series []*profilestorepb.RawProfileSeries) (string, bool) {
	var nodeName string

found:
	for _, s := range series {
		for _, l := range s.Labels.Labels {
			if l.Name == "node" {
				nodeName = l.Value
				break found
			}
		}
	}

	if nodeName == "" {
		return "", false
	}

	return nodeName, true
}

func (s *ProfileColumnStore) WriteRaw(ctx context.Context, req *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	ctx, span := s.tracer.Start(ctx, "write-raw")
	defer span.End()

	start := time.Now()
	writeErr := s.writeSeries(ctx, req)

	// update agent info only when the request is come from agent
	if p, ok := peer.FromContext(ctx); ok && len(req.Series) != 0 {
		nodeName, _ := nodeNameFromLabels(req.Series)
		ag := agent{
			nodeName:         nodeName,
			lastPush:         start,
			lastPushDuration: time.Since(start),
			lastError:        writeErr,
		}
		ipPort := p.Addr.String()
		ip := ipPort[:strings.LastIndex(ipPort, ":")]

		s.updateAgents(nodeName+ip, ag)
	}

	if writeErr != nil {
		return nil, writeErr
	}

	return &profilestorepb.WriteRawResponse{}, nil
}

func (s *ProfileColumnStore) Agents(ctx context.Context, req *profilestorepb.AgentsRequest) (*profilestorepb.AgentsResponse, error) {
	s.mtx.Lock()
	defer s.mtx.Unlock()

	agents := make([]*profilestorepb.Agent, 0, len(s.agents))
	for nodeNameAndIP, ag := range s.agents {
		lastError := ""
		lerr := ag.lastError
		if lerr != nil {
			lastError = lerr.Error()
		}

		id := ag.nodeName
		if id == "" {
			id = nodeNameAndIP
		}

		agents = append(agents, &profilestorepb.Agent{
			Id:               id,
			LastError:        lastError,
			LastPush:         timestamppb.New(ag.lastPush),
			LastPushDuration: durationpb.New(ag.lastPushDuration),
		})
	}

	resp := &profilestorepb.AgentsResponse{
		Agents: agents,
	}

	return resp, nil
}
