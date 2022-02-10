package columnstore

import (
	"testing"

	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/stretchr/testify/require"
)

func TestAggregate(t *testing.T) {
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
					{Name: "label1", Value: "value2"},
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
					{Name: "label1", Value: "value3"},
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

	pool := memory.NewGoAllocator()
	agg := NewHashAggregate(
		pool,
		&SumAggregation{},
		StaticColumnRef("value").ArrowFieldMatcher(),
		DynamicColumnRef("labels").Column("label2").ArrowFieldMatcher(),
	)

	err = table.Iterator(pool, agg.Callback)
	require.NoError(t, err)

	r, err := agg.Aggregate()
	require.NoError(t, err)

	for i, col := range r.Columns() {
		require.Equal(t, 1, col.Len(), "unexpected number of values in column %s", r.Schema().Field(i).Name)
	}
	cols := r.Columns()
	require.Equal(t, []int64{6}, cols[len(cols)-1].(*array.Int64).Int64Values())
}
