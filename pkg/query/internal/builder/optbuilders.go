package builder

import (
	"fmt"
	"math"
	"slices"
	"sync/atomic"
	"unsafe"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// ColumnBuilder is a subset of the array.Builder interface implemented by the
// optimized builders in this file.
type ColumnBuilder interface {
	Retain()
	Release()
	Len() int
	AppendNull()
	Reserve(int)
	NewArray() arrow.Array
}

// OptimizedBuilder is a set of FrostDB specific builder methods.
type OptimizedBuilder interface {
	ColumnBuilder
	AppendNulls(int)
	ResetToLength(int)
	RepeatLastValue(int) error
	IsNull(i int) bool
	IsValid(i int) bool
	SetNull(i int)
}

type builderBase struct {
	dtype          arrow.DataType
	refCount       int64
	length         int
	validityBitmap []byte
}

func (b *builderBase) reset() {
	b.length = 0
	b.validityBitmap = b.validityBitmap[:0]
}

func (b *builderBase) Retain() {
	atomic.AddInt64(&b.refCount, 1)
}

func (b *builderBase) releaseInternal() {
	b.length = 0
	b.validityBitmap = nil
}

func (b *builderBase) Release() {
	atomic.AddInt64(&b.refCount, -1)
	b.releaseInternal()
}

// Len returns the number of elements in the array builder.
func (b *builderBase) Len() int {
	return b.length
}

func (b *builderBase) Reserve(int) {}

// AppendNulls appends n null values to the array being built. This is specific
// to distinct optimizations in FrostDB.
func (b *builderBase) AppendNulls(n int) {
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length+n)
	bitutil.SetBitsTo(b.validityBitmap, int64(b.length), int64(n), false)
	b.length += n
}

// SetNull is setting the value at the index i to null.
func (b *builderBase) SetNull(i int) {
	bitutil.ClearBit(b.validityBitmap, i)
}

func (b *builderBase) IsValid(n int) bool {
	return bitutil.BitIsSet(b.validityBitmap, n)
}

// appendValid does the opposite of appendNulls.
func (b *builderBase) appendValid(n int) {
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length+n)
	bitutil.SetBitsTo(b.validityBitmap, int64(b.length), int64(n), true)
	b.length += n
}

func (b *builderBase) IsNull(n int) bool {
	return bitutil.BitIsNotSet(b.validityBitmap, n)
}

func resizeBitmap(bitmap []byte, valuesToRepresent int) []byte {
	bytesNeeded := int(bitutil.BytesForBits(int64(valuesToRepresent)))
	if cap(bitmap) < bytesNeeded {
		existingBitmap := bitmap
		bitmap = make([]byte, bitutil.NextPowerOf2(bytesNeeded))
		copy(bitmap, existingBitmap)
	}
	return bitmap[:bytesNeeded]
}

var (
	_ OptimizedBuilder = (*OptBinaryBuilder)(nil)
	_ OptimizedBuilder = (*OptInt64Builder)(nil)
	_ OptimizedBuilder = (*OptBooleanBuilder)(nil)
	_ OptimizedBuilder = (*OptFloat64Builder)(nil)
)

// OptBinaryBuilder is an optimized array.BinaryBuilder.
type OptBinaryBuilder struct {
	builderBase

	data []byte
	// offsets are offsets into data. The ith value is
	// data[offsets[i]:offsets[i+1]]. Note however, that during normal operation,
	// len(data) is never appended to the slice until the next value is added,
	// i.e. the last offset is never closed until the offsets slice is appended
	// to or returned to the caller.
	offsets []uint32
}

func NewOptBinaryBuilder(dtype arrow.BinaryDataType) *OptBinaryBuilder {
	b := &OptBinaryBuilder{}
	b.dtype = dtype
	return b
}

// Release decreases the reference count by 1.
// When the reference count goes to zero, the memory is freed.
// Release may be called simultaneously from multiple goroutines.
func (b *OptBinaryBuilder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		b.data = nil
		b.offsets = nil
		b.releaseInternal()
	}
}

// AppendNull adds a new null value to the array being built. This is slow,
// don't use it.
func (b *OptBinaryBuilder) AppendNull() {
	b.offsets = append(b.offsets, uint32(len(b.data)))
	b.builderBase.AppendNulls(1)
}

