package columnstore

import (
	"errors"
	"fmt"
	"strings"
)

type EncodingType int

const (
	PlainEncoding EncodingType = iota
)

func (t EncodingType) New() Encoding {
	return NewEncoding(t)
}

func (t EncodingType) String() string {
	switch t {
	case PlainEncoding:
		return "PlainEncoding"
	default:
		return "unknown"
	}
}

type Encoding interface {
	AppendAt(index int, v interface{}) error
	Iterator(maxIterations int) EncodingIterator
	String() string
}

type EncodingIterator interface {
	Next() bool
	Value() interface{}
	IsNull() bool
	Err() error
	Cardinality() int
}

func NewEncoding(t EncodingType) Encoding {
	switch t {
	case PlainEncoding:
		return NewPlain()
	default:
		panic("unknown encoding type")
	}
}

type Plain struct {
	values []interface{}
}

func NewPlain() *Plain {
	return &Plain{
		values: make([]interface{}, 0, 10), // TODO arbitrary number is arbitrary, this should be optimized using a pool of plain encoding objects to re-use rather than pre-allocating.
	}
}

func (c *Plain) String() string {
	s := "[ "
	for i := 0; i < len(c.values); i++ {
		s += fmt.Sprint(c.values[i])
		s += ","
	}
	s = strings.TrimSuffix(s, ",")

	s += " ]"

	return s
}

var (
	ErrOutOfOrderInsert = errors.New("cannot insert out of order")
)

func (c *Plain) AppendAt(index int, v interface{}) error {
	if index < 0 {
		return errors.New("index out of range")
	}
	if index < len(c.values) {
		return fmt.Errorf("inserting at index %d, but already have %d values: %w", index, len(c.values), ErrOutOfOrderInsert)
	}

	if index > len(c.values) {
		// This could be further optimized by noting the first index where the
		// value is a non-null value as columns are expected to be very sparse,
		// but this decision should be backed by data.
		for i := len(c.values); i < index; i++ {
			c.values = append(c.values, nil)
		}
	}

	c.values = append(c.values, v)
	return nil
}

func (c *Plain) Iterator(maxIterations int) EncodingIterator {
	return &PlainSparseIterator{
		values:        c.values,
		index:         -1,
		maxIterations: maxIterations,
	}
}

type PlainSparseIterator struct {
	values        []interface{}
	index         int
	maxIterations int
}

func (i *PlainSparseIterator) Cardinality() int {
	return i.maxIterations
}

func (i *PlainSparseIterator) Next() bool {
	if i.maxIterations == 0 {
		return false
	}

	i.index++
	i.maxIterations--
	return true
}

func (i *PlainSparseIterator) IsNull() bool {
	if i.index >= len(i.values) {
		return true
	}

	return i.values[i.index] == nil
}

func (i *PlainSparseIterator) Value() interface{} {
	if i.index >= len(i.values) {
		// We allow going over the index to allow for sparse data. The caller
		// is responsible for controlling how many values are read.
		return nil
	}

	return i.values[i.index]
}

func (i *PlainSparseIterator) Err() error {
	return nil
}

func (c *Plain) NonSparseIterator() EncodingIterator {
	return &PlainSparseIterator{
		values:        c.values,
		index:         -1,
		maxIterations: len(c.values),
	}
}
