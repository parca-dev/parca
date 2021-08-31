package sql

import (
	"database/sql"
	"fmt"

	"github.com/parca-dev/parca/pkg/storage/metastore"
	_ "modernc.org/sqlite"
)

var _ metastore.ProfileMetaStore = &InMemoryMetaStore{}

type InMemoryMetaStore struct {
	*sqlMetaStore
}

func NewInMemoryProfileMetaStore(name ...string) (*InMemoryMetaStore, error) {
	dsn := "file::memory:?cache=shared"
	if len(name) > 0 {
		dsn = fmt.Sprintf("file:%s?mode=memory&cache=shared", name[0])
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		db.Close()
		return nil, err
	}

	sqlite := &sqlMetaStore{db}
	if err := sqlite.migrate(); err != nil {
		return nil, fmt.Errorf("migrations failed: %w", err)
	}

	return &InMemoryMetaStore{sqlMetaStore: sqlite}, nil
}
