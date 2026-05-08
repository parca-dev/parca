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

// SchemaColumnType describes the storage type of a profile column.
type SchemaColumnType int

const (
	SchemaColumnTypeUnknown SchemaColumnType = iota
	SchemaColumnTypeInt64
	SchemaColumnTypeString
)

// SchemaColumn is the static definition of a column in the parca write
// schema. Repeated columns become Arrow Lists; Dynamic columns (currently
// only ColumnLabels) get expanded at write time into one field per dynamic
// name.
type SchemaColumn struct {
	Name     string
	Type     SchemaColumnType
	Repeated bool
	Nullable bool
	Dynamic  bool
}

// SchemaDef holds the static column layout used by the parca write/ingest
// path. The struct is intentionally minimal — encoding and compression
// hints from the previous frostdb-driven definition are dropped because
// they were only consumed by the parquet writer.
type SchemaDef struct {
	Name    string
	Columns []SchemaColumn
}

// SchemaDefinition returns the static column layout of the parca write
// schema. Column order is alphabetical (matches the historical FrostDB
// layout).
func SchemaDefinition() SchemaDef {
	return SchemaDef{
		Name: SchemaName,
		Columns: []SchemaColumn{
			{Name: ColumnDuration, Type: SchemaColumnTypeInt64},
			{Name: ColumnLabels, Type: SchemaColumnTypeString, Nullable: true, Dynamic: true},
			{Name: ColumnName, Type: SchemaColumnTypeString},
			{Name: ColumnPeriod, Type: SchemaColumnTypeInt64},
			{Name: ColumnPeriodType, Type: SchemaColumnTypeString},
			{Name: ColumnPeriodUnit, Type: SchemaColumnTypeString},
			{Name: ColumnSampleType, Type: SchemaColumnTypeString},
			{Name: ColumnSampleUnit, Type: SchemaColumnTypeString},
			{Name: ColumnStacktrace, Type: SchemaColumnTypeString, Repeated: true, Nullable: true},
			{Name: ColumnTimestamp, Type: SchemaColumnTypeInt64},
			{Name: ColumnTimeNanos, Type: SchemaColumnTypeInt64},
			{Name: ColumnValue, Type: SchemaColumnTypeInt64},
		},
	}
}

// BuildArrowSchema returns the Arrow schema for the parca write/ingest
// profile data, expanding the dynamic ColumnLabels column into one
// "labels.<name>" field per labelName. The column order matches
// SchemaDefinition's column order, with dynamic labels emitted in place
// of the labels column. Static columns map to:
//   - SchemaColumnTypeInt64  → arrow.PrimitiveTypes.Int64
//   - SchemaColumnTypeString (non-repeated) → dictionary-encoded binary
//   - SchemaColumnTypeString (repeated) → list of dictionary-encoded binary
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

func columnToArrowField(col SchemaColumn) arrow.Field {
	field := arrow.Field{Name: col.Name, Nullable: col.Nullable}
	switch col.Type {
	case SchemaColumnTypeInt64:
		field.Type = arrow.PrimitiveTypes.Int64
	case SchemaColumnTypeString:
		field.Type = dictBinary()
	default:
		panic(fmt.Sprintf("profile: unsupported column %q storage type %v", col.Name, col.Type))
	}
	if col.Repeated {
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
