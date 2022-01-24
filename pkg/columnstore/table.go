package columnstore

import (
	"fmt"

	"github.com/google/btree"
)

type Table struct {
	schema Schema

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
	if t.index.Len() == 0 {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		g := NewGranule(p)
		t.index.ReplaceOrInsert(g)
		return nil
	}

	rowsToInsertPerGranule := t.splitRowsByGranule(rows)
	for granule, rows := range rowsToInsertPerGranule {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		granule.AddPart(p)
		if granule.Cardinality() >= t.schema.GranuleSize {

			// TODO: splits should be performed in the background. Do it now for simplicity

			newpart, err := Merge(granule.parts...) // need to merge all parts in a granule before splitting
			if err != nil {
				return err
			}
			granule.parts = []*Part{newpart}

			granules := granule.Split(t.schema.GranuleSize / 2) // TODO magic numbers
			deleted := t.index.Delete(granule)
			if deleted == nil {
				return fmt.Errorf("failed to delete granule during split")
			}
			for _, g := range granules {
				t.index.ReplaceOrInsert(g)
			}
		}
	}

	return nil
}

// Iterator iterates in order over all granules in the table. It stops iterating when the iterator function returns false.
func (t *Table) Iterator(iterator btree.ItemIterator) {
	t.index.Ascend(iterator)
}

func (t *Table) splitRowsByGranule(rows []Row) map[*Granule][]Row {
	rowsByGranule := map[*Granule][]Row{}

	// Special case: if there is only one granule, insert parts into it until full.
	if t.index.Len() == 1 {
		rowsByGranule[t.index.Min().(*Granule)] = rows
		return rowsByGranule
	}

	// TODO: we might be able to do ascend less than or ascend greater than here?
	j := 0
	var prev *Granule
	t.index.Ascend(func(i btree.Item) bool {
		g := i.(*Granule)

		for ; j < len(rows); j++ {
			if rows[j].Less(g.least) {
				if prev != nil {
					rowsByGranule[prev] = append(rowsByGranule[prev], rows[j])
					continue
				}
				return true // continue btree iteration
			}

			// stop at the first granule where this is not the least
			// this might be the correct granule, but we need to check that it isn't the next granule
			prev = g
			return true // continue btree iteration
		}

		// All rows accounted for
		return false
	})

	// Save any remaining rows that belong into prev
	for ; j < len(rows); j++ {
		rowsByGranule[prev] = append(rowsByGranule[prev], rows[j])
	}

	return rowsByGranule
}
