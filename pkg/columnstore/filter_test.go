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
	err = table.Iterator(pool, Filter(pool, StaticColumnRef("timestamp").GreaterThanOrEqual(Int64Literal(2)), func(ar arrow.Record) error {
		fmt.Println(ar)
		defer ar.Release()

		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("------")

	err = table.Iterator(pool, Filter(pool, DynamicColumnRef("labels").Column("label4").Equal(StringLiteral("value4")), func(ar arrow.Record) error {
		fmt.Println(ar)
		defer ar.Release()

		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}

	fmt.Println("-------")

	err = table.Iterator(pool, Filter(pool, DynamicColumnRef("labels").Column("label1").GreaterThanOrEqual(StringLiteral("value1")), func(ar arrow.Record) error {
		fmt.Println(ar)
		defer ar.Release()

		return nil
	}))
	if err != nil {
		t.Fatal(err)
	}
}

func Test_BuildIndexRanges(t *testing.T) {
	tests := map[string]struct {
		indicies []uint32
		expected []IndexRange
	}{
		"no consecutive": {
			indicies: []uint32{1, 3, 5, 7, 9},
			expected: []IndexRange{{Start: 1, End: 2}, {Start: 3, End: 4}, {Start: 5, End: 6}, {Start: 7, End: 8}, {Start: 9, End: 10}},
		},
		"only consecutive": {
			indicies: []uint32{1, 2},
			expected: []IndexRange{{Start: 1, End: 3}},
		},
		"only 1": {
			indicies: []uint32{1},
			expected: []IndexRange{{Start: 1, End: 2}},
		},
		"multiple": {
			indicies: []uint32{1, 2, 7, 8, 9},
			expected: []IndexRange{{Start: 1, End: 3}, {Start: 7, End: 10}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, test.expected, buildIndexRanges(test.indicies))
		})
	}
}
