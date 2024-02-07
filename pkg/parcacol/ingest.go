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
	"github.com/go-kit/log/level"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"

	"github.com/parca-dev/parca/pkg/normalizer"
	"github.com/parca-dev/parca/pkg/profile"
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
	pBuf, err := ing.schema.GetBuffer(map[string][]string{
		profile.ColumnLabels:         req.AllLabelNames,
		profile.ColumnPprofLabels:    req.AllPprofLabelNames,
		profile.ColumnPprofNumLabels: req.AllPprofNumLabelNames,
	})
	if err != nil {
		return err
	}
	defer ing.schema.PutBuffer(pBuf)

	var r parquet.Row
	for _, s := range req.Series {
		for _, normalizedProfiles := range s.Samples {
			for _, p := range normalizedProfiles {
				if len(p.Samples) == 0 {
					ls := labels.FromMap(s.Labels)
					level.Debug(ing.logger).Log("msg", "no samples found in profile, dropping it", "name", p.Meta.Name, "sample_type", p.Meta.SampleType.Type, "sample_unit", p.Meta.SampleType.Unit, "labels", ls)
					continue
				}

				for _, profileSample := range p.Samples {
					r = SampleToParquetRow(
						ing.schema,
						r[:0],
						req.AllLabelNames,
						req.AllPprofLabelNames,
						req.AllPprofNumLabelNames,
						s.Labels,
						p.Meta,
						profileSample,
					)
					_, err := pBuf.WriteRows([]parquet.Row{r})
					if err != nil {
						return err
					}
				}
			}
		}
	}

	pBuf.Sort()

	// Read sorted rows into an arrow record
	records, err := ParquetBufToArrowRecord(ctx, ing.mem, pBuf.Buffer, ing.schema, 0)
	if err != nil {
		return err
	}
	defer func() {
		for _, record := range records {
			record.Release()
		}
	}()

	for _, record := range records {
		if record.NumRows() == 0 {
			return nil
		}

		if _, err := ing.table.InsertRecord(ctx, record); err != nil {
			return err
		}
	}
	return nil
}
