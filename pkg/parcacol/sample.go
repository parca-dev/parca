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

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow"
	"github.com/polarsignals/frostdb/query/logicalplan"
	"github.com/segmentio/parquet-go"

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
		case ColumnDuration:
			row = append(row, parquet.ValueOf(meta.Duration).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnName:
			row = append(row, parquet.ValueOf(meta.Name).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnPeriod:
			row = append(row, parquet.ValueOf(meta.Period).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnPeriodType:
			row = append(row, parquet.ValueOf(meta.PeriodType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnPeriodUnit:
			row = append(row, parquet.ValueOf(meta.PeriodType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnSampleType:
			row = append(row, parquet.ValueOf(meta.SampleType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnSampleUnit:
			row = append(row, parquet.ValueOf(meta.SampleType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnStacktrace:
			row = append(row, parquet.ValueOf(s.StacktraceID).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnTimestamp:
			row = append(row, parquet.ValueOf(meta.Timestamp).Level(0, 0, columnIndex))
			columnIndex++
		case ColumnValue:
			row = append(row, parquet.ValueOf(s.Value).Level(0, 0, columnIndex))
			columnIndex++

		// All remaining cases take care of dynamic columns
		case ColumnLabels:
			for _, name := range labelNames {
				if value, ok := lset[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case ColumnPprofLabels:
			for _, name := range profileLabelNames {
				if value, ok := s.Label[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case ColumnPprofNumLabels:
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
	schema *dynparquet.Schema,
	series []Series,
	labelNames, profileLabelNames, profileNumLabelNames []string,
) (arrow.Record, error) {
	ps, err := schema.GetDynamicParquetSchema(map[string][]string{
		ColumnLabels:         labelNames,
		ColumnPprofLabels:    profileLabelNames,
		ColumnPprofNumLabels: profileNumLabelNames,
	})
	if err != nil {
		return nil, err
	}
	defer schema.PutPooledParquetSchema(ps)

	ctx := context.Background()
	as, err := pqarrow.ParquetSchemaToArrowSchema(ctx, ps.Schema, logicalplan.IterOptions{})
	if err != nil {
		return nil, err
	}

	bldr := array.NewRecordBuilder(memory.NewGoAllocator(), as)
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
						case ColumnDuration:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Duration)
							i++
						case ColumnName:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.Name)
							i++
						case ColumnPeriod:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Period)
							i++
						case ColumnPeriodType:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.PeriodType.Type)
							i++
						case ColumnPeriodUnit:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.PeriodType.Unit)
							i++
						case ColumnSampleType:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.SampleType.Type)
							i++
						case ColumnSampleUnit:
							err = bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(p.Meta.SampleType.Unit)
							i++
						case ColumnStacktrace:
							bldr.Field(i).(*array.BinaryBuilder).AppendString(sample.StacktraceID)
							i++
						case ColumnTimestamp:
							bldr.Field(i).(*array.Int64Builder).Append(p.Meta.Timestamp)
							i++
						case ColumnValue:
							bldr.Field(i).(*array.Int64Builder).Append(sample.Value)
							i++
						case ColumnLabels:
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
						case ColumnPprofLabels:
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
						case ColumnPprofNumLabels:
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
func ParquetBufToArrowRecord(ctx context.Context, buf *dynparquet.Buffer, rowsPerRecord uint) ([]arrow.Record, error) {
	as, err := pqarrow.ParquetSchemaToArrowSchema(ctx, buf.Schema(), logicalplan.IterOptions{})
	if err != nil {
		return nil, err
	}

	bldr := array.NewRecordBuilder(memory.NewGoAllocator(), as)
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
				case ColumnName:
					fallthrough
				case ColumnPeriodType:
					fallthrough
				case ColumnPeriodUnit:
					fallthrough
				case ColumnSampleType:
					fallthrough
				case ColumnSampleUnit:
					if err := bldr.Field(i).(*array.BinaryDictionaryBuilder).AppendString(r[i].String()); err != nil {
						return nil, err
					}
				case ColumnStacktrace:
					bldr.Field(i).(*array.BinaryBuilder).AppendString(r[i].String())
				case ColumnDuration:
					fallthrough
				case ColumnPeriod:
					fallthrough
				case ColumnTimestamp:
					fallthrough
				case ColumnValue:
					bldr.Field(i).(*array.Int64Builder).Append(r[i].Int64())
				default:
					switch {
					case strings.HasPrefix(as.Field(i).Name, ColumnPprofNumLabels):
						if r[i].IsNull() {
							bldr.Field(i).AppendNull()
						} else {
							bldr.Field(i).(*array.Int64Builder).Append(r[i].Int64())
						}
					case strings.HasPrefix(as.Field(i).Name, ColumnPprofLabels):
						fallthrough
					case strings.HasPrefix(as.Field(i).Name, ColumnLabels):
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
