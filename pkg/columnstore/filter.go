package columnstore

import (
	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

func Filter(pool memory.Allocator, filterExpr BooleanExpression, callback func(r arrow.Record) error) func(r arrow.Record) error {
	return func(r arrow.Record) error {
		filtered, empty, err := filter(pool, filterExpr, r)
		if err != nil {
			return err
		}
		if empty {
			return nil
		}

		defer filtered.Release()
		return callback(filtered)
	}
}

func filter(pool memory.Allocator, filterExpr BooleanExpression, ar arrow.Record) (arrow.Record, bool, error) {
	bitmap, err := filterExpr.Eval(ar)
	if err != nil {
		return nil, true, err
	}

	if bitmap.IsEmpty() {
		return nil, true, nil
	}

	indicesToKeep := bitmap.ToArray()
	ranges := buildIndexRanges(indicesToKeep)

	totalRows := int64(0)
	recordRanges := make([]arrow.Record, len(ranges))
	for j, r := range ranges {
		recordRanges[j] = ar.NewSlice(int64(r.Start), int64(r.End))
		totalRows += int64(r.End - r.Start)
	}

	cols := make([]arrow.Array, ar.NumCols())
	numRanges := len(recordRanges)
	for i := range cols {
		colRanges := make([]arrow.Array, 0, numRanges)
		for _, rr := range recordRanges {
			colRanges = append(colRanges, rr.Column(i))
		}

		cols[i], err = array.Concatenate(colRanges, pool)
		if err != nil {
			return nil, true, err
		}
	}

	return array.NewRecord(ar.Schema(), cols, totalRows), false, nil
}

type IndexRange struct {
	Start uint32
	End   uint32
}

// buildIndexRanges returns a set of continguous index ranges from the given indicies
// ex: [1,2,7,8,9] would return [{Start:1, End:2},{Start:7,End:9}]
func buildIndexRanges(indices []uint32) []IndexRange {
	ranges := []IndexRange{}

	cur := IndexRange{
		Start: indices[0],
		End:   indices[0] + 1,
	}

	for _, i := range indices[1:] {
		if i == cur.End {
			cur.End++
		} else {
			ranges = append(ranges, cur)
			cur = IndexRange{
				Start: i,
				End:   i + 1,
			}
		}
	}

	ranges = append(ranges, cur)
	return ranges
}
