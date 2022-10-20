// Copyright 2022 The Parca Authors
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
	"testing"

	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/segmentio/parquet-go"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
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

	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(MustReadAllGzip(t, "../query/testdata/alloc_objects.pb.gz")))

	nps, err := NewNormalizer(metastore).NormalizePprof(ctx, "memory", map[string]struct{}{}, p, false)
	require.NoError(t, err)

	for i, np := range nps {
		buf := bytes.NewBuffer(nil)
		require.NoError(t, NormalizedProfileToParquetBuffer(buf, schema, labels.Labels{}, np))

		serBuf, err := dynparquet.ReaderFromBytes(buf.Bytes())
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
					require.NoError(t, os.WriteFile(fmt.Sprintf("test-%d.parquet", i), buf.Bytes(), 0o777))
				}
				require.NoError(t, err)
			}
		}
	}
}
