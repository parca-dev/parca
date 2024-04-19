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

package normalizer

import (
	"context"
	"fmt"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/memory"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow"
	"github.com/polarsignals/frostdb/query/logicalplan"

	"github.com/parca-dev/parca/pkg/profile"
)

// ParquetBufToArrowRecord converts a parquet buffer to an arrow record. If rowsPerRecord is 0, then the entire buffer is converted to a single record.
func ParquetBufToArrowRecord(ctx context.Context, mem memory.Allocator, s *dynparquet.Schema, normalizedRequest NormalizedWriteRawRequest) (arrow.Record, error) {
	// Create a buffer with all possible labels, pprof labels and pprof num labels as dynamic columns.
	// We use NewBuffer instead of GetBuffer here since analysis showed a very
	// low hit rate, meaning buffers were GCed faster than they could be reused.
	// The downside of using a pool is that buffers are held around for longer.
	// Using NewBuffer means that we pay the price of reallocating a buffer,
	// but they get GCed a lot sooner.
	buffer, err := s.NewBuffer(map[string][]string{
		profile.ColumnLabels:         normalizedRequest.AllLabelNames,
		profile.ColumnPprofLabels:    normalizedRequest.AllPprofLabelNames,
		profile.ColumnPprofNumLabels: normalizedRequest.AllPprofNumLabelNames,
	})
	if err != nil {
		return nil, err
	}

	var row parquet.Row
	for _, series := range normalizedRequest.Series {
		for _, sample := range series.Samples {
			for _, p := range sample {
				for _, ns := range p.Samples {
					row = SampleToParquetRow(
						s,
						row[:0],
						normalizedRequest.AllLabelNames,
						normalizedRequest.AllPprofLabelNames,
						normalizedRequest.AllPprofNumLabelNames,
						series.Labels,
						p.Meta,
						ns,
					)
					if _, err := buffer.WriteRows([]parquet.Row{row}); err != nil {
						return nil, fmt.Errorf("failed to write row to buffer: %w", err)
					}
				}
			}
		}
	}

	if buffer.NumRows() == 0 {
		// If there are no rows in the buffer we simply return early
		return nil, nil
	}

	// We need to sort the buffer so the rows are inserted in sorted order later
	// on the storage nodes.
	buffer.Sort()

	// Convert the sorted buffer to an arrow record.
	converter := pqarrow.NewParquetConverter(memory.NewGoAllocator(), logicalplan.IterOptions{})
	defer converter.Close()

	if err := converter.Convert(ctx, buffer, s); err != nil {
		return nil, fmt.Errorf("failed to convert parquet to arrow: %w", err)
	}

	return converter.NewRecord(), nil
}
