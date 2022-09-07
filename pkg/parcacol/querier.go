// Copyright 2022 The Parca Authors
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

	"github.com/apache/arrow/go/v8/arrow"
	"github.com/apache/arrow/go/v8/arrow/array"
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

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

var (
	ErrTimestampColumnNotFound = errors.New("timestamp column not found")
	ErrValueColumnNotFound     = errors.New("value column not found")
)

type Engine interface {
	ScanTable(name string) query.Builder
	ScanSchema(name string) query.Builder
}

func NewQuerier(
	tracer trace.Tracer,
	engine Engine,
	tableName string,
	metastore metastorepb.MetastoreServiceClient,
) *Querier {
	return &Querier{
		tracer:    tracer,
		engine:    engine,
		tableName: tableName,
		converter: NewArrowToProfileConverter(
			tracer,
			metastore,
		),
	}
}

type Querier struct {
	engine    Engine
	tableName string
	converter *ArrowToProfileConverter
	tracer    trace.Tracer
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
			stringCol, ok := col.(*array.Binary)
			if !ok {
				return fmt.Errorf("expected string column, got %T", col)
			}

			for i := 0; i < stringCol.Len(); i++ {
				val := stringCol.Value(i)
				vals = append(vals, string(val))
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

func QueryToFilterExprs(query string) (profile.Meta, []logicalplan.Expr, error) {
	parsedSelector, err := parser.ParseMetricSelector(query)
	if err != nil {
		return profile.Meta{}, nil, status.Error(codes.InvalidArgument, "failed to parse query")
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
		return profile.Meta{}, nil, status.Error(codes.InvalidArgument, "query must contain a profile-type selection")
	}

	parts := strings.Split(nameLabel.Value, ":")
	if len(parts) != 5 && len(parts) != 6 {
		return profile.Meta{}, nil, status.Errorf(codes.InvalidArgument, "profile-type selection must be of the form <name>:<sample-type>:<sample-unit>:<period-type>:<period-unit>(:delta), got(%d): %q", len(parts), nameLabel.Value)
	}
	name, sampleType, sampleUnit, periodType, periodUnit, delta := parts[0], parts[1], parts[2], parts[3], parts[4], false
	if len(parts) == 6 && parts[5] == "delta" {
		delta = true
	}

	labelFilterExpressions, err := MatchersToBooleanExpressions(sel)
	if err != nil {
		return profile.Meta{}, nil, status.Error(codes.InvalidArgument, "failed to build query")
	}

	exprs := append([]logicalplan.Expr{
		logicalplan.Col("name").Eq(logicalplan.Literal(name)),
		logicalplan.Col("sample_type").Eq(logicalplan.Literal(sampleType)),
		logicalplan.Col("sample_unit").Eq(logicalplan.Literal(sampleUnit)),
		logicalplan.Col("period_type").Eq(logicalplan.Literal(periodType)),
		logicalplan.Col("period_unit").Eq(logicalplan.Literal(periodUnit)),
	}, labelFilterExpressions...)

	deltaPlan := logicalplan.Col("duration").Eq(logicalplan.Literal(0))
	if delta {
		deltaPlan = logicalplan.Col("duration").NotEq(logicalplan.Literal(0))
	}

	exprs = append(exprs, deltaPlan)

	return profile.Meta{
		Name:       name,
		SampleType: profile.ValueType{Type: sampleType, Unit: sampleUnit},
		PeriodType: profile.ValueType{Type: periodType, Unit: periodUnit},
	}, exprs, nil
}

func (q *Querier) QueryRange(
	ctx context.Context,
	query string,
	startTime, endTime time.Time,
	limit uint32,
) ([]*pb.MetricsSeries, error) {
	_, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, err
	}

	start := timestamp.FromTime(startTime)
	end := timestamp.FromTime(endTime)

	exprs := append(
		selectorExprs,
		logicalplan.Col("timestamp").Gt(logicalplan.Literal(start)),
		logicalplan.Col("timestamp").Lt(logicalplan.Literal(end)),
	)

	filterExpr := logicalplan.And(exprs...)

	resSeries := []*pb.MetricsSeries{}
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
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, err
	}
	if ar == nil || ar.NumRows() == 0 {
		return nil, status.Error(
			codes.NotFound,
			"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
		)
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
			col := ar.Column(labelColumnIndex).(*array.Binary)
			if col.IsNull(i) {
				continue
			}

			v := col.Value(i)
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
			resSeries = append(resSeries, &pb.MetricsSeries{Labelset: &profilestorepb.LabelSet{Labels: pbLabelSet}})
			index = len(resSeries) - 1
			labelsetToIndex[s] = index
		}

		series := resSeries[index]
		series.Samples = append(series.Samples, &pb.MetricsSample{
			Timestamp: timestamppb.New(timestamp.Time(ar.Column(timestampColumnIndex).(*array.Int64).Value(i))),
			Value:     ar.Column(valueColumnIndex).(*array.Int64).Value(i),
		})
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
			logicalplan.Col(ColumnName),
			logicalplan.Col(ColumnSampleType),
			logicalplan.Col(ColumnSampleUnit),
			logicalplan.Col(ColumnPeriodType),
			logicalplan.Col(ColumnPeriodUnit),
			logicalplan.Col(ColumnDuration).Gt(logicalplan.Literal(0)),
		).
		Execute(ctx, func(ctx context.Context, ar arrow.Record) error {
			if ar.NumCols() != 6 {
				return fmt.Errorf("expected 6 column, got %d", ar.NumCols())
			}

			nameColumn, err := BinaryFieldFromRecord(ar, ColumnName)
			if err != nil {
				return err
			}

			sampleTypeColumn, err := BinaryFieldFromRecord(ar, ColumnSampleType)
			if err != nil {
				return err
			}

			sampleUnitColumn, err := BinaryFieldFromRecord(ar, ColumnSampleUnit)
			if err != nil {
				return err
			}

			periodTypeColumn, err := BinaryFieldFromRecord(ar, ColumnPeriodType)
			if err != nil {
				return err
			}

			periodUnitColumn, err := BinaryFieldFromRecord(ar, ColumnPeriodUnit)
			if err != nil {
				return err
			}

			deltaColumn, err := BooleanFieldFromRecord(ar, "duration > 0")
			if err != nil {
				return err
			}

			for i := 0; i < int(ar.NumRows()); i++ {
				name := string(nameColumn.Value(i))
				sampleType := string(sampleTypeColumn.Value(i))
				sampleUnit := string(sampleUnitColumn.Value(i))
				periodType := string(periodTypeColumn.Value(i))
				periodUnit := string(periodUnitColumn.Value(i))
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

func (q *Querier) arrowRecordToProfile(
	ctx context.Context,
	r arrow.Record,
	valueColumn string,
	meta profile.Meta,
) (*profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/arrowRecordToProfile")
	defer span.End()
	return q.converter.Convert(
		ctx,
		r,
		valueColumn,
		meta,
	)
}

func (q *Querier) QuerySingle(
	ctx context.Context,
	query string,
	time time.Time,
) (*profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/QuerySingle")
	defer span.End()

	ar, valueColumn, meta, err := q.findSingle(ctx, query, time)
	if err != nil {
		return nil, err
	}

	p, err := q.arrowRecordToProfile(
		ctx,
		ar,
		valueColumn,
		meta,
	)
	if err != nil {
		// if the column cannot be found the timestamp is too far in the past and we don't have data
		var colErr ErrMissingColumn
		if errors.As(err, &colErr) {
			return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
		}
		return nil, err
	}

	if p == nil {
		return nil, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
	}

	return p, nil
}

func (q *Querier) findSingle(ctx context.Context, query string, t time.Time) (arrow.Record, string, profile.Meta, error) {
	requestedTime := timestamp.FromTime(t)

	ctx, span := q.tracer.Start(ctx, "Querier/findSingle")
	span.SetAttributes(attribute.String("query", query))
	span.SetAttributes(attribute.Int64("time", t.Unix()))
	defer span.End()

	meta, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	filterExpr := logicalplan.And(
		append(
			selectorExprs,
			logicalplan.Col("timestamp").Eq(logicalplan.Literal(requestedTime)),
		)...,
	)

	var ar arrow.Record
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")),
			logicalplan.Col("stacktrace"),
			logicalplan.DynCol("pprof_labels"),
			logicalplan.DynCol("pprof_num_labels"),
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, "", profile.Meta{}, fmt.Errorf("execute query: %w", err)
	}

	return ar,
		"sum(value)",
		profile.Meta{
			Name:       meta.Name,
			SampleType: meta.SampleType,
			PeriodType: meta.PeriodType,
			Timestamp:  requestedTime,
		},
		nil
}

func (q *Querier) QueryMerge(ctx context.Context, query string, start, end time.Time) (*profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/QueryMerge")
	defer span.End()

	r, valueColumn, meta, err := q.selectMerge(ctx, query, start, end)
	if err != nil {
		return nil, err
	}
	defer r.Release()

	p, err := q.arrowRecordToProfile(
		ctx,
		r,
		valueColumn,
		meta,
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (q *Querier) selectMerge(ctx context.Context, query string, startTime, endTime time.Time) (arrow.Record, string, profile.Meta, error) {
	ctx, span := q.tracer.Start(ctx, "Querier/selectMerge")
	defer span.End()

	meta, selectorExprs, err := QueryToFilterExprs(query)
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	start := timestamp.FromTime(startTime)
	end := timestamp.FromTime(endTime)

	filterExpr := logicalplan.And(
		append(
			selectorExprs,
			logicalplan.Col("timestamp").Gt(logicalplan.Literal(start)),
			logicalplan.Col("timestamp").Lt(logicalplan.Literal(end)),
		)...,
	)

	var ar arrow.Record
	err = q.engine.ScanTable(q.tableName).
		Filter(filterExpr).
		Aggregate(
			logicalplan.Sum(logicalplan.Col("value")),
			logicalplan.Col("stacktrace"),
		).
		Execute(ctx, func(ctx context.Context, r arrow.Record) error {
			r.Retain()
			ar = r
			return nil
		})
	if err != nil {
		return nil, "", profile.Meta{}, err
	}

	meta = profile.Meta{
		Name:       meta.Name,
		SampleType: meta.SampleType,
		PeriodType: meta.PeriodType,
		Timestamp:  start,
	}
	return ar, "sum(value)", meta, nil
}
