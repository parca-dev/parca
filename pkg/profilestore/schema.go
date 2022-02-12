package profilestore

import (
	"github.com/parca-dev/parca/pkg/columnstore"
)

func ParcaProfilingTableSchema() columnstore.Schema {
	return columnstore.NewSchema(
		[]columnstore.ColumnDefinition{{
			Name:     "sample_type",
			Type:     columnstore.StringType,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "sample_unit",
			Type:     columnstore.StringType,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "period_type",
			Type:     columnstore.StringType,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "period_unit",
			Type:     columnstore.StringType,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "labels",
			Type:     columnstore.StringType,
			Encoding: columnstore.PlainEncoding,
			Dynamic:  true,
		}, {
			Name: "stacktrace",
			// This should be a UUID, but we don't have a UUID type yet. For
			// now, we'll just use a string. UUIDs might also be best
			// represented as a Uint128 internally.
			Type:     columnstore.List(columnstore.UUIDType),
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "timestamp",
			Type:     columnstore.Int64Type,
			Encoding: columnstore.PlainEncoding,
			// TODO
			//}, {
			//	Name:     "pprof_labels",
			//	Type:     columnstore.StringType,
			//	Encoding: columnstore.PlainEncoding,
			//}, {
			//	Name:     "pprof_num_labels",
			//	Type:     columnstore.Int64Type,
			//	Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "duration",
			Type:     columnstore.Int64Type,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "period",
			Type:     columnstore.Int64Type,
			Encoding: columnstore.PlainEncoding,
		}, {
			Name:     "value",
			Type:     columnstore.Int64Type,
			Encoding: columnstore.PlainEncoding,
		}},
		8192, // 2^13
	)
}
