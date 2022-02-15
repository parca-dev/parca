package columnstore

import (
	"bytes"
	"context"
	"io/ioutil"
	"path"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func BenchmarkPartMerge(b *testing.B) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := New(reg)
	colDB := col.DB("parca")
	schema := ParcaProfilingTableSchema()
	table := colDB.Table("stacktraces", schema, logger)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	b.Cleanup(func() {
		m.Close()
	})

	dir := "./testdata/merge-bench/"
	dirs, err := ioutil.ReadDir(dir)
	require.NoError(b, err)

	rowNum := 0
	for _, labelDir := range dirs {
		if labelDir.Name() == "README.md" {
			continue
		}

		ls, err := parser.ParseMetric(labelDir.Name())
		require.NoError(b, err, "failed to parse label %s", labelDir.Name())

		files, err := ioutil.ReadDir(path.Join(dir, labelDir.Name()))
		require.NoError(b, err)
		for _, f := range files {
			fullPath := path.Join(dir, labelDir.Name(), f.Name())
			fileContent, err := ioutil.ReadFile(fullPath)
			require.NoError(b, err)
			p, err := profile.Parse(bytes.NewBuffer(fileContent))
			require.NoError(b, err)

			profiles, err := parcaprofile.FlatProfilesFromPprof(ctx, logger, m, p)
			require.NoError(b, err)
			for _, prof := range profiles {
				n, err := InsertProfileIntoTable(ctx, logger, table, ls, prof)
				require.NoError(b, err)
				rowNum += n
			}
		}
	}

	require.Equal(b, 4187, rowNum)
	require.Equal(b, 1, table.index.Len())

	g := table.index.Min().(*Granule)
	require.Equal(b, 146, len(g.parts))

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		its := make([]*PartIterator, 0, len(g.parts))
		for _, p := range g.parts {
			its = append(its, p.Iterator())
		}

		p, err := merge(0, &schema, its)
		require.NoError(b, err)
		require.Equal(b, rowNum, p.Cardinality)
	}
}
