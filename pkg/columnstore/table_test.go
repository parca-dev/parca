package columnstore

import (
	"fmt"
	"testing"
)

func TestTable(t *testing.T) {
	schema := Schema{
		Columns: []ColumnDefinition{{
			Name:     "labels",
			Type:     StringType,
			Encoding: PlainEncoding,
			Dynamic:  true,
		}, {
			Name:     "timestamp",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "value",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}},
		OrderedBy: []string{"labels", "timestamp"},
		//WithGranuleSize(2^13), // 8192
		//WithOrderedColumns(
		//	labelsColumn,
		//	timestampColumn,
		//),
	}

	rows := []Row{{
		Values: []interface{}{
			[]DynamicColumnValue{
				{Name: "label1", Value: "value1"},
				{Name: "label2", Value: "value2"},
			},
			int64(1),
			int64(1),
		},
	}, {
		Values: []interface{}{
			[]DynamicColumnValue{
				{Name: "label1", Value: "value1"},
				{Name: "label2", Value: "value2"},
				{Name: "label3", Value: "value3"},
			},
			int64(2),
			int64(2),
		},
	}, {
		Values: []interface{}{
			[]DynamicColumnValue{
				{Name: "label1", Value: "value1"},
				{Name: "label2", Value: "value2"},
				{Name: "label4", Value: "value4"},
			},
			int64(3),
			int64(3),
		},
	}}

	part, err := NewPart(schema, rows)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println(part.String())
}
