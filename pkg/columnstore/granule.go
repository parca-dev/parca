package columnstore

import (
	"fmt"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/google/btree"
	"github.com/prometheus/client_golang/prometheus"
)

type Granule struct {

	// least is the row that exists within the Granule that is the least.
	// This is used for quick insertion into the btree, without requiring an iterator
	least Row
	parts []*Part

	granulesCreated prometheus.Counter
}

func NewGranule(granulesCreated prometheus.Counter, parts ...*Part) *Granule {
	g := &Granule{
		granulesCreated: granulesCreated,
		parts:           parts,
	}

	// Find the least column
	for i, p := range parts {
		it := p.Iterator()
		if it.Next() { // Since we assume a part is sorted, we need only to look at the first row in each Part
			r := Row{Values: it.Values()}
			switch i {
			case 0:
				g.least = r
			default:
				if r.Less(g.least) {
					g.least = r
				}
			}
		}
	}

	granulesCreated.Inc()
	return g
}

func (g *Granule) AddPart(p *Part) {
	g.parts = append(g.parts, p)
	it := p.Iterator()

	if it.Next() {
		r := Row{Values: it.Values()}
		if r.Less(g.least) {
			g.least = r
		}
		return
	}
}

func (g *Granule) Cardinality() int {
	res := 0
	for _, p := range g.parts {
		res += p.Cardinality
	}
	return res
}

// Split a granule into n sized granules. With the last granule containing the remainder.
// Returns the granules in order.
// This assumes the Granule has had it's parts merged into a single part
func (g *Granule) Split(n int) ([]*Granule, error) {
	if len(g.parts) > 1 {
		return []*Granule{g}, nil // do nothing
	}

	// How many granules we'll need to build
	count := g.parts[0].Cardinality / n

	// Build all the new granules
	granules := make([]*Granule, 0, count)

	it := g.parts[0].Iterator()
	rows := make([]Row, 0, n)
	for it.Next() {
		rows = append(rows, Row{Values: it.Values()})
		if len(rows) == n && len(granules) != count-1 { // If we have n rows, and aren't on the last granule, create the n-sized granule
			p, err := NewPart(g.parts[0].schema, rows)
			if err != nil {
				return nil, fmt.Errorf("failed to create new part: %w", err)
			}
			granules = append(granules, NewGranule(g.granulesCreated, p))
			rows = make([]Row, 0, n)
		}
	}

	// Save the remaining Granule
	if len(rows) != 0 {
		p, err := NewPart(g.parts[0].schema, rows)
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("failed to create new part: %w", err)
			}
		}
		granules = append(granules, NewGranule(g.granulesCreated, p))
	}

	return granules, nil
}

// ArrowRecord merges all parts in a Granule before returning an ArrowRecord over that part
func (g *Granule) ArrowRecord(pool memory.Allocator) (arrow.Record, error) {
	// Merge the parts
	p, err := Merge(g.parts...)
	if err != nil {
		return nil, err
	}

	// Prefetch all dynamic columns
	cols := make([]int, len(p.columns))
	names := make([][]string, len(p.columns))
	for i, c := range p.columns {
		if p.schema.Columns[i].Dynamic {
			cols[i] = len(c.(*DynamicColumn).dynamicColumns)
			names[i] = make([]string, cols[i])
			for j, name := range c.(*DynamicColumn).dynamicColumns {
				names[i][j] = name
			}
		}
	}

	// Build the record
	bld := array.NewRecordBuilder(pool, p.schema.ToArrow(names, cols))
	defer bld.Release()

	i := 0 // i is the index into our arrow schema
	for j, c := range p.columns {

		switch p.schema.Columns[j].Dynamic {
		case true: // expand the dynamic columns
			d := c.(*DynamicColumn) // TODO this is gross and we should change this iteration
			for k, name := range d.dynamicColumns {
				buildFromIterator(i, i+k, bld, d.data[name].Iterator(p.Cardinality))
			}
			i += len(d.dynamicColumns)
		default:
			buildFromIterator(i, i, bld, c.Iterator(p.Cardinality))
			i++
		}
	}

	return bld.NewRecord(), nil
}

type SimpleIterator interface {
	Next() bool
	Value() interface{}
}

func buildFromIterator(i, j int, bld *array.RecordBuilder, it SimpleIterator) {
	for it.Next() {
		switch bld.Schema().Field(i).Type.ID() {
		case arrow.BinaryTypes.String.ID():
			if it.Value() == nil {
				bld.Field(j).(*array.StringBuilder).AppendNull()
			} else {
				bld.Field(j).(*array.StringBuilder).Append(it.Value().(string))
			}
		case arrow.PrimitiveTypes.Int64.ID():
			if it.Value() == nil {
				bld.Field(j).(*array.Int64Builder).AppendNull()
			} else {
				bld.Field(j).(*array.Int64Builder).Append(it.Value().(int64))
			}
		}
	}
}

// Less implements the btree.Item interface
func (g *Granule) Less(than btree.Item) bool {
	return g.least.Less(than.(*Granule).least)
}
