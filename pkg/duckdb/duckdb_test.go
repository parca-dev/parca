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

package duckdb_test

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/parca-dev/parca/pkg/duckdb"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbolizer"
)

// nopSymbolizer is a no-op symbolizer for tests — synthetic rows ingested
// here already carry resolved function info.
type nopSymbolizer struct{}

func (nopSymbolizer) Symbolize(context.Context, symbolizer.SymbolizationRequest) error { return nil }

// buildSampleRecord builds a one-row Arrow record matching the parca write
// schema. Caller must Release() the returned record.
func buildSampleRecord(t *testing.T, mem memory.Allocator, ts int64) arrow.RecordBatch {
	t.Helper()

	schema := profile.BuildArrowSchema([]string{"job"})
	b := array.NewRecordBuilder(mem, schema)
	defer b.Release()

	for i, field := range schema.Fields() {
		switch field.Name {
		case profile.ColumnDuration:
			b.Field(i).(*array.Int64Builder).Append(int64(time.Second))
		case profile.ColumnName:
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("process_cpu"))
		case profile.ColumnPeriod:
			b.Field(i).(*array.Int64Builder).Append(10_000_000)
		case profile.ColumnPeriodType:
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("cpu"))
		case profile.ColumnPeriodUnit:
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("nanoseconds"))
		case profile.ColumnSampleType:
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("cpu"))
		case profile.ColumnSampleUnit:
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("nanoseconds"))
		case profile.ColumnStacktrace:
			lb := b.Field(i).(*array.ListBuilder)
			vb := lb.ValueBuilder().(*array.BinaryDictionaryBuilder)
			lb.Append(true)
			require.NoError(t, vb.Append(encodeLocation(0xdeadbeef, "test-build-id", "/lib/test", "main.run")))
		case profile.ColumnTimestamp:
			b.Field(i).(*array.Int64Builder).Append(ts)
		case profile.ColumnTimeNanos:
			b.Field(i).(*array.Int64Builder).Append(ts * int64(time.Millisecond))
		case profile.ColumnValue:
			b.Field(i).(*array.Int64Builder).Append(42)
		case profile.ColumnLabelsPrefix + "job":
			require.NoError(t, b.Field(i).(*array.BinaryDictionaryBuilder).AppendString("test"))
		}
	}

	return b.NewRecordBatch()
}

// encodeLocation produces a varint-encoded location blob in the same
// shape produced by the symbolizer and decoded by the ingester.
//
// Layout (matches pkg/profile + pkg/clickhouse decoders):
//
//	addr (uvarint)
//	numLines (uvarint)
//	hasMapping byte (0|1)
//	  if hasMapping: buildID (len+bytes), filename (len+bytes), 3 zero uvarints
//	per line:
//	  lineNumber (uvarint)
//	  hasFunction byte (0|1)
//	  if hasFunction: startLine (uvarint), name, systemName, filename (each len+bytes)
func encodeLocation(addr uint64, buildID, mappingFile, fnName string) []byte {
	var out []byte
	out = appendUvarint(out, addr)
	out = appendUvarint(out, 1) // 1 line
	out = append(out, 0x01)     // hasMapping
	out = appendBytes(out, []byte(buildID))
	out = appendBytes(out, []byte(mappingFile))
	out = appendUvarint(out, 0) // memoryStart
	out = appendUvarint(out, 0) // memoryLength
	out = appendUvarint(out, 0) // mappingOffset

	out = appendUvarint(out, 7) // line number
	out = append(out, 0x01)     // hasFunction
	out = appendUvarint(out, 1) // startLine
	out = appendBytes(out, []byte(fnName))
	out = appendBytes(out, []byte(fnName)) // system name
	out = appendBytes(out, []byte("main.go"))
	return out
}

func appendUvarint(b []byte, v uint64) []byte {
	for v >= 0x80 {
		b = append(b, byte(v)|0x80)
		v >>= 7
	}
	return append(b, byte(v))
}

func appendBytes(b []byte, payload []byte) []byte {
	b = appendUvarint(b, uint64(len(payload)))
	return append(b, payload...)
}

func newTestClient(t *testing.T) *duckdb.Client {
	t.Helper()
	c, err := duckdb.NewClient(context.Background(), duckdb.Config{Path: "", Table: "stacktraces"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = c.Close() })
	require.NoError(t, c.EnsureSchema(context.Background()))
	return c
}

// TestIngestAndQueryRoundTrip ingests one synthetic profile row and
// verifies the live Querier methods return sensible results against it.
func TestIngestAndQueryRoundTrip(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	client := newTestClient(t)
	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := noop.NewTracerProvider().Tracer("")

	const tsMillis int64 = 1_700_000_000_000
	rec := buildSampleRecord(t, mem, tsMillis)
	defer rec.Release()

	ing := duckdb.NewIngester(logger, client)
	require.NoError(t, ing.Ingest(ctx, rec))

	q := duckdb.NewQuerier(client, logger, tracer, mem, nopSymbolizer{})

	// HasProfileData / ProfileTypes
	has, err := q.HasProfileData(ctx)
	require.NoError(t, err)
	require.True(t, has)

	types, err := q.ProfileTypes(ctx, time.UnixMilli(0), time.UnixMilli(0))
	require.NoError(t, err)
	require.Len(t, types, 1)
	require.Equal(t, "process_cpu", types[0].Name)
	require.Equal(t, "cpu", types[0].SampleType)
	require.True(t, types[0].Delta)

	// Labels / Values
	labels, err := q.Labels(ctx, nil, time.UnixMilli(0), time.UnixMilli(0), "")
	require.NoError(t, err)
	require.Equal(t, []string{"job"}, labels)

	values, err := q.Values(ctx, "job", nil, time.UnixMilli(0), time.UnixMilli(0), "")
	require.NoError(t, err)
	sort.Strings(values)
	require.Equal(t, []string{"test"}, values)

	// QueryRange — wide window covering tsMillis.
	start := time.UnixMilli(tsMillis - 1_000)
	end := time.UnixMilli(tsMillis + 1_000)
	queryStr := `process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{job="test"}`
	series, err := q.QueryRange(ctx, queryStr, start, end, time.Second, 0, []string{"job"})
	require.NoError(t, err)
	require.Len(t, series, 1)
	require.Equal(t, "job", series[0].Labelset.Labels[0].Name)
	require.Equal(t, "test", series[0].Labelset.Labels[0].Value)
	require.NotEmpty(t, series[0].Samples)
	require.Equal(t, int64(42), series[0].Samples[0].Value)

	// QuerySingle — exact timestamp match.
	single, err := q.QuerySingle(ctx, queryStr, time.UnixMilli(tsMillis), false)
	require.NoError(t, err)
	require.NotEmpty(t, single.Samples)
	for _, r := range single.Samples {
		r.Release()
	}

	// QueryMerge — sum across the window.
	merged, err := q.QueryMerge(ctx, queryStr, start, end, []string{"job"}, false, "")
	require.NoError(t, err)
	require.NotEmpty(t, merged.Samples)
	for _, r := range merged.Samples {
		r.Release()
	}

	// Metadata helpers
	mappings, err := q.GetProfileMetadataMappings(ctx, queryStr, start, end)
	require.NoError(t, err)
	require.Equal(t, []string{"/lib/test"}, mappings)

	metaLabels, err := q.GetProfileMetadataLabels(ctx, queryStr, start, end)
	require.NoError(t, err)
	require.Equal(t, []string{"job"}, metaLabels)
}
