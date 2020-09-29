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
	"fmt"
	"io"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/storage"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
)

type grpcStoreClient struct {
	c storepb.ProfileStoreClient
}

func NewGRPCQueryable(c storepb.ProfileStoreClient) *grpcStoreClient {
	return &grpcStoreClient{
		c: c,
	}
}

func (c *grpcStoreClient) Querier(ctx context.Context, mint, maxt int64) (storage.Querier, error) {
	return &grpcStoreQuerier{
		ctx:  ctx,
		mint: mint,
		maxt: maxt,
		c:    c.c,
	}, nil
}

type grpcStoreQuerier struct {
	ctx        context.Context
	mint, maxt int64
	c          storepb.ProfileStoreClient
}

func (q *grpcStoreQuerier) Select(sortSeries bool, hints *storage.SelectHints, matchers ...*labels.Matcher) storage.SeriesSet {
	ss := &grpcSeriesSet{}

	m, err := translatePromMatchers(matchers)
	if err != nil {
		ss.err = fmt.Errorf("translate prom matchers: %w", err)
		return ss
	}

	ss.stream, err = q.c.Series(q.ctx, &storepb.SeriesRequest{
		MinTime:  q.mint,
		MaxTime:  q.maxt,
		Matchers: m,
	})
	if err != nil {
		ss.err = fmt.Errorf("series: %w", err)
		return ss
	}
	return ss
}

type grpcSeriesSet struct {
	stream    storepb.ProfileStore_SeriesClient
	curSeries *protoSeries
	err       error
}

func (s *grpcSeriesSet) Next() bool {
	if s.stream == nil || s.err != nil {
		return false
	}

	res, err := s.stream.Recv()
	if err != nil {
		if err != io.EOF {
			s.err = fmt.Errorf("receive from stream: %w", err)
		}
		return false
	}

	s.curSeries = &protoSeries{s: res.GetSeries()}

	return true
}

type protoSeries struct {
	s *storepb.RawProfileSeries
}

func (s *protoSeries) Labels() labels.Labels {
	return translatePbLabels(s.s.Labels)
}

func (s *protoSeries) Iterator() chunkenc.Iterator {
	return &rawChunkIterator{chunks: s.s.Chunks, pos: -1}
}

type rawChunkIterator struct {
	chunks []storepb.Chunk
	curIt  chunkenc.Iterator
	pos    int
	err    error
}

func (s *rawChunkIterator) Next() bool {
	if s.curIt != nil && s.curIt.Next() {
		return true
	}

	if (s.pos + 1) == len(s.chunks) {
		// No more chunks read.
		return false
	}

	s.pos++
	c, err := chunkenc.FromData(chunkenc.EncBytes, s.chunks[s.pos].Data)
	if err != nil {
		s.err = fmt.Errorf("decode chunk: %w", err)
		return false
	}
	s.curIt = c.Iterator(nil)

	return s.curIt.Next()
}

func (s *rawChunkIterator) Seek(t int64) bool {
	for i, c := range s.chunks {
		if c.MinTime <= t && c.MaxTime >= t {
			s.pos = i
			c, err := chunkenc.FromData(chunkenc.EncBytes, s.chunks[s.pos].Data)
			if err != nil {
				s.err = fmt.Errorf("decode chunk: %w", err)
				return false
			}
			s.curIt = c.Iterator(nil)
			return s.curIt.Seek(t)
		}
	}
	return false
}

func (s *rawChunkIterator) At() (int64, []byte) {
	return s.curIt.At()
}

func (s *rawChunkIterator) Err() error {
	return s.err
}

func (s *grpcSeriesSet) At() storage.Series {
	return s.curSeries
}

func (s *grpcSeriesSet) Err() error {
	return s.err
}

func (s *grpcSeriesSet) Warnings() storage.Warnings {
	return nil
}

func (q *grpcStoreQuerier) LabelValues(name string) ([]string, storage.Warnings, error) {
	return nil, nil, nil
}

func (q *grpcStoreQuerier) LabelNames() ([]string, storage.Warnings, error) {
	return nil, nil, nil
}

func (q *grpcStoreQuerier) Close() error {
	return nil
}
