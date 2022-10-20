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
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
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

	// When the debug-value-log is enabled, every profile is first written to
	// tmp/<labels>/<timestamp>.pb.gz before it's parsed and written to the
	// columnstore. This is primarily for debugging purposes as well as
	// reproducing situations in tests. This has huge overhead, do not enable
	// unless you know what you're doing.
	debugValueLog bool

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
	debugValueLog bool,
) *ProfileColumnStore {
	return &ProfileColumnStore{
		logger:        logger,
		tracer:        tracer,
		metastore:     metastore,
		table:         table,
		debugValueLog: debugValueLog,
		schema:        schema,
		agents:        make(map[string]agent),
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

func (s *ProfileColumnStore) writeSeries(ctx context.Context, req *profilestorepb.WriteRawRequest) error {
	ingester := parcacol.NewIngester(
		s.logger,
		parcacol.NewNormalizer(s.metastore),
		s.table,
		s.schema,
		s.bufferPool,
	)

	for _, series := range req.Series {
		ls := make(labels.Labels, 0, len(series.Labels.Labels))
		for _, l := range series.Labels.Labels {
			if valid := model.LabelName(l.Name).IsValid(); !valid {
				return status.Errorf(codes.InvalidArgument, "invalid label name: %v", l.Name)
			}

			ls = append(ls, labels.Label{
				Name:  l.Name,
				Value: l.Value,
			})
		}

		// Must ensure label-set is sorted and HasDuplicateLabelNames also required a sorted label-set
		sort.Sort(ls)
		if name, has := ls.HasDuplicateLabelNames(); has {
			return status.Errorf(codes.InvalidArgument, "duplicate label names: %v", name)
		}

		for _, sample := range series.Samples {
			r, err := gzip.NewReader(bytes.NewBuffer(sample.RawProfile))
			if err != nil {
				return status.Errorf(codes.Internal, "failed to create gzip reader: %v", err)
			}

			content, err := io.ReadAll(r)
			if err != nil {
				return status.Errorf(codes.InvalidArgument, "failed to decompress profile: %v", err)
			}

			p := &pprofpb.Profile{}
			if err := p.UnmarshalVT(content); err != nil {
				return status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
			}

			if s.debugValueLog {
				dir := fmt.Sprintf("tmp/%s", base64.URLEncoding.EncodeToString([]byte(ls.String())))
				err := os.MkdirAll(dir, os.ModePerm)
				if err != nil {
					level.Error(s.logger).Log("msg", "failed to create debug-value-log directory", "err", err)
				} else {
					err := os.WriteFile(fmt.Sprintf("%s/%d.pb.gz", dir, timestamp.FromTime(time.Now())), sample.RawProfile, 0o644)
					if err != nil {
						level.Error(s.logger).Log("msg", "failed to write debug-value-log", "err", err)
					}
				}
			}

			if err := ingester.Ingest(ctx, ls, p, req.Normalized); err != nil {
				return status.Errorf(codes.Internal, "failed to ingest profile: %v", err)
			}
		}
	}

	return nil
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
