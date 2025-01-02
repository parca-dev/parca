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
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/apache/arrow/go/v17/arrow/ipc"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"github.com/go-kit/log"
	"github.com/gogo/status"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
	otelgrpcprofilingpb "go.opentelemetry.io/proto/otlp/collector/profiles/v1experimental"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/ingester"
	"github.com/parca-dev/parca/pkg/normalizer"
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

	otelgrpcprofilingpb.UnimplementedProfilesServiceServer

	logger log.Logger
	tracer trace.Tracer

	ingester ingester.Ingester

	mtx sync.Mutex
	// ip as the key
	agents map[string]agent

	mem    memory.Allocator
	schema *dynparquet.Schema

	converterMetrics *normalizer.Metrics
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileColumnStore{}

func NewProfileColumnStore(
	reg prometheus.Registerer,
	logger log.Logger,
	tracer trace.Tracer,
	ingester ingester.Ingester,
	schema *dynparquet.Schema,
	mem memory.Allocator,
) *ProfileColumnStore {
	normalizerMetrics := normalizer.NewMetrics(reg)
	return &ProfileColumnStore{
		logger:   logger,
		tracer:   tracer,
		ingester: ingester,
		schema:   schema,
		mem:      mem,
		agents:   make(map[string]agent),

		converterMetrics: normalizerMetrics,
	}
}

func (s *ProfileColumnStore) writeSeries(ctx context.Context, req *profilestorepb.WriteRawRequest) error {
	r, err := normalizer.WriteRawRequestToArrowRecord(
		ctx,
		s.mem,
		req,
		s.schema,
	)
	if err != nil {
		return err
	}
	if r == nil {
		return nil
	}
	defer r.Release()

	if r.NumRows() == 0 {
		return nil
	}

	return s.ingester.Ingest(ctx, r)
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

func (s *ProfileColumnStore) Write(server profilestorepb.ProfileStoreService_WriteServer) error {
	ctx, cancel := context.WithTimeout(server.Context(), 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- s.write(ctx, server)
		close(errChan)
	}()

	select {
	case <-ctx.Done():
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case err := <-errChan:
		return err
	}
}

func (s *ProfileColumnStore) write(ctx context.Context, server profilestorepb.ProfileStoreService_WriteServer) error {
	req, err := server.Recv()
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive request: %v", err)
	}

	r, err := ipc.NewReader(bytes.NewReader(req.Record))
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to create reader: %v", err)
	}
	defer r.Release()

	if !r.Next() {
		return status.Error(codes.InvalidArgument, "no record found")
	}

	if r.Err() != nil {
		return status.Errorf(codes.InvalidArgument, "failed to read record: %v", r.Err())
	}

	c := normalizer.NewArrowToInternalConverter(
		s.mem,
		s.schema,
		s.converterMetrics,
	)
	defer c.Release()

	if err := c.AddSampleRecord(ctx, r.Record()); err != nil {
		return status.Error(codes.InvalidArgument, "failed to add sample record")
	}

	hasUnknownStacktraceIDs, err := c.HasUnknownStacktraceIDs()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to check unknown stacktrace IDs: %v", err)
	}
	if hasUnknownStacktraceIDs {
		rec, err := c.UnknownStacktraceIDsRecord()
		if err != nil {
			return status.Errorf(codes.Internal, "failed to get unknown stacktrace IDs record: %v", err)
		}

		buf := bytes.NewBuffer(nil)
		w := ipc.NewWriter(buf,
			ipc.WithSchema(rec.Schema()),
			ipc.WithAllocator(s.mem),
		)
		if err := w.Write(rec); err != nil {
			return status.Errorf(codes.Internal, "failed to write unknown stacktrace IDs record: %v", err)
		}

		if err := w.Close(); err != nil {
			return status.Errorf(codes.Internal, "failed to close writer")
		}

		if err := server.Send(&profilestorepb.WriteResponse{
			Record: buf.Bytes(),
		}); err != nil {
			return status.Errorf(codes.Internal, "failed to send unknown stacktrace IDs record")
		}

		req, err = server.Recv()
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to receive request: %v", err)
		}

		r, err = ipc.NewReader(bytes.NewReader(req.Record))
		if err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to create reader: %v", err)
		}

		if !r.Next() {
			return status.Error(codes.InvalidArgument, "no record found")
		}

		if r.Err() != nil {
			return status.Errorf(codes.InvalidArgument, "failed to read record: %v", r.Err())
		}

		if err := c.AddLocationsRecord(ctx, r.Record()); err != nil {
			return status.Errorf(codes.InvalidArgument, "failed to add locations record: %v", err)
		}
	}

	if err := c.Validate(); err != nil {
		return status.Errorf(codes.InvalidArgument, "validate record reader: %v", err)
	}

	ir, err := c.NewRecord(ctx)
	if err != nil {
		return fmt.Errorf("new record: %w", err)
	}

	if ir.NumRows() == 0 {
		return nil
	}

	if err := s.ingester.Ingest(ctx, ir); err != nil {
		return status.Errorf(codes.Internal, "failed to ingest record: %v", err)
	}

	return nil
}

func (s *ProfileColumnStore) Export(ctx context.Context, req *otelgrpcprofilingpb.ExportProfilesServiceRequest) (*otelgrpcprofilingpb.ExportProfilesServiceResponse, error) {
	r, err := normalizer.OtlpRequestToArrowRecord(
		ctx,
		req,
		s.schema,
		s.mem,
	)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return &otelgrpcprofilingpb.ExportProfilesServiceResponse{}, nil
	}
	defer r.Release()

	if r.NumRows() == 0 {
		return &otelgrpcprofilingpb.ExportProfilesServiceResponse{}, nil
	}

	if err := s.ingester.Ingest(ctx, r); err != nil {
		return nil, err
	}

	return &otelgrpcprofilingpb.ExportProfilesServiceResponse{}, nil
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
