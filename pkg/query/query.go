package query

import (
	"context"
	"time"

	profilestorepb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/parca-dev/parca/storage"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	sel, err := parser.ParseMetricSelector(req.Query)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	start := req.Start.AsTime()
	end := req.Start.AsTime()

	// Timestamps don't have to match exactly and staleness kicks in within 5
	// minutes of no samples, so we need to search the range of -5min to +5min
	// for possible samples.
	query := q.queryable.Querier(
		ctx,
		timestamp.FromTime(start),
		timestamp.FromTime(end),
	)
	set := query.Select(nil, sel...)
	res := &pb.QueryRangeResponse{}
	for set.Next() {
		series := set.At()

		labels := series.Labels()
		metricsSeries := &pb.MetricsSeries{Labelset: &profilestorepb.LabelSet{Labels: make([]*profilestorepb.Label, 0, len(labels))}}
		for _, l := range labels {
			metricsSeries.Labelset.Labels = append(metricsSeries.Labelset.Labels, &profilestorepb.Label{
				Name:  l.Name,
				Value: l.Value,
			})
		}

		i := series.Iterator()
		for i.Next() {
			p := i.At()
			pit := p.ProfileTree().Iterator()
			if pit.NextChild() {
				metricsSeries.Samples = append(metricsSeries.Samples, &pb.MetricsSample{
					Timestamp: timestamppb.New(timestamp.Time(p.ProfileMeta().Timestamp)),
					Value:     pit.At().CumulativeValue(),
				})
			}
		}
		err := i.Err()
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to iterate")
		}

		res.Series = append(res.Series, metricsSeries)
	}

	return res, nil
}

// Query issues a instant query against the storage
func (q *Query) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	switch *req.Mode {
	case pb.QueryRequest_SINGLE:
		s := req.GetSingle()
		if s == nil {
			return nil, status.Error(codes.InvalidArgument, "requested single mode, but did not provide parameters for single")
		}

		sel, err := parser.ParseMetricSelector(s.Query)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to parse query")
		}

		p, err := q.findSingle(ctx, sel, s)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to search profile")
		}

		if p == nil {
			return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
		}

		return q.renderReport(p, pb.QueryRequest_Flamegraph)
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}
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

func (q *Query) findSingle(ctx context.Context, sel []*labels.Matcher, s *pb.QueryRequest_Single) (storage.InstantProfile, error) {
	t := s.Time.AsTime()
	requestedTime := timestamp.FromTime(t)

	// Timestamps don't have to match exactly and staleness kicks in within 5
	// minutes of no samples, so we need to search the range of -5min to +5min
	// for possible samples.
	query := q.queryable.Querier(
		ctx,
		timestamp.FromTime(t.Add(-5*time.Minute)),
		timestamp.FromTime(t.Add(5*time.Minute)),
	)
	set := query.Select(nil, sel...)
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
