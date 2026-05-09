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
	"fmt"
	"strings"

	"github.com/prometheus/prometheus/model/labels"

	"github.com/parca-dev/parca/pkg/profile"
)

// ProfileTypeFilter returns SQL WHERE-fragment conditions for the profile
// type identifier (name + sample_type + sample_unit + period_type +
// period_unit + delta-by-duration), plus the matching argument list.
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

// LabelMatchersToSQL translates Prometheus label matchers to DuckDB SQL.
//
// We model labels as MAP(VARCHAR, VARCHAR). Reads use:
//
//	element_at(labels, 'foo')[1]
//
// which returns NULL when the key is absent. (`labels['foo']` works on
// recent DuckDB versions but `element_at` is the canonical, version-stable
// form.) Regex uses DuckDB's `regexp_matches`.
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

func matcherToSQL(m *labels.Matcher) (string, interface{}, error) {
	// element_at(labels, 'k')[1] yields NULL when the key is missing,
	// which is what we want for "label absent" semantics.
	labelExpr := fmt.Sprintf("element_at(labels, '%s')[1]", escapeIdent(m.Name))

	switch m.Type {
	case labels.MatchEqual:
		if m.Value == "" {
			return fmt.Sprintf("(%s IS NULL OR %s = '')", labelExpr, labelExpr), nil, nil
		}
		return fmt.Sprintf("%s = ?", labelExpr), m.Value, nil

	case labels.MatchNotEqual:
		if m.Value == "" {
			return fmt.Sprintf("(%s IS NOT NULL AND %s != '')", labelExpr, labelExpr), nil, nil
		}
		return fmt.Sprintf("(%s != ? OR %s IS NULL)", labelExpr, labelExpr), m.Value, nil

	case labels.MatchRegexp:
		return fmt.Sprintf("regexp_matches(coalesce(%s, ''), ?)", labelExpr), m.Value, nil

	case labels.MatchNotRegexp:
		return fmt.Sprintf("NOT regexp_matches(coalesce(%s, ''), ?)", labelExpr), m.Value, nil

	default:
		return "", nil, fmt.Errorf("unsupported matcher type: %v", m.Type)
	}
}

// TimeRangeFilter returns the WHERE-fragment for a time-range filter on
// time_nanos.
func TimeRangeFilter(startNanos, endNanos int64) (string, []interface{}) {
	return "time_nanos >= ? AND time_nanos <= ?", []interface{}{startNanos, endNanos}
}

// BuildWhereClause stitches non-empty fragments into a single WHERE clause.
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

// QueryToFilters parses a parca query string and returns the WHERE clause
// + bind args + parsed query parts for the given time range.
func QueryToFilters(query string, startNanos, endNanos int64) (string, []interface{}, profile.QueryParts, error) {
	qp, err := profile.ParseQuery(query)
	if err != nil {
		return "", nil, qp, err
	}

	profileFilter, profileArgs := ProfileTypeFilter(qp)

	labelFilter, labelArgs, err := LabelMatchersToSQL(qp.Matchers)
	if err != nil {
		return "", nil, qp, err
	}

	timeFilter, timeArgs := TimeRangeFilter(startNanos, endNanos)

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

// escapeIdent escapes a single quote inside a label name embedded as an
// SQL string literal. We can't use a bind parameter here because DuckDB
// (like most engines) doesn't allow parameterising the key argument of
// element_at() across all paths.
func escapeIdent(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}
