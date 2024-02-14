// Copyright 2022-2024 The Parca Authors
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

package parcacol

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/apache/arrow/go/v14/arrow/scalar"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/query"
	"github.com/polarsignals/frostdb/query/logicalplan"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

type Engine interface {
	ScanTable(name string) query.Builder
	ScanSchema(name string) query.Builder
}

func NewQuerier(
	logger log.Logger,
	tracer trace.Tracer,
	engine Engine,
	tableName string,
	symbolizer *ProfileSymbolizer,
	pool memory.Allocator,
) *Querier {
	return &Querier{
		logger:     logger,
		tracer:     tracer,
		engine:     engine,
		tableName:  tableName,
		symbolizer: symbolizer,
		pool:       pool,
	}
}

type Querier struct {
	logger     log.Logger
	engine     Engine
	tableName  string
	symbolizer *ProfileSymbolizer
	tracer     trace.Tracer
	pool       memory.Allocator
}

func (q *Querier) Labels(
	ctx context.Context,
	match []string,
	start, end time.Time,
) ([]string, error) {
	seen := map[string]struct{}{}

	err := q.engine.ScanSchema(q.tableName).
		Distinct(logicalplan.Col("name")).
		Filter(logicalplan.Col("name").RegexMatch("^labels\\..+$")).
		Execute(ctx, func(ctx context.Context, ar arrow.Record) error {
			if ar.NumCols() != 1 {
				return fmt.Errorf("expected 1 column, got %d", ar.NumCols())
			}

			col := ar.Column(0)
			stringCol, ok := col.(*array.String)
			if !ok {
				return fmt.Errorf("expected string column, got %T", col)
			}

			for i := 0; i < stringCol.Len(); i++ {
				// This should usually not happen, but better safe than sorry.
				if stringCol.IsNull(i) {
					continue
				}

				val := stringCol.Value(i)
				seen[strings.TrimPrefix(val, "labels.")] = struct{}{}
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	vals := make([]string, 0, len(seen))
	for val := range seen {
		vals = append(vals, val)
	}

	sort.Strings(vals)

	return vals, nil
}

func (q *Querier) Values(
	ctx context.Context,
	labelName string,
	match []string,
	start, end time.Time,
) ([]string, error) {
	vals := []string{}

	err := q.engine.ScanTable(q.tableName).
		Distinct(logicalplan.Col("labels."+labelName)).
		Execute(ctx, func(ctx context.Context, ar arrow.Record) error {
			if ar.NumCols() != 1 {
				return fmt.Errorf("expected 1 column, got %d", ar.NumCols())
			}

			col := ar.Column(0)
			dict, ok := col.(*array.Dictionary)
			if !ok {
				return fmt.Errorf("expected dictionary column, got %T", col)
			}

			for i := 0; i < dict.Len(); i++ {
				if dict.IsNull(i) {
					continue
				}

				val := StringValueFromDictionary(dict, i)

				// Because of an implementation detail of aggregations in
				// FrostDB resulting columns can have the value of "", but that
				// is equivalent to the label not existing at all, so we need
				// to skip it.
				if len(val) > 0 {
					vals = append(vals, val)
				}
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	sort.Strings(vals)
	return vals, nil
}

func MatcherToBooleanExpression(matcher *labels.Matcher) (logicalplan.Expr, error) {
	label := logicalplan.Col(profile.ColumnLabelsPrefix + matcher.Name)
	labelExpr, err := matcherToBinaryExpression(matcher, label)
	if err != nil {
		return nil, err
	}
	pprofLabel := logicalplan.Col(profile.ColumnPprofLabelsPrefix + matcher.Name)
	pprofLabelExpr, err := matcherToBinaryExpression(matcher, pprofLabel)
	if err != nil {
		return nil, err
	}

	return logicalplan.Or(
			logicalplan.And(pprofLabel.Eq(&logicalplan.LiteralExpr{Value: scalar.ScalarNull}), labelExpr),
			logicalplan.And(label.Eq(&logicalplan.LiteralExpr{Value: scalar.ScalarNull}), pprofLabelExpr),
		),
		nil
}

func matcherToBinaryExpression(matcher *labels.Matcher, ref *logicalplan.Column) (*logicalplan.BinaryExpr, error) {
	switch matcher.Type {
	case labels.MatchEqual:
		if matcher.Value == "" {
			return ref.Eq(&logicalplan.LiteralExpr{Value: scalar.ScalarNull}), nil
		}
		return ref.Eq(logicalplan.Literal(matcher.Value)), nil
	case labels.MatchNotEqual:
		if matcher.Value == "" {
			return ref.NotEq(&logicalplan.LiteralExpr{Value: scalar.ScalarNull}), nil
		}
		return ref.NotEq(logicalplan.Literal(matcher.Value)), nil
	case labels.MatchRegexp:
		return ref.RegexMatch(matcher.Value), nil
	case labels.MatchNotRegexp:
		return ref.RegexNotMatch(matcher.Value), nil
	default:
		return nil, fmt.Errorf("unsupported matcher type %v", matcher.Type.String())
	}
}

func MatchersToBooleanExpressions(matchers []*labels.Matcher) ([]logicalplan.Expr, error) {
	exprs := make([]logicalplan.Expr, 0, len(matchers))

	for _, matcher := range matchers {
		expr, err := MatcherToBooleanExpression(matcher)
		if err != nil {
			return nil, err
		}

		exprs = append(exprs, expr)
	}

	return exprs, nil
}

func QueryToFilterExprs(query string) (QueryParts, []logicalplan.Expr, error) {
	qp, err := ParseQuery(query)
	if err != nil {
		return qp, nil, err
	}

	labelFilterExpressions, err := MatchersToBooleanExpressions(qp.Matchers)
	if err != nil {
		return qp, nil, status.Error(codes.InvalidArgument, "failed to build query")
	}

	exprs := append([]logicalplan.Expr{
		logicalplan.Col(profile.ColumnName).Eq(logicalplan.Literal(qp.Meta.Name)),
		logicalplan.Col(profile.ColumnSampleType).Eq(logicalplan.Literal(qp.Meta.SampleType.Type)),
		logicalplan.Col(profile.ColumnSampleUnit).Eq(logicalplan.Literal(qp.Meta.SampleType.Unit)),
		logicalplan.Col(profile.ColumnPeriodType).Eq(logicalplan.Literal(qp.Meta.PeriodType.Type)),
		logicalplan.Col(profile.ColumnPeriodUnit).Eq(logicalplan.Literal(qp.Meta.PeriodType.Unit)),
	}, labelFilterExpressions...)

	deltaPlan := logicalplan.Col(profile.ColumnDuration).Eq(logicalplan.Literal(0))
	if qp.Delta {
		deltaPlan = logicalplan.Col(profile.ColumnDuration).NotEq(logicalplan.Literal(0))
	}

	exprs = append(exprs, deltaPlan)

	return qp, exprs, nil
}

type QueryParts struct {
	Meta     profile.Meta
	Delta    bool
	Matchers []*labels.Matcher
}

// ParseQuery from a string into the QueryParts struct.
func ParseQuery(query string) (QueryParts, error) {
	parsedSelector, err := parser.ParseMetricSelector(query)
	if err != nil {
		return QueryParts{}, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	sel := make([]*labels.Matcher, 0, len(parsedSelector))
	var nameLabel *labels.Matcher
	for _, matcher := range parsedSelector {
		if matcher.Name == labels.MetricName {
			nameLabel = matcher
		} else {
			sel = append(sel, matcher)
		}
	}
	if nameLabel == nil {
		return QueryParts{}, status.Error(codes.InvalidArgument, "query must contain a profile-type selection")
	}

	parts := strings.Split(nameLabel.Value, ":")
	if len(parts) != 5 && len(parts) != 6 {
		return QueryParts{}, status.Errorf(codes.InvalidArgument, "profile-type selection must be of the form <name>:<sample-type>:<sample-unit>:<period-type>:<period-unit>(:delta), got(%d): %q", len(parts), nameLabel.Value)
	}
	delta := false
	if len(parts) == 6 && parts[5] == "delta" {
		delta = true
	}

	return QueryParts{
		Meta: profile.Meta{
			Name: parts[0],
			SampleType: profile.ValueType{
				Type: parts[1],
				Unit: parts[2],
			},
			PeriodType: profile.ValueType{
				Type: parts[3],
				Unit: parts[4],
			},
		},
		Delta:    delta,
		Matchers: sel,
	}, nil
}

func (q *Querier) QueryRange(
	ctx context.Context,
	query string,
	startTime, endTime time.Time,
	step time.Duration,
	limit uint32,
) ([]*pb.MetricsSeries, error) {
	queryParts, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, err
	}

	start := timestamp.FromTime(startTime)
	end := timestamp.FromTime(endTime)

	// The step cannot be lower than 1s
	if step < time.Second {
		step = time.Second
	}

	exprs := append(
		selectorExprs,
		logicalplan.Col(profile.ColumnTimestamp).Gt(logicalplan.Literal(start)),
		logicalplan.Col(profile.ColumnTimestamp).Lt(logicalplan.Literal(end)),
	)

	filterExpr := logicalplan.And(exprs...)

	if queryParts.Delta {
		return q.queryRangeDelta(
			ctx,
			filterExpr,
			step,
			queryParts.Meta,
		)
	}

	return q.queryRangeNonDelta(ctx, filterExpr, step)
}

const (
	ValuePerSecond = "value_per_second"
)

func (q *Querier) queryRangeDelta(
	ctx context.Context,
	filterExpr logicalplan.Expr,
	step time.Duration,
	m profile.Meta,
) ([]*pb.MetricsSeries, error) {
	resultType := m.SampleType

	records := []arrow.Record{}
	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()
	rows := 0
	preProjection := []logicalplan.Expr{
		logicalplan.Mul(logicalplan.Div(logicalplan.Col(profile.ColumnTimestamp), logicalplan.Literal(step.Milliseconds())), logicalplan.Literal(step.Milliseconds())).Alias(profile.ColumnTimestamp),
		logicalplan.DynCol(profile.ColumnLabels),
		logicalplan.Col(profile.ColumnDuration),
	}

	if m.SampleType.Type == "samples" && m.SampleType.Unit == "count" {
		// 1 CPU sample is equivalent to whatever the period is. Therefore the
		// value * period is the total CPU time spent over the duration.
		preProjection = append(
			preProjection,
			logicalplan.Mul(logicalplan.Col(profile.ColumnValue), logicalplan.Col(profile.ColumnPeriod)).Alias(profile.ColumnValue),
		)

		resultType = m.PeriodType
	} else {
		preProjection = append(
			preProjection,
			logicalplan.Col(profile.ColumnValue),
		)
	}

	totalSum := logicalplan.Sum(logicalplan.Col(profile.ColumnValue))
	totalSumColumn := totalSum.Name()
	durationSum := logicalplan.Sum(logicalplan.Col(profile.ColumnDuration))

	err := q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Project(preProjection...).
		Aggregate(
			[]*logicalplan.AggregationFunction{
				// We need the duration sum, so we can calculate the per-second
				// value at the step-level timestamp.
				durationSum,
				totalSum,
			},
			[]logicalplan.Expr{
				logicalplan.DynCol(profile.ColumnLabels),
				logicalplan.Col(profile.ColumnTimestamp),
			},
		).
		Project(
			// The per second value is the sum of all values at the step-level
			// timestamp, divided by the duration sum. The duration sum is
			// necessary so if multiple timestamps fall into the same step, we
			// account for the total duration within that step.
			logicalplan.Div(
				logicalplan.Convert(totalSum, arrow.PrimitiveTypes.Float64),
				logicalplan.Convert(durationSum, arrow.PrimitiveTypes.Float64),
			).Alias(ValuePerSecond),
			totalSum,
			logicalplan.DynCol(profile.ColumnLabels),
			logicalplan.Col(profile.ColumnTimestamp),
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			records = append(records, r)
			rows += int(r.NumRows())
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || rows == 0 {
		return nil, status.Error(
			codes.NotFound,
			"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
		)
	}

	// Add necessary columns and their found value is false by default.
	columnIndices := struct {
		Timestamp      int
		PerSecondValue int
		ValueSum       int
	}{
		Timestamp:      -1,
		PerSecondValue: -1,
		ValueSum:       -1,
	}

	labelColumnIndices := []int{}
	labelSet := labels.Labels{}
	resSeries := []*pb.MetricsSeries{}
	labelsetToIndex := map[string]int{}

	for _, ar := range records {
		fields := ar.Schema().Fields()
		for i, field := range fields {
			switch field.Name {
			case profile.ColumnTimestamp:
				columnIndices.Timestamp = i
				continue
			case ValuePerSecond:
				columnIndices.PerSecondValue = i
				continue
			case totalSumColumn:
				columnIndices.ValueSum = i
				continue
			}

			if strings.HasPrefix(field.Name, "labels.") {
				labelColumnIndices = append(labelColumnIndices, i)
			}
		}

		if columnIndices.Timestamp == -1 {
			return nil, errors.New("timestamp column not found")
		}
		if columnIndices.PerSecondValue == -1 {
			return nil, errors.New("sum(value_per_second) column not found")
		}
		if columnIndices.ValueSum == -1 {
			return nil, errors.New("sum(value) column not found")
		}

		for i := 0; i < int(ar.NumRows()); i++ {
			labelSet = labelSet[:0]
			for _, labelColumnIndex := range labelColumnIndices {
				col := ar.Column(labelColumnIndex).(*array.Dictionary)
				if col.IsNull(i) {
					continue
				}

				v := col.Dictionary().(*array.Binary).Value(col.GetValueIndex(i))
				if len(v) > 0 {
					labelSet = append(labelSet, labels.Label{Name: strings.TrimPrefix(fields[labelColumnIndex].Name, "labels."), Value: string(v)})
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
				resSeries = append(resSeries, &pb.MetricsSeries{
					Labelset: &profilestorepb.LabelSet{Labels: pbLabelSet},
					PeriodType: &pb.ValueType{
						Type: m.PeriodType.Type,
						Unit: m.PeriodType.Unit,
					},
					SampleType: &pb.ValueType{
						Type: resultType.Type,
						Unit: resultType.Unit,
					},
				})
				index = len(resSeries) - 1
				labelsetToIndex[s] = index
			}

			ts := ar.Column(columnIndices.Timestamp).(*array.Int64).Value(i)
			valueSum := ar.Column(columnIndices.ValueSum).(*array.Int64).Value(i)
			valuePerSecond := ar.Column(columnIndices.PerSecondValue).(*array.Float64).Value(i)

			series := resSeries[index]
			series.Samples = append(series.Samples, &pb.MetricsSample{
				Timestamp:      timestamppb.New(timestamp.Time(ts)),
				Value:          valueSum,
				ValuePerSecond: valuePerSecond,
				Duration:       step.Nanoseconds(),
			})
		}
	}

	// This is horrible and should be fixed. The data is sorted in the storage, we should not have to sort it here.
	for _, series := range resSeries {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].Timestamp.AsTime().Before(series.Samples[j].Timestamp.AsTime())
		})
	}

	return resSeries, nil
}

func (q *Querier) queryRangeNonDelta(ctx context.Context, filterExpr logicalplan.Expr, step time.Duration) ([]*pb.MetricsSeries, error) {
	records := []arrow.Record{}
	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()
	rows := 0

	valueSum := logicalplan.Sum(logicalplan.Col(profile.ColumnValue))
	valueSumColumn := valueSum.Name()
	err := q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			[]*logicalplan.AggregationFunction{
				valueSum,
			},
			[]logicalplan.Expr{
				logicalplan.DynCol(profile.ColumnLabels),
				logicalplan.Col(profile.ColumnTimestamp),
			},
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			records = append(records, r)
			rows += int(r.NumRows())
			return nil
		})
	if err != nil {
		return nil, err
	}
	if len(records) == 0 || rows == 0 {
		return nil, status.Error(
			codes.NotFound,
			"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
		)
	}

	type columnIndex struct {
		index int
		found bool
	}
	// Add necessary columns and their found value is false by default.
	columnIndices := map[string]columnIndex{
		profile.ColumnTimestamp: {},
		valueSumColumn:          {},
	}
	labelColumnIndices := []int{}
	labelSet := labels.Labels{}
	resSeries := []*pb.MetricsSeries{}
	resSeriesBuckets := map[int]map[int64]struct{}{}
	labelsetToIndex := map[string]int{}

	for _, ar := range records {
		fields := ar.Schema().Fields()
		for i, field := range fields {
			if _, ok := columnIndices[field.Name]; ok {
				columnIndices[field.Name] = columnIndex{
					index: i,
					found: true,
				}
				continue
			}

			if strings.HasPrefix(field.Name, "labels.") {
				labelColumnIndices = append(labelColumnIndices, i)
			}
		}

		for name, index := range columnIndices {
			if !index.found {
				return nil, fmt.Errorf("%s column not found", name)
			}
		}

		for i := 0; i < int(ar.NumRows()); i++ {
			labelSet = labelSet[:0]
			for _, labelColumnIndex := range labelColumnIndices {
				col, ok := ar.Column(labelColumnIndex).(*array.Dictionary)
				if col.IsNull(i) || !ok {
					continue
				}

				v := StringValueFromDictionary(col, i)
				if len(v) > 0 {
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
				resSeries = append(resSeries, &pb.MetricsSeries{Labelset: &profilestorepb.LabelSet{Labels: pbLabelSet}})
				index = len(resSeries) - 1
				labelsetToIndex[s] = index
				resSeriesBuckets[index] = map[int64]struct{}{}
			}

			ts := ar.Column(columnIndices[profile.ColumnTimestamp].index).(*array.Int64).Value(i)
			value := ar.Column(columnIndices[valueSumColumn].index).(*array.Int64).Value(i)

			// Each step bucket will only return one of the timestamps and its value.
			// For this reason we'll take each timestamp and divide it by the step seconds.
			// If we have seen a MetricsSample for this bucket before, we'll ignore this one.
			// If we haven't seen one we'll add this sample to the response.

			// TODO: This still queries way too much data from the underlying database.
			// This needs to be moved to FrostDB to not even query all of this data in the first place.
			// With a scrape interval of 10s and a query range of 1d we'd query 8640 samples and at most return 960.
			// Even worse for a week, we'd query 60480 samples and only return 1000.
			tsBucket := ts / 1000 / int64(step.Seconds())
			if _, found := resSeriesBuckets[index][tsBucket]; found {
				// We already have a MetricsSample for this timestamp bucket, ignore it.
				continue
			}

			series := resSeries[index]
			series.Samples = append(series.Samples, &pb.MetricsSample{
				Timestamp:      timestamppb.New(timestamp.Time(ts)),
				Value:          value,
				ValuePerSecond: float64(value),
			})
			// Mark the timestamp bucket as filled by the above MetricsSample.
			resSeriesBuckets[index][tsBucket] = struct{}{}
		}
	}

	// This is horrible and should be fixed. The data is sorted in the storage, we should not have to sort it here.
	for _, series := range resSeries {
		sort.Slice(series.Samples, func(i, j int) bool {
			return series.Samples[i].Timestamp.AsTime().Before(series.Samples[j].Timestamp.AsTime())
		})
	}

	return resSeries, nil
}

func (q *Querier) ProfileTypes(
	ctx context.Context,
) ([]*pb.ProfileType, error) {
	seen := map[string]struct{}{}
	res := []*pb.ProfileType{}

	err := q.engine.ScanTable(q.tableName).
		Distinct(
			logicalplan.Col(profile.ColumnName),
			logicalplan.Col(profile.ColumnSampleType),
			logicalplan.Col(profile.ColumnSampleUnit),
			logicalplan.Col(profile.ColumnPeriodType),
			logicalplan.Col(profile.ColumnPeriodUnit),
			logicalplan.Col(profile.ColumnDuration).Gt(logicalplan.Literal(0)),
		).
		Execute(ctx, func(ctx context.Context, ar arrow.Record) error {
			if ar.NumCols() != 6 {
				return fmt.Errorf("expected 6 column, got %d", ar.NumCols())
			}

			nameColumn, err := DictionaryFromRecord(ar, profile.ColumnName)
			if err != nil {
				return err
			}

			sampleTypeColumn, err := DictionaryFromRecord(ar, profile.ColumnSampleType)
			if err != nil {
				return err
			}

			sampleUnitColumn, err := DictionaryFromRecord(ar, profile.ColumnSampleUnit)
			if err != nil {
				return err
			}

			periodTypeColumn, err := DictionaryFromRecord(ar, profile.ColumnPeriodType)
			if err != nil {
				return err
			}

			periodUnitColumn, err := DictionaryFromRecord(ar, profile.ColumnPeriodUnit)
			if err != nil {
				return err
			}

			deltaColumn, err := BooleanFieldFromRecord(ar, "duration > 0")
			if err != nil {
				return err
			}

			for i := 0; i < int(ar.NumRows()); i++ {
				name := StringValueFromDictionary(nameColumn, i)
				sampleType := StringValueFromDictionary(sampleTypeColumn, i)
				sampleUnit := StringValueFromDictionary(sampleUnitColumn, i)
				periodType := StringValueFromDictionary(periodTypeColumn, i)
				periodUnit := StringValueFromDictionary(periodUnitColumn, i)
				delta := deltaColumn.Value(i)

				key := fmt.Sprintf("%s:%s:%s:%s:%s", name, sampleType, sampleUnit, periodType, periodUnit)
				if delta {
					key = fmt.Sprintf("%s:delta", key)
				}

				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}

				res = append(res, &pb.ProfileType{
					Name:       name,
					SampleType: sampleType,
					SampleUnit: sampleUnit,
					PeriodType: periodType,
					PeriodUnit: periodUnit,
					Delta:      delta,
				})
			}

			return nil
		})
	if err != nil {
		return nil, err
	}

	return res, nil
}

func StringValueFromDictionary(arr *array.Dictionary, i int) string {
	switch dict := arr.Dictionary().(type) {
	case *array.Binary:
		return string(dict.Value(arr.GetValueIndex(i)))
	case *array.String:
		return dict.Value(arr.GetValueIndex(i))
	default:
		panic(fmt.Sprintf("unsupported dictionary type: %T", dict))
	}
}

func DictionaryFromRecord(ar arrow.Record, name string) (*array.Dictionary, error) {
	indices := ar.Schema().FieldIndices(name)
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected 1 column named %q, got %d", name, len(indices))
	}

	col, ok := ar.Column(indices[0]).(*array.Dictionary)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be a dictionary column, got %T", name, ar.Column(indices[0]))
	}

	return col, nil
}

func BinaryFieldFromRecord(ar arrow.Record, name string) (*array.Binary, error) {
	indices := ar.Schema().FieldIndices(name)
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected 1 column named %q, got %d", name, len(indices))
	}

	col, ok := ar.Column(indices[0]).(*array.Binary)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be a binary column, got %T", name, ar.Column(indices[0]))
	}

	return col, nil
}

