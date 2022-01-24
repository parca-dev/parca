package columnstore

import (
	"fmt"
	"testing"

	"github.com/google/btree"
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
		OrderedBy:   []string{"labels", "timestamp"},
		GranuleSize: 2 ^ 13, // 8192
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

	table.Iterator(func(i btree.Item) bool {
		g := i.(*Granule)
		it := g.Iterator()
		for it.Next() {
			fmt.Println(it.Row())
		}

		return true
	})

	// Expect the merge to have left us with one granule with one part
	require.Equal(t, 1, table.index.Len())
	require.Equal(t, 1, len(table.index.Min().(*Granule).parts))
	require.Equal(t, 5, table.index.Min().(*Granule).Cardinality())
	require.Equal(t, []interface{}{
		[]DynamicColumnValue{
			{Name: "label1", Value: "value1"},
			{Name: "label2", Value: "value2"},
		},
		int64(1),
		int64(1),
	}, table.index.Min().(*Granule).least.Values)
	require.Equal(t, 1, table.index.Len())
}

func Test_Table_GranuleSplit(t *testing.T) {
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
		OrderedBy:   []string{"labels", "timestamp"},
		GranuleSize: 4,
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

	table.Iterator(func(i btree.Item) bool {
		g := i.(*Granule)
		it := g.Iterator()
		fmt.Println("-----------------Granule----------------")
		for it.Next() {
			fmt.Println(it.Row())
		}
		fmt.Println("----------------------------------------")

		return true
	})

	require.Equal(t, 2, table.index.Len())
	require.Equal(t, 2, table.index.Min().(*Granule).Cardinality())
	require.Equal(t, 3, table.index.Max().(*Granule).Cardinality())
}
