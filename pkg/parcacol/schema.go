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
	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/segmentio/parquet-go"
)

const (
	schemaName = "parca"
	// The columns are sorted by their name in the schema too.
	columnDuration       = "duration"
	columnLabels         = "labels"
	columnPeriod         = "period"
	columnPeriodType     = "period_type"
	columnPeriodUnit     = "period_unit"
	columnPprofLabels    = "pprof_labels"
	columnPprofNumLabels = "pprof_num_labels"
	columnSampleType     = "sample_type"
	columnSampleUnit     = "sample_unit"
	columnStacktrace     = "stacktrace"
	columnTimestamp      = "timestamp"
	columnValue          = "value"
)

func Schema() *dynparquet.Schema {
	return dynparquet.NewSchema(
		schemaName,
		[]dynparquet.ColumnDefinition{
			{
				Name:          columnDuration,
				StorageLayout: parquet.Int(64),
				Dynamic:       false,
			}, {
				Name:          columnLabels,
				StorageLayout: parquet.Encoded(parquet.Optional(parquet.String()), &parquet.RLEDictionary),
				Dynamic:       true,
			}, {
				Name:          columnPeriod,
				StorageLayout: parquet.Int(64),
				Dynamic:       false,
			}, {
				Name:          columnPeriodType,
				StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
				Dynamic:       false,
			}, {
				Name:          columnPeriodUnit,
				StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
				Dynamic:       false,
			}, {
				Name:          columnPprofLabels,
				StorageLayout: parquet.Encoded(parquet.Optional(parquet.String()), &parquet.RLEDictionary),
				Dynamic:       true,
			}, {
				Name:          columnPprofNumLabels,
				StorageLayout: parquet.Optional(parquet.Int(64)),
				Dynamic:       true,
			}, {
				Name:          columnSampleType,
				StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
				Dynamic:       false,
			}, {
				Name:          columnSampleUnit,
				StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
				Dynamic:       false,
			}, {
				Name:          columnStacktrace,
				StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
				Dynamic:       false,
			}, {
				Name:          columnTimestamp,
				StorageLayout: parquet.Int(64),
				Dynamic:       false,
			}, {
				Name:          columnValue,
				StorageLayout: parquet.Int(64),
				Dynamic:       false,
			},
		},
		[]dynparquet.SortingColumn{
			dynparquet.Ascending(columnSampleType),
			dynparquet.Ascending(columnSampleUnit),
			dynparquet.Ascending(columnPeriodType),
			dynparquet.Ascending(columnPeriodUnit),
			dynparquet.NullsFirst(dynparquet.Ascending(columnLabels)),
			dynparquet.NullsFirst(dynparquet.Ascending(columnStacktrace)),
			dynparquet.Ascending(columnTimestamp),
			dynparquet.NullsFirst(dynparquet.Ascending(columnPprofLabels)),
			dynparquet.NullsFirst(dynparquet.Ascending(columnPprofNumLabels)),
		},
	)
}
