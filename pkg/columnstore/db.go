package columnstore

import (
	"math"
	"sync"
	"sync/atomic"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
)

type ColumnStore struct {
	mtx *sync.RWMutex
	dbs map[string]*DB
	reg prometheus.Registerer
}

func New(reg prometheus.Registerer) *ColumnStore {
	if reg == nil {
		reg = prometheus.NewRegistry()
	}

	return &ColumnStore{
		mtx: &sync.RWMutex{},
		dbs: map[string]*DB{},
		reg: reg,
	}
}

type DB struct {
	name string

	mtx    *sync.RWMutex
	tables map[string]*Table
	reg    prometheus.Registerer

	// Databases monotomically increasing transaction id
	txmtx *sync.RWMutex
	tx    uint64
	// active is the list of active transactions TODO: a gc gorouting should prune this list as parts get merged
	active map[uint64]uint64 // TODO probably not the best choice for active list...
}

func (s *ColumnStore) DB(name string) *DB {
	s.mtx.RLock()
	db, ok := s.dbs[name]
	s.mtx.RUnlock()
	if ok {
		return db
	}

	s.mtx.Lock()
	defer s.mtx.Unlock()

	// Need to double check that in the mean time a database with the same name
	// wasn't concurrently created.
	db, ok = s.dbs[name]
	if ok {
		return db
	}

	db = &DB{
		name:   name,
		mtx:    &sync.RWMutex{},
		tables: map[string]*Table{},
		reg:    prometheus.WrapRegistererWith(prometheus.Labels{"db": name}, s.reg),

		active: map[uint64]uint64{},
		txmtx:  &sync.RWMutex{},
	}

	s.dbs[name] = db
	return db
}

func (db *DB) Table(name string, schema Schema, logger log.Logger) *Table {
	db.mtx.RLock()
	table, ok := db.tables[name]
	db.mtx.RUnlock()
	if ok {
		return table
	}

	db.mtx.Lock()
	defer db.mtx.Unlock()

	// Need to double check that in the mean time another table with the same
	// name wasn't concurrently created.
	table, ok = db.tables[name]
	if ok {
		return table
	}

	table = newTable(db, name, schema, db.reg, logger)
	db.tables[name] = table
	return table
}

// beginRead starts a read transaction
func (db *DB) beginRead() uint64 {
	return atomic.AddUint64(&db.tx, 1)
}

// begin is an internal function that Tables call to start a transaction for writes
func (db *DB) begin() (uint64, func()) {
	tx := atomic.AddUint64(&db.tx, 1)
	db.txmtx.Lock()
	db.active[tx] = math.MaxUint64
	db.txmtx.Unlock()
	return tx, func() {
		// commit the transaction
		db.txmtx.Lock()
		db.active[tx] = atomic.AddUint64(&db.tx, 1)
		db.txmtx.Unlock()
	}
}

// txCompleted returns true if a write transaction has been completed
func (db *DB) txCompleted(tx uint64) uint64 {
	db.txmtx.RLock()
	defer db.txmtx.RUnlock()

	finaltx, ok := db.active[tx]
	if !ok {
		return math.MaxUint64
	}

	return finaltx
}
