// Copyright 2026 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package duckdb

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/marcboeker/go-duckdb/v2"
)

// Config holds DuckDB connection configuration.
//
// Path is the on-disk file path. An empty Path uses an in-memory database.
// Table is the name of the profile data table.
type Config struct {
	Path  string
	Table string
}

// Client wraps a DuckDB database/sql connection.
type Client struct {
	db  *sql.DB
	cfg Config
}

// NewClient opens a DuckDB connection at cfg.Path (file) or in memory if
// the path is empty. The on-disk file is created if it doesn't exist.
func NewClient(_ context.Context, cfg Config) (*Client, error) {
	dsn := cfg.Path // empty string == in-memory per duckdb-go convention
	db, err := sql.Open("duckdb", dsn)
	if err != nil {
		return nil, fmt.Errorf("open duckdb: %w", err)
	}

	// DuckDB is single-writer per process. Pin connection count to 1 so
	// every Appender / Query lands on the same connection and we don't
	// race against a transient one. Reads still work fine because the
	// embedded engine is single-process anyway.
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	return &Client{db: db, cfg: cfg}, nil
}

// Close closes the underlying database/sql connection.
func (c *Client) Close() error { return c.db.Close() }

// DB returns the underlying *sql.DB.
func (c *Client) DB() *sql.DB { return c.db }

// Table returns the profile data table name.
func (c *Client) Table() string { return c.cfg.Table }

// EnsureSchema creates the profile data table if it doesn't already exist.
func (c *Client) EnsureSchema(ctx context.Context) error {
	if _, err := c.db.ExecContext(ctx, CreateTableSQL(c.cfg.Table)); err != nil {
		return fmt.Errorf("create table %q: %w", c.cfg.Table, err)
	}
	return nil
}
