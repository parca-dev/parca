package columnstore

import (
	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

func List(elementType DataType) *ListType {
	return &ListType{elementType: elementType}
}

type ListType struct {
	elementType DataType
}

func (t *ListType) ArrowDataType() arrow.DataType {
	return arrow.ListOf(t.elementType.ArrowDataType())
}

func (t *ListType) NewAppender(enc Encoding) Appender {
	return &ListAppender{
		t:   t,
		enc: enc,
	}
}

func (t *ListType) NewIterator(it EncodingIterator) Iterator {
	return &ListIterator{
		Enc: it,
	}
}

func (t *ListType) String() string {
	return "list<" + t.elementType.String() + ">"
}

type ListAppender struct {
	enc Encoding
	t   *ListType
}

func (a *ListAppender) AppendAt(index int, v interface{}) error {
	enc := NewPlain()
	app := a.t.elementType.NewAppender(enc)
	vs := v.([]interface{})
	for i, v := range vs {
		if err := app.AppendAt(i, v); err != nil {
			return err
		}
	}

	return a.enc.AppendAt(index, enc)
}

func (a *ListAppender) AppendValuesAt(index int, vs []interface{}) error {
	for i, v := range vs {
		if err := a.AppendAt(index+i, v); err != nil {
			return err
		}
		index++
	}

	return nil
}

type ListIterator struct {
	Enc EncodingIterator
}

func (i *ListIterator) Next() bool {
	return i.Enc.Next()
}

func (i *ListIterator) IsNull() bool {
	return i.Enc.IsNull()
}

func (i *ListIterator) Value() interface{} {
	return i.Enc.Value()
}

func (i *ListIterator) Err() error {
	return i.Enc.Err()
}

func (t *ListType) NewArrowArrayFromIterator(pool memory.Allocator, eit EncodingIterator) (array.Interface, error) {
	builder := array.NewListBuilder(pool, t.elementType.ArrowDataType())
	defer builder.Release()

	err := t.AppendIteratorToArrow(eit, builder)
	if err != nil {
		return nil, err
	}

	return builder.NewListArray(), nil
}

func (t *ListType) AppendIteratorToArrow(eit EncodingIterator, builder array.Builder) error {
	lb := builder.(*array.ListBuilder)
	vb := lb.ValueBuilder()

	length := eit.Cardinality()
	it := &ListIterator{Enc: eit}
	i := 0
	for it.Next() {
		if i == length {
			return nil
		}

		if it.IsNull() {
			lb.AppendNull()
			i++
			continue
		}

		lb.Append(true)
		enc := it.Value().(*Plain)
		err := t.elementType.AppendIteratorToArrow(enc.NonSparseIterator(), vb)
		if err != nil {
			return err
		}
		i++
	}

	return nil
}
