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

func (g *Granule) Iterator() *GranuleIterator {
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

// TODO: This iterator implementation is totally wrong. It iterates over all
// parts one by one, but it should be merging them and return them in order.
// But hey ... it does something.
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
