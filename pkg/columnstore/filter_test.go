package columnstore

import (
	"regexp"
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
	if err != nil {
		t.Fatal(err)
	}

	reg, err := regexp.Compile("value.")
	require.NoError(t, err)

	nomatch, err := regexp.Compile("values.*")
	require.NoError(t, err)

	tests := map[string]struct {
		filterExpr BooleanExpression
		rows       int64
	}{
		">= int64": {
			filterExpr: StaticColumnRef("timestamp").GreaterThanOrEqual(Int64Literal(2)),
			rows:       2,
		},
		"== string": {
			filterExpr: DynamicColumnRef("labels").Column("label4").Equal(StringLiteral("value4")),
			rows:       1,
		},
		"regexp simple match": {
			filterExpr: DynamicColumnRef("labels").Column("label1").RegexMatch(&RegexMatcher{regex: reg}),
			rows:       3,
		},
		"regexp no match": {
			filterExpr: DynamicColumnRef("labels").Column("label1").RegexMatch(&RegexMatcher{regex: nomatch}),
			rows:       0,
		},
	}

	pool := memory.NewGoAllocator()
	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rows := int64(0)
			err = table.Iterator(pool, Filter(pool, test.filterExpr, func(ar arrow.Record) error {
				rows += ar.NumRows()
				defer ar.Release()

				return nil
			}))
			require.NoError(t, err)
			require.Equal(t, test.rows, rows)
		})
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
