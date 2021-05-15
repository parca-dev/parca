// Copyright 2021 The conprof Authors
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
	"io/ioutil"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/tsdb/chunkenc"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
	"google.golang.org/grpc/codes"
	grpcstatus "google.golang.org/grpc/status"
)

// EndlessProfileStore is a no-op writable store and returns infinite amounts
// of series when reading. This is meant for testing timeout issues.
type EndlessProfileStore struct{}

func NewEndlessProfileStore() *EndlessProfileStore {
	return &EndlessProfileStore{}
}

func (s *EndlessProfileStore) Write(ctx context.Context, r *storepb.WriteRequest) (*storepb.WriteResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) Series(r *storepb.SeriesRequest, srv storepb.ReadableProfileStore_SeriesServer) error {
	ctx := srv.Context()
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c := chunkenc.NewBytesChunk()
	app, err := c.Appender()
	if err != nil {
		return err
	}

	b, err := ioutil.ReadFile("./testdata/alloc_objects.pb.gz")
	if err != nil {
		return err
	}

	for j := int64(0); j < 12; j++ {
		app.Append(j, b)
	}

	cbytes, err := c.Bytes()
	if err != nil {
		return err
	}

	i := 0
	for {
		if err := srv.Send(storepb.NewSeriesResponse(&storepb.RawProfileSeries{
			Labels: []labelpb.Label{
				{
					Name:  "__name__",
					Value: "allocs",
				},
				{
					Name:  "count",
					Value: fmt.Sprintf("%d", i),
				},
			},
			Chunks: []storepb.AggrChunk{
				{
					MinTime: 0,
					MaxTime: 12,
					Raw: &storepb.Chunk{
						Type: 1,
						Data: cbytes,
					},
				},
			},
		})); err != nil {
			return grpcstatus.Error(codes.Aborted, err.Error())
		}
		i++
	}
}

func (s *EndlessProfileStore) Profile(ctx context.Context, r *storepb.ProfileRequest) (*storepb.ProfileResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) LabelNames(ctx context.Context, r *storepb.LabelNamesRequest) (*storepb.LabelNamesResponse, error) {
	return nil, nil
}

func (s *EndlessProfileStore) LabelValues(ctx context.Context, r *storepb.LabelValuesRequest) (*storepb.LabelValuesResponse, error) {
	return nil, nil
}
