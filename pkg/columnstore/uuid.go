package columnstore

import (
	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

type UUID [16]byte

type UUIDAppender struct {
	enc Encoding
}

func (a *UUIDAppender) AppendAt(index int, v interface{}) error {
	return a.AppendUUIDAt(index, v.(UUID))
}

func (a *UUIDAppender) AppendValuesAt(index int, vs []interface{}) error {
	for i, v := range vs {
		if err := a.AppendUUIDAt(index+i, v.(UUID)); err != nil {
			return err
		}
	}
	return nil
}

func (a *UUIDAppender) AppendUUIDValuesAt(index int, vs []UUID) error {
	for i, v := range vs {
		if err := a.AppendUUIDAt(index+i, v); err != nil {
			return err
		}
	}
	return nil
}

func (a *UUIDAppender) AppendUUIDAt(index int, v UUID) error {
	return a.enc.AppendAt(index, v)
}

type UUIDIterator struct {
	Enc EncodingIterator
}

func (i *UUIDIterator) Next() bool {
	return i.Enc.Next()
}

func (i *UUIDIterator) IsNull() bool {
	return i.Enc.IsNull()
}

func (i *UUIDIterator) Value() interface{} {
	return i.Enc.Value()
}

func (i *UUIDIterator) UUIDValue() UUID {
	return i.Value().(UUID)
}

func (i *UUIDIterator) Err() error {
	return i.Enc.Err()
}

var UUIDFixedSizeBinaryType = &arrow.FixedSizeBinaryType{
	ByteWidth: 16,
}

func NewUUIDArrowArrayFromIterator(pool memory.Allocator, eit EncodingIterator) (array.Interface, error) {
	builder := array.NewFixedSizeBinaryBuilder(pool, UUIDFixedSizeBinaryType)
	defer builder.Release()

	err := AppendUUIDIteratorToArrow(eit, builder)
	if err != nil {
		return nil, err
	}
	return builder.NewFixedSizeBinaryArray(), nil
}

func AppendUUIDIteratorToArrow(eit EncodingIterator, builder array.Builder) error {
	b := builder.(*array.FixedSizeBinaryBuilder)

	length := eit.Cardinality()
	it := &UUIDIterator{Enc: eit}
	i := 0
	for it.Next() {
		if i == length {
			break
		}
		if it.IsNull() {
			b.AppendNull()
			continue
		}

		uuid := it.UUIDValue()
		b.Append(uuid[:])
		i++
	}
	if it.Err() != nil {
		return it.Err()
	}

	return nil
}
