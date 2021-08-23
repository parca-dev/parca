package query

import (
	"context"
	"errors"
	"math"
	"sort"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/parca-dev/parca/pkg/storage"
	profilestorepb "github.com/parca-dev/parca/proto/gen/go/profilestore"
	pb "github.com/parca-dev/parca/proto/gen/go/query"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	minTime = time.Unix(math.MinInt64/1000+62135596801, 0).UTC()
	maxTime = time.Unix(math.MaxInt64/1000-62135596801, 999999999).UTC()
)

// Query is the read api interface for parca
// It implements the proto/query/query.proto APIServer interface
type Query struct {
	logger    log.Logger
	queryable storage.Queryable
	metaStore storage.ProfileMetaStore
}

func New(
	logger log.Logger,
	queryable storage.Queryable,
	metaStore storage.ProfileMetaStore,
) *Query {
	return &Query{
		queryable: queryable,
		metaStore: metaStore,
		logger:    logger,
	}
}

// QueryRange issues a range query against the storage
func (q *Query) QueryRange(ctx context.Context, req *pb.QueryRangeRequest) (*pb.QueryRangeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sel, err := parser.ParseMetricSelector(req.Query)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	start := req.Start.AsTime()
	end := req.End.AsTime()

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
				s := &pb.MetricsSample{
					Timestamp: timestamppb.New(timestamp.Time(p.ProfileMeta().Timestamp)),
					Value:     pit.At().CumulativeValue(),
				}
				metricsSeries.Samples = append(metricsSeries.Samples, s)
			}
		}
		err := i.Err()
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to iterate")
		}

		res.Series = append(res.Series, metricsSeries)

		if req.Limit != 0 && len(res.Series) == int(req.Limit) {
			break
		}
	}
	if err := set.Err(); err != nil {
		return nil, status.Error(codes.Internal, "failed to iterate")
	}

	return res, nil
}

// Query issues a instant query against the storage
func (q *Query) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var p storage.InstantProfile
	switch req.Mode {
	case pb.QueryRequest_SINGLE:
		s := req.GetSingle()
		if s == nil {
			return nil, status.Error(codes.InvalidArgument, "requested single mode, but did not provide parameters for single")
		}

		sel, err := parser.ParseMetricSelector(s.Query)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to parse query")
		}

		p, err = q.findSingle(ctx, sel, s)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to search profile")
		}

		if p == nil {
			return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
		}
	case pb.QueryRequest_MERGE:
		m := req.GetMerge()
		if m == nil {
			return nil, status.Error(codes.InvalidArgument, "requested merge mode, but did not provide parameters for merge")
		}

		sel, err := parser.ParseMetricSelector(m.Query)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "failed to parse query")
		}

		start := m.Start.AsTime()
		end := m.End.AsTime()

		level.Debug(q.logger).Log("msg", "merge request", "query", m.Query, "start", start, "end", end)

		p, err = q.merge(ctx, sel, start, end)
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}

	return q.renderReport(p, pb.QueryRequest_FLAMEGRAPH)
}

func (q *Query) renderReport(p storage.InstantProfile, typ pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	switch typ {
	case pb.QueryRequest_FLAMEGRAPH:
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

func (q *Query) merge(ctx context.Context, sel []*labels.Matcher, start, end time.Time) (storage.InstantProfile, error) {
	startTs := timestamp.FromTime(start)
	endTs := timestamp.FromTime(end)
	query := q.queryable.Querier(
		ctx,
		startTs,
		endTs,
	)

	set := query.Select(&storage.SelectHints{
		Start: startTs,
		End:   endTs,
		Merge: true,
	}, sel...)

	// Naively copy all instant profiles and then merge them. This can probably
	// done streaming, but doing it naively for a first pass.
	profiles := []storage.InstantProfile{}

	for set.Next() {
		series := set.At()
		i := series.Iterator()
		for i.Next() {
			// Have to copy as profile pointer is not stable for more than the
			// current iteration.
			profiles = append(profiles, storage.CopyInstantProfile(i.At()))
		}
	}

	return storage.MergeProfiles(profiles...)
}

// Series issues a series request against the storage
func (q *Query) Series(ctx context.Context, req *pb.SeriesRequest) (*pb.SeriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Labels issues a labels request against the storage
func (q *Query) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	matcherSets, err := parseMatchers(req.Match)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		start time.Time = minTime
		end   time.Time = maxTime
	)

	if req.Start != nil {
		start = req.Start.AsTime()
	}
	if req.End != nil {
		end = req.End.AsTime()
	}

	query := q.queryable.Querier(
		ctx,
		timestamp.FromTime(start),
		timestamp.FromTime(end),
	)

	var (
		names    []string
		warnings storage.Warnings
	)
	if len(matcherSets) > 0 {
		labelNamesSet := make(map[string]struct{})

		for _, matchers := range matcherSets {
			vals, callWarnings, err := query.LabelNames(matchers...)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}

			warnings = append(warnings, callWarnings...)
			for _, val := range vals {
				labelNamesSet[val] = struct{}{}
			}
		}

		// Convert the map to an array.
		names = make([]string, 0, len(labelNamesSet))
		for key := range labelNamesSet {
			names = append(names, key)
		}
		sort.Strings(names)
	} else {
		names, warnings, err = query.LabelNames()
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}
	}

	return &pb.LabelsResponse{
		LabelNames: names,
	}, nil
}

// Values issues a values request against the storage
func (q *Query) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	name := req.LabelName

	matcherSets, err := parseMatchers(req.Match)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		start time.Time = minTime
		end   time.Time = maxTime
	)

	if req.Start != nil {
		start = req.Start.AsTime()
	}
	if req.End != nil {
		end = req.End.AsTime()
	}

	query := q.queryable.Querier(
		ctx,
		timestamp.FromTime(start),
		timestamp.FromTime(end),
	)

	var (
		vals     []string
		warnings storage.Warnings
	)
	if len(matcherSets) > 0 {
		var callWarnings storage.Warnings
		labelValuesSet := make(map[string]struct{})
		for _, matchers := range matcherSets {
			vals, callWarnings, err = query.LabelValues(name, matchers...)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			}
			warnings = append(warnings, callWarnings...)
			for _, val := range vals {
				labelValuesSet[val] = struct{}{}
			}
		}

		vals = make([]string, 0, len(labelValuesSet))
		for val := range labelValuesSet {
			vals = append(vals, val)
		}
	} else {
		vals, warnings, err = query.LabelValues(name)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if vals == nil {
			vals = []string{}
		}
	}

	sort.Strings(vals)

	return &pb.ValuesResponse{
		LabelValues: vals,
	}, nil
}

// Config issues a config request against the storage
func (q *Query) Config(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Targets issues a targets request against the storage
func (q *Query) Targets(ctx context.Context, req *pb.TargetsRequest) (*pb.TargetsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

func parseMatchers(matchers []string) ([][]*labels.Matcher, error) {
	var matcherSets [][]*labels.Matcher
	for _, s := range matchers {
		matchers, err := parser.ParseMetricSelector(s)
		if err != nil {
			return nil, err
		}
		matcherSets = append(matcherSets, matchers)
	}

OUTER:
	for _, ms := range matcherSets {
		for _, lm := range ms {
			if lm != nil && !lm.Matches("") {
				continue OUTER
			}
		}
		return nil, errors.New("match[] must contain at least one non-empty matcher")
	}
	return matcherSets, nil
}
