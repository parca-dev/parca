// Copyright 2023-2026 The Parca Authors
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
	"context"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/go-kit/log"
)

type Ingester interface {
	Ingest(ctx context.Context, record arrow.RecordBatch) error
}

type Table interface {
	InsertRecord(context.Context, arrow.RecordBatch) (tx uint64, err error)
}

type TableIngester struct {
	logger log.Logger
	table  Table
}

func NewIngester(
	logger log.Logger,
	table Table,
) Ingester {
	return TableIngester{
		logger: logger,
		table:  table,
	}
}

func (ing TableIngester) Ingest(ctx context.Context, record arrow.RecordBatch) error {
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