// AppendEmptyValue adds a new empty byte slice to the array being built.
func (b *OptBinaryBuilder) AppendEmptyValue() {
	b.offsets = append(b.offsets, uint32(len(b.data)))
	// Don't append any data, just close the offset for an empty slice
	b.appendValid(1)
}

// AppendNulls appends n null values to the array being built. This is specific
// to distinct optimizations in FrostDB.
func (b *OptBinaryBuilder) AppendNulls(n int) {
	for i := 0; i < n; i++ {
		b.offsets = append(b.offsets, uint32(len(b.data)))
	}
	b.builderBase.AppendNulls(n)
}

// NewArray creates a new array from the memory buffers used
// by the builder and resets the Builder so it can be used to build
// a new array.
func (b *OptBinaryBuilder) NewArray() arrow.Array {
	b.offsets = append(b.offsets, uint32(len(b.data)))
	offsetsAsBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(b.offsets))), len(b.offsets)*arrow.Uint32SizeBytes)
	data := array.NewData(
		b.dtype,
		b.length,
		[]*memory.Buffer{
			memory.NewBufferBytes(b.validityBitmap),
			memory.NewBufferBytes(offsetsAsBytes),
			memory.NewBufferBytes(b.data),
		},
		nil,
		b.length-bitutil.CountSetBits(b.validityBitmap, 0, b.length),
		0,
	)
	b.reset()
	b.offsets = b.offsets[:0]
	b.data = nil

	return array.NewBinaryData(data)
}

var ErrMaxSizeReached = fmt.Errorf("max size reached")

// AppendData appends a flat slice of bytes to the builder, with an accompanying
// slice of offsets. This data is considered to be non-null.
func (b *OptBinaryBuilder) AppendData(data []byte, offsets []uint32) error {
	if len(b.data)+len(data) > math.MaxInt32 { // NOTE: we check against a max int32 here (instead of the uint32 that we're using for offsets) because the arror binary arrays use int32s.
		return ErrMaxSizeReached
	}

	// Trim the last offset since we want this last range to be "open".
	offsets = offsets[:len(offsets)-1]

	offsetConversion := uint32(len(b.data))
	b.data = append(b.data, data...)
	startOffset := len(b.offsets)
	b.offsets = append(b.offsets, offsets...)
	for curOffset := startOffset; curOffset < len(b.offsets); curOffset++ {
		b.offsets[curOffset] += offsetConversion
	}

	b.length += len(offsets)
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBitsTo(b.validityBitmap, int64(startOffset), int64(len(offsets)), true)
	return nil
}

func (b *OptBinaryBuilder) Append(v []byte) error {
	if len(b.data)+len(v) > math.MaxInt32 {
		return ErrMaxSizeReached
	}
	b.offsets = append(b.offsets, uint32(len(b.data)))
	b.data = append(b.data, v...)
	b.length++
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBit(b.validityBitmap, b.length-1)
	return nil
}

// RepeatLastValue is specific to distinct optimizations in FrostDB.
func (b *OptBinaryBuilder) RepeatLastValue(n int) error {
	if bitutil.BitIsNotSet(b.validityBitmap, b.length-1) {
		// Last value is null.
		b.AppendNulls(n)
		return nil
	}

	lastValue := b.data[b.offsets[len(b.offsets)-1]:]
	if len(b.data)+(len(lastValue)*n) > math.MaxInt32 {
		return ErrMaxSizeReached
	}
	for i := 0; i < n; i++ {
		b.offsets = append(b.offsets, uint32(len(b.data)))
		b.data = append(b.data, lastValue...)
	}
	b.appendValid(n)
	return nil
}

// ResetToLength is specific to distinct optimizations in FrostDB.
func (b *OptBinaryBuilder) ResetToLength(n int) {
	if n == b.length {
		return
	}

	b.length = n
	b.data = b.data[:b.offsets[n]]
	b.offsets = b.offsets[:n]
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}

func (b *OptBinaryBuilder) Value(i int) []byte {
	if i == b.length-1 { // last value
		return b.data[b.offsets[i]:]
	}
	return b.data[b.offsets[i]:b.offsets[i+1]]
}

type OptInt64Builder struct {
	builderBase

	data []int64
}

func NewOptInt64Builder(dtype arrow.DataType) *OptInt64Builder {
	b := &OptInt64Builder{}
	b.dtype = dtype
	return b
}

