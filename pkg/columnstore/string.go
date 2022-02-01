package columnstore

import (
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

type StringAppender struct {
	enc Encoding
}

func (a *StringAppender) AppendAt(index int, v interface{}) error {
	return a.AppendStringAt(index, v.(string))
}

func (a *StringAppender) AppendValuesAt(index int, vs interface{}) error {
	return a.AppendStringValuesAt(index, vs.([]string))
}

func (a *StringAppender) AppendStringValuesAt(index int, vs []string) error {
	for i, v := range vs {
		if err := a.AppendStringAt(index+i, v); err != nil {
			return err
		}
	}
	return nil
}

func (a *StringAppender) AppendStringAt(index int, v string) error {
	return a.enc.AppendAt(index, v)
}

type StringIterator struct {
	Enc EncodingIterator
}

func (i *StringIterator) Next() bool {
	return i.Enc.Next()
}

func (i *StringIterator) IsNull() bool {
	return i.Enc.IsNull()
}

func (i *StringIterator) Value() interface{} {
	return i.Enc.Value()
}

func (i *StringIterator) StringValue() string {
	return i.Value().(string)
}

func (i *StringIterator) Err() error {
	return i.Enc.Err()
}

func NewStringArrowArrayFromIterator(pool memory.Allocator, eit EncodingIterator) (array.Interface, error) {
	builder := array.NewStringBuilder(pool)
	defer builder.Release()

	err := AppendStringIteratorToArrow(eit, builder)
	if err != nil {
		return nil, err
	}
	return builder.NewStringArray(), nil
}

func NewStringArrayFromIterator(eit EncodingIterator) (interface{}, error) {
	arr := make([]string, eit.Cardinality())
	sit := &StringIterator{Enc: eit}
	i := 0
	for sit.Next() {
		if sit.IsNull() {
			i++
			continue
		}
		arr[i] = sit.StringValue()
		i++
	}
	if sit.Err() != nil {
		return nil, sit.Err()
	}

	return arr, nil
}

func AppendStringIteratorToArrow(eit EncodingIterator, builder array.Builder) error {
	b := builder.(*array.StringBuilder)

	length := eit.Cardinality()
	it := &StringIterator{Enc: eit}
	i := 0
	for it.Next() {
		if i == length {
			break
		}
		if it.IsNull() {
			b.AppendNull()
			continue
		}
		b.Append(it.StringValue())
		i++
	}
	if it.Err() != nil {
		return it.Err()
	}

	return nil
}
