package columnstore

import (
	"sync"

	"github.com/google/btree"
)

type ColumnStore struct {
	mtx *sync.RWMutex
	dbs map[string]*DB
}

func New() *ColumnStore {
	return &ColumnStore{
		mtx: &sync.RWMutex{},
		dbs: map[string]*DB{},
	}
}

type DB struct {
	name string

	mtx    *sync.RWMutex
	tables map[string]*Table
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
	}

	s.dbs[name] = db
	return db
}

func (db *DB) Table(name string) *Table {
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

	table = &Table{
		db:    db,
		smtx:  &sync.Mutex{},
		mtx:   &sync.RWMutex{},
		index: btree.New(2), // TODO make the degree a setting
	}

	db.tables[name] = table
	return table
}
