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

import "fmt"

// CreateTableSQL returns the SQL statement to create the profile data table.
// The schema uses:
// - JSON type for dynamic labels (native ClickHouse support for dynamic columns)
// - Nested type for stacktrace data with explicit columns for each location field
func CreateTableSQL(database, table string) string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s.%s (
    -- Profile metadata
    name String,
    sample_type String,
    sample_unit String,
    period_type String,
    period_unit String,
    period Int64,
    duration Int64,

    -- Timestamps
    timestamp Int64,
    time_nanos Int64,

    -- Sample value
    value Int64,

    -- Dynamic labels using JSON type
    labels JSON,

    -- Stacktrace as Nested type with explicit location columns
    stacktrace Nested(
        address UInt64,
        mapping_start UInt64,
        mapping_limit UInt64,
        mapping_offset UInt64,
        mapping_file String,
        mapping_build_id String,
        line_number Int64,
        function_name String,
        function_system_name String,
        function_filename String,
        function_start_line Int64
    )
) ENGINE = MergeTree()
ORDER BY (name, sample_type, sample_unit, period_type, period_unit, timestamp, time_nanos)
PARTITION BY toYYYYMMDD(fromUnixTimestamp64Nano(time_nanos))
`, database, table)
}

// Column names matching the profile schema.
const (
	ColName       = "name"
	ColSampleType = "sample_type"
	ColSampleUnit = "sample_unit"
	ColPeriodType = "period_type"
	ColPeriodUnit = "period_unit"
	ColPeriod     = "period"
	ColDuration   = "duration"
	ColTimestamp  = "timestamp"
	ColTimeNanos  = "time_nanos"
	ColValue      = "value"
	ColLabels     = "labels"

	// Stacktrace nested columns
	ColStacktraceAddress            = "stacktrace.address"
	ColStacktraceMappingStart       = "stacktrace.mapping_start"
	ColStacktraceMappingLimit       = "stacktrace.mapping_limit"
	ColStacktraceMappingOffset      = "stacktrace.mapping_offset"
	ColStacktraceMappingFile        = "stacktrace.mapping_file"
	ColStacktraceMappingBuildID     = "stacktrace.mapping_build_id"
	ColStacktraceLineNumber         = "stacktrace.line_number"
	ColStacktraceFunctionName       = "stacktrace.function_name"
	ColStacktraceFunctionSystemName = "stacktrace.function_system_name"
	ColStacktraceFunctionFilename   = "stacktrace.function_filename"
	ColStacktraceFunctionStartLine  = "stacktrace.function_start_line"
)

// InsertSQL returns the SQL statement for inserting data into the table.
func InsertSQL(database, table string) string {
	return fmt.Sprintf(`INSERT INTO %s.%s (
		name,
		sample_type,
		sample_unit,
		period_type,
		period_unit,
		period,
		duration,
		timestamp,
		time_nanos,
		value,
		labels,
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
		stacktrace.function_start_line
	)`, database, table)
}
