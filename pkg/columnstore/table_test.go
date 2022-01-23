package columnstore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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

	table := NewTable(schema)
	err := table.Insert(
		[]Row{{
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
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				int64(2),
				int64(2),
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
					{Name: "label3", Value: "value3"},
				},
				int64(3),
				int64(3),
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	it := table.Iterator()
	for it.Next() {
		fmt.Println(it.Row())
	}

	// Expect the merge to have left us with one granule with one part
	require.Equal(t, 1, len(table.granules))
	require.Equal(t, 1, len(table.granules[0].parts))
	require.Equal(t, []interface{}{
		[]DynamicColumnValue{
			{Name: "label1", Value: "value1"},
			{Name: "label2", Value: "value2"},
		},
		int64(1),
		int64(1),
	}, table.granules[0].least.Values)
	require.Equal(t, 1, table.index.Len())

	// Split the granule
	granuels := table.granules[0].Split(2)
	require.Equal(t, 3, len(granuels))
	require.Equal(t, 1, len(granuels[0].parts))
	require.Equal(t, 1, len(granuels[1].parts))
	require.Equal(t, 1, len(granuels[2].parts))
	require.Equal(t, 2, granuels[0].parts[0].Cardinality)
	require.Equal(t, 2, granuels[1].parts[0].Cardinality)
	require.Equal(t, 1, granuels[2].parts[0].Cardinality)
}
