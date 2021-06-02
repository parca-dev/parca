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

package store

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/conprof/conprof/pkg/runutil"
	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/go-kit/kit/log"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var tracer = otel.Tracer("store-server")

type db interface {
	storage.Queryable
	storage.ChunkQueryable
	storage.Appendable
}

type profileStore struct {
	logger           log.Logger
	db               db
	maxBytesPerFrame int
}

func RegisterSymbolStore(storeSrv storepb.SymbolStoreServer) func(*grpc.Server) {
	return func(s *grpc.Server) {
		if storeSrv != nil {
			storepb.RegisterSymbolStoreServer(s, storeSrv)
		}
	}
}

func RegisterReadableStoreServer(storeSrv storepb.ReadableProfileStoreServer) func(*grpc.Server) {
	return func(s *grpc.Server) {
		storepb.RegisterReadableProfileStoreServer(s, storeSrv)
	}
}

func RegisterWritableStoreServer(storeSrv storepb.WritableProfileStoreServer) func(*grpc.Server) {
	return func(s *grpc.Server) {
		storepb.RegisterWritableProfileStoreServer(s, storeSrv)
	}
}

func NewProfileStore(logger log.Logger, db db, maxBytesPerFrame int) *profileStore {
	return &profileStore{
		logger:           logger,
		db:               db,
		maxBytesPerFrame: maxBytesPerFrame,
	}
}

var _ storepb.ReadableProfileStoreServer = &profileStore{}
var _ storepb.WritableProfileStoreServer = &profileStore{}

func (s *profileStore) Write(ctx context.Context, r *storepb.WriteRequest) (*storepb.WriteResponse, error) {
	app := s.db.Appender(ctx)
	for _, series := range r.ProfileSeries {
		ls := make(labels.Labels, 0, len(series.Labels))
		for _, l := range series.Labels {
			ls = append(ls, labels.Label{
				Name:  l.Name,
				Value: l.Value,
			})
		}
		// Sorting must be ensured at insertion time.
		sort.Sort(ls)

		for _, sample := range series.Samples {
			_, err := app.Add(ls, sample.Timestamp, sample.Value)
			if err != nil {
				return nil, err
			}
		}
	}

	return &storepb.WriteResponse{}, app.Commit()
}

func (s *profileStore) Profile(ctx context.Context, r *storepb.ProfileRequest) (*storepb.ProfileResponse, error) {
	q, err := s.db.Querier(ctx, r.Timestamp, r.Timestamp)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	m, err := translatePbMatchers(r.Matchers)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not translate matchers: %v", err)
	}

	ss := q.Select(false, nil, m...)
	ok := ss.Next()
	if !ok {
		return nil, status.Error(codes.NotFound, "profile series not found")
	}

	i := ss.At().Iterator()
	ok = i.Seek(r.Timestamp)
	if !ok {
		return nil, errors.New("profile not found")
	}

	_, buf := i.At()
	return &storepb.ProfileResponse{
		Data: buf,
	}, nil
}

func (s *profileStore) Series(r *storepb.SeriesRequest, srv storepb.ReadableProfileStore_SeriesServer) error {
	ctx := srv.Context()

	if r.SelectHints != nil && r.SelectHints.Func == "series" {
		return s.noopChunks(r, srv)
	}

	m, err := translatePbMatchers(r.Matchers)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "could not translate matchers: %v", err)
	}

	q, err := s.db.ChunkQuerier(ctx, r.MinTime, r.MaxTime)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer runutil.CloseWithLogOnErr(s.logger, q, "close tsdb chunk querier series")

	_, span := tracer.Start(ctx, "iterate-chunk-series-set")
	if r.SelectHints != nil {
		span.SetAttributes(attribute.Key("select-hint-min").Int64(r.SelectHints.Start))
		span.SetAttributes(attribute.Key("select-hint-max").Int64(r.SelectHints.End))
		span.SetAttributes(attribute.Key("select-hint-func").String(r.SelectHints.Func))
	}
	defer span.End()

	set := q.Select(false, storepb.TsdbSelectHints(r.SelectHints), m...)

	var (
		it chunkenc.Iterator = nil
	)

	for set.Next() {
		series := set.At()
		labels := labelpb.LabelsFromPromLabels(series.Labels())
		bytesLeftForChunks := s.maxBytesPerFrame
		for _, lbl := range labels {
			bytesLeftForChunks -= lbl.Size()
		}
		frameBytesLeft := bytesLeftForChunks

		seriesChunks := []storepb.AggrChunk{}

		chIter := series.Iterator()
		isNext := chIter.Next()
		for isNext {
			chk := chIter.At()
			if chk.Chunk == nil {
				return status.Errorf(codes.Internal, "TSDBStore: found not populated chunk returned by SeriesSet at ref: %v", chk.Ref)
			}

			tc := chk.Chunk
			if r.SelectHints != nil && r.SelectHints.Func == "timestamps" {
				it, tc, err = tsdb.ReencodeChunk(&tsdb.TimestampChunk{Chunk: tc}, it)
				if err != nil {
					return status.Error(codes.Aborted, err.Error())
				}
			}

			tcBytes, err := tc.Bytes()
			if err != nil {
				return err
			}

			c := storepb.AggrChunk{
				MinTime: chk.MinTime,
				MaxTime: chk.MaxTime,
				Raw: &storepb.Chunk{
					Type: storepb.Chunk_Encoding(chk.Chunk.Encoding() - 1), // Proto chunk encoding is one off to TSDB one.
					Data: tcBytes,
				},
			}
			frameBytesLeft -= c.Size()
			seriesChunks = append(seriesChunks, c)

			// We are fine with minor inaccuracy of max bytes per frame. The inaccuracy will be max of full chunk size.
			isNext = chIter.Next()
			if frameBytesLeft > 0 && isNext {
				continue
			}
			if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{Labels: labels, Chunks: seriesChunks})); err != nil {
				return status.Error(codes.Aborted, err.Error())
			}

			if isNext {
				frameBytesLeft = bytesLeftForChunks
				seriesChunks = make([]storepb.AggrChunk, 0, len(seriesChunks))
			}
		}
		if err := chIter.Err(); err != nil {
			return status.Error(codes.Internal, fmt.Errorf("chunk iter: %w", err).Error())
		}
	}

	if err := set.Err(); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	for _, w := range set.Warnings() {
		if err := srv.Send(storepb.NewWarnSeriesResponse(w)); err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}
	return nil
}

