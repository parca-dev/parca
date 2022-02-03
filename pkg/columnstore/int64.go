package columnstore

import (
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

type Int64Appender struct {
	enc Encoding
}

func (a *Int64Appender) AppendAt(index int, v interface{}) error {
	return a.AppendInt64At(index, v.(int64))
}

func (a *Int64Appender) AppendValuesAt(index int, vs interface{}) error {
	return a.AppendInt64ValuesAt(index, vs.([]int64))
}

func (a *Int64Appender) AppendInt64ValuesAt(index int, vs []int64) error {
	for i, v := range vs {
		if err := a.AppendInt64At(index+i, v); err != nil {
			return err
		}
	}

	return nil
}

func (a *Int64Appender) AppendInt64At(index int, v int64) error {
	return a.enc.AppendAt(index, v)
}

type Int64Iterator struct {
	Enc EncodingIterator
}

func (i *Int64Iterator) Next() bool {
	return i.Enc.Next()
}

func (i *Int64Iterator) IsNull() bool {
	return i.Enc.IsNull()
}

func (i *Int64Iterator) Value() interface{} {
	return i.Enc.Value()
}

func (i *Int64Iterator) Int64Value() int64 {
	return i.Value().(int64)
}

func (i *Int64Iterator) Err() error {
	return i.Enc.Err()
}

func NewInt64ArrowArrayFromIterator(pool memory.Allocator, eit EncodingIterator) (array.Interface, error) {
	builder := array.NewInt64Builder(pool)
	defer builder.Release()

	err := AppendInt64IteratorToArrow(eit, builder)
	if err != nil {
		return nil, err
	}
	return builder.NewInt64Array(), nil
}

func NewInt64ArrayFromIterator(eit EncodingIterator) (interface{}, error) {
	arr := make([]int64, eit.Cardinality())
	iit := &Int64Iterator{Enc: eit}
	i := 0
	for iit.Next() {
		if iit.IsNull() {
			i++
			continue
		}
		arr[i] = iit.Int64Value()
		i++
	}
	if iit.Err() != nil {
		return nil, iit.Err()
	}

	return arr, nil
}

func AppendInt64IteratorToArrow(eit EncodingIterator, builder array.Builder) error {
	b := builder.(*array.Int64Builder)

	length := eit.Cardinality()
	it := &Int64Iterator{Enc: eit}
	i := 0
	for it.Next() {
		if i == length {
			break
		}
		if it.IsNull() {
			b.AppendNull()
			continue
		}
		b.Append(it.Int64Value())
		i++
	}
	if it.Err() != nil {
		return it.Err()
	}

	return nil
}

func Int64ArrayScalarEqual(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if left.Value(i) == right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func Int64ArrayScalarNotEqual(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			res.Add(uint32(i))
			continue
		}
		if left.Value(i) != right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func Int64ArrayScalarLessThan(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if left.Value(i) < right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func Int64ArrayScalarLessThanOrEqual(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if left.Value(i) <= right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func Int64ArrayScalarGreaterThan(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if left.Value(i) > right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func Int64ArrayScalarGreaterThanOrEqual(left *array.Int64, right int64) (*Bitmap, error) {
	res := NewBitmap()

	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if left.Value(i) >= right {
			res.Add(uint32(i))
		}
	}

	return res, nil
}
