package columnstore

import (
	"bytes"
	"fmt"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

type UUID [16]byte

type UUIDAppender struct {
	enc Encoding
}

func CompareUUID(a, b UUID) int {
	ab := [16]byte(a)
	bb := [16]byte(b)
	return bytes.Compare(ab[:], bb[:])
}

func (a *UUIDAppender) AppendAt(index int, v interface{}) error {
	return a.AppendUUIDAt(index, v.(UUID))
}

func (a *UUIDAppender) AppendValuesAt(index int, vs interface{}) error {
	return a.AppendUUIDValuesAt(index, vs.([]UUID))
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
	err := a.enc.AppendAt(index, v)
	if err != nil {
		return fmt.Errorf("append uuid at %d: %w", index, err)
	}

	return nil
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

func NewUUIDArrayFromIterator(eit EncodingIterator) (interface{}, error) {
	arr := make([]UUID, eit.Cardinality())
	uit := &UUIDIterator{Enc: eit}
	i := 0
	for uit.Next() {
		if uit.IsNull() {
			i++
			continue
		}
		arr[i] = uit.UUIDValue()
		i++
	}
	if uit.Err() != nil {
		return nil, uit.Err()
	}

	return arr, nil
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

func UUIDArrayScalarEqual(left *array.FixedSizeBinary, right UUID) (*Bitmap, error) {
	rightUUID := right[:]
	res := NewBitmap()
	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			continue
		}
		if bytes.Compare(left.Value(i), rightUUID) == 0 {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func UUIDArrayScalarNotEqual(left *array.FixedSizeBinary, right UUID) (*Bitmap, error) {
	rightUUID := right[:]
	res := NewBitmap()
	for i := 0; i < left.Len(); i++ {
		if left.IsNull(i) {
			res.Add(uint32(i))
			continue
		}
		if bytes.Compare(left.Value(i), rightUUID) != 0 {
			res.Add(uint32(i))
		}
	}

	return res, nil
}

func UUIDsLess(uuids1, uuids2 []UUID) bool {
	uuids1Len := len(uuids1)
	uuids2Len := len(uuids2)

	k := 0
	for {
		switch {
		case k == uuids1Len && k == uuids2Len:
			// This means we've looked at all the elements and they've all been equal.
			return false
		case k >= uuids1Len && k <= uuids2Len:
			// This means the UUIDs are identical up until this point, but uuids1 is ending, and shorter slices are "smaller" than longer ones.
			return true
		case k <= uuids1Len && k >= uuids2Len:
			// This means the UUIDs are identical up until this point, but uuids2 is ending, and shorter slices are "lower" than longer ones.
			return false
		case CompareUUID(uuids1[k], uuids2[k]) == -1:
			return true
		case CompareUUID(uuids1[k], uuids2[k]) == 1:
			return false
		default:
			// This means the slices of UUIDs are identical up until this point. So advance to the next.
			k++
		}
	}
}
