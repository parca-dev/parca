package parcacol

import (
	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/segmentio/parquet-go"
)

const (
	schemaName           = "parca"
	columnSampleType     = "sample_type"
	columnSampleUnit     = "sample_unit"
	columnPeriodType     = "period_type"
	columnPeriodUnit     = "period_unit"
	columnPprofLabels    = "pprof_labels"
	columnPprofNumLabels = "pprof_num_labels"
	columnLabels         = "labels"
	columnStacktrace     = "stacktrace"
	columnTimestamp      = "timestamp"
	columnDuration       = "duration"
	columnPeriod         = "period"
	columnValue          = "value"
)

func Schema() *dynparquet.Schema {
	return dynparquet.NewSchema(
		schemaName,
		[]dynparquet.ColumnDefinition{{
			Name:          columnSampleType,
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          columnSampleUnit,
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
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
			StorageLayout: parquet.Encoded(parquet.Optional(parquet.String()), &parquet.RLEDictionary),
			Dynamic:       true,
		}, {
			Name:          columnLabels,
			StorageLayout: parquet.Encoded(parquet.Optional(parquet.String()), &parquet.RLEDictionary),
			Dynamic:       true,
		}, {
			Name:          columnStacktrace,
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          columnTimestamp,
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          columnDuration,
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          columnPeriod,
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          columnValue,
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}},
		[]dynparquet.SortingColumn{
			dynparquet.Ascending(columnSampleType),
			dynparquet.Ascending(columnSampleUnit),
			dynparquet.Ascending(columnPeriodType),
			dynparquet.Ascending(columnPeriodUnit),
			dynparquet.NullsFirst(dynparquet.Ascending(columnPprofLabels)),
			dynparquet.NullsFirst(dynparquet.Ascending(columnPprofNumLabels)),
			dynparquet.NullsFirst(dynparquet.Ascending(columnLabels)),
			dynparquet.NullsFirst(dynparquet.Ascending(columnStacktrace)),
			dynparquet.Ascending(columnTimestamp),
		},
	)
}
