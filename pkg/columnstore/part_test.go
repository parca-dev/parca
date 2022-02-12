package columnstore

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_PartMerge(t *testing.T) {

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
	}

	p, err := NewPart(0, schema, []Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				int64(1),
				int64(1),
			},
		},
	})
	require.NoError(t, err)

	p1, err := NewPart(0, schema, []Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
					{Name: "label3", Value: "value3"},
				},
				int64(2),
				int64(2),
			},
		},
	})
	require.NoError(t, err)

	p2, err := NewPart(0, schema, []Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
				},
				int64(0),
				int64(0),
			},
		},
	})
	require.NoError(t, err)

	p3, err := NewPart(0, schema, []Row{
		{
			Values: []interface{}{
				[]DynamicColumnValue{
					{Name: "label1", Value: "value1"},
					{Name: "label2", Value: "value2"},
					{Name: "label3", Value: "value3"},
				},
				int64(0),
				int64(2),
			},
		},
	})
	require.NoError(t, err)

	part, err := Merge(0, func(uint64) uint64 { return 0 }, p, p1, p2, p3)
	require.NoError(t, err)
	require.NotNil(t, part)

	it := part.Iterator()
	for it.Next() {
		fmt.Println(it.Values())
	}
}
