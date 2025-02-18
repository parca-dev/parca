// Copyright 2025 The Parca Authors
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

package query

import (
	"fmt"
	"bytes"
	"context"

	"github.com/apache/arrow/go/v17/arrow/math"
	"github.com/apache/arrow/go/v17/arrow/ipc"
	"github.com/apache/arrow/go/v17/arrow/memory"
	queryv1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func GenerateFlamegraphSandwich(
	ctx context.Context,
	mem memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
	groupBy []string,
	trimFraction float32,
) (*queryv1alpha1.FlamegraphArrow, int64, error) {
	ctx, span := tracer.Start(ctx, "GenerateFlamegraphSandwich")
	defer span.End()

	record, cumulative, height, trimmed, err := generateFlamegraphSandwichRecord(ctx, mem, tracer, p, groupBy, trimFraction)
	if err != nil {
		return nil, 0, err
	}
	defer record.Release()

	// TODO: Reuse buffer and potentially writers
	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(mem),
	)
	defer w.Close()

	if err = w.Write(record); err != nil {
		return nil, 0, err
	}

	span.SetAttributes(attribute.Int("record_size", buf.Len()))
	if buf.Len() > 1<<24 { // 16MiB
		span.SetAttributes(attribute.String("record_stats", recordStats(record)))
	}

	return &queryv1alpha1.FlamegraphArrow{
		Record:  buf.Bytes(),
		Unit:    p.Meta.SampleType.Unit,
		Height:  height, // add one for the root
		Trimmed: trimmed,
	}, cumulative, nil
}

func generateFlamegraphSandwichRecord(ctx context.Context, mem memory.Allocator, tracer trace.Tracer, p profile.Profile, groupBy []string, trimFraction float32) (arrow.Record, int64, int32, int64, error) {
	ctx, span := tracer.Start(ctx, "generateFlamegraphSandwichRecord")
	defer span.End()

	totalRows := int64(0)
	for _, r := range p.Samples {
		totalRows += r.NumRows()
	}

	fb, err := newFlamegraphBuilder(mem, totalRows, groupBy)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("create flamegraph builder: %w", err)
	}
	defer fb.Release()

	// these change with every iteration below
	row := fb.builderCumulative.Len()

	profileReader := profile.NewReader(p)
	labelHasher := xxh3.New()
	for _, r := range profileReader.RecordReaders {
		fb.cumulative += math.Int64.Sum(r.Value)
		fb.diff += math.Int64.Sum(r.Diff)

		// This field compares the current sample with the already added values in the builders.
		for i := 0; i < int(r.Record.NumRows()); i++ {
			beg, end := r.Locations.ValueOffsets(i)

			// TODO: Add support for injecting root label rows

			// every new sample resets the childRow to -1 indicating that we start with a leaf again.
			// pprof stores locations in reverse order, thus we loop over locations in reverse order.
			for j := int(end - 1); j >= int(beg); j-- {
				if r.Locations.ListValues().IsNull(j) {
					continue // skip null values; these have been filtered out.
				}

				// iterate over lines
					// iterator over functions names
						// match our function name we filter for
						// set the level to the function's index in the locations
						// functionNameLevel = 1
						// break

				// iterate from functionNameLevel to the location beginning (down)
					// compare each stack to fb.callees
						// append or merge

				// iterate from functionNameLevel to the location end (up)
					// compare each stack to fb.callers
						// append or merge
			}

			// filter for foo
			// [["bar"], ["foo"], ["main"]] 123 42
			// [["baz"], ["foo"], ["main"]] 124 23


			// flame graph with children (height)
			// 0 main [1]
			// 1 foo [2,3]
			// 2 bar
			// 3 baz

			// sandwich with callers and callees (top,bottom)
			// 0 foo [1][2,3]
			// 1 main [][]
			// 2 bar [][]
			// 3 baz [][]


		}
	}
}
