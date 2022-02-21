package columnstore

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"testing"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func basicTable(t *testing.T, granuleSize int) *Table {
	schema := NewSchema(
		[]ColumnDefinition{{
			Name:     "labels",
			Type:     StringType,
			Encoding: PlainEncoding,
			Dynamic:  true,
		}, {
			Name:     "stacktrace",
			Type:     List(UUIDType),
			Encoding: PlainEncoding,
		}, {
			Name:     "timestamp",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "value",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}},
		granuleSize,
		"labels",
		"timestamp",
	)

	c := New(nil)
	db := c.DB("test")
	table := db.Table("test", schema, log.NewNopLogger())

	return table
}

func TestTable(t *testing.T) {
	table := basicTable(t, 2^12)

	err := table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(3),
				int64(3),
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	err = table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
		fmt.Println(ar)
		defer ar.Release()

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// One granule with 3 parts
	require.Equal(t, 1, table.index.Len())
	require.Equal(t, 3, len(table.index.Min().(*Granule).parts))
	require.Equal(t, 5, table.index.Min().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted))
	require.Equal(t, []interface{}{
		[]DynamicColumnValue{
			{Name: "label1", Value: "value1"},
			{Name: "label2", Value: "value2"},
		},
		[]UUID{
			{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
			{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
		},
		int64(1),
		int64(1),
	}, table.index.Min().(*Granule).least.Values)
	require.Equal(t, 1, table.index.Len())
}

func Test_Table_GranuleSplit(t *testing.T) {
	table := basicTable(t, 4)

	err := table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(3),
				int64(3),
			},
		}},
	)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the index to be updated by the asynchronous granule split.
	table.Sync()

	table.Iterator(memory.NewGoAllocator(), func(r arrow.Record) error {
		defer r.Release()
		fmt.Println(r)
		return nil
	})

	require.Equal(t, 2, table.index.Len())
	require.Equal(t, 2, table.index.Min().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted))
	require.Equal(t, 3, table.index.Max().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted))
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
	table := basicTable(t, 4)

	err := table.Insert([]Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label10", Value: "value10"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(2),
				int64(2),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Adding a 5th element should cause a split
	err = table.Insert([]Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label14", Value: "value14"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(2),
				int64(2),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the index to be updated by the asynchronous granule split.
	table.Sync()

	require.Equal(t, 2, table.index.Len())
	require.Equal(t, 2, table.index.Min().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted)) // [10,11]
	require.Equal(t, 3, table.index.Max().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted)) // [12,13,14]

	// Insert a new column that is the lowest column yet; expect it to be added to the minimum column
	err = table.Insert([]Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(10),
				int64(10),
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	require.Equal(t, 2, table.index.Len())
	require.Equal(t, 3, table.index.Min().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted)) // [1,10,11]
	require.Equal(t, 3, table.index.Max().(*Granule).Cardinality(math.MaxUint64, table.db.txCompleted)) // [12,13,14]
}

// This test issues concurrent writes to the database, and expects all of them to be recorded successfully.
func Test_Table_Concurrency(t *testing.T) {
	table := basicTable(t, 1<<13)

	generateRows := func(n int) []Row {
		rows := make([]Row, 0, n)
		for i := 0; i < n; i++ {
			rows = append(rows, Row{
				Values: []interface{}{
					[]DynamicColumnValue{ // TODO would be nice to not have all the same column
						{Name: "label1", Value: "value1"},
						{Name: "label2", Value: "value2"},
					},
					[]UUID{
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
					},
					rand.Int63(),
					rand.Int63(),
				},
			})
		}
		return rows
	}

	// Spawn n workers that will insert values into the table
	n := 8
	inserts := 100
	rows := 10
	wg := &sync.WaitGroup{}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < inserts; i++ {
				if err := table.Insert(generateRows(rows)); err != nil {
					fmt.Println("Received error on insert: ", err)
				}
			}
		}()
	}

	// TODO probably have them generate until a stop event
	wg.Wait()

	totalrows := int64(0)
	err := table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
		totalrows += ar.NumRows()
		defer ar.Release()

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(n*inserts*rows), totalrows)
}

func Benchmark_Table_Insert_10Rows_10Iter_10Writers(b *testing.B) {
	benchmarkTableInserts(b, 10, 10, 10)
}

func Benchmark_Table_Insert_100Row_100Iter_100Writers(b *testing.B) {
	benchmarkTableInserts(b, 100, 100, 100)
}

