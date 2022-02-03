package columnstore

import (
	"errors"

	"github.com/RoaringBitmap/roaring"
	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
)

var (
	ErrColumnNotFound = errors.New("column not found")
)

type Bitmap = roaring.Bitmap

func NewBitmap() *Bitmap {
	return roaring.New()
}

type Expression interface {
	DataType(s Schema) (arrow.DataType, error)
}

type ArrayExpression interface {
	Expression
	GetData(r arrow.Record) (arrow.Array, bool, error)
}

type ScalarExpression interface {
	Expression
	IsScalar() bool
}

type BooleanExpression interface {
	Expression
	Eval(r arrow.Record) (*Bitmap, error)
}

type dynamicColumnRef struct {
	Name string
}

func DynamicColumnRef(name string) dynamicColumnRef {
	return dynamicColumnRef{Name: name}
}

type dynamicColumnInstanceRef struct {
	r    dynamicColumnRef
	Name string
}

func (d dynamicColumnRef) Column(name string) ArrayRef {
	return ArrayRef{array: dynamicColumnInstanceRef{
		r:    d,
		Name: name,
	}}
}

var (
	ErrColumnRefNotDynamic = errors.New("column ref is not dynamic, expected it to be")
)

func (d dynamicColumnInstanceRef) DataType(s Schema) (arrow.DataType, error) {
	def, found := s.ColumnDefinition(d.r.Name)
	if !found {
		return nil, ErrColumnNotFound
	}

	if !def.Dynamic {
		return nil, ErrInvalidBinaryOperation
	}

	return def.Type.ArrowDataType(), nil
}

func (d dynamicColumnInstanceRef) GetData(r arrow.Record) (arrow.Array, bool, error) {
	fields := r.Schema().FieldIndices(d.r.Name + "." + d.Name)
	if len(fields) == 0 {
		return nil, false, nil
	}

	if len(fields) != 1 {
		return nil, false, ErrUnexpectedNumberOfFields
	}

	return r.Column(fields[0]), true, nil
}

type staticColumnRef struct {
	Name string
}

func StaticColumnRef(name string) ArrayRef {
	return ArrayRef{array: staticColumnRef{Name: name}}
}

var (
	ErrColumnRefDynamic = errors.New("column ref is dynamic, expected it not to be")
)

func (c staticColumnRef) DataType(s Schema) (arrow.DataType, error) {
	def, found := s.ColumnDefinition(c.Name)
	if !found {
		return nil, ErrColumnNotFound
	}

	if def.Dynamic {
		return nil, ErrColumnRefNotDynamic
	}

	return def.Type.ArrowDataType(), nil
}

var (
	ErrUnexpectedNumberOfFields = errors.New("unexpected number of fields")
)

func (c staticColumnRef) GetData(r arrow.Record) (arrow.Array, bool, error) {
	fields := r.Schema().FieldIndices(c.Name)
	if len(fields) != 1 {
		return nil, false, ErrUnexpectedNumberOfFields
	}

	return r.Column(fields[0]), true, nil
}

type ArrayRef struct {
	array ArrayExpression
}

func (a ArrayRef) Equal(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: Equal}
}

func (a ArrayRef) NotEqual(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: NotEqual}
}

func (a ArrayRef) GreaterThan(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: GreaterThan}
}

func (a ArrayRef) GreaterThanOrEqual(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: GreaterThanOrEqual}
}

func (a ArrayRef) LessThan(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: LessThan}
}

func (a ArrayRef) LessThanOrEqual(expr ScalarExpression) BinaryScalarExpression {
	return BinaryScalarExpression{Left: a.array, Right: expr, Operator: LessThanOrEqual}
}

func (a ArrayRef) RegexMatch(regex *RegexMatcher) RegexScalarExpression {
	return RegexScalarExpression{left: a.array, right: regex}
}

func (a ArrayRef) RegexNotMatch(regex *RegexMatcher) RegexScalarExpression {
	return RegexScalarExpression{left: a.array, right: regex, notMatch: true}
}