func BooleanFieldFromRecord(ar arrow.Record, name string) (*array.Boolean, error) {
	indices := ar.Schema().FieldIndices(name)
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected 1 column named %q, got %d", name, len(indices))
	}

	col, ok := ar.Column(indices[0]).(*array.Boolean)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be a boolean column, got %T", name, ar.Column(indices[0]))
	}

	return col, nil
}

func (q *Querier) SymbolizeArrowRecord(
	ctx context.Context,
	records []arrow.Record,
	valueColumnName string,
) ([]arrow.Record, error) {
	res := make([]arrow.Record, len(records))

	for i, r := range records {
		schema := r.Schema()

		indices := schema.FieldIndices("stacktrace")
		if len(indices) != 1 {
			return nil, ErrMissingColumn{Column: "stacktrace", Columns: len(indices)}
		}
		stacktraceColumn := r.Column(indices[0]).(*array.Binary)

		rows := int(r.NumRows())
		stacktraceIDs := make([]string, rows)
		for i := 0; i < rows; i++ {
			stacktraceIDs[i] = string(stacktraceColumn.Value(i))
		}

		indices = schema.FieldIndices(valueColumnName)
		if len(indices) != 1 {
			return nil, ErrMissingColumn{Column: "value", Columns: len(indices)}
		}
		valueColumn := r.Column(indices[0]).(*array.Int64)

		profileLabels := []arrow.Field{}
		profileLabelColumns := []arrow.Array{}
		for i, field := range schema.Fields() {
			if strings.HasPrefix(field.Name, profile.ColumnPprofLabelsPrefix) {
				profileLabels = append(profileLabels, field)
				profileLabelColumns = append(profileLabelColumns, r.Column(i))
			}
		}

		stacktraces, locations, locationIndex, err := q.symbolizer.resolveStacktraceLocations(ctx, stacktraceIDs)
		if err != nil {
			return nil, fmt.Errorf("resolve stacktrace locations: %w", err)
		}

		locationsRecord, err := BuildArrowLocations(q.pool, stacktraces, locations, locationIndex)
		if err != nil {
			return nil, fmt.Errorf("build arrow locations: %w", err)
		}
		defer locationsRecord.Release()

		columns := make([]arrow.Array, len(profileLabels)+3) // +3 for stacktrace locations, value and diff
		copy(columns, profileLabelColumns)
		columns[len(columns)-3] = locationsRecord.Column(0)
		columns[len(columns)-2] = valueColumn

		diffColumn := CreateDiffColumn(q.pool, rows)
		defer diffColumn.Release()
		columns[len(columns)-1] = diffColumn

		res[i] = array.NewRecord(profile.ArrowSchema(profileLabels), columns, r.NumRows())
	}

	return res, nil
}

