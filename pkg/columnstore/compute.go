package columnstore

import (
	"errors"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
)

var (
	ErrUnsupportedBinaryOperation = errors.New("unsupported binary operation")
)

func BinaryScalarOperation(left arrow.Array, right ScalarExpression, operator Operator) (*Bitmap, error) {
	leftType := left.DataType()
	switch leftType {
	case UUIDFixedSizeBinaryType:
		rightUUID := right.(UUIDScalarExpression).Value

		switch operator {
		case Equal:
			return UUIDArrayScalarEqual(left.(*array.FixedSizeBinary), rightUUID)
		case NotEqual:
			return UUIDArrayScalarNotEqual(left.(*array.FixedSizeBinary), rightUUID)
		default:
			panic("something terrible has happened, this should have errored previously during validation")
		}
	case arrow.BinaryTypes.String:
		rightString := right.(StringScalarExpression).Value
		switch operator {
		case Equal:
			return StringArrayScalarEqual(left.(*array.String), rightString)
		case NotEqual:
			return StringArrayScalarNotEqual(left.(*array.String), rightString)
		default:
			panic("something terrible has happened, this should have errored previously during validation")
		}
	case arrow.PrimitiveTypes.Int64:
		rightInt64 := right.(Int64ScalarExpression).Value
		switch operator {
		case Equal:
			return Int64ArrayScalarEqual(left.(*array.Int64), rightInt64)
		case NotEqual:
			return Int64ArrayScalarNotEqual(left.(*array.Int64), rightInt64)
		case LessThan:
			return Int64ArrayScalarLessThan(left.(*array.Int64), rightInt64)
		case LessThanOrEqual:
			return Int64ArrayScalarLessThanOrEqual(left.(*array.Int64), rightInt64)
		case GreaterThan:
			return Int64ArrayScalarGreaterThan(left.(*array.Int64), rightInt64)
		case GreaterThanOrEqual:
			return Int64ArrayScalarGreaterThanOrEqual(left.(*array.Int64), rightInt64)
		default:
			panic("something terrible has happened, this should have errored previously during validation")
		}
	}

	switch leftType.(type) {
	case *arrow.ListType:
		panic("TODO: list comparisons unimplemented")
	}

	return nil, ErrUnsupportedBinaryOperation
}
