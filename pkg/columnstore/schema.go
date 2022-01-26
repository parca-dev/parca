package columnstore

import (
	"fmt"

	"github.com/apache/arrow/go/arrow/array"
	"github.com/apache/arrow/go/arrow/memory"
)

type DataType int

const (
	StringType DataType = iota
	Int64Type
)

func (t DataType) String() string {
	switch t {
	case StringType:
		return "string"
	case Int64Type:
		return "int64"
	default:
		return "unknown"
	}
}

func (t DataType) NewAppender(app Appender) Appender {
	switch t {
	case StringType:
		return &StringAppender{app: app}
	case Int64Type:
		return &Int64Appender{app: app}
	default:
		panic("unsupported data type")
	}
}

func (t DataType) NewIterator(it EncodingIterator) Iterator {
	switch t {
	case StringType:
		return &StringIterator{Enc: it}
	case Int64Type:
		return &Int64Iterator{Enc: it}
	default:
		panic("unsupported data type")
	}
}

func (t DataType) NewArrowArrayFromIterator(pool memory.Allocator, it EncodingIterator) (array.Interface, error) {
	length := it.Cardinality()
	switch t {
	case StringType:
		all := make([]string, length)
		notNulls := make([]bool, length)
		it := &StringIterator{Enc: it}
		i := 0
		for it.Next() {
			if i == length {
				break
			}
			if !it.IsNull() {
				notNulls[i] = true
				all[i] = it.StringValue()
			}
			i++
		}
		if it.Err() != nil {
			return nil, it.Err()
		}

		builder := array.NewStringBuilder(pool)
		defer builder.Release()

		builder.AppendValues(all, notNulls)
		return builder.NewStringArray(), nil
	case Int64Type:
		all := make([]int64, length)
		notNulls := make([]bool, length)
		it := &Int64Iterator{Enc: it}
		i := 0
		for it.Next() {
			if i == length {
				break
			}
			if !it.IsNull() {
				notNulls[i] = true
				all[i] = it.Int64Value()
			}
			i++
		}
		if it.Err() != nil {
			return nil, it.Err()
		}

		builder := array.NewInt64Builder(pool)
		defer builder.Release()

		builder.AppendValues(all, notNulls)
		return builder.NewInt64Array(), nil
	default:
		panic("unsupported data type")
	}
}

type StringAppender struct {
	app Appender
}

func (a *StringAppender) AppendAt(index int, v interface{}) error {
	return a.AppendStringAt(index, v.(string))
}

func (a *StringAppender) AppendStringAt(index int, v string) error {
	return a.app.AppendAt(index, v)
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

type Int64Appender struct {
	app Appender
}

func (a *Int64Appender) AppendAt(index int, v interface{}) error {
	return a.AppendInt64At(index, v.(int64))
}

func (a *Int64Appender) AppendInt64At(index int, v int64) error {
	return a.app.AppendAt(index, v)
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

type ColumnDefinition struct {
	Name     string
	Type     DataType
	Encoding EncodingType
	Dynamic  bool

	// This doesn't do anything in the in-memory representation yet but it is
	// passed onto the Apache Arrow frames.
	Nullable bool
}

func (d ColumnDefinition) String() string {
	return fmt.Sprintf("%q (Type: %s, Encoding: %s, Dynamic: %t)", d.Name, d.Type.String(), d.Encoding.String(), d.Dynamic)
}

type Schema struct {
	Columns     []ColumnDefinition
	OrderedBy   []string
	GranuleSize int
}
