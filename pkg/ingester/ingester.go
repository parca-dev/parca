// Copyright 2023-2024 The Parca Authors
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

package ingester

import (
	"bytes"
	"context"
	"sync"

	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/array"
	"github.com/apache/arrow/go/v16/arrow/memory"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/dynparquet"

	"github.com/parca-dev/parca/pkg/normalizer"
)

type Ingester interface {
	Ingest(ctx context.Context, req normalizer.NormalizedWriteRawRequest) error
	Close() error
}

type Table interface {
	Schema() *dynparquet.Schema
	InsertRecord(context.Context, arrow.Record) (tx uint64, err error)
}

type TableIngester struct {
	logger     log.Logger
	mem        memory.Allocator
	table      Table
	schema     *dynparquet.Schema
	bufferPool *sync.Pool
}

func NewIngester(
	logger log.Logger,
	mem memory.Allocator,
	table Table,
	schema *dynparquet.Schema,
) Ingester {
	return TableIngester{
		logger: logger,
		mem:    mem,
		table:  table,
		schema: schema,
		bufferPool: &sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
	}
}

func (ing TableIngester) Close() error { return nil }

func (ing TableIngester) Ingest(ctx context.Context, req normalizer.NormalizedWriteRawRequest) error {
	// Read sorted rows into an arrow record
	record, err := normalizer.ParquetBufToArrowRecord(ctx, ing.mem, ing.schema, req)
	if err != nil {
		return err
	}
	if record == nil {
		return nil
	}
	defer record.Release()

	if record.NumRows() == 0 {
		return nil
	}

	for _, col := range record.Columns() {
		switch col := col.(type) {
		case *array.Dictionary:
			// Dictionaries are lazily initialized, we need to do this here
			// to make them concurrency safe. This should be solved
			// upstream, but for now this is a fix to avoid data races with
			// what we have.
			col.Dictionary()
		default:
			// Do nothing
		}
	}

	if _, err := ing.table.InsertRecord(ctx, record); err != nil {
		return err
	}

	return nil
}