func (s *profileStore) noopChunks(r *storepb.SeriesRequest, srv storepb.ReadableProfileStore_SeriesServer) error {
	ctx := srv.Context()

	m, err := translatePbMatchers(r.Matchers)
	if err != nil {
		return status.Errorf(codes.InvalidArgument, "could not translate matchers: %v", err)
	}

	q, err := s.db.Querier(ctx, r.MinTime, r.MaxTime)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}
	defer runutil.CloseWithLogOnErr(s.logger, q, "close tsdb chunk querier series")

	_, span := tracer.Start(ctx, "iterate-series-set-noop-chunks")
	defer span.End()

	set := q.Select(false, storepb.TsdbSelectHints(r.SelectHints), m...)

	for set.Next() {
		series := set.At()
		labels := labelpb.LabelsFromPromLabels(series.Labels())
		if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{Labels: labels})); err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}

	if err := set.Err(); err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	for _, w := range set.Warnings() {
		if err := srv.Send(storepb.NewWarnSeriesResponse(w)); err != nil {
			return status.Error(codes.Aborted, err.Error())
		}
	}
	return nil
}

func (s *profileStore) LabelNames(ctx context.Context, r *storepb.LabelNamesRequest) (*storepb.LabelNamesResponse, error) {
	q, err := s.db.Querier(ctx, r.Start, r.End)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer runutil.CloseWithLogOnErr(s.logger, q, "close tsdb chunk querier series")

	labelNames, warnings, err := q.LabelNames()

	warningStrings := make([]string, 0, len(warnings))
	for _, w := range warnings {
		warningStrings = append(warningStrings, w.Error())
	}

	return &storepb.LabelNamesResponse{
		Names:    labelNames,
		Warnings: warningStrings,
	}, err
}

func (s *profileStore) LabelValues(ctx context.Context, r *storepb.LabelValuesRequest) (*storepb.LabelValuesResponse, error) {
	q, err := s.db.Querier(ctx, r.Start, r.End)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	defer runutil.CloseWithLogOnErr(s.logger, q, "close tsdb chunk querier series")

	labelNames, warnings, err := q.LabelValues(r.Label)

	warningStrings := make([]string, 0, len(warnings))
	for _, w := range warnings {
		warningStrings = append(warningStrings, w.Error())
	}

	return &storepb.LabelValuesResponse{
		Values:   labelNames,
		Warnings: warningStrings,
	}, err
}

func translatePbMatchers(ms []storepb.LabelMatcher) (res []*labels.Matcher, err error) {
	for _, m := range ms {
		r, err := translatePbMatcher(m)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func translatePbMatcher(m storepb.LabelMatcher) (*labels.Matcher, error) {
	switch m.Type {
	case storepb.LabelMatcher_EQ:
		return labels.NewMatcher(labels.MatchEqual, m.Name, m.Value)

	case storepb.LabelMatcher_NEQ:
		return labels.NewMatcher(labels.MatchNotEqual, m.Name, m.Value)

	case storepb.LabelMatcher_RE:
		return labels.NewMatcher(labels.MatchRegexp, m.Name, m.Value)

	case storepb.LabelMatcher_NRE:
		return labels.NewMatcher(labels.MatchNotRegexp, m.Name, m.Value)
	}
	return nil, fmt.Errorf("unknown label matcher type %d", m.Type)
}

func translatePromMatchers(ms []*labels.Matcher) (res []storepb.LabelMatcher, err error) {
	for _, m := range ms {
		r, err := translatePromMatcher(m)
		if err != nil {
			return nil, err
		}
		res = append(res, r)
	}
	return res, nil
}

func translatePromMatcher(m *labels.Matcher) (storepb.LabelMatcher, error) {
	switch m.Type {
	case labels.MatchEqual:
		return storepb.LabelMatcher{
			Type:  storepb.LabelMatcher_EQ,
			Name:  m.Name,
			Value: m.Value,
		}, nil

	case labels.MatchNotEqual:
		return storepb.LabelMatcher{
			Type:  storepb.LabelMatcher_NEQ,
			Name:  m.Name,
			Value: m.Value,
		}, nil

	case labels.MatchRegexp:
		return storepb.LabelMatcher{
			Type:  storepb.LabelMatcher_RE,
			Name:  m.Name,
			Value: m.Value,
		}, nil

	case labels.MatchNotRegexp:
		return storepb.LabelMatcher{
			Type:  storepb.LabelMatcher_NRE,
			Name:  m.Name,
			Value: m.Value,
		}, nil
	}
	return storepb.LabelMatcher{}, fmt.Errorf("unknown label matcher type %d", m.Type)
}
