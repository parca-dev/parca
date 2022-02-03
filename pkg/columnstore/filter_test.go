package columnstore

import (
	"fmt"
	"testing"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/stretchr/testify/require"
)

func TestBuildIndexRanges(t *testing.T) {
	arr := []uint32{4, 6, 7, 8, 10}
	ranges := buildIndexRanges(arr)

	require.Equal(t, 3, len(ranges))
	require.Equal(t, []IndexRange{
		{Start: 4, End: 5},
		{Start: 6, End: 9},
		{Start: 10, End: 11},
	}, ranges)
}

func TestFilter(t *testing.T) {
	table := basicTable(t, 2^12)

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

	pool := memory.NewGoAllocator()
	err = table.Iterator(pool, Filter(pool, DynamicColumnRef("labels").Column("label4").Equal(StringLiteral("value4")), func(ar arrow.Record) error {
		fmt.Println(ar)
		defer ar.Release()

		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
}
