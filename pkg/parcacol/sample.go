package parcacol

import (
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
		r = sample.ToParquetRow(r[:0], names, pprofLabels, pprofNumLabels)
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

func (s Sample) ToParquetRow(row parquet.Row, labelNames, pprofLabelNames, pprofNumLabelNames []string) parquet.Row {
	// The order of these appends is important. Parquet values must be in the
	// order of the schema and the schema orders columns by their names.
	labelNamesLen := len(labelNames)
	pprofLabelsNamesLen := len(pprofLabelNames)
	pprofNumLabelsNamesLen := len(pprofNumLabelNames)
	dynamicNum := labelNamesLen + pprofLabelsNamesLen + pprofNumLabelsNamesLen

	if row == nil {
		row = make([]parquet.Value, 0, 9+dynamicNum)
	}

	row = append(row, parquet.ValueOf(s.Duration).Level(0, 0, 0))

	// Labels
	i, j := 0, 0
	for i < labelNamesLen {
		columnIndex := i + 1

		if labelNames[i] == s.Labels[j].Name {
			value := parquet.ValueOf(s.Labels[j].Value).Level(0, 1, columnIndex)
			row = append(row, value)
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
			i++
		}
	}

	// We add these columns at their index with the initial padding of the size of dynamic columns.
	row = append(row, parquet.ValueOf(s.Period).Level(0, 0, labelNamesLen+1))
	row = append(row, parquet.ValueOf(s.PeriodType).Level(0, 0, labelNamesLen+2))
	row = append(row, parquet.ValueOf(s.PeriodUnit).Level(0, 0, labelNamesLen+3))

	// pprofLabels
	i, j = 0, 0
	for i < pprofLabelsNamesLen {
		// add the previous labelName column index on top
		// duration, period, period_type, period_unit are index 4 on top
		columnIndex := labelNamesLen + i + 4

		if pprofLabelNames[i] == s.PprofLabels[j].Name {
			value := parquet.ValueOf(s.PprofLabels[j].Value).Level(0, 1, columnIndex)
			row = append(row, value)
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
			i++
		}
	}

	// pprofNumLabels
	i, j = 0, 0
	for i < pprofNumLabelsNamesLen {
		// add the previous labelNames and pprofLabelsNames column index on top
		// duration, period, period_type, period_unit are index 4 on top
		columnIndex := labelNamesLen + pprofLabelsNamesLen + i + 4

		if pprofNumLabelNames[i] == s.PprofNumLabels[j].Name {
			value := parquet.ValueOf(s.PprofNumLabels[j].Value).Level(0, 1, columnIndex)
			row = append(row, value)
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
			i++
		}
	}

	// the indexes start at dynamicNum which is the sum of all dynamic columns.
	// on top of that start with the 4 for: duration, period, period_type, period_unit
	row = append(row, parquet.ValueOf(s.SampleType).Level(0, 0, dynamicNum+4))
	row = append(row, parquet.ValueOf(s.SampleUnit).Level(0, 0, dynamicNum+5))
	row = append(row, parquet.ValueOf(s.Stacktrace).Level(0, 0, dynamicNum+6))
	row = append(row, parquet.ValueOf(s.Timestamp).Level(0, 0, dynamicNum+7))
	row = append(row, parquet.ValueOf(s.Value).Level(0, 0, dynamicNum+8))

	return row
}
