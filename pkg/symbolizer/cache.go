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

package symbolizer

import (
	"context"
	"fmt"

	"github.com/dgraph-io/badger/v4"

	"github.com/parca-dev/parca/pkg/profile"
)

type BadgerCache struct {
	db *badger.DB
}

func NewBadgerCache(db *badger.DB) *BadgerCache {
	return &BadgerCache{db: db}
}

func (c *BadgerCache) Get(ctx context.Context, buildID string, addr uint64) ([]profile.LocationLine, bool, error) {
	var (
		found bool
		res   []profile.LocationLine
	)
	err := c.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(c.makeKey(buildID, addr))
		if err == badger.ErrKeyNotFound {
			return nil
		}
		if err != nil {
			return fmt.Errorf("get badger: %w", err)
		}

		if err := item.Value(func(val []byte) error {
			res = decodeLines(val)
			found = true
			return nil
		}); err != nil {
			return fmt.Errorf("badger value: %w", err)
		}

		return nil
	})
	if err != nil {
		return nil, false, fmt.Errorf("view badger: %w", err)
	}

	return res, found, nil
}

func (c *BadgerCache) makeKey(buildID string, addr uint64) []byte {
	return []byte(buildID + "/" + fmt.Sprintf("0x%x", addr))
}

func (c *BadgerCache) Set(ctx context.Context, buildID string, addr uint64, lines []profile.LocationLine) error {
	return c.db.Update(func(txn *badger.Txn) error {
		return txn.Set(c.makeKey(buildID, addr), encodeLines(lines))
	})
}
