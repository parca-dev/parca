// Copyright 2022-2023 The Parca Authors
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
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow"
	"github.com/polarsignals/frostdb/query/logicalplan"

	"github.com/parca-dev/parca/pkg/normalizer"
	"github.com/parca-dev/parca/pkg/profile"
)

// SampleToParquetRow converts a sample to a Parquet row. The passed labels
// must be sorted.
func SampleToParquetRow(
	schema *dynparquet.Schema,
	row parquet.Row,
	labelNames, profileLabelNames, profileNumLabelNames []string,
	lset map[string]string,
	meta profile.Meta,
	s *profile.NormalizedSample,
) parquet.Row {
	// schema.Columns() returns a sorted list of all columns.
	// We match on the column's name to insert the correct values.
	// We track the columnIndex to insert each column at the correct index.
	columnIndex := 0
	for _, column := range schema.Columns() {
		switch column.Name {
		case profile.ColumnDuration:
			row = append(row, parquet.ValueOf(meta.Duration).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnName:
			row = append(row, parquet.ValueOf(meta.Name).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriod:
			row = append(row, parquet.ValueOf(meta.Period).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriodType:
			row = append(row, parquet.ValueOf(meta.PeriodType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriodUnit:
			row = append(row, parquet.ValueOf(meta.PeriodType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnSampleType:
			row = append(row, parquet.ValueOf(meta.SampleType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnSampleUnit:
			row = append(row, parquet.ValueOf(meta.SampleType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnStacktrace:
			row = append(row, parquet.ValueOf(s.StacktraceID).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnTimestamp:
			row = append(row, parquet.ValueOf(meta.Timestamp).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnValue:
			row = append(row, parquet.ValueOf(s.Value).Level(0, 0, columnIndex))
			columnIndex++

		// All remaining cases take care of dynamic columns
		case profile.ColumnLabels:
			for _, name := range labelNames {
				if value, ok := lset[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case profile.ColumnPprofLabels:
			for _, name := range profileLabelNames {
				if value, ok := s.Label[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case profile.ColumnPprofNumLabels:
			for _, name := range profileNumLabelNames {
				if value, ok := s.NumLabel[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		default:
			panic(fmt.Errorf("conversion not implement for column: %s", column.Name))
		}
	}

	return row
}

func SeriesToArrowRecord(
	mem memory.Allocator,
	schema *dynparquet.Schema,
	series []normalizer.Series,
	labelNames, profileLabelNames, profileNumLabelNames []string,
) (arrow.Record, error) {
	ps, err := schema.GetDynamicParquetSchema(map[string][]string{
		profile.ColumnLabels:         labelNames,
		profile.ColumnPprofLabels:    profileLabelNames,
		profile.ColumnPprofNumLabels: profileNumLabelNames,
	})
	if err != nil {
		return nil, err
	}
	defer schema.PutPooledParquetSchema(ps)

	ctx := context.Background()
	as, err := pqarrow.ParquetSchemaToArrowSchema(ctx, ps.Schema, schema, logicalplan.IterOptions{})
	if err != nil {
		return nil, err
	}

	bldr := array.NewRecordBuilder(mem, as)
	defer bldr.Release()

	for _, s := range series {
		for _, np := range s.Samples {
			for _, p := range np {
				if len(p.Samples) == 0 {
					continue
				}

				for _, sample := range p.Samples {
					i := 0
					for _, col := range schema.Columns() {
						switch col.Name {
						case profile.ColumnDuration:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Duration)
							i++
						case profile.ColumnName:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.Name)
							i++
						case profile.ColumnPeriod:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Period)
							i++
						case profile.ColumnPeriodType:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.PeriodType.Type)
							i++
						case profile.ColumnPeriodUnit:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.PeriodType.Unit)
							i++
						case profile.ColumnSampleType:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.SampleType.Type)
							i++
						case profile.ColumnSampleUnit:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.SampleType.Unit)
							i++
						case profile.ColumnStacktrace:
							bldr.Field(i).(*array.BinaryBuilder).AppendString(sample.StacktraceID)
							i++
						case profile.ColumnTimestamp:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Timestamp)
							i++
						case profile.ColumnValue:
							bldr.Field(i).(*array.Int64Builder).Append(sample.Value)
							i++
						case profile.ColumnLabels:
							for _, name := range labelNames {
								if value, ok := s.Labels[name]; ok {
									if err := bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(value); err != nil {
										return nil, err
									}
								} else {
									bldr.Field(i).AppendNull()
								}
								i++
							}
						case profile.ColumnPprofLabels:
							for _, name := range profileLabelNames {
								if value, ok := sample.Label[name]; ok {
									if err := bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(value); err != nil {
										return nil, err
									}
								} else {
									bldr.Field(i).AppendNull()
								}
								i++
							}
						case profile.ColumnPprofNumLabels:
							for _, name := range profileNumLabelNames {
								if value, ok := sample.NumLabel[name]; ok {
									bldr.Field(i).(*array.Int64Builder).Append(value)
								} else {
									bldr.Field(i).AppendNull()
								}
								i++
							}
						default:
							panic(fmt.Sprintf("unknown column %v", col.Name))
						}
					}

					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return bldr.NewRecord(), nil
}

// ParquetBufToArrowRecord converts a parquet buffer to an arrow record. If rowsPerRecord is 0, then the entire buffer is converted to a single record.
func ParquetBufToArrowRecord(ctx context.Context, mem memory.Allocator, buf *dynparquet.Buffer, s *dynparquet.Schema, rowsPerRecord uint) ([]arrow.Record, error) {
	as, err := pqarrow.ParquetSchemaToArrowSchema(ctx, buf.Schema(), s, logicalplan.IterOptions{})
	if err != nil {
		return nil, err
	}

	bldr := array.NewRecordBuilder(mem, as)
	defer bldr.Release()

	rows := buf.Rows()
	defer rows.Close()

	buffSize := 256
	rowBuf := make([]parquet.Row, buffSize)

	records := []arrow.Record{}
	recordSize := uint(0)
	for {
		n, err := rows.ReadRows(rowBuf)
		if err != nil && err != io.EOF {
			return nil, err
		}

		if n == 0 {
			break
		}

		// Write each row to the arrow record
		for _, r := range rowBuf[:n] {
			for i := range r {
				switch as.Field(i).Name {
				case profile.ColumnName:
					fallthrough
				case profile.ColumnPeriodType:
					fallthrough
				case profile.ColumnPeriodUnit:
					fallthrough
				case profile.ColumnSampleType:
					fallthrough
				case profile.ColumnSampleUnit:
					if err := bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(r[i].String()); err != nil {
						return nil, err
					}
				case profile.ColumnStacktrace:
					bldr.Field(i).(*array.BinaryBuilder).AppendString(r[i].String())
				case profile.ColumnDuration:
					fallthrough
				case profile.ColumnPeriod:
					fallthrough
				case profile.ColumnTimestamp:
					fallthrough
				case profile.ColumnValue:
					bldr.Field(i).(*array.Int64Builder).Append(r[i].Int64())
				default:
					switch {
					case strings.HasPrefix(as.Field(i).Name, profile.ColumnPprofNumLabels):
						if r[i].IsNull() {
							bldr.Field(i).AppendNull()
						} else {
							bldr.Field(i).(*array.Int64Builder).Append(r[i].Int64())
						}
					case strings.HasPrefix(as.Field(i).Name, profile.ColumnPprofLabels):
						fallthrough
					case strings.HasPrefix(as.Field(i).Name, profile.ColumnLabels):
						if r[i].IsNull() {
							bldr.Field(i).AppendNull()
						} else {
							if err := bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(r[i].String()); err != nil {
								return nil, err
							}
						}
					default:
						panic(fmt.Sprintf("unknown column %v", as.Field(i).Name))
					}
				}
			}

			recordSize++
			if rowsPerRecord != 0 && recordSize >= rowsPerRecord {
				records = append(records, bldr.NewRecord())
				recordSize = 0
			}
		}
	}

	// Append final record
	r := bldr.NewRecord()
	if r.NumRows() > 0 {
		records = append(records, r)
	}

	return records, nil
}
