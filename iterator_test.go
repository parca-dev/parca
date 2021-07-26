package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileTreeIterator(t *testing.T) {
	pt := &ProfileTree{}
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt.Insert(makeSample(1, []uint64{3, 3, 1}))

	it := pt.Iterator()

	res := []uint64{}
	for {
		if !it.HasMore() {
			break
		}

		if it.NextChild() {
			res = append(res, it.At().LocationID())
			it.StepInto()
			continue
		}
		it.StepUp()
	}

	require.Equal(t, []uint64{0, 1, 2, 3, 4, 5, 3, 3}, res)
}
