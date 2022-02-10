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
	schema      Schema
	columns     []Iterable
	Cardinality int
}

func NewPart(schema Schema, rows []Row) (*Part, error) {
	p := &Part{
		schema:      schema,
		Cardinality: len(rows),
	}
	p.columns = make([]Iterable, len(schema.Columns))

	var err error
	for i, c := range schema.Columns {
		p.columns[i], err = NewImmutableColumn(c, func(app Appender) error {
			for j := range rows {
				err := app.AppendAt(j, rows[j].Values[i])
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
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
func Merge(parts ...*Part) (*Part, error) {

	rows := SortableRows{}
	// Convert all the parts into a set of rows
	for _, p := range parts {
		it := p.Iterator()
		for it.Next() {
			rows = append(rows, Row{Values: it.Values()})
		}
	}

	// Sort the rows
	sort.Sort(rows)

	return NewPart(parts[0].schema, rows)
}

// SortableRows is a slice of Rows that can be sorted
type SortableRows []Row

// Len implements the sort.Interface interface
func (s SortableRows) Len() int { return len(s) }

// Less implements the sort.Interface interface
func (s SortableRows) Less(i, j int) bool {
	return s[i].Less(s[j])
}

// Swap implements the sort.Interface interface
func (s SortableRows) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

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
func (r Row) Less(than Row) bool {
	if than.Values == nil { // in the 0 case always return true
		return true
	}
	for k := 0; k < len(r.Values); k++ {
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
