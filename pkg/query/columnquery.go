package query

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/go-kit/log"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/columnstore"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ColumnQuery is the read api interface for parca
// It implements the proto/query/query.proto APIServer interface
type ColumnQueryAPI struct {
	pb.UnimplementedQueryServiceServer

	logger    log.Logger
	tracer    trace.Tracer
	table     *columnstore.Table
	metaStore metastore.ProfileMetaStore
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
	table *columnstore.Table,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:    logger,
		tracer:    tracer,
		table:     table,
		metaStore: metaStore,
	}
}

// Labels issues a labels request against the storage
func (q *ColumnQueryAPI) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	return &pb.LabelsResponse{
		LabelNames: nil,
	}, nil
}

// Values issues a values request against the storage
func (q *ColumnQueryAPI) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	name := req.LabelName
	vals := []string{}
	seen := map[string]struct{}{}

	pool := memory.NewGoAllocator()
	err := q.table.Iterator(pool, columnstore.Distinct(pool, []columnstore.ArrowFieldMatcher{columnstore.DynamicColumnRef("labels").Column(name).ArrowFieldMatcher()}, func(ar arrow.Record) error {
		defer ar.Release()

		if ar.NumCols() != 1 {
			return fmt.Errorf("expected 1 column, got %d", ar.NumCols())
		}

		col := ar.Column(0)
		stringCol, ok := col.(*array.String)
		if !ok {
			return fmt.Errorf("expected string column, got %T", col)
		}

		for i := 0; i < stringCol.Len(); i++ {
			val := stringCol.Value(i)
			if _, ok := seen[val]; !ok {
				vals = append(vals, val)
				seen[val] = struct{}{}
			}
		}

		return nil
	}).Callback)
	if err != nil {
		return nil, err
	}

	sort.Strings(vals)

	return &pb.ValuesResponse{
		LabelValues: vals,
	}, nil
}

func matcherToBooleanExpression(matcher *labels.Matcher) (columnstore.BooleanExpression, error) {
	ref := columnstore.DynamicColumnRef("labels").Column(matcher.Name)
	switch matcher.Type {
	case labels.MatchEqual:
		return ref.Equal(columnstore.StringLiteral(matcher.Value)), nil
	case labels.MatchNotEqual:
		return ref.NotEqual(columnstore.StringLiteral(matcher.Value)), nil
	case labels.MatchRegexp:
		r, err := columnstore.NewRegexMatcher(matcher.Value)
		if err != nil {
			return nil, err
		}

		return ref.RegexMatch(r), nil
	case labels.MatchNotRegexp:
		r, err := columnstore.NewRegexMatcher(matcher.Value)
		if err != nil {
			return nil, err
		}

		return ref.RegexNotMatch(r), nil
	default:
		return nil, fmt.Errorf("unsupported matcher type %v", matcher.Type.String())
	}
}

func matchersToBooleanExpressions(matchers []*labels.Matcher) ([]columnstore.BooleanExpression, error) {
	exprs := make([]columnstore.BooleanExpression, 0, len(matchers))

	for _, matcher := range matchers {
		expr, err := matcherToBooleanExpression(matcher)
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, expr)
	}

	return exprs, nil
}

var (
	ErrTimestampColumnNotFound = errors.New("timestamp column not found")
	ErrValueColumnNotFound     = errors.New("timestamp column not found")
)

// QueryRange issues a range query against the storage
func (q *ColumnQueryAPI) QueryRange(ctx context.Context, req *pb.QueryRangeRequest) (*pb.QueryRangeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	sel, err := parser.ParseMetricSelector(req.Query)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	start := timestamp.FromTime(req.Start.AsTime())
	end := timestamp.FromTime(req.End.AsTime())

	labelFilterExpressions, err := matchersToBooleanExpressions(sel)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to build query")
	}

	filterExpr := columnstore.And(
		columnstore.StaticColumnRef("timestamp").GreaterThan(columnstore.Int64Literal(start)),
		columnstore.StaticColumnRef("timestamp").LessThan(columnstore.Int64Literal(end)),
		labelFilterExpressions...,
	)
	pool := memory.NewGoAllocator()

	res := &pb.QueryRangeResponse{}
	labelsetToIndex := map[string]int{}

	labelSet := labels.Labels{}
	err = q.table.Iterator(pool, columnstore.Filter(pool, filterExpr, func(ar arrow.Record) error {
		defer ar.Release()

		timestampColumnIndex := 0
		timestampColumnFound := false
		valueColumnIndex := 0
		valueColumnFound := false
		labelColumnIndices := []int{}

		fields := ar.Schema().Fields()
		for i, field := range fields {
			if field.Name == "timestamp" {
				timestampColumnIndex = i
				timestampColumnFound = true
				continue
			}
			if field.Name == "value" {
				valueColumnIndex = i
				valueColumnFound = true
				continue
			}

			if strings.HasPrefix(field.Name, "labels.") {
				labelColumnIndices = append(labelColumnIndices, i)
			}
		}

		if !timestampColumnFound {
			return ErrTimestampColumnNotFound
		}

		if !valueColumnFound {
			return ErrValueColumnNotFound
		}

		for i := 0; i < int(ar.NumRows()); i++ {
			labelSet = labelSet[:0]
			for _, labelColumnIndex := range labelColumnIndices {
				col := ar.Column(labelColumnIndex).(*array.String)
				if col.IsNull(i) {
					continue
				}

				v := col.Value(i)
				if v != "" {
					labelSet = append(labelSet, labels.Label{Name: strings.TrimPrefix(fields[labelColumnIndex].Name, "labels."), Value: v})
				}
			}

			sort.Sort(labelSet)
			s := labelSet.String()
			index, ok := labelsetToIndex[s]
			if !ok {
				pbLabelSet := make([]*profilestorepb.Label, 0, len(labelSet))
				for _, l := range labelSet {
					pbLabelSet = append(pbLabelSet, &profilestorepb.Label{
						Name:  l.Name,
						Value: l.Value,
					})
				}
				res.Series = append(res.Series, &pb.MetricsSeries{Labelset: &profilestorepb.LabelSet{Labels: pbLabelSet}})
				index = len(res.Series) - 1
				labelsetToIndex[s] = index
			}

			series := res.Series[index]
			series.Samples = append(series.Samples, &pb.MetricsSample{
				Timestamp: timestamppb.New(timestamp.Time(ar.Column(timestampColumnIndex).(*array.Int64).Value(i))),
				Value:     ar.Column(valueColumnIndex).(*array.Int64).Value(i),
			})
		}

		return nil
	}))
	if err != nil {
		return nil, err
	}

	// This is horrible and should be fixed. The data is sorted in the storage, we should not have to sort it here.
	for _, series := range res.Series {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].Timestamp.AsTime().Before(series.Samples[j].Timestamp.AsTime())
		})
	}

	return res, nil
}
