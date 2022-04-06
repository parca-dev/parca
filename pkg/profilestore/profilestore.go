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
	"sort"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/parca-dev/parca/pkg/metastore"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage"
)

type ProfileStore struct {
	profilestorepb.UnimplementedProfileStoreServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	app       storage.Appendable
	metaStore metastore.ProfileMetaStore
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileStore{}

func NewProfileStore(
	logger log.Logger,
	tracer trace.Tracer,
	app storage.Appendable,
	metaStore metastore.ProfileMetaStore,
) *ProfileStore {
	return &ProfileStore{
		logger:    logger,
		tracer:    tracer,
		app:       app,
		metaStore: metaStore,
	}
}

func (s *ProfileStore) WriteRaw(ctx context.Context, r *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
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

			if err := p.CheckValid(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
			}

			convertCtx, convertSpan := s.tracer.Start(ctx, "profile-from-pprof")
			profiles, err := parcaprofile.ProfilesFromPprof(convertCtx, s.logger, s.metaStore, p, r.Normalized)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to normalize pprof: %v", err)
			}
			convertSpan.End()

			appendCtx, appendSpan := s.tracer.Start(ctx, "append-profiles")
			for _, prof := range profiles {
				profLabelset := ls.Copy()
				found := false
				for i, label := range profLabelset {
					if label.Name == "__name__" {
						found = true
						profLabelset[i] = labels.Label{
							Name:  "__name__",
							Value: label.Value + "_" + prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
						}
					}
				}
				if !found {
					profLabelset = append(profLabelset, labels.Label{
						Name:  "__name__",
						Value: prof.Meta.SampleType.Type + "_" + prof.Meta.SampleType.Unit,
					})
				}
				sort.Sort(profLabelset)

				level.Debug(s.logger).Log("msg", "writing sample", "label_set", profLabelset.String(), "timestamp", prof.Meta.Timestamp)

				app, err := s.app.Appender(appendCtx, profLabelset)
				if err != nil {
					return nil, err
				}

				if err := app.AppendFlat(appendCtx, prof); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to append sample: %v", err)
				}
			}
			appendSpan.End()
		}
	}

	return &profilestorepb.WriteRawResponse{}, nil
}
