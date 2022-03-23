package query

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/polarsignals/arcticdb/query"
	"github.com/polarsignals/arcticdb/query/logicalplan"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"go.opentelemetry.io/otel/attribute"
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
	engine    *query.Engine
	tableName string
	metaStore metastore.ProfileMetaStore
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
	engine *query.Engine,
	tableName string,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:    logger,
		tracer:    tracer,
		engine:    engine,
		tableName: tableName,
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

	err := q.engine.ScanTable(q.tableName).
		Distinct(logicalplan.Col("labels." + name)).
		Execute(func(ar arrow.Record) error {
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
		})
	if err != nil {
		return nil, err
	}

	sort.Strings(vals)

	return &pb.ValuesResponse{
		LabelValues: vals,
	}, nil
}

func matcherToBooleanExpression(matcher *labels.Matcher) (logicalplan.Expr, error) {
	ref := logicalplan.Col("labels." + matcher.Name)
	switch matcher.Type {
	case labels.MatchEqual:
		return ref.Eq(logicalplan.Literal(matcher.Value)), nil
	case labels.MatchNotEqual:
		return ref.NotEq(logicalplan.Literal(matcher.Value)), nil
	case labels.MatchRegexp:
		return ref.RegexMatch(matcher.Value), nil
	case labels.MatchNotRegexp:
		return ref.RegexNotMatch(matcher.Value), nil
	default:
		return nil, fmt.Errorf("unsupported matcher type %v", matcher.Type.String())
	}
}

func matchersToBooleanExpressions(matchers []*labels.Matcher) ([]logicalplan.Expr, error) {
	exprs := make([]logicalplan.Expr, 0, len(matchers))

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
	ErrValueColumnNotFound     = errors.New("value column not found")
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

	filterExpr := logicalplan.And(
		logicalplan.Col("timestamp").GT(logicalplan.Literal(start)),
		logicalplan.Col("timestamp").LT(logicalplan.Literal(end)),
		labelFilterExpressions...,
	)

	res := &pb.QueryRangeResponse{}
	labelsetToIndex := map[string]int{}

	labelSet := labels.Labels{}

	var ar arrow.Record
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")),
			logicalplan.DynCol("labels"),
			logicalplan.Col("timestamp"),
		).
		Execute(func(r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, err
	}

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
		if field.Name == "sum(value)" {
			valueColumnIndex = i
			valueColumnFound = true
			continue
		}

		if strings.HasPrefix(field.Name, "labels.") {
			labelColumnIndices = append(labelColumnIndices, i)
		}
	}

	if !timestampColumnFound {
		return nil, ErrTimestampColumnNotFound
	}

	if !valueColumnFound {
		return nil, ErrValueColumnNotFound
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

	// This is horrible and should be fixed. The data is sorted in the storage, we should not have to sort it here.
	for _, series := range res.Series {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].Timestamp.AsTime().Before(series.Samples[j].Timestamp.AsTime())
		})
	}

	return res, nil
}

// Query issues a instant query against the storage
func (q *ColumnQueryAPI) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	switch req.Mode {
	case pb.QueryRequest_MODE_SINGLE_UNSPECIFIED:
		return q.singleRequest(ctx, req.GetSingle(), req.GetReportType())
	case pb.QueryRequest_MODE_MERGE:
		return q.mergeRequest(ctx, req.GetMerge(), req.GetReportType())
	case pb.QueryRequest_MODE_DIFF:
		return q.diffRequest(ctx, req.GetDiff(), req.GetReportType())
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}
}

