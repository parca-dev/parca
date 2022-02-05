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
	index  *btree.BTree

	sync.RWMutex
	sync.WaitGroup
}

type tableMetrics struct {
	granulesCreated  prometheus.Counter
	granulesSplits   prometheus.Counter
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

	t := &Table{
		db:     db,
		schema: schema,
		index:  btree.New(2), // TODO make the degree a setting
		metrics: &tableMetrics{
			granulesCreated: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "granules_created",
				Help: "Number of granules created.",
			}),
			granulesSplits: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "granules_splits",
				Help: "Number of granules splits executed.",
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

	promauto.With(reg).NewGaugeFunc(prometheus.GaugeOpts{
		Name: "index_size",
		Help: "Number of granules in the table index currently.",
	}, func() float64 {
		t.RLock()
		defer t.RUnlock()
		return float64(t.index.Len())
	})

	g := NewGranule(t.metrics.granulesCreated, []*Part{}...)
	t.index.ReplaceOrInsert(g)

	return t
}

// Sync the table. This will return once all split operations have completed.
// Currently it does not prevent new inserts from happening, so this is only
// safe to rely on if you control all writers. In the future we may need to add a way to
// block new writes as well.
func (t *Table) Sync() {
	t.Wait()
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

	t.RLock()
	defer t.RUnlock()

	rowsToInsertPerGranule := t.splitRowsByGranule(rows)
	for granule, rows := range rowsToInsertPerGranule {
		p, err := NewPart(t.schema, rows)
		if err != nil {
			return err
		}

		granule.AddPart(p)
		if granule.Cardinality() >= t.schema.GranuleSize {
			t.Add(1)
			go t.splitGranule(granule) // TODO there may be a better way to schedule this
		}
	}

	return nil
}

func (t *Table) splitGranule(granule *Granule) {
	defer t.Done()
	t.Lock()
	defer t.Unlock()
	granule.Lock()
	defer granule.Unlock()

	// Recheck to ensure the granule still needs to be split
	if granule.pruned || granule.cardinality() < t.schema.GranuleSize {
		return
	}

	newpart, err := Merge(granule.parts...) // need to merge all parts in a granule before splitting
	if err != nil {
		panic("failed to merge: TODO log this")
	}
	granule.parts = []*Part{newpart}

	granules, err := granule.split(t.schema.GranuleSize / 2) // TODO magic numbers
	if err != nil {
		panic("granule split failed after AddPart: TODO log this")
	}

	deleted := t.index.Delete(granule)
	if deleted == nil {
		panic("failed to delete granule during split: TODO log this")
	}

	// mark this granule as having been pruned
	granule.pruned = true

	for _, g := range granules {
		t.index.ReplaceOrInsert(g)
	}
}

// Iterator iterates in order over all granules in the table. It stops iterating when the iterator function returns false.
func (t *Table) Iterator(pool memory.Allocator, iterator func(r arrow.Record) error) error {
	t.RLock()
	defer t.RUnlock()
	var err error
	t.granuleIterator(func(g *Granule) bool {
		var r arrow.Record
		r, err = g.ArrowRecord(pool)
		if err != nil {
			return false
		}
		err = iterator(r)
		r.Release()
		return err == nil
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
		g.RLock()
		defer g.RUnlock()

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
