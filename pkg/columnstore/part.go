package columnstore

import (
	"math"
	"reflect"
	"sort"
)

// Comparison is a result from the compare function
type Comparison int

const (
	LessThan    Comparison = iota
	GreaterThan Comparison = iota
	Equal       Comparison = iota
)

type Row struct {
	Values []interface{}
}

type Part struct {
	schema      Schema
	columns     []Column
	Cardinality int
}

func NewPart(schema Schema, rows []Row) (*Part, error) {
	p := &Part{
		schema:      schema,
		Cardinality: len(rows),
	}
	p.columns = make([]Column, len(schema.Columns))

	for i, c := range schema.Columns {
		p.columns[i] = NewColumn(c)
		app, err := p.columns[i].Appender()
		if err != nil {
			return nil, err
		}

		for j := range rows {
			err := app.AppendAt(j, rows[j].Values[i])
			if err != nil {
				return nil, err
			}
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
func compare(a, b interface{}) Comparison {
	switch a.(type) {
	case string:
		switch {
		case a.(string) < b.(string):
			return LessThan
		case a.(string) > b.(string):
			return GreaterThan
		default:
			return Equal
		}
	case uint64:
		switch {
		case a.(uint64) < b.(uint64):
			return LessThan
		case a.(uint64) > b.(uint64):
			return GreaterThan
		default:
			return Equal
		}
	case int64:
		switch {
		case a.(int64) < b.(int64):
			return LessThan
		case a.(int64) > b.(int64):
			return GreaterThan
		default:
			return Equal
		}
	case UUID:
		res := CompareUUID(a.(UUID), b.(UUID))
		switch res {
		case -1:
			return LessThan
		case 1:
			return GreaterThan
		default:
			return Equal
		}
	default:
		panic("unsupported compare for type " + reflect.TypeOf(a).String())
	}
}

// Less returns true if the row is Less than the given row
func (r Row) Less(than Row) bool {
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
				case compare(dci[l].Value, dcj[l].Value) == LessThan:
					return true
				case compare(dci[l].Value, dcj[l].Value) == GreaterThan:
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
			case LessThan:
				return true
			case GreaterThan:
				return false
			}
		}
	}

	return false
}

func UUIDsLess(uuids1, uuids2 []UUID) bool {
	uuids1Len := len(uuids1)
	uuids2Len := len(uuids2)

	k := 0
	for {
		switch {
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
}
