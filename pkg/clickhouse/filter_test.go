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
	"testing"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantName    string
		wantDelta   bool
		wantErr     bool
		wantMatches int
	}{
		{
			name:        "simple profile type",
			query:       "process_cpu:samples:count:cpu:nanoseconds{}",
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 0,
		},
		{
			name:        "delta profile type",
			query:       "process_cpu:samples:count:cpu:nanoseconds:delta{}",
			wantName:    "process_cpu",
			wantDelta:   true,
			wantErr:     false,
			wantMatches: 0,
		},
		{
			name:        "profile type with label matcher",
			query:       `process_cpu:samples:count:cpu:nanoseconds{node="test"}`,
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 1,
		},
		{
			name:        "profile type with multiple label matchers",
			query:       `process_cpu:samples:count:cpu:nanoseconds{node="test",container="app"}`,
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 2,
		},
		{
			name:    "invalid query",
			query:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qp, err := ParseQuery(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, qp.Meta.Name)
			require.Equal(t, tt.wantDelta, qp.Delta)
			require.Len(t, qp.Matchers, tt.wantMatches)
		})
	}
}

func TestProfileTypeFilter(t *testing.T) {
	qp, err := ParseQuery("process_cpu:samples:count:cpu:nanoseconds{}")
	require.NoError(t, err)

	filter, args := ProfileTypeFilter(qp)

	require.Contains(t, filter, "name = ?")
	require.Contains(t, filter, "sample_type = ?")
	require.Contains(t, filter, "sample_unit = ?")
	require.Contains(t, filter, "period_type = ?")
	require.Contains(t, filter, "period_unit = ?")
	require.Contains(t, filter, "duration = 0")
	require.Len(t, args, 5)
	require.Equal(t, "process_cpu", args[0])
}

func TestProfileTypeFilterDelta(t *testing.T) {
	qp, err := ParseQuery("process_cpu:samples:count:cpu:nanoseconds:delta{}")
	require.NoError(t, err)

	filter, _ := ProfileTypeFilter(qp)

	require.Contains(t, filter, "duration != 0")
}

func TestLabelMatchersToSQL(t *testing.T) {
	tests := []struct {
		name       string
		matchers   []*labels.Matcher
		wantSQL    string
		wantArgs   []interface{}
		wantErr    bool
	}{
		{
			name:     "empty matchers",
			matchers: []*labels.Matcher{},
			wantSQL:  "",
			wantArgs: nil,
			wantErr:  false,
		},
		{
			name: "equal matcher",
			matchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "node", "test"),
			},
			wantSQL:  "labels.node = ?",
			wantArgs: []interface{}{"test"},
			wantErr:  false,
		},
		{
			name: "not equal matcher",
			matchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchNotEqual, "node", "test"),
			},
			wantSQL:  "(labels.node != ? OR labels.node IS NULL)",
			wantArgs: []interface{}{"test"},
			wantErr:  false,
		},
		{
			name: "regex matcher",
			matchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchRegexp, "node", "test.*"),
			},
			wantSQL:  "match(toString(labels.node), ?)",
			wantArgs: []interface{}{"test.*"},
			wantErr:  false,
		},
		{
			name: "not regex matcher",
			matchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchNotRegexp, "node", "test.*"),
			},
			wantSQL:  "NOT match(toString(labels.node), ?)",
			wantArgs: []interface{}{"test.*"},
			wantErr:  false,
		},
		{
			name: "empty value equal",
			matchers: []*labels.Matcher{
				labels.MustNewMatcher(labels.MatchEqual, "node", ""),
			},
			wantSQL:  "(labels.node IS NULL OR labels.node = '')",
			wantArgs: []interface{}{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sql, args, err := LabelMatchersToSQL(tt.matchers)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantSQL, sql)
			require.Equal(t, tt.wantArgs, args)
		})
	}
}

func TestTimeRangeFilter(t *testing.T) {
	filter, args := TimeRangeFilter(1000000000, 2000000000)

	require.Equal(t, "time_nanos >= ? AND time_nanos <= ?", filter)
	require.Len(t, args, 2)
	require.Equal(t, int64(1000000000), args[0])
	require.Equal(t, int64(2000000000), args[1])
}

func TestBuildWhereClause(t *testing.T) {
	tests := []struct {
		name       string
		conditions []string
		args       []interface{}
		wantClause string
	}{
		{
			name:       "empty conditions",
			conditions: []string{},
			args:       nil,
			wantClause: "",
		},
		{
			name:       "single condition",
			conditions: []string{"name = ?"},
			args:       []interface{}{"test"},
			wantClause: "WHERE name = ?",
		},
		{
			name:       "multiple conditions",
			conditions: []string{"name = ?", "value > ?"},
			args:       []interface{}{"test", 10},
			wantClause: "WHERE name = ? AND value > ?",
		},
		{
			name:       "with empty condition",
			conditions: []string{"name = ?", "", "value > ?"},
			args:       []interface{}{"test", 10},
			wantClause: "WHERE name = ? AND value > ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clause, _ := BuildWhereClause(tt.conditions, tt.args)
			require.Equal(t, tt.wantClause, clause)
		})
	}
}
