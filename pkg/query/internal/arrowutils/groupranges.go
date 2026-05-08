package arrowutils

import (
	"bytes"
	"container/heap"
	"fmt"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

// GetGroupsAndOrderedSetRanges returns a min-heap of group ranges and ordered
// set ranges of the given arrow arrays in that order. For the given input with
// a single array:
// a a c d a b c
// This function will return [2, 3, 4, 5, 6] for the group ranges and [4] for
// the ordered set ranges. A group is a collection of values that are equal and
// an ordered set is a collection of groups that are in increasing order.
// The ranges are determined by iterating over the arrays and comparing the
// current group value for each column. The firstGroup to compare against must
// be provided (it can be initialized to the values at index 0 of each array).
// The last group found is returned.
func GetGroupsAndOrderedSetRanges(
	firstGroup []any, arrs []arrow.Array,
) (*Int64Heap, *Int64Heap, []any, error) {
	if len(firstGroup) != len(arrs) {
		return nil,
			nil,
			nil,
			fmt.Errorf(
				"columns mismatch (%d != %d) when getting group ranges",
				len(firstGroup),
				len(arrs),
			)
	}

	// Safe copy the group in order to not overwrite the input slice values.
	curGroup := make([]any, len(firstGroup))
	for i, v := range firstGroup {
		switch concreteV := v.(type) {
		case []byte:
			curGroup[i] = append([]byte(nil), concreteV...)
		default:
			curGroup[i] = v
		}
	}

	// groupRanges keeps track of the bounds of the group by columns.
	groupRanges := &Int64Heap{}
	heap.Init(groupRanges)
	// setRanges keeps track of the bounds of ordered sets. i.e. in the
	// following slice, (a, a, b, c) is an ordered set of three groups. The
	// second ordered set is (a, e): [a, a, b, c, a, e]
	setRanges := &Int64Heap{}
	heap.Init(setRanges)

	// handleCmpResult is a closure that encapsulates the handling of the result
	// of comparing a current grouping column with a value in a group array.
	handleCmpResult := func(cmp, column int, t arrow.Array, j int) error {
		switch cmp {
		case -1, 1:
			// New group, append range index.
			heap.Push(groupRanges, int64(j))
			if cmp == 1 {
				// New ordered set encountered.
				heap.Push(setRanges, int64(j))
			}

			// And update the current group.
			v := t.GetOneForMarshal(j)
			switch concreteV := v.(type) {
			case []byte:
				// Safe copy, otherwise the value might get overwritten.
				curGroup[column] = append([]byte(nil), concreteV...)
			default:
				curGroup[column] = v
			}
		case 0:
			// Equal to group, do nothing.
		}
		return nil
	}
	for i, arr := range arrs {
		switch t := arr.(type) {
		case *array.Binary:
			for j := 0; j < arr.Len(); j++ {
				var curGroupValue []byte
				if curGroup[i] != nil {
					curGroupValue = curGroup[i].([]byte)
				}
				vIsNull := t.IsNull(j)
				cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
				if !ok {
					cmp = bytes.Compare(curGroupValue, t.Value(j))
				}
				if err := handleCmpResult(cmp, i, t, j); err != nil {
					return nil, nil, nil, err
				}
			}
		case *array.String:
			for j := 0; j < arr.Len(); j++ {
				var curGroupValue *string
				if curGroup[i] != nil {
					g := curGroup[i].(string)
					curGroupValue = &g
				}
				vIsNull := t.IsNull(j)
				cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
				if !ok {
					cmp = strings.Compare(*curGroupValue, t.Value(j))
				}
				if err := handleCmpResult(cmp, i, t, j); err != nil {
					return nil, nil, nil, err
				}
			}
		case *array.Int64:
			for j := 0; j < arr.Len(); j++ {
				var curGroupValue *int64
				if curGroup[i] != nil {
					g := curGroup[i].(int64)
					curGroupValue = &g
				}
				vIsNull := t.IsNull(j)
				cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
				if !ok {
					cmp = compareInt64(*curGroupValue, t.Value(j))
				}
				if err := handleCmpResult(cmp, i, t, j); err != nil {
					return nil, nil, nil, err
				}
			}
		case *array.Boolean:
			for j := 0; j < arr.Len(); j++ {
				var curGroupValue *bool
				if curGroup[i] != nil {
					g := curGroup[i].(bool)
					curGroupValue = &g
				}
				vIsNull := t.IsNull(j)
				cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
				if !ok {
					cmp = compareBools(*curGroupValue, t.Value(j))
				}
				if err := handleCmpResult(cmp, i, t, j); err != nil {
					return nil, nil, nil, err
				}
			}
		case VirtualNullArray:
			for j := 0; j < arr.Len(); j++ {
				cmp, ok := nullComparison(curGroup[i] == nil, true)
				if !ok {
					return nil, nil, nil, fmt.Errorf(
						"null comparison should always be valid but group was: %v", curGroup[i],
					)
				}
				if err := handleCmpResult(cmp, i, t, j); err != nil {
					return nil, nil, nil, err
				}
			}
		case *array.Dictionary:
			switch dict := t.Dictionary().(type) {
			case *array.Binary:
				for j := 0; j < arr.Len(); j++ {
					var curGroupValue []byte
					if curGroup[i] != nil {
						curGroupValue = curGroup[i].([]byte)
					}
					vIsNull := t.IsNull(j)
					cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
					if !ok {
						cmp = bytes.Compare(curGroupValue, dict.Value(t.GetValueIndex(j)))
					}
					if err := handleCmpResult(cmp, i, t, j); err != nil {
						return nil, nil, nil, err
					}
				}

			case *array.String:
				for j := 0; j < arr.Len(); j++ {
					var curGroupValue *string
					if curGroup[i] != nil {
						g := curGroup[i].(string)
						curGroupValue = &g
					}
					vIsNull := t.IsNull(j)
					cmp, ok := nullComparison(curGroupValue == nil, vIsNull)
					if !ok {
						cmp = strings.Compare(*curGroupValue,
							dict.Value(t.GetValueIndex(j)),
						)
					}
					if err := handleCmpResult(cmp, i, t, j); err != nil {
						return nil, nil, nil, err
					}
				}

			default:
				panic(fmt.Sprintf("unsupported dictionary type: %T", dict))
			}
		default:
			panic(fmt.Sprintf("unsupported type: %T", t))
		}
	}
	return groupRanges, setRanges, curGroup, nil
}

// nullComparison encapsulates null comparison. leftNull is whether the current
// Note that this function observes default SQL semantics as well as our own,
// i.e. nulls sort first.
// The comparison integer is returned, as well as whether either value was null.
// If the returned boolean is false, the comparison should be disregarded.
func nullComparison(leftNull, rightNull bool) (int, bool) {
	if !leftNull && !rightNull {
		// Both are not null, this implies that the null comparison should be
		// disregarded.
		return 0, false
	}

	if leftNull {
		if !rightNull {
			return -1, true
		}
		return 0, true
	}
	return 1, true
}

func compareInt64(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

func compareBools(a, b bool) int {
	if a == b {
		return 0
	}

	if !a {
		return -1
	}
	return 1
}

type Int64Heap []int64

func (h Int64Heap) Len() int {
	return len(h)
}

func (h Int64Heap) Less(i, j int) bool {
	return h[i] < h[j]
}

func (h Int64Heap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h *Int64Heap) Push(x any) {
	*h = append(*h, x.(int64))
}

func (h *Int64Heap) Pop() any {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// PopNextNotEqual returns the next least element not equal to compare.
func (h *Int64Heap) PopNextNotEqual(compare int64) (int64, bool) {
	for h.Len() > 0 {
		v := heap.Pop(h).(int64)
		if v != compare {
			return v, true
		}
	}
	return 0, false
}

// Unwrap unwraps the heap into the provided scratch space. The result is a
// slice that will have distinct ints in order. This helps with reiterating over
// the same heap.
func (h *Int64Heap) Unwrap(scratch []int64) []int64 {
	scratch = scratch[:0]
	if h.Len() == 0 {
		return scratch
	}
	cmp := (*h)[0]
	scratch = append(scratch, cmp)
	for h.Len() > 0 {
		if v := heap.Pop(h).(int64); v != cmp {
			scratch = append(scratch, v)
			cmp = v
		}
	}
	return scratch
}
