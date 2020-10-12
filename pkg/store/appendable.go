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

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/storage"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/thanos-io/thanos/pkg/store/labelpb"
)

type grpcStoreAppendable struct {
	c storepb.ProfileStoreClient
}

func NewGRPCAppendable(c storepb.ProfileStoreClient) *grpcStoreAppendable {
	return &grpcStoreAppendable{
		c: c,
	}
}

type grpcStoreAppender struct {
	c storepb.ProfileStoreClient

	ctx context.Context
	l   labels.Labels
	t   int64
	v   []byte
}

func (a *grpcStoreAppendable) Appender(ctx context.Context) storage.Appender {
	return &grpcStoreAppender{
		c:   a.c,
		ctx: ctx,
	}
}

func (a *grpcStoreAppender) Add(l labels.Labels, t int64, v []byte) (uint64, error) {
	a.l = l
	a.t = t
	a.v = v
	return 0, nil
}

func (a *grpcStoreAppender) AddFast(ref uint64, t int64, v []byte) error {
	return errors.New("not implemented")
}

func (a *grpcStoreAppender) Commit() error {
	_, err := a.c.Write(a.ctx, &storepb.WriteRequest{
		ProfileSeries: []storepb.ProfileSeries{
			{
				Labels: labelpb.LabelsFromPromLabels(a.l),
				Samples: []storepb.Sample{
					{
						Timestamp: a.t,
						Value:     a.v,
					},
				},
			},
		},
	})
	return err
}

func (a *grpcStoreAppender) Rollback() error {
	return nil
}
