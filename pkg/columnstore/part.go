package columnstore

import (
	"math"
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

	for k := 0; k < len(s[i].Values); k++ {
		vi := s[i].Values[k]
		vj := s[j].Values[k]

		switch vi.(type) {
		case string:
			switch {
			case vi.(string) < vj.(string):
				return true
			case vi.(string) > vj.(string):
				return false
			}
		case uint64:
			switch {
			case vi.(uint64) < vj.(uint64):
				return true
			case vi.(uint64) > vj.(uint64):
				return false
			}
		case int:
			switch {
			case vi.(int) < vj.(int):
				return true
			case vi.(int) > vj.(int):
				return false
			}
		case int64:
			switch {
			case vi.(int64) < vj.(int64):
				return true
			case vi.(int64) > vj.(int64):
				return false
			}
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

		default:
			panic("at the disco")
		}
	}

	return true
}

// Swap implements the sort.Interface interface
func (s SortableRows) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

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
	default:
		panic("unsupported compare")
	}
}
