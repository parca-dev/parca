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

	profilestorepb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	"github.com/parca-dev/parca/storage"
)

type ProfileStore struct {
	logger    log.Logger
	app       storage.Appendable
	metaStore storage.ProfileMetaStore
}

var _ profilestorepb.ProfileStoreServer = &ProfileStore{}

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
		sort.Sort(ls)

		app := s.app.Appender(ctx, ls)
		for _, sample := range series.Samples {
			p, err := profile.Parse(bytes.NewBuffer(sample.RawProfile))
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
			}

			if err := p.CheckValid(); err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
			}

			level.Debug(s.logger).Log("msg", "writing sample", "label_set", ls.String())

			if err := app.Append(storage.ProfileFromPprof(s.metaStore, p)); err != nil {
				return nil, status.Errorf(codes.Internal, "failed to append sample: %v", err)
			}
		}
	}

	return &profilestorepb.WriteRawResponse{}, nil
}
