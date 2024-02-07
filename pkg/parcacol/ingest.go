// Copyright 2022-2024 The Parca Authors
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

package parcacol

import (
	"bytes"
	"context"
	"sync"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/dynparquet"

	"github.com/parca-dev/parca/pkg/normalizer"
)

type Table interface {
	Schema() *dynparquet.Schema
	InsertRecord(context.Context, arrow.Record) (tx uint64, err error)
}

type Ingester struct {
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
	return Ingester{
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

func (ing Ingester) Close() error { return nil }

func (ing Ingester) Ingest(ctx context.Context, req normalizer.NormalizedWriteRawRequest) error {
	r, err := SeriesToArrowRecord(
		ing.mem,
		ing.schema,
		req.Series,
		req.AllLabelNames,
		req.AllPprofLabelNames,
		req.AllPprofNumLabelNames,
	)
	if err != nil {
		return err
	}
	defer r.Release()

	if _, err := ing.table.InsertRecord(ctx, r); err != nil {
		return err
	}
	return nil
}
