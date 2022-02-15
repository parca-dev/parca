package columnstore

import (
	"container/heap"
	"fmt"
	"math"
	"reflect"
)

type Row struct {
	Values []interface{}
}

type Part struct {
	columns     []Iterable
	Cardinality int

	// transaction id that this part was indserted under
	tx uint64
}

type RowWriter interface {
	WriteTo(appenders []Appender) (int, error)
}

func NewSimpleRowWriter(rows []Row) *SimpleRowWriter {
	return &SimpleRowWriter{
		rows: rows,
	}
}

type SimpleRowWriter struct {
	rows []Row
}

func (w *SimpleRowWriter) WriteTo(appenders []Appender) (int, error) {
	var err error

	for i, row := range w.rows {
		for j, v := range row.Values {
			err = appenders[j].AppendAt(i, v)
			if err != nil {
				return i, err
			}
		}
	}

	return len(w.rows), nil
}

func NewEmptyPart(tx uint64, colDefs []ColumnDefinition) (*Part, error) {
	return NewPart(tx, colDefs, &emptyRowWriter{})
}

type emptyRowWriter struct{}

func (w *emptyRowWriter) WriteTo(appenders []Appender) (int, error) { return 0, nil }

func NewPart(tx uint64, colDefs []ColumnDefinition, w RowWriter) (*Part, error) {
	p := &Part{
		tx: tx,
	}
	p.columns = make([]Iterable, len(colDefs))
	appenders := make([]Appender, len(colDefs))

	var err error
	for i, c := range colDefs {
		p.columns[i], appenders[i], err = NewAppendOnceColumn(c)
		if err != nil {
			return nil, err
		}
	}
	p.Cardinality, err = w.WriteTo(appenders)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Part) Iterator() *PartIterator {
	columnIterators := make([]Iterator, len(p.columns))

	for i, c := range p.columns {
		columnIterators[i] = c.Iterator(p.Cardinality)
	}

	return &PartIterator{
		columnIterators: columnIterators,
	}
}

type PartIterator struct {
	columnIterators []Iterator
}

func (pi *PartIterator) Next() bool {
	for _, ci := range pi.columnIterators {
		if !ci.Next() {
			return false
		}
	}

	return true
}

func (pi *PartIterator) Values() []interface{} {
	res := make([]interface{}, len(pi.columnIterators))

	for i, ci := range pi.columnIterators {
		res[i] = ci.Value()
	}

	return res
}

func (pi *PartIterator) Err() error {
	for _, ci := range pi.columnIterators {
		if err := ci.Err(); err != nil {
			return err
		}
	}

	return nil
}

// Merge merges all parts into a single part
func Merge(tx uint64, txCompleted func(uint64) uint64, schema *Schema, parts ...*Part) (*Part, error) {
	its := make([]*PartIterator, 0, len(parts))

	// Convert all the parts into a set of rows
	for _, p := range parts {
		// Don't merge parts from an newer tx, or from an uncompleted tx, or a completed tx that finished after this tx started
		if p.tx > tx || txCompleted(p.tx) > tx {
			continue
		}

		its = append(its, p.Iterator())
	}

	return merge(tx, schema, its)
}

func merge(tx uint64, schema *Schema, its []*PartIterator) (*Part, error) {
	partsWithData := make([]*PartIterator, 0, len(its))
	for _, it := range its {
		if it.Next() {
			partsWithData = append(partsWithData, it)
			continue
		}
		if it.Err() != nil {
			return nil, fmt.Errorf("start part iterators: %w", it.Err())
		}
	}

	if len(partsWithData) == 0 {
		return NewEmptyPart(tx, schema.Columns)
	}

	return NewPart(tx, schema.Columns, &streamingRowWriter{
		it: newMultiPartIterator(schema, partsWithData),
	})
}

type streamingRowWriter struct {
	it *multiPartIterator
}