func (q *ColumnQueryAPI) renderReport(ctx context.Context, p *profile.StacktraceSamples, typ pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	switch typ {
	case pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED:
		fg, err := GenerateFlamegraphFlat(ctx, q.tracer, q.metaStore, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate flamegraph: %v", err.Error())
		}
		return &pb.QueryResponse{
			Report: &pb.QueryResponse_Flamegraph{
				Flamegraph: fg,
			},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_PPROF:
		pp, err := GenerateFlatPprof(ctx, q.metaStore, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		var buf bytes.Buffer
		if err := pp.Write(&buf); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		return &pb.QueryResponse{
			Report: &pb.QueryResponse_Pprof{Pprof: buf.Bytes()},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_TOP:
		top, err := GenerateTopTable(ctx, q.metaStore, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		return &pb.QueryResponse{
			Report: &pb.QueryResponse_Top{Top: top},
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "requested report type does not exist")
	}
}

func (q *ColumnQueryAPI) singleRequest(ctx context.Context, s *pb.SingleProfile, reportType pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	p, err := q.selectSingle(ctx, s)
	if err != nil {
		return nil, err
	}

	return q.renderReport(ctx, p, reportType)
}

func (q *ColumnQueryAPI) selectSingle(ctx context.Context, s *pb.SingleProfile) (*profile.StacktraceSamples, error) {
	sel, err := parser.ParseMetricSelector(s.Query)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	t := s.Time.AsTime()
	p, err := q.findSingle(ctx, sel, t)
	if err != nil {
		level.Error(q.logger).Log("msg", "failed to find single profile", "err", err)
		return nil, status.Errorf(codes.Internal, "failed to search profile: %v", err.Error())
	}

	if p == nil {
		return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
	}

	return p, nil
}

func (q *ColumnQueryAPI) findSingle(ctx context.Context, sel []*labels.Matcher, t time.Time) (*profile.StacktraceSamples, error) {
	requestedTime := timestamp.FromTime(t)

	ctx, span := q.tracer.Start(ctx, "findSingle")
	for i, m := range sel {
		span.SetAttributes(attribute.String(fmt.Sprintf("matcher-%d", i), m.String()))
	}
	span.SetAttributes(attribute.Int64("time", t.Unix()))
	defer span.End()

	labelFilterExpressions, err := matchersToBooleanExpressions(sel)
	if err != nil {
		return nil, fmt.Errorf("convert matchers to boolean expressions: %w", err)
	}

	filterExpr := logicalplan.And(
		logicalplan.Col("timestamp").Eq(logicalplan.Literal(requestedTime)),
		labelFilterExpressions[0],
		labelFilterExpressions[1:]...,
	)

	var ar arrow.Record
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")),
			logicalplan.Col("stacktrace"),
		).
		Execute(func(r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, err
	}
	defer ar.Release()

	return arrowRecordToStacktraceSamples(ctx, q.metaStore, ar)
}

func (q *ColumnQueryAPI) mergeRequest(ctx context.Context, m *pb.MergeProfile, reportType pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	ctx, span := q.tracer.Start(ctx, "mergeRequest")
	defer span.End()

	p, err := q.selectMerge(ctx, m)
	if err != nil {
		return nil, err
	}

	return q.renderReport(ctx, p, reportType)
}

func (q *ColumnQueryAPI) selectMerge(ctx context.Context, m *pb.MergeProfile) (*profile.StacktraceSamples, error) {
	ctx, span := q.tracer.Start(ctx, "selectMerge")
	defer span.End()

	sel, err := parser.ParseMetricSelector(m.Query)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	start := timestamp.FromTime(m.Start.AsTime())
	end := timestamp.FromTime(m.End.AsTime())

	labelFilterExpressions, err := matchersToBooleanExpressions(sel)
	if err != nil {
		return nil, fmt.Errorf("convert matchers to boolean expressions: %w", err)
	}

	filterExpr := logicalplan.And(
		logicalplan.Col("timestamp").GT(logicalplan.Literal(start)),
		logicalplan.Col("timestamp").LT(logicalplan.Literal(end)),
		labelFilterExpressions...,
	)

	var ar arrow.Record
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")),
			logicalplan.Col("stacktrace"),
		).
		Execute(func(r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, err
	}
	defer ar.Release()

	return arrowRecordToStacktraceSamples(ctx, q.metaStore, ar)
}

func (q *ColumnQueryAPI) diffRequest(ctx context.Context, d *pb.DiffProfile, reportType pb.QueryRequest_ReportType) (*pb.QueryResponse, error) {
	ctx, span := q.tracer.Start(ctx, "diffRequest")
	defer span.End()

	if d == nil {
		return nil, status.Error(codes.InvalidArgument, "requested diff mode, but did not provide parameters for diff")
	}

	base, err := q.selectProfileForDiff(ctx, d.A)
	if err != nil {
		return nil, err
	}

	compare, err := q.selectProfileForDiff(ctx, d.B)
	if err != nil {
		return nil, err
	}

	// TODO: This is cheating a bit. This should be done with a sub-query in the columnstore.
	diff := &profile.StacktraceSamples{}
	stacktraceIndices := map[string]int{}
	for i, s := range base.Samples {
		stacktraceIndices[string(profile.MakeStacktraceKey(s))] = i
	}

	for _, s := range compare.Samples {
		if i, ok := stacktraceIndices[string(profile.MakeStacktraceKey(s))]; ok {
			s.DiffValue = s.Value - base.Samples[i].Value
		}
		diff.Samples = append(diff.Samples, s)
	}

	return q.renderReport(ctx, diff, reportType)
}

func (q *ColumnQueryAPI) selectProfileForDiff(ctx context.Context, s *pb.ProfileDiffSelection) (*profile.StacktraceSamples, error) {
	var (
		p   *profile.StacktraceSamples
		err error
	)
	switch s.Mode {
	case pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		p, err = q.selectSingle(ctx, s.GetSingle())
	case pb.ProfileDiffSelection_MODE_MERGE:
		p, err = q.selectMerge(ctx, s.GetMerge())
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown mode for diff profile selection")
	}

	return p, err
}

func arrowRecordToStacktraceSamples(ctx context.Context, metaStore metastore.ProfileMetaStore, ar arrow.Record) (*profile.StacktraceSamples, error) {
	s := ar.Schema()
	indices := s.FieldIndices("stacktrace")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one stacktrace column, got %d", len(indices))
	}
	stacktraceColumn := ar.Column(indices[0]).(*array.List)
	stacktraceValues := stacktraceColumn.ListValues().(*array.FixedSizeBinary)
	stacktraceOffsets := stacktraceColumn.Offsets()[1:]

	indices = s.FieldIndices("sum(value)")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one value column, got %d", len(indices))
	}
	valueColumn := ar.Column(indices[0]).(*array.Int64)

	locationUUIDSeen := map[string]struct{}{}
	locationUUIDs := [][]byte{}
	rows := int(ar.NumRows())
	samples := make([]sample, rows)
	pos := 0
	for i := 0; i < rows; i++ {
		s := sample{
			value: valueColumn.Value(i),
		}

		for j := pos; j < int(stacktraceOffsets[i]); j++ {
			locID := stacktraceValues.Value(j)
			s.locationIDs = append(s.locationIDs, locID)

			if _, ok := locationUUIDSeen[string(locID)]; !ok {
				locationUUIDSeen[string(locID)] = struct{}{}
				locationUUIDs = append(locationUUIDs, locID)
			}
		}

		samples[i] = s
		pos = int(stacktraceOffsets[i])
	}

	// Get the full locations for the location UUIDs
	locationsMap, err := metastore.GetLocationsByIDs(ctx, metaStore, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	stackSamples := make([]*profile.Sample, 0, len(samples))
	for _, s := range samples {
		stackSample := &profile.Sample{
			Value:    s.value,
			Location: make([]*metastore.Location, 0, len(s.locationIDs)),
		}

		// LocationIDs are stored in the opposite order than the flamegraph
		// builder expects, so we need to iterate over them in reverse.
		for i := len(s.locationIDs) - 1; i >= 0; i-- {
			locID := s.locationIDs[i]
			stackSample.Location = append(stackSample.Location, locationsMap[string(locID)])
		}

		stackSamples = append(stackSamples, stackSample)
	}

	return &profile.StacktraceSamples{
		Samples: stackSamples,
	}, nil
}

type sample struct {
	locationIDs [][]byte
	value       int64
}
