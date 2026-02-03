// Copyright 2024-2026 The Parca Authors
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

package clickhouse

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

// Config holds ClickHouse connection configuration.
type Config struct {
	Address  string
	Database string
	Username string
	Password string
	Table    string
	Secure   bool
}

// Client is a wrapper around the ClickHouse connection.
type Client struct {
	conn driver.Conn
	cfg  Config
}

// NewClient creates a new ClickHouse client with the given configuration.
// It first connects without a database to ensure the database can be created,
// then reconnects with the database specified.
func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	// First, connect without specifying a database to allow database creation
	opts := &clickhouse.Options{
		Addr: []string{cfg.Address},
		Auth: clickhouse.Auth{
			Username: cfg.Username,
			Password: cfg.Password,
		},
	}

	if cfg.Secure {
		opts.TLS = &tls.Config{}
	}

	conn, err := clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse: %w", err)
	}

	// Create database if it doesn't exist
	if err := conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", cfg.Database)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	// Close the initial connection
	conn.Close()

	// Now connect with the database specified
	opts.Auth.Database = cfg.Database
	conn, err = clickhouse.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to open ClickHouse connection with database: %w", err)
	}

	if err := conn.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping ClickHouse with database: %w", err)
	}

	return &Client{
		conn: conn,
		cfg:  cfg,
	}, nil
}

// Close closes the ClickHouse connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// Conn returns the underlying ClickHouse connection.
func (c *Client) Conn() driver.Conn {
	return c.conn
}

// Config returns the client configuration.
func (c *Client) Config() Config {
	return c.cfg
}

// Database returns the database name.
func (c *Client) Database() string {
	return c.cfg.Database
}

// Table returns the table name.
func (c *Client) Table() string {
	return c.cfg.Table
}

// FullTableName returns the fully qualified table name (database.table).
func (c *Client) FullTableName() string {
	return fmt.Sprintf("%s.%s", c.cfg.Database, c.cfg.Table)
}

// EnsureSchema creates the table if it doesn't exist.
// Note: The database is already created in NewClient.
func (c *Client) EnsureSchema(ctx context.Context) error {
	// Create table using the schema definition
	schema := CreateTableSQL(c.cfg.Database, c.cfg.Table)
	if err := c.conn.Exec(ctx, schema); err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	return nil
}

// Query executes a query and returns the rows.
func (c *Client) Query(ctx context.Context, query string, args ...interface{}) (driver.Rows, error) {
	return c.conn.Query(ctx, query, args...)
}

// Exec executes a query without returning rows.
func (c *Client) Exec(ctx context.Context, query string, args ...interface{}) error {
	return c.conn.Exec(ctx, query, args...)
}

// PrepareBatch prepares a batch for insertion.
func (c *Client) PrepareBatch(ctx context.Context, query string) (driver.Batch, error) {
	return c.conn.PrepareBatch(ctx, query)
}
