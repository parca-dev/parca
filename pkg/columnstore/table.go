package columnstore

import (
	"github.com/google/btree"
)

type Table struct {
	schema   Schema
	granules []*Granule

	index *btree.BTree
}

func NewTable(schema Schema) *Table {
	return &Table{
		schema: schema,
		index:  btree.New(2), // TODO make the degree a setting
	}
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
