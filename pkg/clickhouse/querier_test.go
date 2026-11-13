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
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/parca-dev/parca/pkg/symbolizer"
)

type fakeRows struct {
	rows []sampleData
	pos  int
}

func (f *fakeRows) Next() bool {
	f.pos++
	return f.pos <= len(f.rows)
}

func (f *fakeRows) Err() error { return nil }

func (f *fakeRows) Scan(dest ...interface{}) error {
	s := f.rows[f.pos-1]
	*dest[0].(*[]uint64) = s.addresses
	*dest[1].(*[]uint64) = s.mappingStarts
	*dest[2].(*[]uint64) = s.mappingLimits
	*dest[3].(*[]uint64) = s.mappingOffsets
	*dest[4].(*[]string) = s.mappingFiles
	*dest[5].(*[]string) = s.mappingBuildIDs
	*dest[6].(*[]int64) = s.lineNumbers
	*dest[7].(*[]string) = s.functionNames
	*dest[8].(*[]string) = s.functionSystemNames
	*dest[9].(*[]string) = s.functionFilenames
	*dest[10].(*[]int64) = s.functionStartLines
	*dest[11].(*int64) = s.value
	*dest[12].(*string) = s.labelsJSON
	*dest[13].(*int64) = s.duration
	*dest[14].(*int64) = s.period
	return nil
}

type noopSymbolizer struct{}

func (noopSymbolizer) Symbolize(context.Context, symbolizer.SymbolizationRequest) error {
	return nil
}

func TestRowsToArrowRecordsStoredFunctions(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	defer mem.AssertSize(t, 0)

	q := &Querier{
		logger:     log.NewNopLogger(),
		tracer:     noop.NewTracerProvider().Tracer("test"),
		mem:        mem,
		symbolizer: noopSymbolizer{},
	}

	rows := &fakeRows{rows: []sampleData{{
		addresses:           []uint64{0x1, 0x2},
		mappingStarts:       []uint64{0, 0},
		mappingLimits:       []uint64{0, 0},
		mappingOffsets:      []uint64{0, 0},
		mappingFiles:        []string{"a", "a"},
		mappingBuildIDs:     []string{"", ""},
		lineNumbers:         []int64{10, 20},
		functionNames:       []string{"foo", "bar"},
		functionSystemNames: []string{"foo", "bar"},
		functionFilenames:   []string{"foo.go", "bar.go"},
		functionStartLines:  []int64{1, 2},
		value:               1,
		period:              1,
	}}}

	records, err := q.rowsToArrowRecords(context.Background(), rows, false)
	require.NoError(t, err)
	require.Len(t, records, 1)
	defer records[0].Release()
	require.EqualValues(t, 1, records[0].NumRows())
}
