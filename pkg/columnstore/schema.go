package columnstore

import (
	"fmt"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
)

type DataType interface {
	String() string
	NewAppender(enc Encoding) Appender
	NewIterator(it EncodingIterator) Iterator
	NewArrowArrayFromIterator(memory.Allocator, EncodingIterator) (array.Interface, error)
	NewArrayFromIterator(EncodingIterator) (interface{}, error)
	AppendIteratorToArrow(EncodingIterator, array.Builder) error
	ArrowDataType() arrow.DataType
}

type PrimitiveType int

const (
	StringType PrimitiveType = iota
	Int64Type
	UUIDType
)

func (t PrimitiveType) String() string {
	switch t {
	case StringType:
		return "string"
	case Int64Type:
		return "int64"
	case UUIDType:
		return "uuid"
	default:
		return "unknown"
	}
}

func (t PrimitiveType) NewAppender(enc Encoding) Appender {
	switch t {
	case StringType:
		return &StringAppender{enc: enc}
	case Int64Type:
		return &Int64Appender{enc: enc}
	case UUIDType:
		return &UUIDAppender{enc: enc}
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) NewIterator(it EncodingIterator) Iterator {
	switch t {
	case StringType:
		return &StringIterator{Enc: it}
	case Int64Type:
		return &Int64Iterator{Enc: it}
	case UUIDType:
		return &UUIDIterator{Enc: it}
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) NewArrowArrayFromIterator(pool memory.Allocator, it EncodingIterator) (array.Interface, error) {
	switch t {
	case StringType:
		return NewStringArrowArrayFromIterator(pool, it)
	case Int64Type:
		return NewInt64ArrowArrayFromIterator(pool, it)
	case UUIDType:
		return NewUUIDArrowArrayFromIterator(pool, it)
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) NewArrayFromIterator(it EncodingIterator) (interface{}, error) {
	switch t {
	case StringType:
		return NewStringArrayFromIterator(it)
	case Int64Type:
		return NewInt64ArrayFromIterator(it)
	case UUIDType:
		return NewUUIDArrayFromIterator(it)
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) AppendIteratorToArrow(it EncodingIterator, builder array.Builder) error {
	switch t {
	case StringType:
		return AppendStringIteratorToArrow(it, builder)
	case Int64Type:
		return AppendInt64IteratorToArrow(it, builder)
	case UUIDType:
		return AppendUUIDIteratorToArrow(it, builder)
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) ArrowDataType() arrow.DataType {
	switch t {
	case StringType:
		return &arrow.StringType{}
	case Int64Type:
		return &arrow.Int64Type{}
	default:
		panic("unsupported data type")
	}
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

func (s Schema) Equals(other Schema) bool {
	if len(s.Columns) != len(other.Columns) {
		return false
	}
	for i, c := range s.Columns {
		if c != other.Columns[i] {
			return false
		}
	}

	for i, c := range s.OrderedBy {
		if c != other.OrderedBy[i] {
			return false
		}
	}

	return s.GranuleSize == other.GranuleSize
}

// ToArrow returns the schema in arrow schema format
func (s Schema) ToArrow(dynamicColNames [][]string, dynamicColCounts []int) *arrow.Schema {

	fields := make([]arrow.Field, 0, len(s.Columns))
	for i, c := range s.Columns {

		switch c.Dynamic {
		case true: // split out the dynamic columns into multiple arrow cols
			for j := 0; j < dynamicColCounts[i]; j++ {
				fields = append(fields, arrow.Field{
					Name: dynamicColNames[i][j],
					Type: c.Type.ArrowDataType(),
				})
			}
		default:
			fields = append(fields, arrow.Field{
				Name: c.Name,
				Type: c.Type.ArrowDataType(),
			})
		}

	}

	return arrow.NewSchema(fields, nil)
}
