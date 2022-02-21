package columnstore

import (
	"fmt"
	"sync/atomic"

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
	parts *PartList

	// card is the raw commited, and uncommited cardinality of the granule. It is used as a suggestion for potential compaction
	card uint64

	schema *Schema

	granulesCreated prometheus.Counter

	// pruned indicates if this Granule is longer found in the index
	pruned uint64

	// newGranules are the granules that were created after a split
	newGranules []*Granule
}

func NewGranule(granulesCreated prometheus.Counter, schema *Schema, parts ...*Part) *Granule {
	g := &Granule{
		granulesCreated: granulesCreated,
		parts:           &PartList{},
		schema:          schema,
	}

	// Find the least column
	for i, p := range parts {
		g.card += uint64(p.Cardinality)
		g.parts.Prepend(p)
		it := p.Iterator()
		if it.Next() { // Since we assume a part is sorted, we need only to look at the first row in each Part
			r := Row{Values: it.Values()}
			switch i {
			case 0:
				g.least = r
			default:
				if schema.RowLessThan(r.Values, g.least.Values) {
					g.least = r
				}
			}
		}
	}

	granulesCreated.Inc()
	return g
}

// AddPart returns the new cardinality of the Granule
func (g *Granule) AddPart(p *Part) uint64 {

	g.parts.Prepend(p)
	newcard := atomic.AddUint64(&g.card, uint64(p.Cardinality))
	it := p.Iterator()

	if it.Next() {
		r := Row{Values: it.Values()}
		if g.schema.RowLessThan(r.Values, g.least.Values) { // TODO load g.least ptr
			g.least = r // TODO atomic set the least pointer
		}

		// If the granule was pruned, copy part to new granule
		if atomic.LoadUint64(&g.pruned) != 0 {
			addPartToGranule(g.newGranules, p)
		}
	}

	return newcard
}

func (g *Granule) Cardinality(tx uint64, txCompleted func(uint64) uint64) int {
	return g.cardinality(tx, txCompleted)
}

func (g *Granule) cardinality(tx uint64, txCompleted func(uint64) uint64) int {
	res := 0
	g.parts.Iterate(func(p *Part) bool {
		if p.tx > tx || txCompleted(p.tx) > tx {
			return true
		}
		res += p.Cardinality
		return true
	})
	return res
}

// split a granule into n sized granules. With the last granule containing the remainder.
// Returns the granules in order.
// This assumes the Granule has had it's parts merged into a single part
func (g *Granule) split(tx uint64, n int) ([]*Granule, error) {

	// How many granules we'll need to build
	count := 0
	var it *PartIterator
	g.parts.Iterate(func(p *Part) bool {
		count = p.Cardinality / n
		it = p.Iterator()
		return false
	})

	// Build all the new granules
	granules := make([]*Granule, 0, count)

	rows := make([]Row, 0, n)
	for it.Next() {
		rows = append(rows, Row{Values: it.Values()})
		if len(rows) == n && len(granules) != count-1 { // If we have n rows, and aren't on the last granule, create the n-sized granule
			p, err := NewPart(tx, g.schema.columns, NewSimpleRowWriter(rows))
			if err != nil {
				return nil, fmt.Errorf("failed to create new part: %w", err)
			}
			granules = append(granules, NewGranule(g.granulesCreated, g.schema, p))
			rows = make([]Row, 0, n)
		}
	}

	// Save the remaining Granule
	if len(rows) != 0 {
		p, err := NewPart(tx, g.schema.columns, NewSimpleRowWriter(rows))
		if err != nil {
			if err != nil {
				return nil, fmt.Errorf("failed to create new part: %w", err)
			}
		}
		granules = append(granules, NewGranule(g.granulesCreated, g.schema, p))
	}

	return granules, nil
}

// ArrowRecord merges all parts in a Granule before returning an ArrowRecord over that part
func (g *Granule) ArrowRecord(tx uint64, txCompleted func(uint64) uint64, pool memory.Allocator) (arrow.Record, error) {

	// Merge the parts
	p, err := Merge(tx, txCompleted, g.schema, g.parts)
	if err != nil {
		return nil, err
	}

	// Prefetch all dynamic columns
	cols := make([]int, len(p.columns))
	names := make([][]string, len(p.columns))
	for i, c := range p.columns {
		if g.schema.columns[i].Dynamic {
			cols[i] = len(c.(*DynamicColumn).dynamicColumns)
			names[i] = make([]string, cols[i])
			for j, name := range c.(*DynamicColumn).dynamicColumns {
				names[i][j] = name
			}
		}
	}

	// Build the record
	bld := array.NewRecordBuilder(pool, g.schema.ToArrow(names, cols))
	defer bld.Release()

	i := 0 // i is the index into our arrow schema
	for j, c := range p.columns {

		switch g.schema.columns[j].Dynamic {
		case true: // expand the dynamic columns
			d := c.(*DynamicColumn) // TODO this is gross and we should change this iteration
			for k, name := range d.dynamicColumns {
				err := d.def.Type.AppendIteratorToArrow(d.data[name].Iterator(p.Cardinality), bld.Field(i+k))
				if err != nil {
					return nil, err
				}
			}
			i += len(d.dynamicColumns)
		default:
			col := c.(*StaticColumn)
			err := col.def.Type.AppendIteratorToArrow(col.data.Iterator(p.Cardinality), bld.Field(i))
			if err != nil {
				return nil, err
			}
			i++
		}
	}

	maxValues := 0
	for _, c := range bld.Fields() {
		l := c.Len()
		if l > maxValues {
			maxValues = l
		}
	}

	for _, c := range bld.Fields() {
		for i := c.Len(); i < maxValues; i++ {
			c.AppendNull()
		}
	}

	return bld.NewRecord(), nil
}

// Less implements the btree.Item interface
func (g *Granule) Less(than btree.Item) bool {
	return g.schema.RowLessThan(g.least.Values, than.(*Granule).least.Values)
}
