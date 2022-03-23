package parcaparquet

import (
	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/segmentio/parquet-go"
)

func Schema() *dynparquet.Schema {
	return dynparquet.NewSchema(
		"parca",
		[]dynparquet.ColumnDefinition{{
			Name:          "sample_type",
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          "sample_unit",
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          "period_type",
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          "period_unit",
			StorageLayout: parquet.Encoded(parquet.String(), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          "labels",
			StorageLayout: parquet.Encoded(parquet.Optional(parquet.String()), &parquet.RLEDictionary),
			Dynamic:       true,
		}, {
			Name:          "stacktrace",
			StorageLayout: parquet.Encoded(parquet.Repeated(parquet.UUID()), &parquet.RLEDictionary),
			Dynamic:       false,
		}, {
			Name:          "timestamp",
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          "duration",
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          "period",
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}, {
			Name:          "value",
			StorageLayout: parquet.Int(64),
			Dynamic:       false,
		}},
		[]dynparquet.SortingColumn{
			dynparquet.Ascending("sample_type"),
			dynparquet.Ascending("sample_unit"),
			dynparquet.Ascending("period_type"),
			dynparquet.Ascending("period_unit"),
			dynparquet.NullsFirst(dynparquet.Ascending("labels")),
			dynparquet.NullsFirst(dynparquet.Ascending("stacktrace")),
			dynparquet.Ascending("timestamp"),
		},
	)
}
