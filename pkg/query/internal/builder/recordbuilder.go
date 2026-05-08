package builder

import (
	"fmt"
	"sync/atomic"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
)

// The code in this file is based heavily on Apache arrow's array.RecordBuilder,
// with some modifications to use our own optimized record builders. Ideally, we
// would eventually merge this upstream.

// RecordBuilder eases the process of building a Record, iteratively, from
// a known Schema.
type RecordBuilder struct {
	refCount int64
	mem      memory.Allocator
	schema   *arrow.Schema
	fields   []ColumnBuilder
}

// NewRecordBuilder returns a builder, using the provided memory allocator and a schema.
func NewRecordBuilder(mem memory.Allocator, schema *arrow.Schema) *RecordBuilder {
	b := &RecordBuilder{
		refCount: 1,
		mem:      mem,
		schema:   schema,
		fields:   make([]ColumnBuilder, schema.NumFields()),
	}

	for i := 0; i < schema.NumFields(); i++ {
		b.fields[i] = NewBuilder(mem, schema.Field(i).Type)
	}

	return b
}

// Retain increases the reference count by 1.
// Retain may be called simultaneously from multiple goroutines.
func (b *RecordBuilder) Retain() {
	atomic.AddInt64(&b.refCount, 1)
}

// Release decreases the reference count by 1.
func (b *RecordBuilder) Release() {
	if atomic.AddInt64(&b.refCount, -1) == 0 {
		for _, f := range b.fields {
			f.Release()
		}
		b.fields = nil
	}
}

func (b *RecordBuilder) Schema() *arrow.Schema     { return b.schema }
func (b *RecordBuilder) Fields() []ColumnBuilder   { return b.fields }
func (b *RecordBuilder) Field(i int) ColumnBuilder { return b.fields[i] }

func (b *RecordBuilder) Reserve(size int) {
	for _, f := range b.fields {
		f.Reserve(size)
	}
}

// NewRecord creates a new record from the memory buffers and resets the
// RecordBuilder so it can be used to build a new record.
//
// The returned Record must be Release()'d after use.
//
// NewRecord panics if the fields' builder do not have the same length.
func (b *RecordBuilder) NewRecord() arrow.Record {
	cols := make([]arrow.Array, len(b.fields))
	rows := int64(0)

	defer func(cols []arrow.Array) {
		for _, col := range cols {
			if col == nil {
				continue
			}
			col.Release()
		}
	}(cols)

	for i, f := range b.fields {
		cols[i] = f.NewArray()
		irow := int64(cols[i].Len())
		if i > 0 && irow != rows {
			panic(fmt.Errorf("arrow/array: field %d has %d rows. want=%d", i, irow, rows))
		}
		rows = irow
	}

	return array.NewRecord(b.schema, cols, rows)
}

// ExpandSchema expands the record builder schema by adding new fields.
func (b *RecordBuilder) ExpandSchema(schema *arrow.Schema) {
	for i := 0; i < schema.NumFields(); i++ {
		f := schema.Field(i)
		found := false
		for j := 0; j < b.schema.NumFields(); j++ {
			old := b.schema.Field(j)
			if f.Equal(old) {
				found = true
				break
			}
		}
		if found { // field already exists
			continue
		}

		// Add the new field
		b.fields = append(b.fields[:i], append([]ColumnBuilder{NewBuilder(b.mem, f.Type)}, b.fields[i:]...)...)
	}

	b.schema = schema
}

// Reset will call ResetFull on any dictionary builders to prevent memo tables from growing unbounded.
func (b *RecordBuilder) Reset() {
	for _, f := range b.fields {
		if lb, ok := f.(*ListBuilder); ok {
			if vb, ok := lb.ValueBuilder().(array.DictionaryBuilder); ok {
				vb.ResetFull()
			}
		}
	}
}