func CreateDiffColumn(pool memory.Allocator, rows int) arrow.Array {
	b := array.NewInt64Builder(pool)
	defer b.Release()

	values := make([]int64, 0, rows)
	valid := make([]bool, 0, rows)
	for i := 0; i < rows; i++ {
		values = append(values, 0)
		valid = append(valid, true)
	}
	b.AppendValues(values, valid)
	arr := b.NewInt64Array()

	return arr
}

func (q *Querier) QuerySingle(
	ctx context.Context,
	query string,
	time time.Time,
) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/QuerySingle")
	defer span.End()

	records, valueColumn, meta, err := q.findSingle(ctx, query, time)
	if err != nil {
		return profile.Profile{}, err
	}
	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()

	symbolizedRecords, err := q.SymbolizeArrowRecord(
		ctx,
		records,
		valueColumn,
	)
	if err != nil {
		// if the column cannot be found the timestamp is too far in the past and we don't have data
		var colErr ErrMissingColumn
		if errors.As(err, &colErr) {
			return profile.Profile{}, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
		}
		return profile.Profile{}, err
	}

	totalRows := int64(0)
	for _, r := range symbolizedRecords {
		totalRows += r.NumRows()
	}

	if totalRows == 0 {
		return profile.Profile{}, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
	}

	return profile.Profile{
		Meta:    meta,
		Samples: symbolizedRecords,
	}, nil
}

