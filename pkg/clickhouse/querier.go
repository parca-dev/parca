// Copyright 2024-2026 The Parca Authors
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

package clickhouse

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/prometheus/model/timestamp"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	metapb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbolizer"
)

// Querier implements the query.Querier interface for ClickHouse.
type Querier struct {
	client     *Client
	logger     log.Logger
	tracer     trace.Tracer
	mem        memory.Allocator
	symbolizer symbolizer.SymbolizationClient
}

// NewQuerier creates a new ClickHouse querier.
func NewQuerier(
	client *Client,
	logger log.Logger,
	tracer trace.Tracer,
	mem memory.Allocator,
	sym symbolizer.SymbolizationClient,
) *Querier {
	return &Querier{
		client:     client,
		logger:     logger,
		tracer:     tracer,
		mem:        mem,
		symbolizer: sym,
	}
}

// Labels returns the unique label names within the given time range.
func (q *Querier) Labels(
	ctx context.Context,
	match []string,
	start, end time.Time,
	profileType string,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/Labels")
	defer span.End()

	table := q.client.FullTableName()
	query := fmt.Sprintf(`
		SELECT DISTINCT arrayJoin(JSONAllPaths(labels)) as label_name
		FROM %s
	`, table)

	var args []interface{}
	var conditions []string

	// Only apply time filter if both start and end are non-zero
	if start.Unix() != 0 && end.Unix() != 0 {
		conditions = append(conditions, "time_nanos > ? AND time_nanos < ?")
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	if profileType != "" {
		// profileType is a string like "process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta".
		// ParseQuery expects a full query string like "profile_type{label=value}", so we
		// append "{}" to create a valid query with empty matchers, then extract the profile
		// type components (name, sample type/unit, period type/unit, delta) for filtering.
		qp, err := profile.ParseQuery(profileType + "{}")
		if err == nil {
			profileFilter, profileArgs := ProfileTypeFilter(qp)
			conditions = append(conditions, profileFilter)
			args = append(args, profileArgs...)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY label_name"

	rows, err := q.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query labels: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var labelName string
		if err := rows.Scan(&labelName); err != nil {
			return nil, fmt.Errorf("failed to scan label name: %w", err)
		}
		result = append(result, labelName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// Values returns the unique values for a given label name.
func (q *Querier) Values(
	ctx context.Context,
	labelName string,
	match []string,
	start, end time.Time,
	profileType string,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/Values")
	defer span.End()

	table := q.client.FullTableName()
	labelPath := fmt.Sprintf("labels.%s", labelName)

	// Cast to String to avoid dynamic type issues with DISTINCT and ORDER BY
	query := fmt.Sprintf(`
		SELECT DISTINCT CAST(%s AS String) as label_value
		FROM %s
		WHERE %s IS NOT NULL
	`, labelPath, table, labelPath)

	var args []interface{}

	// Only apply time filter if both start and end are non-zero
	if start.Unix() != 0 && end.Unix() != 0 {
		query += " AND time_nanos > ? AND time_nanos < ?"
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	if profileType != "" {
		// profileType is a string like "process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta".
		// ParseQuery expects a full query string like "profile_type{label=value}", so we
		// append "{}" to create a valid query with empty matchers, then extract the profile
		// type components (name, sample type/unit, period type/unit, delta) for filtering.
		qp, err := profile.ParseQuery(profileType + "{}")
		if err == nil {
			profileFilter, profileArgs := ProfileTypeFilter(qp)
			query += " AND " + profileFilter
			args = append(args, profileArgs...)
		}
	}

	query += " ORDER BY label_value"

	rows, err := q.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query values: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("failed to scan value: %w", err)
		}
		if value != "" {
			result = append(result, value)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ProfileTypes returns the available profile types within the given time range.
func (q *Querier) ProfileTypes(
	ctx context.Context,
	start, end time.Time,
) ([]*pb.ProfileType, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/ProfileTypes")
	defer span.End()

	table := q.client.FullTableName()
	query := fmt.Sprintf(`
		SELECT DISTINCT
			name,
			sample_type,
			sample_unit,
			period_type,
			period_unit,
			(duration > 0) as delta
		FROM %s
	`, table)

	var args []interface{}

	// Only apply time filter if both start and end are non-zero
	if start.Unix() != 0 && end.Unix() != 0 {
		query += " WHERE time_nanos > ? AND time_nanos < ?"
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	rows, err := q.client.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query profile types: %w", err)
	}
	defer rows.Close()

	var result []*pb.ProfileType

	for rows.Next() {
		var (
			name       string
			sampleType string
			sampleUnit string
			periodType string
			periodUnit string
			delta      bool
		)
		if err := rows.Scan(&name, &sampleType, &sampleUnit, &periodType, &periodUnit, &delta); err != nil {
			return nil, fmt.Errorf("failed to scan profile type: %w", err)
		}

		result = append(result, &pb.ProfileType{
			Name:       name,
			SampleType: sampleType,
			SampleUnit: sampleUnit,
			PeriodType: periodType,
			PeriodUnit: periodUnit,
			Delta:      delta,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// HasProfileData checks if there is any profile data in the store.
func (q *Querier) HasProfileData(ctx context.Context) (bool, error) {
	types, err := q.ProfileTypes(ctx, time.UnixMilli(0), time.UnixMilli(0))
	if err != nil {
		return false, err
	}
	return len(types) > 0, nil
}

// QueryRange executes a range query and returns time series data.
// It aggregates samples by timestamp bucket and labels in ClickHouse,
// returning one row per unique labelset with an array of time series samples.
func (q *Querier) QueryRange(
	ctx context.Context,
	query string,
	startTime, endTime time.Time,
	step time.Duration,
	limit uint32,
	sumBy []string,
) ([]*pb.MetricsSeries, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/QueryRange")
	defer span.End()

	qp, err := profile.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	// The step cannot be lower than 1s
	if step < time.Second {
		step = time.Second
	}

	table := q.client.FullTableName()
	start := startTime.UnixNano()
	end := endTime.UnixNano()

	// Build profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Build label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	// Build sumBy label selections - cast to String because by default it is forbidden to group by
	// dynamic types in ClickHouse
	sumBySelects := ""
	outerLabelSelects := ""
	if len(sumBy) > 0 {
		selects := make([]string, len(sumBy))
		outerSelects := make([]string, len(sumBy))
		for i, s := range sumBy {
			labelPath := fmt.Sprintf("labels.%s", s)
			selects[i] = fmt.Sprintf("CAST(%s AS String) AS label_%s", labelPath, s)
			outerSelects[i] = fmt.Sprintf("label_%s", s)
		}
		sumBySelects = ", " + strings.Join(selects, ", ")
		outerLabelSelects = strings.Join(outerSelects, ", ") + ", "
	}

	// Build the query with a subquery that aggregates by timestamp and labels,
	// then an outer query that groups by labels and collects samples into an array.
	// This reduces the number of rows returned and moves grouping logic to ClickHouse.
	// GROUP BY ALL automatically groups by all non-aggregate columns.
	innerQuery := fmt.Sprintf(`
		SELECT
			intDiv(time_nanos, ?) * ? as timestamp_bucket,
			sum(value) as total_sum,
			min(duration) as duration_min
			%s
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?
	`, sumBySelects, table, profileFilter)

	// Build args in the correct order matching placeholder positions
	args := []interface{}{step.Nanoseconds(), step.Nanoseconds()}
	args = append(args, profileArgs...)
	args = append(args, start, end)

	if labelFilter != "" {
		innerQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	innerQuery += `
		GROUP BY ALL
		ORDER BY timestamp_bucket
	`

	// Outer query groups by labels and collects (timestamp, value, duration) tuples.
	// GROUP BY ALL handles both cases: with labels it groups by them, without labels
	// it returns a single row with all samples.
	sqlQuery := fmt.Sprintf(`
		SELECT
			%s
			groupArray(tuple(timestamp_bucket, total_sum, duration_min)) as samples
		FROM (%s)
		GROUP BY ALL
	`, outerLabelSelects, innerQuery)

	rows, err := q.client.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query range: %w", err)
	}
	defer rows.Close()

	var resSeries []*pb.MetricsSeries

	// Each row represents one unique labelset with all its samples
	for rows.Next() {
		// Scan label columns
		labelValues := make([]string, len(sumBy))
		scanArgs := make([]interface{}, 0, len(sumBy)+1)
		for i := range sumBy {
			scanArgs = append(scanArgs, &labelValues[i])
		}

		// Scan samples array - ClickHouse returns array of tuples as slice of slices
		// Each tuple element becomes []interface{}{timestamp_bucket, total_sum, duration_min}
		var samples [][]interface{}
		scanArgs = append(scanArgs, &samples)

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Build labelset
		pbLabelSet := make([]*profilestorepb.Label, len(sumBy))
		for i, s := range sumBy {
			pbLabelSet[i] = &profilestorepb.Label{
				Name:  s,
				Value: labelValues[i],
			}
		}

		// Build samples for this series
		pbSamples := make([]*pb.MetricsSample, 0, len(samples))
		for _, sample := range samples {
			// Extract values from tuple (timestamp_bucket, total_sum, duration_min)
			timestampBucket := sample[0].(int64)
			totalSum := sample[1].(int64)
			durationMin := sample[2].(int64)

			// Calculate value per second
			valuePerSecond := float64(totalSum)
			if durationMin > 0 {
				valuePerSecond = float64(totalSum) / (float64(durationMin) / float64(time.Second.Nanoseconds()))
			}

			pbSamples = append(pbSamples, &pb.MetricsSample{
				Timestamp:      timestamppb.New(time.Unix(0, timestampBucket)),
				Value:          totalSum,
				ValuePerSecond: valuePerSecond,
				Duration:       durationMin,
			})
		}

		resSeries = append(resSeries, &pb.MetricsSeries{
			Labelset: &profilestorepb.LabelSet{Labels: pbLabelSet},
			PeriodType: &pb.ValueType{
				Type: qp.Meta.PeriodType.Type,
				Unit: qp.Meta.PeriodType.Unit,
			},
			SampleType: &pb.ValueType{
				Type: qp.Meta.SampleType.Type,
				Unit: qp.Meta.SampleType.Unit,
			},
			Samples: pbSamples,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	if len(resSeries) == 0 {
		return nil, status.Error(
			codes.NotFound,
			"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
		)
	}

	return resSeries, nil
}

// QuerySingle executes a point query for a single timestamp.
func (q *Querier) QuerySingle(
	ctx context.Context,
	query string,
	t time.Time,
	invertCallStacks bool,
) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/QuerySingle")
	defer span.End()

	qp, err := profile.ParseQuery(query)
	if err != nil {
		return profile.Profile{}, err
	}

	table := q.client.FullTableName()
	requestedTime := timestamp.FromTime(t)

	// Build profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Build label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return profile.Profile{}, err
	}

	// Query aggregates by stacktrace and labels, summing values and durations.
	// sum(duration) matches FrostDB behavior for per-second value calculation.
	// Period is grouped by since it should be constant for a profile type.
	// Cast to Int64 explicitly to ensure correct type for Go scanning.
	sqlQuery := fmt.Sprintf(`
		SELECT
			stacktrace.address,
			stacktrace.mapping_start,
			stacktrace.mapping_limit,
			stacktrace.mapping_offset,
			stacktrace.mapping_file,
			stacktrace.mapping_build_id,
			stacktrace.line_number,
			stacktrace.function_name,
			stacktrace.function_system_name,
			stacktrace.function_filename,
			stacktrace.function_start_line,
			sum(value) as value,
			toString(labels) as labels_json,
			CAST(sum(duration) AS Int64) as sample_duration,
			period as sample_period
		FROM %s
		WHERE %s
		  AND timestamp = ?
	`, table, profileFilter)

	// Build args in the correct order matching placeholder positions
	args := append([]interface{}{}, profileArgs...)
	args = append(args, requestedTime)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	sqlQuery += `
		GROUP BY
			stacktrace.address,
			stacktrace.mapping_start,
			stacktrace.mapping_limit,
			stacktrace.mapping_offset,
			stacktrace.mapping_file,
			stacktrace.mapping_build_id,
			stacktrace.line_number,
			stacktrace.function_name,
			stacktrace.function_system_name,
			stacktrace.function_filename,
			stacktrace.function_start_line,
			labels_json,
			period
	`

	rows, err := q.client.Query(ctx, sqlQuery, args...)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("failed to execute query single: %w", err)
	}
	defer rows.Close()

	// Build Arrow record from results
	records, err := q.rowsToArrowRecords(ctx, rows, invertCallStacks)
	if err != nil {
		return profile.Profile{}, err
	}

	if len(records) == 0 {
		return profile.Profile{}, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
	}

	qp.Meta.Timestamp = requestedTime

	return profile.Profile{
		Meta:    qp.Meta,
		Samples: records,
	}, nil
}

// QueryMerge executes a merge query over a time range.
func (q *Querier) QueryMerge(
	ctx context.Context,
	query string,
	start, end time.Time,
	aggregateByLabels []string,
	invertCallStacks bool,
	functionToFilterBy string,
) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/QueryMerge")
	defer span.End()

	qp, err := profile.ParseQuery(query)
	if err != nil {
		return profile.Profile{}, err
	}

	table := q.client.FullTableName()
	startNanos := start.UnixNano()
	endNanos := end.UnixNano()

	// Build profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Build label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return profile.Profile{}, err
	}

	// Build group by labels - cast to String to avoid dynamic type issues
	groupByLabels := ""
	if len(aggregateByLabels) > 0 {
		labels := make([]string, len(aggregateByLabels))
		for i, l := range aggregateByLabels {
			labelPath := l
			if !strings.HasPrefix(l, "labels.") {
				labelPath = fmt.Sprintf("labels.%s", l)
			}
			// Cast to String to avoid ClickHouse dynamic type GROUP BY issues
			labels[i] = fmt.Sprintf("CAST(%s AS String)", labelPath)
		}
		groupByLabels = ", " + strings.Join(labels, ", ")
	}

	// Query aggregates by stacktrace, period, and optional labels.
	// Period is grouped by (like FrostDB's ColumnPeriod in selectMerge).
	// Duration uses the query time range for per-second calculation (matching FrostDB).
	// Cast to Int64 explicitly to ensure correct type for Go scanning.
	queryDuration := endNanos - startNanos
	sqlQuery := fmt.Sprintf(`
		SELECT
			stacktrace.address,
			stacktrace.mapping_start,
			stacktrace.mapping_limit,
			stacktrace.mapping_offset,
			stacktrace.mapping_file,
			stacktrace.mapping_build_id,
			stacktrace.line_number,
			stacktrace.function_name,
			stacktrace.function_system_name,
			stacktrace.function_filename,
			stacktrace.function_start_line,
			sum(value) as value_sum,
			'' as labels_json,
			CAST(%d AS Int64) as sample_duration,
			period as sample_period
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?
	`, queryDuration, table, profileFilter)

	// Build args in the correct order matching placeholder positions
	args := append([]interface{}{}, profileArgs...)
	args = append(args, startNanos, endNanos)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	sqlQuery += fmt.Sprintf(`
		GROUP BY
			stacktrace.address,
			stacktrace.mapping_start,
			stacktrace.mapping_limit,
			stacktrace.mapping_offset,
			stacktrace.mapping_file,
			stacktrace.mapping_build_id,
			stacktrace.line_number,
			stacktrace.function_name,
			stacktrace.function_system_name,
			stacktrace.function_filename,
			stacktrace.function_start_line,
			period
			%s
	`, groupByLabels)

	rows, err := q.client.Query(ctx, sqlQuery, args...)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("failed to execute query merge: %w", err)
	}
	defer rows.Close()

	// Build Arrow record from results
	records, err := q.rowsToArrowRecords(ctx, rows, invertCallStacks)
	if err != nil {
		return profile.Profile{}, err
	}

	qp.Meta.Timestamp = startNanos

	return profile.Profile{
		Meta:    qp.Meta,
		Samples: records,
	}, nil
}

// GetProfileMetadataMappings returns unique mapping files for the given query.
func (q *Querier) GetProfileMetadataMappings(
	ctx context.Context,
	query string,
	start, end time.Time,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/GetProfileMetadataMappings")
	defer span.End()

	qp, err := profile.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	table := q.client.FullTableName()
	startNanos := start.UnixNano()
	endNanos := end.UnixNano()

	// Build profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Build label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	sqlQuery := fmt.Sprintf(`
		SELECT DISTINCT arrayJoin(stacktrace.mapping_file) as mapping_file
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?
	`, table, profileFilter)

	// Args must be in same order as placeholders: profileArgs, then time args, then label args
	args := append([]interface{}{}, profileArgs...)
	args = append(args, startNanos, endNanos)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	sqlQuery += " ORDER BY mapping_file"

	rows, err := q.client.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query mapping files: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var mappingFile string
		if err := rows.Scan(&mappingFile); err != nil {
			return nil, fmt.Errorf("failed to scan mapping file: %w", err)
		}
		result = append(result, mappingFile)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// GetProfileMetadataLabels returns unique label names for the given query.
func (q *Querier) GetProfileMetadataLabels(
	ctx context.Context,
	query string,
	start, end time.Time,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "ClickHouse/GetProfileMetadataLabels")
	defer span.End()

	qp, err := profile.ParseQuery(query)
	if err != nil {
		return nil, err
	}

	table := q.client.FullTableName()
	startNanos := start.UnixNano()
	endNanos := end.UnixNano()

	// Build profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Build label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	sqlQuery := fmt.Sprintf(`
		SELECT DISTINCT arrayJoin(JSONAllPaths(labels)) as label_name
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?
	`, table, profileFilter)

	// Args must be in same order as placeholders: profileArgs, then time args, then label args
	args := append([]interface{}{}, profileArgs...)
	args = append(args, startNanos, endNanos)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	sqlQuery += " ORDER BY label_name"

	rows, err := q.client.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query labels: %w", err)
	}
	defer rows.Close()

	var result []string
	for rows.Next() {
		var labelName string
		if err := rows.Scan(&labelName); err != nil {
			return nil, fmt.Errorf("failed to scan label name: %w", err)
		}
		result = append(result, labelName)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// sampleData holds the data for a single sample row from ClickHouse.
type sampleData struct {
	addresses           []uint64
	mappingStarts       []uint64
	mappingLimits       []uint64
	mappingOffsets      []uint64
	mappingFiles        []string
	mappingBuildIDs     []string
	lineNumbers         []int64
	functionNames       []string
	functionSystemNames []string
	functionFilenames   []string
	functionStartLines  []int64
	value               int64
	labelsJSON          string
	duration            int64
	period              int64
}

// rowsToArrowRecords converts ClickHouse query results to Arrow records.
func (q *Querier) rowsToArrowRecords(
	ctx context.Context,
	rows interface {
		Next() bool
		Scan(dest ...interface{}) error
		Err() error
	},
	invertCallStacks bool,
) ([]arrow.RecordBatch, error) {
	_, span := q.tracer.Start(ctx, "ClickHouse/rowsToArrowRecords")
	defer span.End()

	// First pass: collect all sample data and build symbolization requests
	var samples []sampleData
	locationIndex := make(map[string]map[uint64]*profile.Location) // buildID -> address -> location

	for rows.Next() {
		var s sampleData
		if err := rows.Scan(
			&s.addresses,
			&s.mappingStarts,
			&s.mappingLimits,
			&s.mappingOffsets,
			&s.mappingFiles,
			&s.mappingBuildIDs,
			&s.lineNumbers,
			&s.functionNames,
			&s.functionSystemNames,
			&s.functionFilenames,
			&s.functionStartLines,
			&s.value,
			&s.labelsJSON,
			&s.duration,
			&s.period,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		samples = append(samples, s)

		// Collect locations that need symbolization
		for i := 0; i < len(s.addresses); i++ {
			buildID := ""
			if i < len(s.mappingBuildIDs) {
				buildID = s.mappingBuildIDs[i]
			}
			addr := s.addresses[i]

			// Check if this location needs symbolization (no function name but has build ID)
			needsSymbolization := (i >= len(s.functionNames) || s.functionNames[i] == "") && buildID != "" && addr != 0

			if needsSymbolization {
				if _, ok := locationIndex[buildID]; !ok {
					locationIndex[buildID] = make(map[uint64]*profile.Location)
				}
				if _, ok := locationIndex[buildID][addr]; !ok {
					mappingFile := ""
					if i < len(s.mappingFiles) {
						mappingFile = s.mappingFiles[i]
					}
					var mappingStart, mappingLimit, mappingOffset uint64
					if i < len(s.mappingStarts) {
						mappingStart = s.mappingStarts[i]
					}
					if i < len(s.mappingLimits) {
						mappingLimit = s.mappingLimits[i]
					}
					if i < len(s.mappingOffsets) {
						mappingOffset = s.mappingOffsets[i]
					}

					locationIndex[buildID][addr] = &profile.Location{
						Address: addr,
						Mapping: &metapb.Mapping{
							BuildId: buildID,
							File:    mappingFile,
							Start:   mappingStart,
							Limit:   mappingLimit,
							Offset:  mappingOffset,
						},
					}
				}
			}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Call symbolizer for each build ID
	for buildID, addrMap := range locationIndex {
		locs := make([]*profile.Location, 0, len(addrMap))
		for _, loc := range addrMap {
			locs = append(locs, loc)
		}

		symReq := symbolizer.SymbolizationRequest{
			BuildID: buildID,
			Mappings: []symbolizer.SymbolizationRequestMappingAddrs{
				{Locations: locs},
			},
		}

		if err := q.symbolizer.Symbolize(ctx, symReq); err != nil {
			level.Error(q.logger).Log("msg", "failed to symbolize locations", "buildID", buildID, "err", err)
			// Continue even if symbolization fails
		}
	}

	// Second pass: build Arrow records with symbolized data
	w := profile.NewWriter(q.mem, []string{})
	defer w.Release()

	for _, s := range samples {
		w.LocationsList.Append(true)

		numLocations := len(s.addresses)
		for i := 0; i < numLocations; i++ {
			idx := i
			if invertCallStacks {
				idx = numLocations - 1 - i
			}

			w.Locations.Append(true)
			w.Addresses.Append(s.addresses[idx])

			var mappingStart, mappingLimit, mappingOffset uint64
			if idx < len(s.mappingStarts) {
				mappingStart = s.mappingStarts[idx]
			}
			if idx < len(s.mappingLimits) {
				mappingLimit = s.mappingLimits[idx]
			}
			if idx < len(s.mappingOffsets) {
				mappingOffset = s.mappingOffsets[idx]
			}
			w.MappingStart.Append(mappingStart)
			w.MappingLimit.Append(mappingLimit)
			w.MappingOffset.Append(mappingOffset)

			if idx < len(s.mappingFiles) {
				if err := w.MappingFile.Append([]byte(s.mappingFiles[idx])); err != nil {
					level.Error(q.logger).Log("msg", "failed to append mapping file", "err", err)
				}
			} else {
				w.MappingFile.AppendNull()
			}

			if idx < len(s.mappingBuildIDs) {
				if err := w.MappingBuildID.Append([]byte(s.mappingBuildIDs[idx])); err != nil {
					level.Error(q.logger).Log("msg", "failed to append mapping build id", "err", err)
				}
			} else {
				w.MappingBuildID.AppendNull()
			}

			// Check if we have symbolized data for this location
			buildID := ""
			if idx < len(s.mappingBuildIDs) {
				buildID = s.mappingBuildIDs[idx]
			}
			addr := s.addresses[idx]

			var symbolizedLoc *profile.Location
			if buildID != "" {
				if addrMap, ok := locationIndex[buildID]; ok {
					symbolizedLoc = addrMap[addr]
				}
			}

			// Use symbolized data if available, otherwise use stored data
			if symbolizedLoc != nil && len(symbolizedLoc.Lines) > 0 {
				w.Lines.Append(true)
				for _, line := range symbolizedLoc.Lines {
					w.Line.Append(true)
					w.LineNumber.Append(line.Line)
					if line.Function != nil {
						if err := w.FunctionName.Append([]byte(line.Function.Name)); err != nil {
							level.Error(q.logger).Log("msg", "failed to append function name", "err", err)
						}
						if err := w.FunctionSystemName.Append([]byte(line.Function.SystemName)); err != nil {
							level.Error(q.logger).Log("msg", "failed to append function system name", "err", err)
						}
						if err := w.FunctionFilename.Append([]byte(line.Function.Filename)); err != nil {
							level.Error(q.logger).Log("msg", "failed to append function filename", "err", err)
						}
						w.FunctionStartLine.Append(line.Function.StartLine)
					} else {
						w.FunctionName.AppendNull()
						w.FunctionSystemName.AppendNull()
						w.FunctionFilename.AppendNull()
						w.FunctionStartLine.AppendNull()
					}
				}
			} else if idx < len(s.functionNames) && s.functionNames[idx] != "" {
				// Use stored function data
				w.Lines.Append(true)
				w.Line.Append(true)
				w.LineNumber.Append(s.lineNumbers[idx])
				if err := w.FunctionName.Append([]byte(s.functionNames[idx])); err != nil {
					level.Error(q.logger).Log("msg", "failed to append function name", "err", err)
				}
				if idx < len(s.functionSystemNames) {
					if err := w.FunctionSystemName.Append([]byte(s.functionSystemNames[idx])); err != nil {
						level.Error(q.logger).Log("msg", "failed to append function system name", "err", err)
					}
				} else {
					w.FunctionSystemName.AppendNull()
				}
				if idx < len(s.functionFilenames) {
					if err := w.FunctionFilename.Append([]byte(s.functionFilenames[idx])); err != nil {
						level.Error(q.logger).Log("msg", "failed to append function filename", "err", err)
					}
				} else {
					w.FunctionFilename.AppendNull()
				}
				if idx < len(s.functionStartLines) {
					w.FunctionStartLine.Append(s.functionStartLines[idx])
				} else {
					w.FunctionStartLine.AppendNull()
				}
			} else {
				w.Lines.AppendNull()
			}
		}

		w.Value.Append(s.value)

		// Create zero diff column
		w.Diff.Append(0)

		// Append timestamp and period
		w.TimeNanos.Append(0)
		w.Period.Append(s.period)
	}

	if len(samples) == 0 {
		return nil, nil
	}

	record := w.RecordBuilder.NewRecordBatch()
	return []arrow.RecordBatch{record}, nil
}
