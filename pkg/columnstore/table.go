package columnstore

import (
	"fmt"
	"sync"

	"github.com/apache/arrow/go/v7/arrow"
	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/google/btree"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ErrNoSchema = fmt.Errorf("no schema")

type Table struct {
	db      *DB
	metrics *tableMetrics

	schema Schema

	mtx   *sync.RWMutex
	index *btree.BTree
}

type tableMetrics struct {
	granulesCreated  prometheus.Counter
	rowsInserted     prometheus.Counter
	zeroRowsInserted prometheus.Counter
	rowInsertSize    prometheus.Histogram
}

func newTable(
	db *DB,
	name string,
	schema Schema,
	reg prometheus.Registerer,
) *Table {
	reg = prometheus.WrapRegistererWith(prometheus.Labels{"table": name}, reg)

	return &Table{
		db:     db,
		schema: schema,
		mtx:    &sync.RWMutex{},
		index:  btree.New(2), // TODO make the degree a setting
		metrics: &tableMetrics{
			granulesCreated: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "granules_created",
				Help: "Number of granules created.",
			}),
			rowsInserted: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rows_inserted",
				Help: "Number of rows inserted into table.",
			}),
			zeroRowsInserted: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "zero_rows_inserted",
				Help: "Number of times it was attempted to insert zero rows into the table.",
			}),
			rowInsertSize: promauto.With(reg).NewHistogram(prometheus.HistogramOpts{
				Name:    "row_insert_size",
				Help:    "Size of batch inserts into table.",
				Buckets: prometheus.ExponentialBuckets(1, 2, 10),
			}),
		},
	}
}

func (t *Table) Insert(rows []Row) error {
	defer func() {
		t.metrics.rowsInserted.Add(float64(len(rows)))
		t.metrics.rowInsertSize.Observe(float64(len(rows)))
	}()

	if len(rows) == 0 {
		t.metrics.zeroRowsInserted.Add(float64(len(rows)))
		return nil
	}

	t.mtx.Lock()
	defer t.mtx.Unlock()

	// Special case: if there are no granules, create the very first one and immediately insert the first part.
	if t.index.Len() == 0 {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		g := NewGranule(t.metrics.granulesCreated, p)
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

			granules, err := granule.Split(t.schema.GranuleSize / 2) // TODO magic numbers
			if err != nil {
				return fmt.Errorf("granule split failed after AddPart: %w", err)
			}
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
func (t *Table) Iterator(pool memory.Allocator, iterator func(r arrow.Record) bool) error {
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	var err error
	t.granuleIterator(func(g *Granule) bool {
		var r arrow.Record
		r, err = g.ArrowRecord(pool)
		if err != nil {
			return false
		}
		res := iterator(r)
		r.Release()
		return res
	})
	return err
}

func (t *Table) granuleIterator(iterator func(g *Granule) bool) {
	t.index.Ascend(func(i btree.Item) bool {
		g := i.(*Granule)
		return iterator(g)
	})
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
