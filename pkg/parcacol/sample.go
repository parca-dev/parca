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
	"fmt"
	"sort"

	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/segmentio/parquet-go"

	"github.com/parca-dev/parca/pkg/profile"
)

// NormalizedProfileToParquetBuffer converts a normalized profile to a Parquet
// buffer. The passed labels must be sorted.
func NormalizedProfileToParquetBuffer(schema *dynparquet.Schema, ls labels.Labels, p *profile.NormalizedProfile) (*dynparquet.Buffer, error) {
	names := labelNames(ls)
	pprofLabels := profileLabelNames(p)
	pprofNumLabels := profileNumLabelNames(p)

	pb, err := schema.NewBuffer(map[string][]string{
		ColumnLabels:         names,
		ColumnPprofLabels:    pprofLabels,
		ColumnPprofNumLabels: pprofNumLabels,
	})
	if err != nil {
		return nil, err
	}

	var r parquet.Row
	for _, sample := range p.Samples {
		r = SampleToParquetRow(
			schema,
			r[:0],
			pprofLabels,
			pprofNumLabels,
			ls,
			p.Meta,
			sample,
		)
		_, err := pb.WriteRows([]parquet.Row{r})
		if err != nil {
			return nil, err
		}
	}

	return pb, nil
}

func labelNames(ls labels.Labels) []string {
	names := []string{}

	for _, label := range ls {
		names = append(names, label.Name)
	}

	return names
}

func profileLabelNames(p *profile.NormalizedProfile) []string {
	names := []string{}
	seen := map[string]struct{}{}

	for _, sample := range p.Samples {
		for name := range sample.Label {
			if _, ok := seen[name]; !ok {
				names = append(names, name)
				seen[name] = struct{}{}
			}
		}
	}
	sort.Strings(names)

	return names
}

func profileNumLabelNames(p *profile.NormalizedProfile) []string {
	names := []string{}
	seen := map[string]struct{}{}

	for _, sample := range p.Samples {
		for name := range sample.NumLabel {
			if _, ok := seen[name]; !ok {
				names = append(names, name)
				seen[name] = struct{}{}
			}
		}
	}
	sort.Strings(names)

	return names
}

// SampleToParquetRow converts a sample to a Parquet row. The passed labels
// must be sorted.
func SampleToParquetRow(
	schema *dynparquet.Schema,
	row parquet.Row,
	profileLabelNames, profileNumLabelNames []string,
	ls labels.Labels,
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
			for _, label := range ls {
				row = append(row, parquet.ValueOf(label.Value).Level(0, 1, columnIndex))
				columnIndex++
			}
		case ColumnPprofLabels:
			for _, name := range profileLabelNames {
				if value, ok := s.Label[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
					columnIndex++
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
					columnIndex++
				}
			}
		case ColumnPprofNumLabels:
			for _, name := range profileNumLabelNames {
				if value, ok := s.NumLabel[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
					columnIndex++
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
					columnIndex++
				}
			}
		default:
			panic(fmt.Errorf("conversion not implement for column: %s", column.Name))
		}
	}

	return row
}
