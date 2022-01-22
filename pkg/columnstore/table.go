package columnstore

type Table struct {
	schema   Schema
	granules []*Granule
}

func NewTable(schema Schema) *Table {
	return &Table{schema: schema}
}

func (t *Table) Insert(rows []Row) error {
	// Special case: if there are no granules, create the very first one and immediately insert the first part.
	if len(t.granules) == 0 {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		t.granules = append(t.granules, NewGranule(p))
		return nil
	}

	rowsToInsertPerGranule := t.splitRowsByGranule(rows)
	for granule, rows := range rowsToInsertPerGranule {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		granule.AddPart(p)

		// TODO: somewhere here compactions of a granule need to be scheduled.
	}

	return nil
}

func (t *Table) Iterator() *TableIterator {
	its := make([]*GranuleIterator, len(t.granules))

	for i, g := range t.granules {
		its[i] = g.Iterator()
	}

	return &TableIterator{
		its: its,
	}
}

type TableIterator struct {
	its              []*GranuleIterator
	currGranuleIndex int
}

func (ti *TableIterator) Next() bool {
	if ti.its[ti.currGranuleIndex].Next() {
		return true
	}

	ti.currGranuleIndex++
	if ti.currGranuleIndex >= len(ti.its) {
		return false
	}
	return ti.its[ti.currGranuleIndex].Next()
}

func (ti *TableIterator) Row() Row {
	return ti.its[ti.currGranuleIndex].Row()
}

func (ti *TableIterator) Err() error {
	return ti.its[ti.currGranuleIndex].Err()
}

func (t *Table) splitRowsByGranule(rows []Row) map[*Granule][]Row {
	rowsByGranule := map[*Granule][]Row{}

	// Special case: if there is only one granule, insert parts into it until full.
	if len(t.granules) == 1 {
		rowsByGranule[t.granules[0]] = rows
		return rowsByGranule
	}

	// TODO: general case: split rows into groups of rows belonging to the respective granule.

	return rowsByGranule
}

type Granule struct {
	parts []*Part
}

func NewGranule(parts ...*Part) *Granule {
	return &Granule{parts: parts}
}

func (g *Granule) AddPart(p *Part) {
	g.parts = append(g.parts, p)
}

func (g *Granule) Cardinality() int {
	res := 0
	for _, p := range g.parts {
		res += p.Cardinality
	}
	return res
}

// Split a granule into n sized granules. Returns the granules in order.
// This assumes the Granule has had it's parts merged into a single part
func (g *Granule) Split(n int) []*Granule {
	if len(g.parts) > 1 {
		return []*Granule{g} // do nothing
	}

	// How many granules we'll need to build
	count := g.parts[0].Cardinality / n
	if g.parts[0].Cardinality%n != 0 {
		count++
	}

	// Build all the new granules
	granules := make([]*Granule, 0, count)

	it := g.parts[0].Iterator()
	rows := make([]Row, 0, n)
	for it.Next() {
		rows = append(rows, Row{Values: it.Values()})
		if len(rows) == n {
			p, err := NewPart(g.parts[0].schema, rows)
			if err != nil {
				panic("dun goofed")
			}
			granules = append(granules, NewGranule(p))
			rows = make([]Row, 0, n)
		}
	}

	// Save the remaining Granule
	if len(rows) != 0 {
		p, err := NewPart(g.parts[0].schema, rows)
		if err != nil {
			panic("dun goofed")
		}
		granules = append(granules, NewGranule(p))
	}

	return granules
}

// Iterator merges all parts iin a Granule before returning an iterator over that part
// NOTE: this may not be the optimal way to perform a merge during iteration. But it's technically correct
func (g *Granule) Iterator() *GranuleIterator {

	// Merge the parts
	p, err := Merge(g.parts...)
	if err != nil {
		panic("merge failure")
	}

	// replace the granules parts with the merged part
	g.parts = []*Part{p}

	its := make([]*PartIterator, len(g.parts))
	for i, p := range g.parts {
		its[i] = p.Iterator()
	}

	return &GranuleIterator{
		its: its,
	}
}

type GranuleIterator struct {
	its           []*PartIterator
	currPartIndex int
}

func (gi *GranuleIterator) Next() bool {
	if gi.its[gi.currPartIndex].Next() {
		return true
	}

	gi.currPartIndex++
	if gi.currPartIndex >= len(gi.its) {
		return false
	}
	return gi.its[gi.currPartIndex].Next()
}

func (gi *GranuleIterator) Row() Row {
	return Row{Values: gi.its[gi.currPartIndex].Values()}
}

func (gi *GranuleIterator) Err() error {
	return gi.its[gi.currPartIndex].Err()
}