type RegexScalarExpression struct {
	left     ArrayExpression
	notMatch bool
	right    *RegexMatcher
}

var (
	ErrInvalidRegexLeftOperand = errors.New("left side of regex expression must be string")
)

func (e RegexScalarExpression) DataType(s Schema) (arrow.DataType, error) {
	leftDataType, err := e.left.DataType(s)
	if err != nil {
		return nil, err
	}

	if leftDataType != arrow.BinaryTypes.String {
		return nil, ErrInvalidRegexLeftOperand
	}

	return arrow.FixedWidthTypes.Boolean, nil
}

func (e RegexScalarExpression) Eval(r arrow.Record) (*Bitmap, error) {
	leftData, exists, err := e.left.GetData(r)
	if err != nil {
		return nil, err
	}

	// TODO: This needs a bunch of test cases to validate edge cases like non
	// existant columns or null values.
	if !exists {
		res := NewBitmap()
		if e.notMatch {
			for i := uint32(0); i < uint32(r.NumRows()); i++ {
				res.Add(i)
			}
			return res, nil
		}
		return res, nil
	}

	if e.notMatch {
		return StringArrayScalarRegexNotMatch(leftData.(*array.String), e.right)
	}

	return StringArrayScalarRegexMatch(leftData.(*array.String), e.right)
}

type StringScalarExpression struct {
	Value string
}

func StringLiteral(value string) StringScalarExpression {
	return StringScalarExpression{Value: value}
}

func (s StringScalarExpression) DataType(_ Schema) (arrow.DataType, error) {
	return arrow.BinaryTypes.String, nil
}

func (s StringScalarExpression) IsScalar() bool {
	return true
}

type Int64ScalarExpression struct {
	Value int64
}

func Int64Literal(value int64) Int64ScalarExpression {
	return Int64ScalarExpression{Value: value}
}

func (e Int64ScalarExpression) DataType(_ Schema) (arrow.DataType, error) {
	return arrow.PrimitiveTypes.Int64, nil
}

func (e Int64ScalarExpression) IsScalar() bool {
	return true
}

type UUIDScalarExpression struct {
	Value UUID
}

func UUIDLiteral(value UUID) UUIDScalarExpression {
	return UUIDScalarExpression{Value: value}
}

func (e UUIDScalarExpression) DataType(_ Schema) (arrow.DataType, error) {
	return UUIDFixedSizeBinaryType, nil
}

func (e UUIDScalarExpression) IsScalar() bool {
	return true
}

type LogicalAndExpression struct {
	expressions []BooleanExpression
}

func And(rhs, lhs BooleanExpression, expr ...BooleanExpression) LogicalAndExpression {
	return LogicalAndExpression{
		expressions: append([]BooleanExpression{rhs, lhs}, expr...),
	}
}

var (
	ErrInvalidBinaryOperation = errors.New("logical expression must be built of expressions of type boolean")
	ErrInvalidBinaryScalar    = errors.New("logical expression must be built of expressions that are not scalars")
)

func (e LogicalAndExpression) Eval(r arrow.Record) (*Bitmap, error) {
	res, err := e.expressions[0].Eval(r)
	if err != nil {
		return nil, err
	}

	for _, expr := range e.expressions[1:] {
		// Early return because if the bitmap is empty every consecutive AND
		// will result in an empty bitmap.
		if res.IsEmpty() {
			return res, nil
		}

		r, err := expr.Eval(r)
		if err != nil {
			return nil, err
		}

		res.And(r)
	}

	return res, nil
}

type Operator int

const (
	Equal Operator = iota
	NotEqual
	LessThan
	LessThanOrEqual
	GreaterThan
	GreaterThanOrEqual
	ContainsOneOf
	StartsWithList
)

type BinaryScalarExpression struct {
	Left     ArrayExpression
	Right    ScalarExpression
	Operator Operator
}

var (
	ErrBinaryExpressionOnlyScalars = errors.New("binary expression must be built of expressions that are not only scalars")
)

