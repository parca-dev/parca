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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateTableSQL(t *testing.T) {
	sql := CreateTableSQL("testdb", "testtable")

	// Verify the SQL contains expected components
	require.Contains(t, sql, "CREATE TABLE IF NOT EXISTS testdb.testtable")
	require.Contains(t, sql, "name String")
	require.Contains(t, sql, "sample_type String")
	require.Contains(t, sql, "sample_unit String")
	require.Contains(t, sql, "period_type String")
	require.Contains(t, sql, "period_unit String")
	require.Contains(t, sql, "period Int64")
	require.Contains(t, sql, "duration Int64")
	require.Contains(t, sql, "timestamp Int64")
	require.Contains(t, sql, "time_nanos Int64")
	require.Contains(t, sql, "value Int64")
	require.Contains(t, sql, "labels JSON")
	require.Contains(t, sql, "stacktrace Nested")
	require.Contains(t, sql, "ENGINE = MergeTree()")
	require.Contains(t, sql, "ORDER BY")
	require.Contains(t, sql, "PARTITION BY")
}

func TestInsertSQL(t *testing.T) {
	sql := InsertSQL("testdb", "testtable")

	// Verify the SQL contains expected components
	require.Contains(t, sql, "INSERT INTO testdb.testtable")
	require.Contains(t, sql, "name")
	require.Contains(t, sql, "sample_type")
	require.Contains(t, sql, "sample_unit")
	require.Contains(t, sql, "period_type")
	require.Contains(t, sql, "period_unit")
	require.Contains(t, sql, "period")
	require.Contains(t, sql, "duration")
	require.Contains(t, sql, "timestamp")
	require.Contains(t, sql, "time_nanos")
	require.Contains(t, sql, "value")
	require.Contains(t, sql, "labels")
	require.Contains(t, sql, "stacktrace.address")
	require.Contains(t, sql, "stacktrace.mapping_file")
	require.Contains(t, sql, "stacktrace.function_name")
}

func TestSchemaConstants(t *testing.T) {
	// Verify column constants match expected values
	require.Equal(t, "name", ColName)
	require.Equal(t, "sample_type", ColSampleType)
	require.Equal(t, "sample_unit", ColSampleUnit)
	require.Equal(t, "period_type", ColPeriodType)
	require.Equal(t, "period_unit", ColPeriodUnit)
	require.Equal(t, "period", ColPeriod)
	require.Equal(t, "duration", ColDuration)
	require.Equal(t, "timestamp", ColTimestamp)
	require.Equal(t, "time_nanos", ColTimeNanos)
	require.Equal(t, "value", ColValue)
	require.Equal(t, "labels", ColLabels)

	// Verify stacktrace column constants
	require.True(t, strings.HasPrefix(ColStacktraceAddress, "stacktrace."))
	require.True(t, strings.HasPrefix(ColStacktraceMappingFile, "stacktrace."))
	require.True(t, strings.HasPrefix(ColStacktraceFunctionName, "stacktrace."))
}
