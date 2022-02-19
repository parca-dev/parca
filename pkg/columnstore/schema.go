package columnstore

import (
	"fmt"
	"math"

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
	Less(interface{}, interface{}) bool
	ListLess(interface{}, interface{}) bool
	Equal(interface{}, interface{}) bool
	ListEqual(interface{}, interface{}) bool
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
	case UUIDType:
		return UUIDFixedSizeBinaryType
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) Less(a, b interface{}) bool {
	switch t {
	case StringType:
		return a.(string) < b.(string)
	case Int64Type:
		return a.(int64) < b.(int64)
	case UUIDType:
		return CompareUUID(a.(UUID), b.(UUID)) == -1
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) ListLess(a, b interface{}) bool {
	switch t {
	case UUIDType:
		uuids1 := a.([]UUID)
		uuids2 := b.([]UUID)
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
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) Equal(a, b interface{}) bool {
	switch t {
	case StringType:
		return a.(string) == b.(string)
	case Int64Type:
		return a.(int64) == b.(int64)
	case UUIDType:
		return CompareUUID(a.(UUID), b.(UUID)) == 0
	default:
		panic("unsupported data type")
	}
}

func (t PrimitiveType) ListEqual(a, b interface{}) bool {
	switch t {
	case StringType:
		as := a.([]string)
		bs := b.([]string)
		if len(as) != len(bs) {
			return false
		}

		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}

		return true
	case Int64Type:
		as := a.([]int64)
		bs := b.([]int64)
		if len(as) != len(bs) {
			return false
		}

		for i := range as {
			if as[i] != bs[i] {
				return false
			}
		}

		return true
	case UUIDType:
		as := a.([]UUID)
		bs := b.([]UUID)
		if len(as) != len(bs) {
			return false
		}

		for i := range as {
			if CompareUUID(as[i], bs[i]) != 0 {
				return false
			}
		}

		return true
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
	columns     []ColumnDefinition
	orderedBy   []string
	granuleSize int

	// ordered are the indicies to the Columns array that are to be ordered. If OrderedBy is empty this will contain all indicies of Columns
	ordered []int
}

// NewSchema returns a new schema
func NewSchema(cols []ColumnDefinition, granuleSize int, orderedBy ...string) Schema {
	s := Schema{
		columns:     cols,
		granuleSize: granuleSize,
		orderedBy:   orderedBy,

		ordered: []int{},
	}

	// initialize the ordered indicies
	switch len(orderedBy) {
	case 0: // if no ordering is provided, use all columns in the order provided
		for i := range cols {
			s.ordered = append(s.ordered, i)
		}
	default:
		for _, row := range orderedBy {
			for i, col := range cols {
				// If the column matches the ordering, add it to the ordered index
				if col.Name == row {
					s.ordered = append(s.ordered, i)
				}
			}
		}
	}

	return s
}

func (s Schema) ColumnDefinition(name string) (ColumnDefinition, bool) {
	for _, c := range s.columns {
		if !c.Dynamic && c.Name == name {
			return c, true
		}
	}
	return ColumnDefinition{}, false
}

func (s Schema) Equals(other Schema) bool {
	if len(s.columns) != len(other.columns) {
		return false
	}
	for i, c := range s.columns {
		if c != other.columns[i] {
			return false
		}
	}

	for i, c := range s.orderedBy {
		if c != other.orderedBy[i] {
			return false
		}
	}

	return s.granuleSize == other.granuleSize
}

// ToArrow returns the schema in arrow schema format
func (s Schema) ToArrow(dynamicColNames [][]string, dynamicColCounts []int) *arrow.Schema {
	fields := make([]arrow.Field, 0, len(s.columns))
	for i, c := range s.columns {

		switch c.Dynamic {
		case true: // split out the dynamic columns into multiple arrow cols
			for j := 0; j < dynamicColCounts[i]; j++ {
				fields = append(fields, arrow.Field{
					Name: c.Name + "." + dynamicColNames[i][j],
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

// RowLessThan returns true if the first row is less than the second row.
func (s Schema) RowLessThan(a, b []interface{}) bool {
	if b == nil { // in the 0 case always return true
		return true
	}

	for _, k := range s.ordered {
		vi := a[k]
		vj := b[k]
		less := s.columns[k].Type.Less
		equal := s.columns[k].Type.Equal
		switch s.columns[k].Dynamic {
		case true:
			dci := vi.([]DynamicColumnValue)
			dcj := vj.([]DynamicColumnValue)
			end := int(math.Min(float64(len(dci)), float64(len(dcj))))
			for l := 0; l < end; l++ {
				if dci[l].Name != dcj[l].Name {
					return dci[l].Name < dcj[l].Name
				}

				if !equal(dci[l].Value, dcj[l].Value) {
					return less(dci[l].Value, dcj[l].Value)
				}
			}

			// The dynamic columns are equal unless their lengths aren't the same
			switch {
			case len(dci) < len(dcj):
				return true
			case len(dci) > len(dcj):
				return false
			}
		default:
			if !equal(vi, vj) {
				return less(vi, vj)
			}
		}
	}

	return false
}
