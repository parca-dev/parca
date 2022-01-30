package columnstore

import (
	"fmt"

	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/google/btree"
)

type Granule struct {

	// least is the row that exists within the Granule that is the least.
	// This is used for quick insertion into the btree, without requiring an iterator
	least Row
	parts []*Part
}

func NewGranule(parts ...*Part) *Granule {
	g := &Granule{
		parts: parts,
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
			granules = append(granules, NewGranule(p))
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
		granules = append(granules, NewGranule(p))
	}

	return granules, nil
}

// Iterator merges all parts iin a Granule before returning an iterator over that part
// NOTE: this may not be the optimal way to perform a merge during iteration. But it's technically correct
func (g *Granule) ArrowRecord(pool memory.Allocator) (*ArrowRecord, error) {
	// Merge the parts
	p, err := Merge(g.parts...)
	if err != nil {
		return nil, err
	}

	cols, err := p.ArrowColumns(pool)
	if err != nil {
		return nil, err
	}

	return NewArrowRecord(cols), nil
}

// Less implements the btree.Item interface
func (g *Granule) Less(than btree.Item) bool {
	return g.least.Less(than.(*Granule).least)
}
