package columnstore

import (
	"sync"

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
	}

	s.dbs[name] = db
	return db
}

func (db *DB) Table(name string, schema Schema) *Table {
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

	table = newTable(db, name, schema, db.reg)
	db.tables[name] = table
	return table
}
