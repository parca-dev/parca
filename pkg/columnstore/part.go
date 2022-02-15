package columnstore

import (
	"math"
	"reflect"
	"sort"
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
	rows := SortableRows{
		rows:   []Row{},
		schema: schema,
	}

	for _, it := range its {
		for it.Next() {
			rows.rows = append(rows.rows, Row{Values: it.Values()})
		}
		if it.Err() != nil {
			return nil, it.Err()
		}
	}

	// Sort the rows
	sort.Sort(rows)

	return NewPart(tx, schema.Columns, NewSimpleRowWriter(rows.rows))
}

// SortableRows is a slice of Rows that can be sorted
type SortableRows struct {
	rows   []Row
	schema *Schema
}

// Len implements the sort.Interface interface
func (s SortableRows) Len() int { return len(s.rows) }

// Less implements the sort.Interface interface
func (s SortableRows) Less(i, j int) bool {
	return s.rows[i].Less(s.rows[j], s.schema.ordered)
}

// Swap implements the sort.Interface interface
func (s SortableRows) Swap(i, j int) { s.rows[i], s.rows[j] = s.rows[j], s.rows[i] }

// TODO comparison int values are well defined in Go, -1 for less than, 0 for
// equal, 1 for greater than. We should use that instead of the custom return
// values.
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

// Less returns true if the row is Less than the given row
func (r Row) Less(than Row, orderedBy []int) bool {
	if than.Values == nil { // in the 0 case always return true
		return true
	}
	for _, k := range orderedBy {
		vi := r.Values[k]
		vj := than.Values[k]

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
