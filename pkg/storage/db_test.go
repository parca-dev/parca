// Copyright 2021 The Parca Authors
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

package storage

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime/pprof"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestDB(t *testing.T) {
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"testdb",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	db := OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	ctx := context.Background()
	app1, err := db.Appender(ctx, labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test1"}})
	require.NoError(t, err)

	b := bytes.NewBuffer(nil)
	err = pprof.WriteHeapProfile(b)
	require.NoError(t, err)
	p, err := profile.Parse(b)
	require.NoError(t, err)

	prof1, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)
	require.NoError(t, app1.Append(ctx, prof1))

	app2, err := db.Appender(ctx, labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test2"}})
	require.NoError(t, err)

	b = bytes.NewBuffer(nil)
	err = pprof.WriteHeapProfile(b)
	require.NoError(t, err)
	p, err = profile.Parse(b)
	require.NoError(t, err)

	prof2, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)
	require.NoError(t, app2.Append(ctx, prof2))

	q := db.Querier(
		ctx,
		timestamp.FromTime(time.Now().Add(-10*time.Second)),
		timestamp.FromTime(time.Now().Add(10*time.Second)),
	)
	m, err := labels.NewMatcher(labels.MatchEqual, "namespace", "default")
	require.NoError(t, err)

	set := q.Select(nil, m)
	seen := map[string]bool{
		labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test1"}}.String(): false,
		labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test2"}}.String(): false,
	}
	for set.Next() {
		seen[set.At().Labels().String()] = true
	}
	require.NoError(t, set.Err())

	for labels, seenLabels := range seen {
		if !seenLabels {
			t.Fatalf("expected to see %s but did not", labels)
		}
	}
}

func TestDBConsistency(t *testing.T) {
	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"testdb",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	db := OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	ctx := context.Background()
	app, err := db.Appender(ctx, labels.FromStrings("__name__", "cpu"))
	require.NoError(t, err)

	profiles := []*Profile{}
	dir := "./testdata/many-samples"
	items, err := ioutil.ReadDir(dir)
	require.NoError(t, err)
	j := 0
	for _, item := range items {
		if item.IsDir() {
			continue
		}

		f, err := os.Open(filepath.Join(dir, item.Name()))
		require.NoError(t, err)

		p, err := profile.Parse(f)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		prof, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
		require.NoError(t, err)

		profiles = append(profiles, prof)

		require.NoError(t, app.Append(ctx, prof))
		j++
	}

	q := db.Querier(
		ctx,
		profiles[0].Meta.Timestamp-1,
		profiles[len(profiles)-1].Meta.Timestamp+1,
	)
	m, err := labels.NewMatcher(labels.MatchEqual, "__name__", "cpu")
	require.NoError(t, err)

	set := q.Select(nil, m)
	i := 0
	for set.Next() {
		series := set.At()
		it := series.Iterator()
		for it.Next() {
			p := it.At()
			cp := CopyInstantProfile(p)
			require.NoError(t, validateProfile(cp), "sample #%d not valid", i)
			i++
		}
		require.NoError(t, it.Err())
	}
	require.NoError(t, set.Err())
	require.Equal(t, j, i)
}

func validateProfile(p *Profile) error {
	it := NewProfileTreeIterator(p.Tree)

	for it.HasMore() {
		if it.NextChild() {
			n := it.At().(*ProfileTreeNode)
			cumulative := n.CumulativeValue()
			childrenCumulative := int64(0)
			for _, c := range n.Children {
				childrenCumulative += c.CumulativeValue()
			}
			if childrenCumulative > cumulative {
				stackTrace := []int{}
				for _, e := range it.stack {
					stackTrace = append(stackTrace, int(e.node.LocationID()))
				}
				children := []struct {
					loc int
					cum int
				}{}
				for _, c := range n.Children {
					children = append(children, struct {
						loc int
						cum int
					}{loc: int(c.LocationID()), cum: int(c.CumulativeValue())})
				}
				return fmt.Errorf("unexpected sum %d of children %v at %v, node %d, expected %d", childrenCumulative, children, stackTrace, n.LocationID(), cumulative)
			}

			it.StepInto()
			continue
		}
		it.StepUp()
	}

	return nil
}

func TestSliceSeriesSet(t *testing.T) {
	ss := SliceSeriesSet{
		series: []Series{},
		i:      -1,
	}

	for i := 0; i < 100; i++ {
		ss.series = append(ss.series, &MemSeries{
			id:      uint64(i),
			minTime: 0,
			maxTime: int64(i),
		})
	}

	// Iterate over all series
	for i := 0; i < 100; i++ {
		require.True(t, ss.Next())
		require.Equal(t, &MemSeries{id: uint64(i), maxTime: int64(i)}, ss.At())
	}
	require.NoError(t, ss.Err())
	require.False(t, ss.Next())
}