func (q *Querier) findSingle(ctx context.Context, query string, t time.Time) ([]arrow.Record, string, profile.Meta, error) {
	requestedTime := timestamp.FromTime(t)

	ctx, span := q.tracer.Start(ctx, "Querier/findSingle")
	span.SetAttributes(attribute.String("query", query))
	span.SetAttributes(attribute.Int64("time", t.Unix()))
	defer span.End()

	queryParts, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	filterExpr := logicalplan.And(
		append(
			selectorExprs,
			logicalplan.Col("timestamp").Eq(logicalplan.Literal(requestedTime)),
		)...,
	)

	records := []arrow.Record{}
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			[]*logicalplan.AggregationFunction{
				logicalplan.Sum(logicalplan.Col(profile.ColumnValue)),
			},
			[]logicalplan.Expr{
				logicalplan.Col(profile.ColumnStacktrace),
				logicalplan.DynCol(profile.ColumnPprofLabels),
				logicalplan.DynCol(profile.ColumnPprofNumLabels),
			},
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			records = append(records, r)
			return nil
		})
	if err != nil {
		return nil, "", profile.Meta{}, fmt.Errorf("execute query: %w", err)
	}

	return records,
		"sum(value)",
		profile.Meta{
			Name:       queryParts.Meta.Name,
			SampleType: queryParts.Meta.SampleType,
			PeriodType: queryParts.Meta.PeriodType,
			Timestamp:  requestedTime,
		},
		nil
}

