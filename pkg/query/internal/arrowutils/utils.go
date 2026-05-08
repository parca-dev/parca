package arrowutils

import (
	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// ArrayConcatenator is an object that helps callers keep track of a slice of
// arrays and concatenate them into a single one when needed. This is more
// efficient and memory safe than using a builder.
type ArrayConcatenator struct {
	arrs []arrow.Array
}

func (c *ArrayConcatenator) Add(arr arrow.Array) {
	c.arrs = append(c.arrs, arr)
}

func (c *ArrayConcatenator) NewArray(mem memory.Allocator) (arrow.Array, error) {
	arr, err := array.Concatenate(c.arrs, mem)
	if err != nil {
		return nil, err
	}
	c.arrs = c.arrs[:0]
	return arr, err
}

func (c *ArrayConcatenator) Len() int {
	return len(c.arrs)
}

func (c *ArrayConcatenator) Release() {
	for _, arr := range c.arrs {
		arr.Release()
	}
	c.arrs = c.arrs[:0]
}
