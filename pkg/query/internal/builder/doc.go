// Package builder vendors the optimized Arrow array builders from
// github.com/polarsignals/frostdb/pqarrow/builder. These builders expose
// random-access mutation methods (Set/Add/Value/AppendData) that the
// flamegraph and table query algorithms rely on, which the upstream
// arrow-go array.Builder API does not provide.
//
// AppendParquetValues methods and the parquet-go dependency from the
// upstream package have been dropped — parca uses these builders only on
// the query side and never to convert parquet values.
package builder
