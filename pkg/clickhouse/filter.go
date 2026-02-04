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
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/model/labels"

	"github.com/parca-dev/parca/pkg/profile"
)

// ProfileTypeFilter generates SQL WHERE clause conditions for profile type filtering.
func ProfileTypeFilter(qp profile.QueryParts) (string, []interface{}) {
	conditions := []string{
		"name = ?",
		"sample_type = ?",
		"sample_unit = ?",
		"period_type = ?",
		"period_unit = ?",
	}
	args := []interface{}{
		qp.Meta.Name,
		qp.Meta.SampleType.Type,
		qp.Meta.SampleType.Unit,
		qp.Meta.PeriodType.Type,
		qp.Meta.PeriodType.Unit,
	}

	if qp.Delta {
		conditions = append(conditions, "duration != 0")
	} else {
		conditions = append(conditions, "duration = 0")
	}

	return strings.Join(conditions, " AND "), args
}

// LabelMatchersToSQL converts Prometheus label matchers to SQL WHERE clause conditions.
func LabelMatchersToSQL(matchers []*labels.Matcher) (string, []interface{}, error) {
	if len(matchers) == 0 {
		return "", nil, nil
	}

	conditions := make([]string, 0, len(matchers))
	args := make([]interface{}, 0, len(matchers))

	for _, m := range matchers {
		condition, arg, err := matcherToSQL(m)
		if err != nil {
			return "", nil, err
		}
		conditions = append(conditions, condition)
		if arg != nil {
			args = append(args, arg)
		}
	}

	return strings.Join(conditions, " AND "), args, nil
}

// matcherToSQL converts a single Prometheus label matcher to a SQL condition.
func matcherToSQL(m *labels.Matcher) (string, interface{}, error) {
	// Use ClickHouse JSON path syntax for label access
	labelPath := fmt.Sprintf("labels.%s", m.Name)

	switch m.Type {
	case labels.MatchEqual:
		if m.Value == "" {
			// Empty value means label should not exist or be null
			return fmt.Sprintf("(%s IS NULL OR %s = '')", labelPath, labelPath), nil, nil
		}
		return fmt.Sprintf("%s = ?", labelPath), m.Value, nil

	case labels.MatchNotEqual:
		if m.Value == "" {
			// Not empty means label should exist and not be null/empty
			return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", labelPath, labelPath), nil, nil
		}
		return fmt.Sprintf("(%s != ? OR %s IS NULL)", labelPath, labelPath), m.Value, nil

	case labels.MatchRegexp:
		// ClickHouse uses match() for regex
		return fmt.Sprintf("match(toString(%s), ?)", labelPath), m.Value, nil

	case labels.MatchNotRegexp:
		return fmt.Sprintf("NOT match(toString(%s), ?)", labelPath), m.Value, nil

	default:
		return "", nil, fmt.Errorf("unsupported matcher type: %v", m.Type)
	}
}

// TimeRangeFilter generates SQL WHERE clause conditions for time range filtering.
func TimeRangeFilter(startNanos, endNanos int64) (string, []interface{}) {
	return "time_nanos >= ? AND time_nanos <= ?", []interface{}{startNanos, endNanos}
}

// BuildWhereClause combines multiple filter conditions into a single WHERE clause.
func BuildWhereClause(conditions []string, allArgs []interface{}) (string, []interface{}) {
	nonEmpty := make([]string, 0, len(conditions))
	for _, c := range conditions {
		if c != "" {
			nonEmpty = append(nonEmpty, c)
		}
	}

	if len(nonEmpty) == 0 {
		return "", nil
	}

	return "WHERE " + strings.Join(nonEmpty, " AND "), allArgs
}

// QueryToFilters converts a query string and time range to SQL filter components.
func QueryToFilters(query string, startNanos, endNanos int64) (string, []interface{}, profile.QueryParts, error) {
	qp, err := profile.ParseQuery(query)
	if err != nil {
		return "", nil, qp, err
	}

	// Profile type filter
	profileFilter, profileArgs := ProfileTypeFilter(qp)

	// Label matchers filter
	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return "", nil, qp, err
	}

	// Time range filter
	timeFilter, timeArgs := TimeRangeFilter(startNanos, endNanos)

	// Combine all conditions
	conditions := []string{profileFilter}
	args := append([]interface{}{}, profileArgs...)

	if labelFilter != "" {
		conditions = append(conditions, labelFilter)
		args = append(args, labelArgs...)
	}

	if startNanos != 0 || endNanos != 0 {
		conditions = append(conditions, timeFilter)
		args = append(args, timeArgs...)
	}

	whereClause, _ := BuildWhereClause(conditions, args)

	return whereClause, args, qp, nil
}
