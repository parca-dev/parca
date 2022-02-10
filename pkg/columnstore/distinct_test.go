package columnstore

import (
	"testing"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/stretchr/testify/require"
)

func TestDistinct(t *testing.T) {
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
					{Name: "label1", Value: "value2"},
					{Name: "label2", Value: "value2"},
					{Name: "label3", Value: "value3"},
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
				int64(3),
				int64(3),
			},
		}},
	)
	require.NoError(t, err)

	tests := map[string]struct {
		columns []ArrowFieldMatcher
		rows    int64
	}{
		"label1": {
			columns: []ArrowFieldMatcher{
				DynamicColumnRef("labels").Column("label1").ArrowFieldMatcher(),
			},
			rows: 3,
		},
		"label2": {
			columns: []ArrowFieldMatcher{
				DynamicColumnRef("labels").Column("label2").ArrowFieldMatcher(),
			},
			rows: 1,
		},
		"label1,label2": {
			columns: []ArrowFieldMatcher{
				DynamicColumnRef("labels").Column("label1").ArrowFieldMatcher(),
				DynamicColumnRef("labels").Column("label2").ArrowFieldMatcher(),
			},
			rows: 3,
		},
		"label1,label2,label3": {
			columns: []ArrowFieldMatcher{
				DynamicColumnRef("labels").Column("label1").ArrowFieldMatcher(),
				DynamicColumnRef("labels").Column("label2").ArrowFieldMatcher(),
				DynamicColumnRef("labels").Column("label3").ArrowFieldMatcher(),
			},
			rows: 3,
		},
		"label1,label2,label4": {
			columns: []ArrowFieldMatcher{
				DynamicColumnRef("labels").Column("label1").ArrowFieldMatcher(),
				DynamicColumnRef("labels").Column("label2").ArrowFieldMatcher(),
				DynamicColumnRef("labels").Column("label4").ArrowFieldMatcher(),
			},
			rows: 3,
		},
	}

	pool := memory.NewGoAllocator()
	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rows := int64(0)
			err = table.Iterator(pool, Distinct(pool, test.columns, func(ar arrow.Record) error {
				rows += ar.NumRows()
				defer ar.Release()

				return nil
			}).Callback)
			require.NoError(t, err)
			require.Equal(t, test.rows, rows)
		})
	}
}
