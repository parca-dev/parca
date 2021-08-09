package query

import (
	"context"
	"time"

	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/parca-dev/parca/storage"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Query is the read api interface for parca
// It implements the proto/query/query.proto APIServer interface
type Query struct {
	queryable storage.Queryable
	metaStore storage.ProfileMetaStore
}

func New(
	queryable storage.Queryable,
	metaStore storage.ProfileMetaStore,
) *Query {
	return &Query{
		queryable: queryable,
		metaStore: metaStore,
	}
}

// QueryRange issues a range query against the storage
func (q *Query) QueryRange(ctx context.Context, req *pb.QueryRangeRequest) (*pb.QueryRangeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Query issues a instant query against the storage
func (q *Query) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	switch *req.Mode {
	case pb.QueryRequest_SINGLE:
		s := req.GetSingle()
		if s == nil {
			return nil, status.Error(codes.InvalidArgument, "requested single mode, but did not provide parameters for single")
		}

		p, err := q.findSingle(ctx, s)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to search profile")
		}

		if p == nil {
			return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
		}

		return q.renderReport(p, pb.QueryRequest_Flamegraph)
	}
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func (q *Query) renderReport(p storage.InstantProfile, typ pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	switch typ {
	case pb.QueryRequest_Flamegraph:
		fg, err := storage.GenerateFlamegraph(q.metaStore, p)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to generate flamegraph")
		}

		return &pb.QueryResponse{
			Report: &pb.QueryResponse_Flamegraph{
				Flamegraph: fg,
			},
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "requested report type does not exist")
	}
}

func (q *Query) findSingle(ctx context.Context, s *pb.QueryRequest_Single) (storage.InstantProfile, error) {
	t := s.Time.AsTime()
	requestedTime := timestamp.FromTime(t)

	ms := []*labels.Matcher{}

	// Timestamps don't have to match exactly and staleness kicks in within 5
	// minutes of no samples, so we need to search the range of -5min to +5min
	// for possible samples.
	query := q.queryable.Querier(
		ctx,
		timestamp.FromTime(t.Add(-5*time.Minute)),
		timestamp.FromTime(t.Add(5*time.Minute)),
	)
	set := query.Select(nil, ms...)
	for set.Next() {
		series := set.At()
		i := series.Iterator()
		for i.Next() {
			p := i.At()
			if p.ProfileMeta().Timestamp >= requestedTime {
				return p, nil
			}
		}
		err := i.Err()
		if err != nil {
			return nil, err
		}
	}

	return nil, nil
}

// Series issues a series request against the storage
func (q *Query) Series(ctx context.Context, req *pb.SeriesRequest) (*pb.SeriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Labels issues a labels request against the storage
func (q *Query) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Values issues a values request against the storage
func (q *Query) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Config issues a config request against the storage
func (q *Query) Config(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Targets issues a targets request against the storage
func (q *Query) Targets(ctx context.Context, req *pb.TargetsRequest) (*pb.TargetsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