func (b *OptInt64Builder) resizeData(neededLength int) {
	if cap(b.data) < neededLength {
		oldData := b.data
		b.data = make([]int64, bitutil.NextPowerOf2(neededLength))
		copy(b.data, oldData)
	}
	b.data = b.data[:neededLength]
}

func (b *OptInt64Builder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		b.data = nil
		b.releaseInternal()
	}
}

func (b *OptInt64Builder) AppendNull() {
	b.AppendNulls(1)
}

// AppendEmptyValue adds a new zero value (0) to the array being built.
func (b *OptInt64Builder) AppendEmptyValue() {
	b.Append(0)
}

func (b *OptInt64Builder) AppendNulls(n int) {
	b.resizeData(b.length + n)
	b.builderBase.AppendNulls(n)
}

func (b *OptInt64Builder) NewArray() arrow.Array {
	dataAsBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(b.data))), len(b.data)*arrow.Int64SizeBytes)
	data := array.NewData(
		b.dtype,
		b.length,
		[]*memory.Buffer{
			memory.NewBufferBytes(b.validityBitmap),
			memory.NewBufferBytes(dataAsBytes),
		},
		nil,
		b.length-bitutil.CountSetBits(b.validityBitmap, 0, b.length),
		0,
	)
	b.reset()
	b.data = nil
	return array.NewInt64Data(data)
}

// AppendData appends a slice of int64s to the builder. This data is considered
// to be non-null.
func (b *OptInt64Builder) AppendData(data []int64) {
	oldLength := b.length
	b.data = append(b.data, data...)
	b.length += len(data)
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBitsTo(b.validityBitmap, int64(oldLength), int64(len(data)), true)
}

func (b *OptInt64Builder) Append(v int64) {
	b.data = append(b.data, v)
	b.length++
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBit(b.validityBitmap, b.length-1)
}

func (b *OptInt64Builder) Set(i int, v int64) {
	b.data[i] = v
}

func (b *OptInt64Builder) Add(i int, v int64) {
	b.data[i] += v
}

// Value returns the ith value of the builder.
func (b *OptInt64Builder) Value(i int) int64 {
	return b.data[i]
}

func (b *OptInt64Builder) RepeatLastValue(n int) error {
	if bitutil.BitIsNotSet(b.validityBitmap, b.length-1) {
		b.AppendNulls(n)
		return nil
	}

	lastValue := b.data[b.length-1]
	b.resizeData(b.length + n)
	for i := b.length; i < b.length+n; i++ {
		b.data[i] = lastValue
	}
	b.appendValid(n)
	return nil
}

// ResetToLength is specific to distinct optimizations in FrostDB.
func (b *OptInt64Builder) ResetToLength(n int) {
	if n == b.length {
		return
	}

	b.length = n
	b.data = b.data[:n]
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}

type OptBooleanBuilder struct {
	builderBase
	data []byte
}

func NewOptBooleanBuilder(dtype arrow.DataType) *OptBooleanBuilder {
	b := &OptBooleanBuilder{}
	b.dtype = dtype
	return b
}

func (b *OptBooleanBuilder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		b.data = nil
		b.releaseInternal()
	}
}

func (b *OptBooleanBuilder) AppendNull() {
	b.AppendNulls(1)
}

// AppendEmptyValue adds a new zero value (false) to the array being built.
func (b *OptBooleanBuilder) AppendEmptyValue() {
	b.AppendSingle(false)
}

func (b *OptBooleanBuilder) AppendNulls(n int) {
	v := b.length + n
	b.data = resizeBitmap(b.data, v)
	b.validityBitmap = resizeBitmap(b.validityBitmap, v)

	for i := 0; i < n; i++ {
		bitutil.SetBitTo(b.data, b.length, false)
		bitutil.SetBitTo(b.validityBitmap, b.length, false)
		b.length++
	}
}

func (b *OptBooleanBuilder) NewArray() arrow.Array {
	data := array.NewData(
		b.dtype,
		b.length,
		[]*memory.Buffer{
			memory.NewBufferBytes(b.validityBitmap),
			memory.NewBufferBytes(b.data),
		},
		nil,
		b.length-bitutil.CountSetBits(b.validityBitmap, 0, b.length),
		0,
	)
	b.reset()
	b.data = nil
	array := array.NewBooleanData(data)
	return array
}

