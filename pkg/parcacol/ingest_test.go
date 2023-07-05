// Copyright 2022-2023 The Parca Authors
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
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"testing"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
)

func MustReadAllGzip(t require.TestingT, filename string) []byte {
	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

type fakeTable struct {
	schema *dynparquet.Schema

	inserts [][]byte
}

func (t *fakeTable) Schema() *dynparquet.Schema {
	return t.schema
}

func (t *fakeTable) InsertRecord(ctx context.Context, record arrow.Record) (uint64, error) {
	return 0, nil
}

func (t *fakeTable) Insert(ctx context.Context, data []byte) (uint64, error) {
	cpy := make([]byte, len(data))
	copy(cpy, data)
	t.inserts = append(t.inserts, cpy)

	return 0, nil
}

func TestPprofToParquet(t *testing.T) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx := context.Background()

	schema, err := Schema()
	require.NoError(t, err)

	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)
	metastore := metastore.NewInProcessClient(m)

	fileContent, err := os.ReadFile("../query/testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	tests := map[string]struct {
		arrow bool
	}{
		"parquet": {false},
		"arrow":   {true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.arrow {
				ExperimentalArrow = true
				t.Cleanup(func() {
					ExperimentalArrow = false
				})
			}

			table := &fakeTable{
				schema: schema,
			}
			req := &profilestorepb.WriteRawRequest{
				Series: []*profilestorepb.RawProfileSeries{{
					Labels: &profilestorepb.LabelSet{
						Labels: []*profilestorepb.Label{
							{
								Name:  "__name__",
								Value: "memory",
							},
							{
								Name:  "job",
								Value: "default",
							},
						},
					},
					Samples: []*profilestorepb.RawSample{{
						RawProfile: fileContent,
					}},
				}},
			}
			err := NormalizedIngest(
				ctx,
				req,
				logger,
				table,
				schema,
				metastore,
				&sync.Pool{
					New: func() interface{} {
						return bytes.NewBuffer(nil)
					},
				},
				true,
			)
			require.NoError(t, err)

			for i, insert := range table.inserts {
				serBuf, err := dynparquet.ReaderFromBytes(insert)
				require.NoError(t, err)

				rows := serBuf.Reader()
				rowBuf := []parquet.Row{{}}
				for {
					_, err := rows.ReadRows(rowBuf)
					if err == io.EOF {
						break
					}
					if err != io.EOF {
						if err != nil {
							require.NoError(t, os.WriteFile(fmt.Sprintf("test-%d.parquet", i), insert, 0o777))
						}
						require.NoError(t, err)
					}
				}
			}
		})
	}
}

func TestUncompressedPprofToParquet(t *testing.T) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx := context.Background()

	schema, err := Schema()
	require.NoError(t, err)

	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)
	metastore := metastore.NewInProcessClient(m)

	fileContent, err := os.ReadFile("../query/testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	r, err := gzip.NewReader(bytes.NewReader(fileContent))
	require.NoError(t, err)

	fileContent, err = io.ReadAll(r)
	require.NoError(t, err)
	require.NoError(t, r.Close())

	tests := map[string]struct {
		arrow bool
	}{
		"parquet": {false},
		"arrow":   {true},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.arrow {
				ExperimentalArrow = true
				t.Cleanup(func() {
					ExperimentalArrow = false
				})
			}

			table := &fakeTable{
				schema: schema,
			}
			req := &profilestorepb.WriteRawRequest{
				Series: []*profilestorepb.RawProfileSeries{{
					Labels: &profilestorepb.LabelSet{
						Labels: []*profilestorepb.Label{
							{
								Name:  "__name__",
								Value: "memory",
							},
							{
								Name:  "job",
								Value: "default",
							},
						},
					},
					Samples: []*profilestorepb.RawSample{{
						RawProfile: fileContent,
					}},
				}},
			}
			err := NormalizedIngest(
				ctx,
				req,
				logger,
				table,
				schema,
				metastore,
				&sync.Pool{
					New: func() interface{} {
						return bytes.NewBuffer(nil)
					},
				},
				true,
			)
			require.NoError(t, err)

			for i, insert := range table.inserts {
				serBuf, err := dynparquet.ReaderFromBytes(insert)
				require.NoError(t, err)

				rows := serBuf.Reader()
				rowBuf := []parquet.Row{{}}
				for {
					_, err := rows.ReadRows(rowBuf)
					if err == io.EOF {
						break
					}
					if err != io.EOF {
						if err != nil {
							require.NoError(t, os.WriteFile(fmt.Sprintf("test-%d.parquet", i), insert, 0o777))
						}
						require.NoError(t, err)
					}
				}
			}
		})
	}
}

func BenchmarkNormalizeWriteRawRequest(b *testing.B) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	ctx := context.Background()

	m := metastoretest.NewTestMetastore(
		b,
		logger,
		reg,
		tracer,
	)
	metastore := metastore.NewInProcessClient(m)

	fileContent, err := os.ReadFile("../query/testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	normalizer := NewNormalizer(metastore, true)
	req := &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "memory",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err = NormalizeWriteRawRequest(ctx, normalizer, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
