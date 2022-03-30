package parcacol

import (
	"fmt"
	"sort"

	"github.com/polarsignals/arcticdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/segmentio/parquet-go"
)

type Sample struct {
	SampleType     string
	SampleUnit     string
	PeriodType     string
	PeriodUnit     string
	PprofLabels    labels.Labels
	PprofNumLabels labels.Labels
	Labels         labels.Labels
	Stacktrace     []byte
	Timestamp      int64
	Duration       int64
	Period         int64
	Value          int64
}

type Samples []Sample

func (s Samples) ToBuffer(schema *dynparquet.Schema) (*dynparquet.Buffer, error) {
	names := s.SampleLabelNames()
	pprofLabels := s.pprofLabelsNames()
	pprofNumLabels := s.pprofNumLabelsNames()

	pb, err := schema.NewBuffer(map[string][]string{
		columnLabels:         names,
		columnPprofLabels:    pprofLabels,
		columnPprofNumLabels: pprofNumLabels,
	})
	if err != nil {
		return nil, err
	}

	var r parquet.Row
	for _, sample := range s {
		r = sample.ToParquetRow(schema, r[:0], names, pprofLabels, pprofNumLabels)
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

func (s Samples) pprofLabelsNames() []string {
	names := []string{}
	seen := map[string]struct{}{}

	for _, sample := range s {
		for _, label := range sample.PprofLabels {
			if _, ok := seen[label.Name]; !ok {
				names = append(names, label.Name)
				seen[label.Name] = struct{}{}
			}
		}
	}
	sort.Strings(names)

	return names
}

func (s Samples) pprofNumLabelsNames() []string {
	names := []string{}
	seen := map[string]struct{}{}

	for _, sample := range s {
		for _, label := range sample.PprofNumLabels {
			if _, ok := seen[label.Name]; !ok {
				names = append(names, label.Name)
				seen[label.Name] = struct{}{}
			}
		}
	}
	sort.Strings(names)

	return names
}

func (s Sample) ToParquetRow(schema *dynparquet.Schema, row parquet.Row, labelNames, pprofLabelNames, pprofNumLabelNames []string) parquet.Row {
	// schema.Columns() returns a sorted list of all columns.
	// We match on the column's name to insert the correct values.
	// We track the columnIndex to insert each column at the correct index.
	columnIndex := 0
	for _, column := range schema.Columns() {
		switch column.Name {
		case columnDuration:
			row = append(row, parquet.ValueOf(s.Duration).Level(0, 0, columnIndex))
			columnIndex++
		case columnPeriod:
			row = append(row, parquet.ValueOf(s.Period).Level(0, 0, columnIndex))
			columnIndex++
		case columnPeriodType:
			row = append(row, parquet.ValueOf(s.PeriodType).Level(0, 0, columnIndex))
			columnIndex++
		case columnPeriodUnit:
			row = append(row, parquet.ValueOf(s.PeriodUnit).Level(0, 0, columnIndex))
			columnIndex++
		case columnSampleType:
			row = append(row, parquet.ValueOf(s.SampleType).Level(0, 0, columnIndex))
			columnIndex++
		case columnSampleUnit:
			row = append(row, parquet.ValueOf(s.SampleUnit).Level(0, 0, columnIndex))
			columnIndex++
		case columnStacktrace:
			row = append(row, parquet.ValueOf(s.Stacktrace).Level(0, 0, columnIndex))
			columnIndex++
		case columnTimestamp:
			row = append(row, parquet.ValueOf(s.Timestamp).Level(0, 0, columnIndex))
			columnIndex++
		case columnValue:
			row = append(row, parquet.ValueOf(s.Value).Level(0, 0, columnIndex))
			columnIndex++

		// All remaining cases take care of dynamic columns
		case columnLabels:
			labelNamesLen := len(labelNames)
			i, j := 0, 0
			for i < labelNamesLen {
				if labelNames[i] == s.Labels[j].Name {
					value := parquet.ValueOf(s.Labels[j].Value).Level(0, 1, columnIndex)
					row = append(row, value)
					columnIndex++
					i++
					j++

					if j >= len(s.Labels) {
						for ; i < labelNamesLen; i++ {
							row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
						}
						break
					}
				} else {
					// If nothing matches we add a NULL to the column
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
					columnIndex++
					i++
				}
			}
		case columnPprofLabels:
			i, j := 0, 0
			pprofLabelsNamesLen := len(pprofLabelNames)
			for i < pprofLabelsNamesLen {
				if pprofLabelNames[i] == s.PprofLabels[j].Name {
					value := parquet.ValueOf(s.PprofLabels[j].Value).Level(0, 1, columnIndex)
					row = append(row, value)
					columnIndex++
					i++
					j++

					if j >= len(s.PprofLabels) {
						for ; i < pprofLabelsNamesLen; i++ {
							row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
						}
						break
					}
				} else {
					// If nothing matches we add a NULL to the column
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
					columnIndex++
					i++
				}
			}
		case columnPprofNumLabels:
			i, j := 0, 0
			pprofNumLabelsNamesLen := len(pprofNumLabelNames)
			for i < pprofNumLabelsNamesLen {
				if pprofNumLabelNames[i] == s.PprofNumLabels[j].Name {
					value := parquet.ValueOf(s.PprofNumLabels[j].Value).Level(0, 1, columnIndex)
					row = append(row, value)
					columnIndex++
					i++
					j++

					if j >= len(s.PprofNumLabels) {
						for ; i < pprofNumLabelsNamesLen; i++ {
							row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
						}
						break
					}
				} else {
					// If nothing matches we add a NULL to the column
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
					columnIndex++
					i++
				}
			}
		default:
			panic(fmt.Errorf("conversion not implement for column: %s", column.Name))
		}
	}

	return row
}
