package store

import (
	"context"
	"errors"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/conprof/db/storage"
	"github.com/prometheus/prometheus/pkg/labels"
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
				Labels: translatePromLabels(a.l),
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