func benchmarkTableInserts(b *testing.B, rows, iterations, writers int) {
	schema := NewSchema(
		[]ColumnDefinition{{
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
		2<<13,
		"labels",
		"timestamp",
	)

	c := New(nil)
	db := c.DB("test")
	generateRows := func(id string, n int) []Row {
		rows := make([]Row, 0, n)
		for i := 0; i < n; i++ {
			rows = append(rows, Row{
				Values: []interface{}{
					[]DynamicColumnValue{ // TODO would be nice to not have all the same column
						{Name: "label1", Value: id},
						{Name: "label2", Value: "value2"},
					},
					rand.Int63(),
					int64(i),
				},
			})
		}
		return rows
	}

	// Pre-generate all rows we're inserting
	inserts := make(map[string][]Row, writers)
	for i := 0; i < writers; i++ {
		id := uuid.New().String()
		inserts[id] = generateRows(id, rows*iterations)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {

		// Create table for test
		table := db.Table(uuid.New().String(), schema, log.NewNopLogger())

		// Spawn n workers that will insert values into the table
		wg := &sync.WaitGroup{}
		for id := range inserts {
			wg.Add(1)
			go func(id string, tbl *Table, w *sync.WaitGroup) {
				defer w.Done()
				for i := 0; i < iterations; i++ {
					if err := tbl.Insert(inserts[id][i : i+rows]); err != nil {
						fmt.Println("Received error on insert: ", err)
					}
				}
			}(id, table, wg)
		}
		wg.Wait()

		b.StopTimer()

		// Wait for all compaction routines to complete
		table.Sync()

		// Calculate the number of entries in database
		totalrows := int64(0)
		err := table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
			totalrows += ar.NumRows()
			defer ar.Release()

			return nil
		})
		require.NoError(b, err)
		require.Equal(b, int64(rows*iterations*writers), totalrows)

		b.StartTimer()
	}
}

func Test_Table_ReadIsolation(t *testing.T) {
	table := basicTable(t, 2<<12)

	err := table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
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
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(3),
				int64(3),
			},
		}},
	)
	require.NoError(t, err)

	// Perform a new insert that will have a higher tx id
	err = table.Insert(
		[]Row{{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "blarg", Value: "blarg"},
					{Name: "blah", Value: "blah"},
				},
				[]UUID{
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
					{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
				},
				int64(1),
				int64(1),
			},
		}},
	)
	require.NoError(t, err)

	// Now we cheat and reset our tx so that we can perform a read in the past.
	prev := table.db.tx
	table.db.tx = 1

	rows := int64(0)
	err = table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
		rows += ar.NumRows()
		defer ar.Release()

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(3), rows)

	// Now set the tx back to what it was, and perform the same read, we should return all 4 rows
	table.db.tx = prev

	rows = int64(0)
	err = table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
		rows += ar.NumRows()
		defer ar.Release()

		return nil
	})
	require.NoError(t, err)
	require.Equal(t, int64(4), rows)
}

func Test_Table_Sorting(t *testing.T) {
	granuleSize := 2 << 12
	schema1 := NewSchema(
		[]ColumnDefinition{{
			Name:     "labels",
			Type:     StringType,
			Encoding: PlainEncoding,
			Dynamic:  true,
		}, {
			Name:     "stacktrace",
			Type:     List(UUIDType),
			Encoding: PlainEncoding,
		}, {
			Name:     "timestamp",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "value",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}},
		granuleSize,
		"labels",
		"timestamp",
	)

	schema2 := NewSchema(
		[]ColumnDefinition{{
			Name:     "labels",
			Type:     StringType,
			Encoding: PlainEncoding,
			Dynamic:  true,
		}, {
			Name:     "stacktrace",
			Type:     List(UUIDType),
			Encoding: PlainEncoding,
		}, {
			Name:     "timestamp",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}, {
			Name:     "value",
			Type:     Int64Type,
			Encoding: PlainEncoding,
		}},
		granuleSize,
		"timestamp",
		"labels",
	)

	c := New(nil)
	db := c.DB("test")
	table1 := db.Table("test1", schema1, log.NewNopLogger())
	table2 := db.Table("test2", schema2, log.NewNopLogger())

	tables := []*Table{
		table1,
		table2,
	}

	for _, table := range tables {

		err := table.Insert(
			[]Row{{
				Values: []interface{}{
					[]DynamicColumnValue{
						{Name: "label1", Value: "value1"},
						{Name: "label2", Value: "value2"},
					},
					[]UUID{
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
					},
					int64(3),
					int64(3),
				},
			}},
		)
		require.NoError(t, err)

		err = table.Insert(
			[]Row{{
				Values: []interface{}{
					[]DynamicColumnValue{
						{Name: "label1", Value: "value1"},
						{Name: "label2", Value: "value2"},
						{Name: "label3", Value: "value3"},
					},
					[]UUID{
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
					},
					int64(2),
					int64(2),
				},
			}},
		)
		require.NoError(t, err)

		err = table.Insert(
			[]Row{{
				Values: []interface{}{
					[]DynamicColumnValue{
						{Name: "label1", Value: "value1"},
						{Name: "label2", Value: "value2"},
						{Name: "label4", Value: "value4"},
					},
					[]UUID{
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1},
						{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2},
					},
					int64(1),
					int64(1),
				},
			}},
		)
		require.NoError(t, err)
	}

	for i, table := range tables {
		err := table.Iterator(memory.NewGoAllocator(), func(ar arrow.Record) error {
			switch i {
			case 0:
				require.Equal(t, "[3 2 1]", fmt.Sprintf("%v", ar.Column(6)))
			case 1:
				require.Equal(t, "[1 2 3]", fmt.Sprintf("%v", ar.Column(6)))
			}
			defer ar.Release()

			return nil
		})
		require.NoError(t, err)
	}
}

func Test_Granule_Less(t *testing.T) {

	schema := NewSchema(
		[]ColumnDefinition{{
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
		2<<13,
		"labels",
		"timestamp",
	)
	g := &Granule{
		schema: &schema,
		least: Row{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "06e32507-cda3-49db-8093-53a8a4c8da76"},
					{Name: "label2", Value: "value2"},
				},
				int64(6375179957311426905),
				int64(0),
			},
		},
	}
	g1 := &Granule{
		schema: &schema,
		least: Row{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "06e32507-cda3-49db-8093-53a8a4c8da76"},
					{Name: "label2", Value: "value2"},
				},
				int64(8825936717838690748),
				int64(0),
			},
		},
	}

	require.NotEqual(t, g.Less(g1), g1.Less(g))
}
