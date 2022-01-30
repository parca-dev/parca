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
	"github.com/parca-dev/parca/pkg/columnstore"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/prometheus/prometheus/model/labels"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

type ProfileStore struct {
	profilestorepb.UnimplementedProfileStoreServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	metaStore metastore.ProfileMetaStore
	table     *columnstore.Table
}

var _ profilestorepb.ProfileStoreServiceServer = &ProfileStore{}

func NewProfileStore(
	logger log.Logger,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
) *ProfileStore {
	s := columnstore.New()
	db := s.DB("parca")
	table := db.Table("stacktraces") // TODO we need to define a schema here

	return &ProfileStore{
		logger:    logger,
		tracer:    tracer,
		metaStore: metaStore,
		table:     table,
	}
}

func (s *ProfileStore) WriteRaw(ctx context.Context, r *profilestorepb.WriteRawRequest) (*profilestorepb.WriteRawResponse, error) {
	ctx, span := s.tracer.Start(ctx, "write-raw")
	defer span.End()

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

			convertCtx, convertSpan := s.tracer.Start(ctx, "profile-from-pprof")
			profiles, err := parcaprofile.FlatProfilesFromPprof(convertCtx, s.logger, s.metaStore, p)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "failed to normalize pprof: %v", err)
			}
			convertSpan.End()

			_, appendSpan := s.tracer.Start(ctx, "append-profiles")
			for _, prof := range profiles {
				// TODO all of this should be done in the flat profile
				// extraction in the first place.
				labels := make([]columnstore.DynamicColumnValue, 0, len(ls))
				for _, l := range ls {
					labels = append(labels, columnstore.DynamicColumnValue{
						Name:  l.Name,
						Value: l.Value,
					})
				}

				rows := make([]*SampleRow, 0, len(prof.FlatSamples))
				for _, s := range prof.FlatSamples {
					rows = append(rows, &SampleRow{
						Stacktrace: metastoreLocationsToSampleStacktrace(s.Location),
						Value:      s.Value,
					})
				}

				level.Debug(s.logger).Log("msg", "writing sample", "label_set", ls.String(), "timestamp", prof.Meta.Timestamp)

				sortSampleRows(rows)
				err := s.table.Insert(makeRows(prof, labels, rows))
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to insert profile: %v", err)
				}
			}
			appendSpan.End()
		}
	}

	return &profilestorepb.WriteRawResponse{}, nil
}

func makeRows(prof *parcaprofile.FlatProfile, labels []columnstore.DynamicColumnValue, rows []*SampleRow) []columnstore.Row {
	res := make([]columnstore.Row, len(rows))
	for i, r := range rows {
		res[i] = columnstore.Row{
			Values: []interface{}{
				prof.Meta.SampleType.Type,
				prof.Meta.SampleType.Unit,
				prof.Meta.PeriodType.Type,
				prof.Meta.PeriodType.Unit,
				labels,
				r.Stacktrace,
				prof.Meta.Timestamp,
				prof.Meta.Duration,
				prof.Meta.Period,
				r.Value,
			},
		}
	}

	return res
}

func metastoreLocationsToSampleStacktrace(locs []*metastore.Location) []columnstore.UUID {
	length := len(locs) - 1
	stacktrace := make([]columnstore.UUID, length+1)
	for i := range locs {
		cUUID := columnstore.UUID(locs[length-i].ID)
		stacktrace[i] = cUUID
	}

	return stacktrace
}

type SampleRow struct {
	// Array of Location IDs.
	Stacktrace []columnstore.UUID

	PprofStringLabels  map[string]string
	PprofNumLabels     map[string]int64
	PprofNumLabelUnits map[string]string

	Value int64
}

func sortSampleRows(samples []*SampleRow) {
	sort.Slice(samples, func(i, j int) bool {
		// TODO need to take labels into account
		return stacktraceLess(samples[i].Stacktrace, samples[j].Stacktrace)
	})
}

func stacktraceLess(stacktrace1, stacktrace2 []columnstore.UUID) bool {
	stacktrace1Len := len(stacktrace1)
	stacktrace2Len := len(stacktrace2)

	k := 0
	for {
		switch {
		case k >= stacktrace1Len && k <= stacktrace2Len:
			// This means the stacktraces are identical up until this point, but stacktrace1 is ending, and shorter stactraces are "smaller" than longer ones.
			return true
		case k <= stacktrace1Len && k >= stacktrace2Len:
			// This means the stacktraces are identical up until this point, but stacktrace2 is ending, and shorter stactraces are "lower" than longer ones.
			return false
		case uuidCompare(stacktrace1[k], stacktrace2[k]) == -1:
			return true
		case uuidCompare(stacktrace1[k], stacktrace2[k]) == 1:
			return false
		default:
			// This means the stack traces are identical up until this point. So advance to the next.
			k++
		}
	}
}

func uuidCompare(a, b columnstore.UUID) int {
	ab := [16]byte(a)
	bb := [16]byte(b)
	return bytes.Compare(ab[:], bb[:])
}