func (b *OptBooleanBuilder) Append(data []byte, valid int) {
	n := b.length + valid
	b.data = resizeBitmap(b.data, n)
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)

	// TODO: This isn't ideal setting bits 1 by 1, when we could copy in all the bits
	for i := 0; i < valid; i++ {
		bitutil.SetBitTo(b.data, b.length, bitutil.BitIsSet(data, i))
		bitutil.SetBitTo(b.validityBitmap, b.length, true)
		b.length++
	}
}

func (b *OptBooleanBuilder) Set(i int, v bool) {
	bitutil.SetBitTo(b.data, i, v)
}

func (b *OptBooleanBuilder) Value(i int) bool {
	return bitutil.BitIsSet(b.data, i)
}

func (b *OptBooleanBuilder) AppendData(_ []byte) {
	panic("do not use AppendData for opt boolean builder, use Append instead")
}

func (b *OptBooleanBuilder) AppendSingle(v bool) {
	b.length++
	b.data = resizeBitmap(b.data, b.length)
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBitTo(b.data, b.length-1, v)
	bitutil.SetBit(b.validityBitmap, b.length-1)
}

func (b *OptBooleanBuilder) RepeatLastValue(n int) error {
	if bitutil.BitIsNotSet(b.validityBitmap, b.length-1) {
		b.AppendNulls(n)
		return nil
	}

	lastValue := bitutil.BitIsSet(b.data, b.length-1)
	b.data = resizeBitmap(b.data, b.length+n)
	bitutil.SetBitsTo(b.data, int64(b.length), int64(n), lastValue)
	b.appendValid(n)
	return nil
}

// ResetToLength is specific to distinct optimizations in FrostDB.
func (b *OptBooleanBuilder) ResetToLength(n int) {
	if n == b.length {
		return
	}

	b.length = n
	b.data = resizeBitmap(b.data, n)
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}

type OptInt32Builder struct {
	builderBase

	data []int32
}

func NewOptInt32Builder(dtype arrow.DataType) *OptInt32Builder {
	b := &OptInt32Builder{}
	b.dtype = dtype
	return b
}

func (b *OptInt32Builder) resizeData(neededLength int) {
	if cap(b.data) < neededLength {
		oldData := b.data
		b.data = make([]int32, bitutil.NextPowerOf2(neededLength))
		copy(b.data, oldData)
	}
	b.data = b.data[:neededLength]
}

func (b *OptInt32Builder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		b.data = nil
		b.releaseInternal()
	}
}

func (b *OptInt32Builder) AppendNull() {
	b.AppendNulls(1)
}

// AppendEmptyValue adds a new zero value (0) to the array being built.
func (b *OptInt32Builder) AppendEmptyValue() {
	b.Append(0)
}

func (b *OptInt32Builder) AppendNulls(n int) {
	b.resizeData(b.length + n)
	b.builderBase.AppendNulls(n)
}

func (b *OptInt32Builder) NewArray() arrow.Array {
	dataAsBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(b.data))), len(b.data)*arrow.Int32SizeBytes)
	data := array.NewData(
		b.dtype,
		b.length,
		[]*memory.Buffer{
			memory.NewBufferBytes(b.validityBitmap),
			memory.NewBufferBytes(dataAsBytes),
		},
		nil,
		b.length-bitutil.CountSetBits(b.validityBitmap, 0, b.length),
		0,
	)
	b.reset()
	b.data = nil
	return array.NewInt32Data(data)
}

// AppendData appends a slice of int32s to the builder. This data is considered
// to be non-null.
func (b *OptInt32Builder) AppendData(data []int32) {
	oldLength := b.length
	b.data = append(b.data, data...)
	b.length += len(data)
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBitsTo(b.validityBitmap, int64(oldLength), int64(len(data)), true)
}

func (b *OptInt32Builder) Append(v int32) {
	b.data = append(b.data, v)
	b.length++
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBit(b.validityBitmap, b.length-1)
}

// Set sets value v at index i. THis will panic if i is out of bounds. Use this
// after calling Reserve.
func (b *OptInt32Builder) Set(i int, v int32) {
	b.data[i] = v
	bitutil.SetBit(b.validityBitmap, i)
}

// Swap swaps values at i and j index.
func (b *OptInt32Builder) Swap(i, j int) {
	b.data[i], b.data[j] = b.data[j], b.data[i]
}

