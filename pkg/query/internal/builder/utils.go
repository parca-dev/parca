package builder

import (
	"fmt"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

func NewBuilder(mem memory.Allocator, t arrow.DataType) ColumnBuilder {
	switch t := t.(type) {
	case *arrow.BinaryType:
		return NewOptBinaryBuilder(arrow.BinaryTypes.Binary)
	case *arrow.Int64Type:
		return NewOptInt64Builder(arrow.PrimitiveTypes.Int64)
	case *arrow.ListType:
		return NewListBuilder(mem, t.Elem())
	case *arrow.BooleanType:
		return NewOptBooleanBuilder(arrow.FixedWidthTypes.Boolean)
	default:
		return array.NewBuilder(mem, t)
	}
}

func RollbackPrevious(cb ColumnBuilder) error {
	switch b := cb.(type) {
	case *OptBinaryBuilder:
		b.ResetToLength(b.Len() - 1)
	case *OptInt64Builder:
		b.ResetToLength(b.Len() - 1)
	case *OptBooleanBuilder:
		b.ResetToLength(b.Len() - 1)
	case *array.Int64Builder:
		b.Resize(b.Len() - 1)

	case *array.StringBuilder:
		b.Resize(b.Len() - 1)
	case *array.BinaryBuilder:
		b.Resize(b.Len() - 1)
	case *array.FixedSizeBinaryBuilder:
		b.Resize(b.Len() - 1)
	case *array.BooleanBuilder:
		b.Resize(b.Len() - 1)
	case *array.BinaryDictionaryBuilder:
		b.Resize(b.Len() - 1)
	default:
		return fmt.Errorf("unsupported type for RollbackPrevious %T", b)
	}
	return nil
}

func AppendValue(cb ColumnBuilder, arr arrow.Array, i int) error {
	if arr == nil || arr.IsNull(i) {
		cb.AppendNull()
		return nil
	}

	switch b := cb.(type) {
	case *OptBinaryBuilder:
		return b.Append(arr.(*array.Binary).Value(i))
	case *OptInt64Builder:
		b.Append(arr.(*array.Int64).Value(i))
	case *OptBooleanBuilder:
		b.AppendSingle(arr.(*array.Boolean).Value(i))
	case *array.Int64Builder:
		b.Append(arr.(*array.Int64).Value(i))
	case *array.Int32Builder:
		b.Append(arr.(*array.Int32).Value(i))
	case *array.Float64Builder:
		b.Append(arr.(*array.Float64).Value(i))
	case *array.Uint64Builder:
		b.Append(arr.(*array.Uint64).Value(i))
	case *array.StringBuilder:
		b.Append(arr.(*array.String).Value(i))
	case *array.BinaryBuilder:
		b.Append(arr.(*array.Binary).Value(i))
	case *array.FixedSizeBinaryBuilder:
		b.Append(arr.(*array.FixedSizeBinary).Value(i))
	case *array.BooleanBuilder:
		b.Append(arr.(*array.Boolean).Value(i))
	case *array.StructBuilder:
		arrStruct := arr.(*array.Struct)

		b.Append(true)
		for j := 0; j < b.NumField(); j++ {
			if err := AppendValue(b.FieldBuilder(j), arrStruct.Field(j), i); err != nil {
				return fmt.Errorf("failed to append struct field: %w", err)
			}
		}
	case *array.BinaryDictionaryBuilder:
		switch a := arr.(type) {
		case *array.Dictionary:
			idx := a.GetValueIndex(i)
			switch dict := a.Dictionary().(type) {
			case *array.Binary:
				if idx < 0 || idx >= dict.Len() {
					b.AppendNull()
					return nil
				}
				if err := b.Append(dict.Value(idx)); err != nil {
					return err
				}
			case *array.String:
				if idx < 0 || idx >= dict.Len() {
					b.AppendNull()
					return nil
				}
				if err := b.AppendString(dict.Value(idx)); err != nil {
					return err
				}
			default:
				return fmt.Errorf("dictionary type %T unsupported", dict)
			}
		default:
			return fmt.Errorf("non-dictionary array %T provided for dictionary builder", a)
		}
	case *array.ListBuilder:
		return buildList(b.ValueBuilder(), b, arr, i)
	case *ListBuilder:
		return buildList(b.ValueBuilder(), b, arr, i)
	default:
		return fmt.Errorf("unsupported type for arrow append %T", b)
	}
	return nil
}

type ListLikeBuilder interface {
	Append(bool)
}

func buildList(vb any, b ListLikeBuilder, arr arrow.Array, i int) error {
	list := arr.(*array.List)
	start, end := list.ValueOffsets(i)

	data := list.ListValues().Data()
	if start > int64(data.Len()) || start > end || data.Offset()+int(start) > data.Offset()+data.Len() {
		return fmt.Errorf("invalid data range: start=%d end=%d for list with %v", start, end, list.Offsets())
	}

	values := array.NewSlice(list.ListValues(), start, end)
	defer values.Release()

	switch v := values.(type) {
	case *array.Int64:
		int64Builder := vb.(*OptInt64Builder)
		b.Append(true)
		for j := 0; j < v.Len(); j++ {
			int64Builder.Append(v.Value(j))
		}
	case *array.Dictionary:
		switch dict := v.Dictionary().(type) {
		case *array.Binary:
			b.Append(true)
			for j := 0; j < v.Len(); j++ {
				switch bldr := vb.(type) {
				case *array.BinaryDictionaryBuilder:
					if err := bldr.Append(dict.Value(v.GetValueIndex(j))); err != nil {
						return err
					}
				default:
					return fmt.Errorf("unknown value builder type %T", bldr)
				}
			}
		}
	case *array.Struct:
		structBuilder, ok := vb.(*array.StructBuilder)
		if !ok {
			return fmt.Errorf("unsupported type for ListLikeBuilder: %T", vb)
		}

		b.Append(true)
		for j := 0; j < v.Len(); j++ {
			structBuilder.Append(true)
			for k := 0; k < v.NumField(); k++ {
				if err := AppendValue(structBuilder.FieldBuilder(k), v.Field(k), j); err != nil {
					return err
				}
			}
		}
	default:
		return fmt.Errorf("unsupported type for List builder %T", v)
	}

	return nil
}

// TODO(asubiotto): This function doesn't handle NULLs in the case of optimized
// builders.
func AppendArray(cb ColumnBuilder, arr arrow.Array) error {
	switch b := cb.(type) {
	case *OptBinaryBuilder:
		v := arr.(*array.Binary)
		offsets := v.ValueOffsets()
		return b.AppendData(v.ValueBytes(), *(*[]uint32)(unsafe.Pointer(&offsets)))
	case *OptInt64Builder:
		b.AppendData(arr.(*array.Int64).Int64Values())
	default:
		// TODO(asubiotto): Handle OptBooleanBuilder. It needs some way to
		// append data.
		for i := 0; i < arr.Len(); i++ {
			// This is an interface conversion on each call, but we should care
			// more about porting our uses of arrow builders to optimized
			// builders for exactly these use cases.
			if err := AppendValue(cb, arr, i); err != nil {
				return err
			}
		}
	}
	return nil
}

func AppendGoValue(cb ColumnBuilder, v any) error {
	if v == nil {
		cb.AppendNull()
		return nil
	}

	switch b := cb.(type) {
	case *OptBinaryBuilder:
		return b.Append(v.([]byte))
	case *OptBooleanBuilder:
		b.AppendSingle(v.(bool))
	case *OptFloat64Builder:
		b.Append(v.(float64))
	case *OptInt32Builder:
		b.Append(v.(int32))
	case *OptInt64Builder:
		b.Append(v.(int64))
	case *array.Int64Builder:
		b.Append(v.(int64))
	case *array.Int32Builder:
		b.Append(v.(int32))
	case *array.StringBuilder:
		b.Append(v.(string))
	case *array.BinaryBuilder:
		b.Append(v.([]byte))
	case *array.FixedSizeBinaryBuilder:
		b.Append(v.([]byte))
	case *array.BooleanBuilder:
		b.Append(v.(bool))
	case *array.BinaryDictionaryBuilder:
		switch e := v.(type) {
		case string:
			return b.Append([]byte(e))
		case []byte:
			return b.Append(e)
		default:
			return fmt.Errorf("unsupported type %T for append go value %T", e, b)
		}
	default:
		return fmt.Errorf("unsupported type for append go value %T", b)
	}
	return nil
}
