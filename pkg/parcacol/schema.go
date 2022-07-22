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
	"github.com/polarsignals/frostdb/dynparquet"
	schemapb "github.com/polarsignals/frostdb/gen/proto/go/frostdb/schema/v1alpha1"
)

const (
	SchemaName = "parca"
	// The columns are sorted by their name in the schema too.
	ColumnDuration       = "duration"
	ColumnLabels         = "labels"
	ColumnName           = "name"
	ColumnPeriod         = "period"
	ColumnPeriodType     = "period_type"
	ColumnPeriodUnit     = "period_unit"
	ColumnPprofLabels    = "pprof_labels"
	ColumnPprofNumLabels = "pprof_num_labels"
	ColumnSampleType     = "sample_type"
	ColumnSampleUnit     = "sample_unit"
	ColumnStacktrace     = "stacktrace"
	ColumnTimestamp      = "timestamp"
	ColumnValue          = "value"
)

func Schema() (*dynparquet.Schema, error) {
	return dynparquet.SchemaFromDefinition(&schemapb.Schema{
		Name: SchemaName,
		Columns: []*schemapb.Column{
			{
				Name: ColumnDuration,
				StorageLayout: &schemapb.StorageLayout{
					Type: schemapb.StorageLayout_TYPE_INT64,
				},
				Dynamic: false,
			}, {
				Name: ColumnLabels,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
					Nullable: true,
				},
				Dynamic: true,
			}, {
				Name: ColumnName,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnPeriod,
				StorageLayout: &schemapb.StorageLayout{
					Type: schemapb.StorageLayout_TYPE_INT64,
				},
				Dynamic: false,
			}, {
				Name: ColumnPeriodType,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnPeriodUnit,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnPprofLabels,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
					Nullable: true,
				},
				Dynamic: true,
			}, {
				Name: ColumnPprofNumLabels,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_INT64,
					Nullable: true,
				},
				Dynamic: true,
			}, {
				Name: ColumnSampleType,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnSampleUnit,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnStacktrace,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_STRING,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
				},
				Dynamic: false,
			}, {
				Name: ColumnTimestamp,
				StorageLayout: &schemapb.StorageLayout{
					Type: schemapb.StorageLayout_TYPE_INT64,
				},
				Dynamic: false,
			}, {
				Name: ColumnValue,
				StorageLayout: &schemapb.StorageLayout{
					Type: schemapb.StorageLayout_TYPE_INT64,
				},
				Dynamic: false,
			},
		},
		SortingColumns: []*schemapb.SortingColumn{
			{
				Name:      ColumnName,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:      ColumnSampleType,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:      ColumnSampleUnit,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:      ColumnPeriodType,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:      ColumnPeriodUnit,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:       ColumnLabels,
				Direction:  schemapb.SortingColumn_DIRECTION_ASCENDING,
				NullsFirst: true,
			}, {
				Name:       ColumnStacktrace,
				Direction:  schemapb.SortingColumn_DIRECTION_ASCENDING,
				NullsFirst: true,
			}, {
				Name:      ColumnTimestamp,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:       ColumnPprofLabels,
				Direction:  schemapb.SortingColumn_DIRECTION_ASCENDING,
				NullsFirst: true,
			}, {
				Name:       ColumnPprofNumLabels,
				Direction:  schemapb.SortingColumn_DIRECTION_ASCENDING,
				NullsFirst: true,
			},
		},
	})
}
