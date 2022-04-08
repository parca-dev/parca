// Copyright 2021 The Parca Authors
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
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/polarsignals/arcticdb"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

type ProfileColumnStore struct {
	profilestorepb.UnimplementedProfileStoreServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	metaStore metastore.ProfileMetaStore

	table *arcticdb.Table

	// When the debug-value-log is enabled, every profile is first written to
	// tmp/<labels>/<timestamp>.pb.gz before it's parsed and written to the
	// columnstore. This is primarily for debugging purposes as well as
	// reproducing situations in tests. This has huge overhead, do not enable
	// unless you know what you're doing.
	debugValueLog bool
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileColumnStore{}

func NewProfileColumnStore(
	logger log.Logger,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
	table *arcticdb.Table,
	debugValueLog bool,
) *ProfileColumnStore {
	return &ProfileColumnStore{
		logger:        logger,
		tracer:        tracer,
		metaStore:     metaStore,
		table:         table,
		debugValueLog: debugValueLog,
	}
}

func (s *ProfileColumnStore) WriteRaw(ctx context.Context, r *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	ctx, span := s.tracer.Start(ctx, "write-raw")
	defer span.End()

	for _, series := range r.Series {
		ls := make(labels.Labels, 0, len(series.Labels.Labels))
		for _, l := range series.Labels.Labels {
			if valid := model.LabelName(l.Name).IsValid(); !valid {
				return nil, status.Errorf(codes.InvalidArgument, "invalid label name: %v", l.Name)
			}

			ls = append(ls, labels.Label{
				Name:  l.Name,
				Value: l.Value,
			})
		}

		for _, sample := range series.Samples {
			p, err := profile.Parse(bytes.NewBuffer(sample.RawProfile))
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
			}

			if s.debugValueLog {
				dir := fmt.Sprintf("tmp/%s", base64.URLEncoding.EncodeToString([]byte(ls.String())))
				err := os.MkdirAll(dir, os.ModePerm)
				if err != nil {
					level.Error(s.logger).Log("msg", "failed to create debug-value-log directory", "err", err)
				} else {
					err := ioutil.WriteFile(fmt.Sprintf("%s/%d.pb.gz", dir, timestamp.FromTime(time.Now())), sample.RawProfile, 0o644)
					if err != nil {
						level.Error(s.logger).Log("msg", "failed to write debug-value-log", "err", err)
					}
				}
			}

			if err := p.CheckValid(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
			}

			convertCtx, convertSpan := s.tracer.Start(ctx, "profile-from-pprof")
			profiles, err := parcaprofile.ProfilesFromPprof(convertCtx, s.logger, s.metaStore, p, r.Normalized)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to normalize pprof: %v", err)
			}
			convertSpan.End()

			_, appendSpan := s.tracer.Start(ctx, "append-profiles")
			for _, prof := range profiles {
				_, err := parcacol.InsertProfileIntoTable(ctx, s.logger, s.table, ls, prof)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to insert profile: %v", err)
				}
			}
			appendSpan.End()
		}
	}

	return &profilestorepb.WriteRawResponse{}, nil
}
