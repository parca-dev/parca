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

import "fmt"

// Column names matching the parca write schema.
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
	ColStacktrace = "stacktrace"
)

// Stacktrace STRUCT field names.
const (
	StFieldAddress            = "address"
	StFieldMappingStart       = "mapping_start"
	StFieldMappingLimit       = "mapping_limit"
	StFieldMappingOffset      = "mapping_offset"
	StFieldMappingFile        = "mapping_file"
	StFieldMappingBuildID     = "mapping_build_id"
	StFieldLineNumber         = "line_number"
	StFieldFunctionName       = "function_name"
	StFieldFunctionSystemName = "function_system_name"
	StFieldFunctionFilename   = "function_filename"
	StFieldFunctionStartLine  = "function_start_line"
)

// CreateTableSQL returns the DDL for the profile data table.
//
// Schema choices vs ClickHouse:
//   - labels: MAP(VARCHAR, VARCHAR). DuckDB-native, queryable via labels[key]
//     and map_keys/map_values. Avoids the JSON extension dependency.
//   - stacktrace: LIST(STRUCT(...)). Each location is a struct in a list,
//     unlike ClickHouse's parallel-array Nested layout. UNNEST flattens it
//     for queries that iterate locations.
//   - DuckDB doesn't have ClickHouse's MergeTree partitioning. We rely on
//     min/max statistics on time_nanos for time-range pruning instead.
func CreateTableSQL(table string) string {
	return fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %[1]s (
    name                 VARCHAR NOT NULL,
    sample_type          VARCHAR NOT NULL,
    sample_unit          VARCHAR NOT NULL,
    period_type          VARCHAR NOT NULL,
    period_unit          VARCHAR NOT NULL,
    period               BIGINT  NOT NULL,
    duration             BIGINT  NOT NULL,
    timestamp            BIGINT  NOT NULL,
    time_nanos           BIGINT  NOT NULL,
    value                BIGINT  NOT NULL,
    labels               MAP(VARCHAR, VARCHAR),
    stacktrace           STRUCT(
                             address              UBIGINT,
                             mapping_start        UBIGINT,
                             mapping_limit        UBIGINT,
                             mapping_offset       UBIGINT,
                             mapping_file         VARCHAR,
                             mapping_build_id     VARCHAR,
                             line_number          BIGINT,
                             function_name        VARCHAR,
                             function_system_name VARCHAR,
                             function_filename    VARCHAR,
                             function_start_line  BIGINT
                         )[]
);
`, quoteIdent(table))
}

// quoteIdent wraps an identifier in double quotes per SQL standard.
// We don't need full escaping because table names come from operator config.
func quoteIdent(s string) string {
	return `"` + s + `"`
}
