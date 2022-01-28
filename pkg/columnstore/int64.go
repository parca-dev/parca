package columnstore

import (
	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/memory"
)

type Int64Appender struct {
	enc Encoding
}

func (a *Int64Appender) AppendAt(index int, v interface{}) error {
	return a.AppendInt64At(index, v.(int64))
}

func (a *Int64Appender) AppendValuesAt(index int, vs []interface{}) error {
	for i, v := range vs {
		if err := a.AppendInt64At(index+i, v.(int64)); err != nil {
			return err
		}
	}

	return nil
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
