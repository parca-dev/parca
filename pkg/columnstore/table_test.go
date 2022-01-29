package columnstore

import (
	"fmt"
	"testing"

	"github.com/apache/arrow/go/v7/arrow/memory"
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

	c := New()
	db := c.DB("test")
	table := db.Table("test")
	err := table.EnsureSchema(schema)
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

	err = table.Iterator(memory.NewGoAllocator(), func(ar *ArrowRecord) bool {
		fmt.Println(ar)

		return true
	})
	if err != nil {
		t.Fatal(err)
	}

	// One granule with 3 parts
	require.Equal(t, 1, table.index.Len())
	require.Equal(t, 3, len(table.index.Min().(*Granule).parts))
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

	c := New()
	db := c.DB("test")
	table := db.Table("test")
	err := table.EnsureSchema(schema)
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

	table.granuleIterator(func(g *Granule) bool {
		ar, err := g.ArrowRecord(memory.NewGoAllocator())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("-----------------Granule----------------")
		fmt.Println(ar)
		fmt.Println("----------------------------------------")

		return true
	})

	require.Equal(t, 2, table.index.Len())
	require.Equal(t, 2, table.index.Min().(*Granule).Cardinality())
	require.Equal(t, 3, table.index.Max().(*Granule).Cardinality())
}

/*

	This test is meant for the following case
	If the table index is as follows

	[10,11]
		\
		[12,13,14]


	And we try and insert [8,9], we expect them to be inserted into the top granule

	[8,9,10,11]
		\
		[12,13]

*/
func Test_Table_InsertLowest(t *testing.T) {
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

	c := New()
	db := c.DB("test")
	table := db.Table("test")
	err := table.EnsureSchema(schema)
	if err != nil {
		t.Fatal(err)
	}

	err = table.Insert([]Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label10", Value: "value10"},
				},
				int64(2),
				int64(2),
			},
		},
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label11", Value: "value11"},
				},
				int64(2),
				int64(2),
			},
		},
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label12", Value: "value12"},
				},
				int64(2),
				int64(2),
			},
		},
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label13", Value: "value13"},
				},
				int64(2),
				int64(2),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	err = table.Insert([]Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label14", Value: "value14"},
				},
				int64(2),
				int64(2),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	table.granuleIterator(func(g *Granule) bool {
		ar, err := g.ArrowRecord(memory.NewGoAllocator())
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("-----------------Granule----------------")
		fmt.Println(ar)
		fmt.Println("----------------------------------------")

		return true
	})
}
