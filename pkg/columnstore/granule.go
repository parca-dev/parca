package columnstore

import (
	"fmt"
	"sync"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/array"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/google/btree"
	"github.com/prometheus/client_golang/prometheus"
)

type Granule struct {
	sync.RWMutex

	// least is the row that exists within the Granule that is the least.
	// This is used for quick insertion into the btree, without requiring an iterator
	least  Row
	parts  []*Part
	schema *Schema

	granulesCreated prometheus.Counter

	// pruned indicates if this Granule is longer found in the index
	pruned bool
}

func NewGranule(granulesCreated prometheus.Counter, schema *Schema, parts ...*Part) *Granule {
	g := &Granule{
		granulesCreated: granulesCreated,
		parts:           parts,
		schema:          schema,
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
				if schema.RowLessThan(r.Values, g.least.Values) {
					g.least = r
				}
			}
		}
	}

	granulesCreated.Inc()
	return g
}

func (g *Granule) AddPart(p *Part) {
	g.Lock()
	defer g.Unlock()

	g.parts = append(g.parts, p)
	it := p.Iterator()

	if it.Next() {
		r := Row{Values: it.Values()}
		if g.schema.RowLessThan(r.Values, g.least.Values) {
			g.least = r
		}
		return
	}
}

func (g *Granule) Cardinality() int {
	g.RLock()
	defer g.RUnlock()

	return g.cardinality()
}

func (g *Granule) cardinality() int {
	res := 0
	for _, p := range g.parts {
		res += p.Cardinality
	}
	return res
}

// split a granule into n sized granules. With the last granule containing the remainder.
// Returns the granules in order.
// This assumes the Granule has had it's parts merged into a single part
func (g *Granule) split(n int) ([]*Granule, error) {
	if len(g.parts) > 1 {
		return []*Granule{g}, nil // do nothing
	}

	tx := uint64(0) // TODO what is the tx during a split?

	// How many granules we'll need to build
	count := g.parts[0].Cardinality / n

	// Build all the new granules
	granules := make([]*Granule, 0, count)

	it := g.parts[0].Iterator()
	rows := make([]Row, 0, n)
	for it.Next() {
		rows = append(rows, Row{Values: it.Values()})
		if len(rows) == n && len(granules) != count-1 { // If we have n rows, and aren't on the last granule, create the n-sized granule
			p, err := NewPart(tx, g.schema.Columns, NewSimpleRowWriter(rows))
			if err != nil {
				return nil, fmt.Errorf("failed to create new part: %w", err)
			}
			granules = append(granules, NewGranule(g.granulesCreated, g.schema, p))
			rows = make([]Row, 0, n)
		}
	}

	// Save the remaining Granule
	if len(rows) != 0 {
		p, err := NewPart(tx, g.schema.Columns, NewSimpleRowWriter(rows))
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
	g.RLock()
	defer g.RUnlock()

	// Merge the parts
	p, err := Merge(tx, txCompleted, g.schema, g.parts...)
	if err != nil {
		return nil, err
	}

	// Prefetch all dynamic columns
	cols := make([]int, len(p.columns))
	names := make([][]string, len(p.columns))
	for i, c := range p.columns {
		if g.schema.Columns[i].Dynamic {
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

		switch g.schema.Columns[j].Dynamic {
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
