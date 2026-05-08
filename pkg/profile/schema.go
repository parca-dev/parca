// Copyright 2022-2026 The Parca Authors
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

package profile

import (
	"fmt"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/polarsignals/frostdb/dynparquet"
	schemapb "github.com/polarsignals/frostdb/gen/proto/go/frostdb/schema/v1alpha1"
)

const (
	SchemaName = "parca"
	// The columns are sorted by their name in the schema too.
	ColumnDuration   = "duration"
	ColumnLabels     = "labels"
	ColumnName       = "name"
	ColumnPeriod     = "period"
	ColumnPeriodType = "period_type"
	ColumnPeriodUnit = "period_unit"
	ColumnSampleType = "sample_type"
	ColumnSampleUnit = "sample_unit"
	ColumnStacktrace = "stacktrace"
	ColumnTimestamp  = "timestamp"
	ColumnTimeNanos  = "time_nanos"
	ColumnValue      = "value"
)

func SchemaDefinition() *schemapb.Schema {
	return &schemapb.Schema{
		Name: SchemaName,
		Columns: []*schemapb.Column{
			{
				Name: ColumnDuration,
				StorageLayout: &schemapb.StorageLayout{
					Type:     schemapb.StorageLayout_TYPE_INT64,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
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
					Type:     schemapb.StorageLayout_TYPE_INT64,
					Encoding: schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
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
					Type:        schemapb.StorageLayout_TYPE_STRING,
					Encoding:    schemapb.StorageLayout_ENCODING_RLE_DICTIONARY,
					Compression: schemapb.StorageLayout_COMPRESSION_LZ4_RAW,
					Repeated:    true,
					Nullable:    true,
				},
				Dynamic: false,
			}, {
				Name: ColumnTimestamp,
				StorageLayout: &schemapb.StorageLayout{
					Type:        schemapb.StorageLayout_TYPE_INT64,
					Encoding:    schemapb.StorageLayout_ENCODING_DELTA_BINARY_PACKED,
					Compression: schemapb.StorageLayout_COMPRESSION_LZ4_RAW,
				},
				Dynamic: false,
			}, {
				Name: ColumnTimeNanos,
				StorageLayout: &schemapb.StorageLayout{
					Type:        schemapb.StorageLayout_TYPE_INT64,
					Encoding:    schemapb.StorageLayout_ENCODING_DELTA_BINARY_PACKED,
					Compression: schemapb.StorageLayout_COMPRESSION_LZ4_RAW,
				},
				Dynamic: false,
			}, {
				Name: ColumnValue,
				StorageLayout: &schemapb.StorageLayout{
					Type:        schemapb.StorageLayout_TYPE_INT64,
					Encoding:    schemapb.StorageLayout_ENCODING_DELTA_BINARY_PACKED,
					Compression: schemapb.StorageLayout_COMPRESSION_LZ4_RAW,
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
				Name:      ColumnTimestamp,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			}, {
				Name:      ColumnTimeNanos,
				Direction: schemapb.SortingColumn_DIRECTION_ASCENDING,
			},
		},
	}
}

func Schema() (*dynparquet.Schema, error) {
	return dynparquet.SchemaFromDefinition(SchemaDefinition())
}

// BuildArrowSchema returns the Arrow schema for the parca write/ingest
// profile data, expanding the dynamic ColumnLabels column into one
// "labels.<name>" field per labelName. The column order matches the proto
// definition order, with dynamic labels emitted in place of the labels
// column. Static columns map to:
//   - TYPE_INT64 → arrow.PrimitiveTypes.Int64
//   - TYPE_STRING (non-repeated) → dictionary-encoded binary
//   - TYPE_STRING (repeated) → list of dictionary-encoded binary
func BuildArrowSchema(labelNames []string) *arrow.Schema {
	def := SchemaDefinition()
	fields := make([]arrow.Field, 0, len(def.Columns)+len(labelNames))
	for _, col := range def.Columns {
		if col.Dynamic && col.Name == ColumnLabels {
			for _, name := range labelNames {
				fields = append(fields, arrow.Field{
					Name:     ColumnLabelsPrefix + name,
					Type:     dictBinary(),
					Nullable: true,
				})
			}
			continue
		}
		fields = append(fields, columnToArrowField(col))
	}
	return arrow.NewSchema(fields, nil)
}

func columnToArrowField(col *schemapb.Column) arrow.Field {
	layout := col.StorageLayout
	field := arrow.Field{Name: col.Name, Nullable: layout.Nullable}
	switch layout.Type {
	case schemapb.StorageLayout_TYPE_INT64:
		field.Type = arrow.PrimitiveTypes.Int64
	case schemapb.StorageLayout_TYPE_STRING:
		field.Type = dictBinary()
	default:
		panic(fmt.Sprintf("profile: unsupported column %q storage type %v", col.Name, layout.Type))
	}
	if layout.Repeated {
		field.Type = arrow.ListOf(field.Type)
	}
	return field
}

func dictBinary() arrow.DataType {
	return &arrow.DictionaryType{
		IndexType: arrow.PrimitiveTypes.Uint32,
		ValueType: arrow.BinaryTypes.Binary,
	}
}