func (b *OptInt32Builder) Add(i int, v int32) {
	b.data[i] += v
}

func (b *OptInt32Builder) Value(i int) int32 {
	return b.data[i]
}

func (b *OptInt32Builder) RepeatLastValue(n int) error {
	if bitutil.BitIsNotSet(b.validityBitmap, b.length-1) {
		b.AppendNulls(n)
		return nil
	}

	lastValue := b.data[b.length-1]
	b.resizeData(b.length + n)
	for i := b.length; i < b.length+n; i++ {
		b.data[i] = lastValue
	}
	b.appendValid(n)
	return nil
}

// ResetToLength is specific to distinct optimizations in FrostDB.
func (b *OptInt32Builder) ResetToLength(n int) {
	if n == b.length {
		return
	}

	b.length = n
	b.data = b.data[:n]
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}

func (b *OptInt32Builder) Reserve(n int) {
	b.length = n
	b.data = slices.Grow(b.data, n)[:n]
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}

type OptFloat64Builder struct {
	builderBase

	data []float64
}

func NewOptFloat64Builder(dtype arrow.DataType) *OptFloat64Builder {
	b := &OptFloat64Builder{}
	b.dtype = dtype
	return b
}

func (b *OptFloat64Builder) resizeData(neededLength int) {
	if cap(b.data) < neededLength {
		oldData := b.data
		b.data = make([]float64, bitutil.NextPowerOf2(neededLength))
		copy(b.data, oldData)
	}
	b.data = b.data[:neededLength]
}

func (b *OptFloat64Builder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		b.data = nil
		b.releaseInternal()
	}
}

func (b *OptFloat64Builder) AppendNull() {
	b.AppendNulls(1)
}

// AppendEmptyValue adds a new zero value (0.0) to the array being built.
func (b *OptFloat64Builder) AppendEmptyValue() {
	b.Append(0.0)
}

func (b *OptFloat64Builder) AppendNulls(n int) {
	b.resizeData(b.length + n)
	b.builderBase.AppendNulls(n)
}

func (b *OptFloat64Builder) NewArray() arrow.Array {
	dataAsBytes := unsafe.Slice((*byte)(unsafe.Pointer(unsafe.SliceData(b.data))), len(b.data)*arrow.Float64SizeBytes)
	data := array.NewData(
		b.dtype,
		b.length,
		[]*memory.Buffer{
			memory.NewBufferBytes(b.validityBitmap),
			memory.NewBufferBytes(dataAsBytes),
		},
		nil,
		b.length-bitutil.CountSetBits(b.validityBitmap, 0, b.length),
		0,
	)
	b.reset()
	b.data = nil
	return array.NewFloat64Data(data)
}

// AppendData appends a slice of float64s to the builder.
// This data is considered to be non-null.
func (b *OptFloat64Builder) AppendData(data []float64) {
	oldLength := b.length
	b.data = append(b.data, data...)
	b.length += len(data)
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBitsTo(b.validityBitmap, int64(oldLength), int64(len(data)), true)
}

func (b *OptFloat64Builder) Append(v float64) {
	b.data = append(b.data, v)
	b.length++
	b.validityBitmap = resizeBitmap(b.validityBitmap, b.length)
	bitutil.SetBit(b.validityBitmap, b.length-1)
}

func (b *OptFloat64Builder) Set(i int, v float64) {
	b.data[i] = v
}

func (b *OptFloat64Builder) Add(i int, v float64) {
	b.data[i] += v
}

// Value returns the ith value of the builder.
func (b *OptFloat64Builder) Value(i int) float64 {
	return b.data[i]
}

func (b *OptFloat64Builder) RepeatLastValue(n int) error {
	if bitutil.BitIsNotSet(b.validityBitmap, b.length-1) {
		b.AppendNulls(n)
		return nil
	}

	lastValue := b.data[b.length-1]
	b.resizeData(b.length + n)
	for i := b.length; i < b.length+n; i++ {
		b.data[i] = lastValue
	}
	b.appendValid(n)
	return nil
}

// ResetToLength is specific to distinct optimizations in FrostDB.
func (b *OptFloat64Builder) ResetToLength(n int) {
	if n == b.length {
		return
	}

	b.length = n
	b.data = b.data[:n]
	b.validityBitmap = resizeBitmap(b.validityBitmap, n)
}
