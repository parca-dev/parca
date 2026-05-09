// Copyright 2026 The Parca Authors
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

package duckdb

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	duckdb "github.com/marcboeker/go-duckdb/v2"
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

// Querier implements the query.Querier interface against DuckDB.
type Querier struct {
	client     *Client
	logger     log.Logger
	tracer     trace.Tracer
	mem        memory.Allocator
	symbolizer symbolizer.SymbolizationClient
}

// NewQuerier returns a Querier reading from client.
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

// stacktraceLoc mirrors the STRUCT layout of stacktrace[i] in the table.
// Field tags must match the SQL column names exactly because go-duckdb's
// Composite scanner uses field names to map STRUCT entries.
type stacktraceLoc struct {
	Address            uint64 `db:"address"`
	MappingStart       uint64 `db:"mapping_start"`
	MappingLimit       uint64 `db:"mapping_limit"`
	MappingOffset      uint64 `db:"mapping_offset"`
	MappingFile        string `db:"mapping_file"`
	MappingBuildID     string `db:"mapping_build_id"`
	LineNumber         int64  `db:"line_number"`
	FunctionName       string `db:"function_name"`
	FunctionSystemName string `db:"function_system_name"`
	FunctionFilename   string `db:"function_filename"`
	FunctionStartLine  int64  `db:"function_start_line"`
}

func quotedTable(c *Client) string { return quoteIdent(c.Table()) }