func (q *Querier) QueryMerge(ctx context.Context, query string, start, end time.Time, aggregateByLabels bool) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/QueryMerge")
	defer span.End()

	records, valueColumn, meta, err := q.selectMerge(ctx, query, start, end, aggregateByLabels)
	if err != nil {
		return profile.Profile{}, err
	}
	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()

	symbolizedRecords, err := q.SymbolizeArrowRecord(
		ctx,
		records,
		valueColumn,
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return profile.Profile{
		Meta:    meta,
		Samples: symbolizedRecords,
	}, nil
}

func (q *Querier) selectMerge(
	ctx context.Context,
	query string,
	startTime,
	endTime time.Time,
	aggregateByLabels bool,
) ([]arrow.Record, string, profile.Meta, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/selectMerge")
	defer span.End()

	queryParts, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	start := timestamp.FromTime(startTime)
	end := timestamp.FromTime(endTime)
	resultType := queryParts.Meta.SampleType

	filterExpr := logicalplan.And(
		append(
			selectorExprs,
			logicalplan.Col(profile.ColumnTimestamp).GtEq(logicalplan.Literal(start)),
			logicalplan.Col(profile.ColumnTimestamp).LtEq(logicalplan.Literal(end)),
		)...,
	)

	aggrCols := []logicalplan.Expr{
		logicalplan.Col(profile.ColumnStacktrace),
	}

	if aggregateByLabels {
		aggrCols = append(
			aggrCols,
			logicalplan.DynCol(profile.ColumnPprofLabels),
			logicalplan.DynCol(profile.ColumnPprofNumLabels),
		)
	}

	var valueCol logicalplan.Expr = logicalplan.Col(profile.ColumnValue)
	if resultType.Type == "samples" && resultType.Unit == "count" {
		valueCol = logicalplan.Mul(logicalplan.Col(profile.ColumnValue), logicalplan.Col(profile.ColumnPeriod)).Alias(profile.ColumnValue)
		resultType = queryParts.Meta.PeriodType
	}

	records := []arrow.Record{}
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Project(
			append(aggrCols, valueCol)...,
		).
		Aggregate(
			[]*logicalplan.AggregationFunction{
				logicalplan.Sum(logicalplan.Col(profile.ColumnValue)),
			},
			aggrCols,
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			records = append(records, r)
			return nil
		})
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	meta := profile.Meta{
		Name:       queryParts.Meta.Name,
		SampleType: resultType,
		PeriodType: queryParts.Meta.PeriodType,
		Timestamp:  start,
	}
	return records, "sum(value)", meta, nil
}
