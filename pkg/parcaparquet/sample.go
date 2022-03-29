package parcaparquet

import (
	"sort"

	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/segmentio/parquet-go"
)

type Sample struct {
	SampleType string
	SampleUnit string
	PeriodType string
	PeriodUnit string
	Labels     labels.Labels
	Stacktrace []byte
	Timestamp  int64
	Duration   int64
	Period     int64
	Value      int64
}

type Samples []Sample

func (s Samples) ToBuffer(schema *dynparquet.Schema) (*dynparquet.Buffer, error) {
	names := s.SampleLabelNames()

	pb, err := schema.NewBuffer(map[string][]string{
		"labels": names,
	})
	if err != nil {
		return nil, err
	}

	var r parquet.Row
	for _, sample := range s {
		r = sample.ToParquetRow(r[:0], names)
		err := pb.WriteRow(r)
		if err != nil {
			return nil, err
		}
	}

	return pb, nil
}

func (s Samples) SampleLabelNames() []string {
	names := []string{}
	seen := map[string]struct{}{}

	for _, sample := range s {
		for _, label := range sample.Labels {
			if _, ok := seen[label.Name]; !ok {
				names = append(names, label.Name)
				seen[label.Name] = struct{}{}
			}
		}
	}
	sort.Strings(names)

	return names
}

func (s Sample) ToParquetRow(row parquet.Row, labelNames []string) parquet.Row {
	// The order of these appends is important. Parquet values must be in the
	// order of the schema and the schema orders columns by their names.

	nameNumber := len(labelNames)
	labelLen := len(s.Labels)

	if row == nil {
		row = make([]parquet.Value, 0, nameNumber+9)
	}

	row = append(row, parquet.ValueOf(s.Duration).Level(0, 0, 0))

	i, j := 0, 0
	for i < nameNumber {
		if labelNames[i] == s.Labels[j].Name {
			row = append(row, parquet.ValueOf(s.Labels[j].Value).Level(0, 1, i+1))
			i++
			j++

			if j >= labelLen {
				for ; i < nameNumber; i++ {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, i+1))
				}
				break
			}
		} else {
			row = append(row, parquet.ValueOf(nil).Level(0, 0, i+1))
			i++
		}
	}

	row = append(row, parquet.ValueOf(s.Period).Level(0, 0, nameNumber+1))
	row = append(row, parquet.ValueOf(s.PeriodType).Level(0, 0, nameNumber+2))
	row = append(row, parquet.ValueOf(s.PeriodUnit).Level(0, 0, nameNumber+3))
	row = append(row, parquet.ValueOf(s.SampleType).Level(0, 0, nameNumber+4))
	row = append(row, parquet.ValueOf(s.SampleUnit).Level(0, 0, nameNumber+5))
	row = append(row, parquet.ValueOf(s.Stacktrace).Level(0, 0, nameNumber+6))
	row = append(row, parquet.ValueOf(s.Timestamp).Level(0, 0, nameNumber+7))
	row = append(row, parquet.ValueOf(s.Value).Level(0, 0, nameNumber+8))

	return row
}