func (w *streamingRowWriter) WriteTo(appenders []Appender) (int, error) {
	var err error

	i := 0
	for w.it.Next() {
		for j, v := range w.it.Values() {
			err = appenders[j].AppendAt(i, v)
			if err != nil {
				return i, fmt.Errorf("append value at index %d: %w", i, err)
			}
		}
		i++
	}

	return i, w.it.Err()
}

type multiPartIterator struct {
	schema  *Schema
	parts   []*PartIterator
	cur     [][]interface{}
	err     error
	started bool
}

func newMultiPartIterator(schema *Schema, parts []*PartIterator) *multiPartIterator {
	it := &multiPartIterator{
		schema: schema,
		parts:  parts,
		cur:    make([][]interface{}, len(parts)),
	}

	for i, p := range parts {
		it.cur[i] = p.Values()
	}

	heap.Init(it)
	return it
}

func (m *multiPartIterator) Next() bool {
	if !m.started {
		m.started = true
		return true
	}

	next := m.parts[0].Next()
	if !next {
		if m.parts[0].Err() != nil {
			m.err = m.parts[0].Err()
			return false
		}
		heap.Pop(m)
		if len(m.parts) == 0 {
			return false
		}
		return true
	}
	m.cur[0] = m.parts[0].Values()
	heap.Fix(m, 0)
	return true
}

func (m *multiPartIterator) Err() error {
	return m.err
}

func (m *multiPartIterator) Values() []interface{} {
	return m.cur[0]
}

func (m *multiPartIterator) Len() int { return len(m.parts) }

func (m *multiPartIterator) Less(i, j int) bool {
	return valuesLess(m.cur[i], m.cur[j], m.schema.ordered)
}

func (m *multiPartIterator) Swap(i, j int) {
	m.parts[i], m.parts[j] = m.parts[j], m.parts[i]
	m.cur[i], m.cur[j] = m.cur[j], m.cur[i]
}

func (m *multiPartIterator) Pop() interface{} {
	n := len(m.parts)
	m.parts = m.parts[0 : n-1]
	m.cur = m.cur[0 : n-1]
	return nil
}

func (m *multiPartIterator) Push(v interface{}) {
	panic("not implemented")
}

func compare(a, b interface{}) int {
	switch a.(type) {
	case string:
		switch {
		case a.(string) < b.(string):
			return -1
		case a.(string) > b.(string):
			return 1
		default:
			return 0
		}
	case uint64:
		switch {
		case a.(uint64) < b.(uint64):
			return -1
		case a.(uint64) > b.(uint64):
			return 1
		default:
			return 0
		}
	case int64:
		switch {
		case a.(int64) < b.(int64):
			return -1
		case a.(int64) > b.(int64):
			return 1
		default:
			return 0
		}
	case UUID:
		return CompareUUID(a.(UUID), b.(UUID))
	default:
		panic("unsupported compare for type " + reflect.TypeOf(a).String())
	}
}

// valuesLess returns true if the row is Less than the given row
func valuesLess(a, b []interface{}, orderedBy []int) bool {
	if b == nil { // in the 0 case always return true
		return true
	}
	for _, k := range orderedBy {
		vi := a[k]
		vj := b[k]

		switch vi.(type) {
		case []DynamicColumnValue:

			dci := vi.([]DynamicColumnValue)
			dcj := vj.([]DynamicColumnValue)
			end := int(math.Min(float64(len(dci)), float64(len(dcj))))
			for l := 0; l < end; l++ {
				switch {
				case dci[l].Name < dcj[l].Name:
					return true
				case dci[l].Name < dcj[l].Name:
					return false
				case compare(dci[l].Value, dcj[l].Value) == -1:
					return true
				case compare(dci[l].Value, dcj[l].Value) == 1:
					return false
				}
			}

			// The dynamic columns are equal unless their lengths aren't the same
			switch {
			case len(dci) < len(dcj):
				return true
			case len(dci) > len(dcj):
				return false
			}
		case []UUID:
			return UUIDsLess(vi.([]UUID), vj.([]UUID))
		default:
			switch compare(vi, vj) {
			case -1:
				return true
			case 1:
				return false
			}
		}
	}

	return false
}
