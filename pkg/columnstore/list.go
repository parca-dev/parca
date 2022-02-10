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
		t:   t,
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
	if err := app.AppendValuesAt(0, v); err != nil {
		return err
	}

	return a.enc.AppendAt(index, enc)
}

func (a *ListAppender) AppendValuesAt(index int, vs interface{}) error {
	values := vs.([]interface{})
	for i, v := range values {
		if err := a.AppendAt(index+i, v); err != nil {
			return err
		}
		index++
	}

	return nil
}

type ListIterator struct {
	Enc       EncodingIterator
	t         *ListType
	err       error
	cur       interface{}
	curIsNull bool
}

func (i *ListIterator) Next() bool {
	next := i.Enc.Next()
	if !next {
		return false
	}

	if i.IsNull() {
		i.curIsNull = true
		return true
	}

	i.curIsNull = false
	enc := i.Enc.Value().(*Plain)
	it := enc.NonSparseIterator()
	listType := i.t
	t := listType.elementType
	v, err := t.NewArrayFromIterator(it)
	if err != nil {
		i.err = err
		return false
	}

	i.cur = v
	return true
}

func (i *ListIterator) IsNull() bool {
	return i.curIsNull
}

func (i *ListIterator) Value() interface{} {
	return i.cur
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

func (t *ListType) NewArrayFromIterator(eit EncodingIterator) (interface{}, error) {
	arr := make([]interface{}, eit.Cardinality())
	it := t.NewIterator(eit)
	i := 0
	for it.Next() {
		if it.IsNull() {
			arr[i] = nil
			i++
			continue
		}

		arr[i] = it.Value()
		i++
	}
	if it.Err() != nil {
		return nil, it.Err()
	}

	return arr, nil
}

func (t *ListType) AppendIteratorToArrow(it EncodingIterator, builder array.Builder) error {
	lb := builder.(*array.ListBuilder)
	vb := lb.ValueBuilder()

	length := it.Cardinality()
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
