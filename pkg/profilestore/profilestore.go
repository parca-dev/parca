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
	"github.com/prometheus/prometheus/pkg/labels"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage"
)

type ProfileStore struct {
	logger    log.Logger
	app       storage.Appendable
	metaStore storage.ProfileMetaStore
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileStore{}

func NewProfileStore(logger log.Logger, app storage.Appendable, metaStore storage.ProfileMetaStore) *ProfileStore {
	return &ProfileStore{
		logger:    logger,
		app:       app,
		metaStore: metaStore,
	}
}

func (s *ProfileStore) WriteRaw(ctx context.Context, r *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	for _, series := range r.Series {
		ls := make(labels.Labels, 0, len(series.Labels.Labels))
		for _, l := range series.Labels.Labels {
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

			profiles := storage.ProfilesFromPprof(s.metaStore, p)
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

				app, err := s.app.Appender(ctx, profLabelset)
				if err != nil {
					return nil, err
				}

				if err := app.Append(prof); err != nil {
					return nil, status.Errorf(codes.Internal, "failed to append sample: %v", err)
				}
			}
		}
	}

	return &profilestorepb.WriteRawResponse{}, nil
}