func (e BinaryScalarExpression) DataType(s Schema) (arrow.DataType, error) {
	if !e.Right.IsScalar() {
		return nil, ErrBinaryExpressionOnlyScalars
	}

	leftDataType, err := e.Left.DataType(s)
	if err != nil {
		return nil, err
	}

	rightDataType, err := e.Right.DataType(s)
	if err != nil {
		return nil, err
	}

	return binaryOperationResult(leftDataType, rightDataType, e.Operator)
}

func (e BinaryScalarExpression) Eval(r arrow.Record) (*Bitmap, error) {
	leftData, exists, err := e.Left.GetData(r)
	if err != nil {
		return nil, err
	}

	// TODO: This needs a bunch of test cases to validate edge cases like non
	// existant columns or null values. I'm pretty sure this is completely
	// wrong and needs per operation, per type specific behavior.
	if !exists {
		res := NewBitmap()
		for i := uint32(0); i < uint32(r.NumRows()); i++ {
			res.Add(i)
		}
		return res, nil
	}

	return BinaryScalarOperation(leftData, e.Right, e.Operator)
}

var (
	ErrUnsupportedLeftExpressionType = errors.New("unknown left expression type")
	ErrRightExpressionNotUUID        = errors.New("right expression is not of UUID type")
	ErrUUIDExpressionNotSupported    = errors.New("UUID expression not supported, only Equal and NotEqual are supported")
	ErrRightExpressionNotString      = errors.New("right expression is not of string type")
	ErrStringExpressionNotSupported  = errors.New("string expression not supported, only Equal, NotEqual, AnchoredRegexMatch and AnchoredRegexNotMatch are supported")
	ErrRightExpressionNotInt64       = errors.New("right expression is not of int64 type")
	ErrInt64ExpressionNotSupported   = errors.New("int64 expression not supported, only Equal, NotEqual, LessThan, LessThanOrEqual, GreaterThan, GreaterThanOrEqual are supported")
	ErrRightExpressionNotList        = errors.New("right expression is not of list type or does not contain the same element type")
	ErrListExpressionNotSupported    = errors.New("list expression not supported, only Equal, NotEqual, ContainsOneOf and StartsWithList are supported")
)

func binaryOperationResult(leftDataType, rightDataType arrow.DataType, operator Operator) (arrow.DataType, error) {
	switch leftDataType {
	case UUIDFixedSizeBinaryType:
		if rightDataType != UUIDFixedSizeBinaryType {
			return nil, ErrRightExpressionNotUUID
		}
		switch operator {
		case Equal, NotEqual:
			return arrow.FixedWidthTypes.Boolean, nil
		default:
			return nil, ErrUUIDExpressionNotSupported
		}
	case arrow.BinaryTypes.String:
		if rightDataType != arrow.BinaryTypes.String {
			return nil, ErrRightExpressionNotString
		}
		switch operator {
		case Equal, NotEqual:
			return arrow.FixedWidthTypes.Boolean, nil
		default:
			return nil, ErrUUIDExpressionNotSupported
		}
	case arrow.PrimitiveTypes.Int64:
		if rightDataType != arrow.PrimitiveTypes.Int64 {
			return nil, ErrRightExpressionNotInt64
		}
		switch operator {
		case Equal, NotEqual, LessThan, LessThanOrEqual, GreaterThan, GreaterThanOrEqual:
			return arrow.FixedWidthTypes.Boolean, nil
		default:
			return nil, ErrInt64ExpressionNotSupported
		}
	}

	switch leftDataType.(type) {
	case *arrow.ListType:
		if rightDataType.Fingerprint() != leftDataType.Fingerprint() {
			return nil, ErrRightExpressionNotList
		}
		switch operator {
		case Equal, NotEqual, ContainsOneOf, StartsWithList:
			return arrow.FixedWidthTypes.Boolean, nil
		default:
			return nil, ErrListExpressionNotSupported
		}
	}

	return nil, ErrUnsupportedLeftExpressionType
}