// Labels returns the unique label names within the time range and matching
// the optional profile type.
func (q *Querier) Labels(
	ctx context.Context,
	_ []string,
	start, end time.Time,
	profileType string,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/Labels")
	defer span.End()

	var conditions []string
	var args []interface{}

	if start.Unix() != 0 && end.Unix() != 0 {
		conditions = append(conditions, "time_nanos > ? AND time_nanos < ?")
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	if profileType != "" {
		// profileType is "name:st:su:pt:pu[:delta]"; ParseQuery wants a full
		// `query{}` form so add empty matchers.
		if qp, err := profile.ParseQuery(profileType + "{}"); err == nil {
			profileFilter, profileArgs := ProfileTypeFilter(qp)
			conditions = append(conditions, profileFilter)
			args = append(args, profileArgs...)
		}
	}

	where := ""
	if len(conditions) > 0 {
		where = " WHERE " + strings.Join(conditions, " AND ")
	}

	// UNNEST(map_keys(labels)) flattens the per-row label keys into rows,
	// then DISTINCT collapses duplicates across the result. ORDER BY is
	// applied at the outer level.
	query := fmt.Sprintf(
		"SELECT DISTINCT k FROM (SELECT UNNEST(map_keys(labels)) AS k FROM %s%s) ORDER BY k",
		quotedTable(q.client), where,
	)

	rows, err := q.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query labels: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan label name: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// Values returns the unique values seen for labelName within the time range.
func (q *Querier) Values(
	ctx context.Context,
	labelName string,
	_ []string,
	start, end time.Time,
	profileType string,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/Values")
	defer span.End()

	labelExpr := fmt.Sprintf("element_at(labels, '%s')[1]", escapeIdent(labelName))

	conditions := []string{fmt.Sprintf("%s IS NOT NULL", labelExpr)}
	var args []interface{}

	if start.Unix() != 0 && end.Unix() != 0 {
		conditions = append(conditions, "time_nanos > ? AND time_nanos < ?")
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	if profileType != "" {
		if qp, err := profile.ParseQuery(profileType + "{}"); err == nil {
			profileFilter, profileArgs := ProfileTypeFilter(qp)
			conditions = append(conditions, profileFilter)
			args = append(args, profileArgs...)
		}
	}

	query := fmt.Sprintf(
		"SELECT DISTINCT %s AS v FROM %s WHERE %s ORDER BY v",
		labelExpr, quotedTable(q.client), strings.Join(conditions, " AND "),
	)

	rows, err := q.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query values: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var v sql.NullString
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan value: %w", err)
		}
		if v.Valid && v.String != "" {
			out = append(out, v.String)
		}
	}
	return out, rows.Err()
}

// ProfileTypes returns the distinct profile-type identifiers in the time
// range.
func (q *Querier) ProfileTypes(
	ctx context.Context,
	start, end time.Time,
) ([]*pb.ProfileType, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/ProfileTypes")
	defer span.End()

	query := fmt.Sprintf(`
		SELECT DISTINCT
			name,
			sample_type,
			sample_unit,
			period_type,
			period_unit,
			(duration > 0) AS delta
		FROM %s`, quotedTable(q.client))

	var args []interface{}
	if start.Unix() != 0 && end.Unix() != 0 {
		query += " WHERE time_nanos > ? AND time_nanos < ?"
		args = append(args, start.UnixNano(), end.UnixNano())
	}

	rows, err := q.client.DB().QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query profile types: %w", err)
	}
	defer rows.Close()

	var out []*pb.ProfileType
	for rows.Next() {
		t := &pb.ProfileType{}
		if err := rows.Scan(
			&t.Name, &t.SampleType, &t.SampleUnit,
			&t.PeriodType, &t.PeriodUnit, &t.Delta,
		); err != nil {
			return nil, fmt.Errorf("scan profile type: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// HasProfileData returns true if the table has any rows.
func (q *Querier) HasProfileData(ctx context.Context) (bool, error) {
	types, err := q.ProfileTypes(ctx, time.UnixMilli(0), time.UnixMilli(0))
	if err != nil {
		return false, err
	}
	return len(types) > 0, nil
}

// QueryRange returns time-bucketed metric series for the query.
func (q *Querier) QueryRange(
	ctx context.Context,
	queryStr string,
	startTime, endTime time.Time,
	step time.Duration,
	_ uint32,
	sumBy []string,
) ([]*pb.MetricsSeries, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/QueryRange")
	defer span.End()

	qp, err := profile.ParseQuery(queryStr)
	if err != nil {
		return nil, err
	}

	if step < time.Second {
		step = time.Second
	}

	profileFilter, profileArgs := ProfileTypeFilter(qp)
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	// Build optional sumBy projections. labels is a MAP so we extract
	// each requested label via element_at(labels, 'k')[1].
	var (
		innerSumBy   []string
		outerSumBy   []string
		groupBySumBy []string
	)
	for i, name := range sumBy {
		alias := fmt.Sprintf("label_%d", i)
		innerSumBy = append(innerSumBy,
			fmt.Sprintf("element_at(labels, '%s')[1] AS %s", escapeIdent(name), alias),
		)
		outerSumBy = append(outerSumBy, alias)
		groupBySumBy = append(groupBySumBy, alias)
	}

	innerSelectExtras := ""
	outerSelectExtras := ""
	if len(innerSumBy) > 0 {
		innerSelectExtras = ", " + strings.Join(innerSumBy, ", ")
		outerSelectExtras = strings.Join(outerSumBy, ", ") + ", "
	}

	innerQuery := fmt.Sprintf(`
		SELECT
			(time_nanos / ?)::BIGINT * ? AS bucket,
			SUM(value)::BIGINT AS total_sum,
			MIN(duration) AS duration_min%s
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?`,
		innerSelectExtras, quotedTable(q.client), profileFilter,
	)

	args := []interface{}{step.Nanoseconds(), step.Nanoseconds()}
	args = append(args, profileArgs...)
	args = append(args, startTime.UnixNano(), endTime.UnixNano())

	if labelFilter != "" {
		innerQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	groupByInner := "bucket"
	if len(groupBySumBy) > 0 {
		groupByInner = "bucket, " + strings.Join(groupBySumBy, ", ")
	}
	innerQuery += " GROUP BY " + groupByInner + " ORDER BY bucket"

	groupByOuter := "GROUP BY ALL"
	if len(outerSumBy) == 0 {
		groupByOuter = ""
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			%sLIST({bucket: bucket, value: total_sum, duration: duration_min} ORDER BY bucket) AS samples
		FROM (%s)
		%s`,
		outerSelectExtras, innerQuery, groupByOuter,
	)

	rows, err := q.client.DB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query range: %w", err)
	}
	defer rows.Close()

	type sampleRow struct {
		Bucket   int64 `db:"bucket"`
		Value    int64 `db:"value"`
		Duration int64 `db:"duration"`
	}

	var resSeries []*pb.MetricsSeries
	for rows.Next() {
		labelValues := make([]sql.NullString, len(sumBy))
		scanArgs := make([]interface{}, 0, len(sumBy)+1)
		for i := range sumBy {
			scanArgs = append(scanArgs, &labelValues[i])
		}
		var samples duckdb.Composite[[]sampleRow]
		scanArgs = append(scanArgs, &samples)

		if err := rows.Scan(scanArgs...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}

		pbLabels := make([]*profilestorepb.Label, len(sumBy))
		for i, n := range sumBy {
			val := ""
			if labelValues[i].Valid {
				val = labelValues[i].String
			}
			pbLabels[i] = &profilestorepb.Label{Name: n, Value: val}
		}

		entries := samples.Get()
		pbSamples := make([]*pb.MetricsSample, 0, len(entries))
		for _, e := range entries {
			vps := float64(e.Value)
			if e.Duration > 0 {
				vps = float64(e.Value) / (float64(e.Duration) / float64(time.Second.Nanoseconds()))
			}
			pbSamples = append(pbSamples, &pb.MetricsSample{
				Timestamp:      timestamppb.New(time.Unix(0, e.Bucket)),
				Value:          e.Value,
				ValuePerSecond: vps,
				Duration:       e.Duration,
			})
		}

		resSeries = append(resSeries, &pb.MetricsSeries{
			Labelset: &profilestorepb.LabelSet{Labels: pbLabels},
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
		return nil, fmt.Errorf("iterate rows: %w", err)
	}
	if len(resSeries) == 0 {
		return nil, status.Error(
			codes.NotFound,
			"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
		)
	}
	return resSeries, nil
}

// QuerySingle returns the symbolised profile at a single timestamp.
func (q *Querier) QuerySingle(
	ctx context.Context,
	queryStr string,
	t time.Time,
	invertCallStacks bool,
) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/QuerySingle")
	defer span.End()

	qp, err := profile.ParseQuery(queryStr)
	if err != nil {
		return profile.Profile{}, err
	}

	requestedTime := timestamp.FromTime(t)

	profileFilter, profileArgs := ProfileTypeFilter(qp)
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return profile.Profile{}, err
	}

	sqlQuery := fmt.Sprintf(`
		SELECT
			stacktrace,
			SUM(value)::BIGINT AS value,
			SUM(duration)::BIGINT AS sample_duration,
			period AS sample_period
		FROM %s
		WHERE %s
		  AND timestamp = ?`,
		quotedTable(q.client), profileFilter,
	)

	args := append([]interface{}{}, profileArgs...)
	args = append(args, requestedTime)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}
	sqlQuery += " GROUP BY stacktrace, period"

	records, err := q.runStacktraceQuery(ctx, sqlQuery, args, invertCallStacks)
	if err != nil {
		return profile.Profile{}, err
	}
	if len(records) == 0 {
		return profile.Profile{}, status.Error(codes.NotFound, "could not find profile at requested time and selectors")
	}

	qp.Meta.Timestamp = requestedTime
	return profile.Profile{Meta: qp.Meta, Samples: records}, nil
}

// QueryMerge returns the symbolised profile aggregated across the time range.
func (q *Querier) QueryMerge(
	ctx context.Context,
	queryStr string,
	start, end time.Time,
	aggregateByLabels []string,
	invertCallStacks bool,
	_ string,
) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/QueryMerge")
	defer span.End()

	qp, err := profile.ParseQuery(queryStr)
	if err != nil {
		return profile.Profile{}, err
	}

	startNanos := start.UnixNano()
	endNanos := end.UnixNano()

	profileFilter, profileArgs := ProfileTypeFilter(qp)
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return profile.Profile{}, err
	}

	groupByLabels := ""
	if len(aggregateByLabels) > 0 {
		labels := make([]string, len(aggregateByLabels))
		for i, l := range aggregateByLabels {
			name := strings.TrimPrefix(l, "labels.")
			labels[i] = fmt.Sprintf("element_at(labels, '%s')[1]", escapeIdent(name))
		}
		groupByLabels = ", " + strings.Join(labels, ", ")
	}

	queryDuration := endNanos - startNanos

	sqlQuery := fmt.Sprintf(`
		SELECT
			stacktrace,
			SUM(value)::BIGINT AS value,
			%d::BIGINT AS sample_duration,
			period AS sample_period
		FROM %s
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?`,
		queryDuration, quotedTable(q.client), profileFilter,
	)

	args := append([]interface{}{}, profileArgs...)
	args = append(args, startNanos, endNanos)

	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}

	sqlQuery += " GROUP BY stacktrace, period" + groupByLabels

	records, err := q.runStacktraceQuery(ctx, sqlQuery, args, invertCallStacks)
	if err != nil {
		return profile.Profile{}, err
	}

	qp.Meta.Timestamp = startNanos
	return profile.Profile{Meta: qp.Meta, Samples: records}, nil
}

// GetProfileMetadataMappings returns the distinct mapping files seen in
// the matched samples.
func (q *Querier) GetProfileMetadataMappings(
	ctx context.Context,
	queryStr string,
	start, end time.Time,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/GetProfileMetadataMappings")
	defer span.End()

	qp, err := profile.ParseQuery(queryStr)
	if err != nil {
		return nil, err
	}

	profileFilter, profileArgs := ProfileTypeFilter(qp)
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	sqlQuery := fmt.Sprintf(`
		SELECT DISTINCT loc.mapping_file AS f
		FROM %s, UNNEST(stacktrace) AS t(loc)
		WHERE %s
		  AND time_nanos >= ? AND time_nanos <= ?`,
		quotedTable(q.client), profileFilter,
	)

	args := append([]interface{}{}, profileArgs...)
	args = append(args, start.UnixNano(), end.UnixNano())
	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}
	sqlQuery += " ORDER BY f"

	rows, err := q.client.DB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query mapping files: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var s sql.NullString
		if err := rows.Scan(&s); err != nil {
			return nil, fmt.Errorf("scan mapping file: %w", err)
		}
		if s.Valid {
			out = append(out, s.String)
		}
	}
	return out, rows.Err()
}

// GetProfileMetadataLabels returns the distinct label names seen in the
// matched samples.
func (q *Querier) GetProfileMetadataLabels(
	ctx context.Context,
	queryStr string,
	start, end time.Time,
) ([]string, error) {
	ctx, span := q.tracer.Start(ctx, "DuckDB/GetProfileMetadataLabels")
	defer span.End()

	qp, err := profile.ParseQuery(queryStr)
	if err != nil {
		return nil, err
	}

	profileFilter, profileArgs := ProfileTypeFilter(qp)
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return nil, err
	}

	sqlQuery := fmt.Sprintf(`
		SELECT DISTINCT k FROM (
			SELECT UNNEST(map_keys(labels)) AS k
			FROM %s
			WHERE %s
			  AND time_nanos >= ? AND time_nanos <= ?`,
		quotedTable(q.client), profileFilter,
	)

	args := append([]interface{}{}, profileArgs...)
	args = append(args, start.UnixNano(), end.UnixNano())
	if labelFilter != "" {
		sqlQuery += " AND " + labelFilter
		args = append(args, labelArgs...)
	}
	sqlQuery += ") ORDER BY k"

	rows, err := q.client.DB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("query labels: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan label name: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// runStacktraceQuery executes a query whose rows have the shape
// (stacktrace LIST<STRUCT>, value BIGINT, sample_duration BIGINT,
// sample_period BIGINT). It symbolises any unsymbolised locations and
// emits one Arrow record describing the whole result set.
func (q *Querier) runStacktraceQuery(
	ctx context.Context,
	sqlQuery string,
	args []interface{},
	invertCallStacks bool,
) ([]arrow.RecordBatch, error) {
	rows, err := q.client.DB().QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("execute stacktrace query: %w", err)
	}
	defer rows.Close()

	type row struct {
		stack    []stacktraceLoc
		value    int64
		duration int64
		period   int64
	}

	var samples []row
	// buildID -> address -> Location, used to deduplicate symbolisation
	// requests across all rows in this result.
	locationIndex := make(map[string]map[uint64]*profile.Location)

	for rows.Next() {
		var stack duckdb.Composite[[]stacktraceLoc]
		var r row
		if err := rows.Scan(&stack, &r.value, &r.duration, &r.period); err != nil {
			return nil, fmt.Errorf("scan stacktrace row: %w", err)
		}
		r.stack = stack.Get()

		for _, loc := range r.stack {
			needsSym := loc.FunctionName == "" && loc.MappingBuildID != "" && loc.Address != 0
			if !needsSym {
				continue
			}
			if _, ok := locationIndex[loc.MappingBuildID]; !ok {
				locationIndex[loc.MappingBuildID] = make(map[uint64]*profile.Location)
			}
			if _, ok := locationIndex[loc.MappingBuildID][loc.Address]; ok {
				continue
			}
			locationIndex[loc.MappingBuildID][loc.Address] = &profile.Location{
				Address: loc.Address,
				Mapping: &metapb.Mapping{
					BuildId: loc.MappingBuildID,
					File:    loc.MappingFile,
					Start:   loc.MappingStart,
					Limit:   loc.MappingLimit,
					Offset:  loc.MappingOffset,
				},
			}
		}

		samples = append(samples, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate stacktrace rows: %w", err)
	}

	for buildID, addrMap := range locationIndex {
		locs := make([]*profile.Location, 0, len(addrMap))
		for _, l := range addrMap {
			locs = append(locs, l)
		}
		req := symbolizer.SymbolizationRequest{
			BuildID: buildID,
			Mappings: []symbolizer.SymbolizationRequestMappingAddrs{
				{Locations: locs},
			},
		}
		if err := q.symbolizer.Symbolize(ctx, req); err != nil {
			level.Error(q.logger).Log("msg", "failed to symbolize locations", "buildID", buildID, "err", err)
			// continue with raw addresses if symbolisation fails
		}
	}

	if len(samples) == 0 {
		return nil, nil
	}

	w := profile.NewWriter(q.mem, []string{})
	defer w.Release()

	for _, s := range samples {
		w.LocationsList.Append(true)

		n := len(s.stack)
		for i := 0; i < n; i++ {
			idx := i
			if invertCallStacks {
				idx = n - 1 - i
			}
			loc := s.stack[idx]

			w.Locations.Append(true)
			w.Addresses.Append(loc.Address)
			w.MappingStart.Append(loc.MappingStart)
			w.MappingLimit.Append(loc.MappingLimit)
			w.MappingOffset.Append(loc.MappingOffset)

			if err := w.MappingFile.Append([]byte(loc.MappingFile)); err != nil {
				level.Error(q.logger).Log("msg", "append mapping file", "err", err)
			}
			if err := w.MappingBuildID.Append([]byte(loc.MappingBuildID)); err != nil {
				level.Error(q.logger).Log("msg", "append mapping build id", "err", err)
			}

			// Prefer freshly symbolised function info over the row's stored
			// data when the symbolizer returned something.
			var sym *profile.Location
			if m, ok := locationIndex[loc.MappingBuildID]; ok {
				sym = m[loc.Address]
			}

			switch {
			case sym != nil && len(sym.Lines) > 0:
				w.Lines.Append(true)
				for _, line := range sym.Lines {
					w.Line.Append(true)
					w.LineNumber.Append(line.Line)
					if line.Function != nil {
						_ = w.FunctionName.Append([]byte(line.Function.Name))
						_ = w.FunctionSystemName.Append([]byte(line.Function.SystemName))
						_ = w.FunctionFilename.Append([]byte(line.Function.Filename))
						w.FunctionStartLine.Append(line.Function.StartLine)
					} else {
						w.FunctionName.AppendNull()
						w.FunctionSystemName.AppendNull()
						w.FunctionFilename.AppendNull()
						w.FunctionStartLine.AppendNull()
					}
				}
			case loc.FunctionName != "":
				w.Lines.Append(true)
				w.Line.Append(true)
				w.LineNumber.Append(loc.LineNumber)
				_ = w.FunctionName.Append([]byte(loc.FunctionName))
				_ = w.FunctionSystemName.Append([]byte(loc.FunctionSystemName))
				_ = w.FunctionFilename.Append([]byte(loc.FunctionFilename))
				w.FunctionStartLine.Append(loc.FunctionStartLine)
			default:
				w.Lines.AppendNull()
			}
		}

		w.Value.Append(s.value)
		w.Diff.Append(0)
		w.TimeNanos.Append(0)
		w.Period.Append(s.period)
	}

	return []arrow.RecordBatch{w.RecordBuilder.NewRecordBatch()}, nil
}
