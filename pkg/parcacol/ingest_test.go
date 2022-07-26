package parcacol

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
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
	content, err := ioutil.ReadAll(r)
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

	nps, err := NewNormalizer(metastore).NormalizePprof(ctx, "memory", p, false)
	require.NoError(t, err)

	for i, np := range nps {
		buf, err := NormalizedProfileToParquetBuffer(schema, labels.Labels{}, np)
		require.NoError(t, err)

		b, err := schema.SerializeBuffer(buf)
		require.NoError(t, err)

		serBuf, err := dynparquet.ReaderFromBytes(b)
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
					require.NoError(t, os.WriteFile(fmt.Sprintf("test-%d.parquet", i), b, 0o777))
				}
				require.NoError(t, err)
			}
		}
	}
}
