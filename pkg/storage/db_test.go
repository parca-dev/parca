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
	"runtime/pprof"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestDB(t *testing.T) {
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})
	db := OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	ctx := context.Background()
	app1, err := db.Appender(ctx, labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test1"}})
	require.NoError(t, err)

	b := bytes.NewBuffer(nil)
	err = pprof.WriteHeapProfile(b)
	require.NoError(t, err)
	p, err := profile.Parse(b)
	require.NoError(t, err)

	prof1, err := parcaprofile.FromPprof(ctx, log.NewNopLogger(), l, p, 0, false)
	require.NoError(t, err)
	require.NoError(t, app1.AppendFlat(ctx, prof1))

	app2, err := db.Appender(ctx, labels.Labels{{Name: "namespace", Value: "default"}, {Name: "container", Value: "test2"}})
	require.NoError(t, err)

	b = bytes.NewBuffer(nil)
	err = pprof.WriteHeapProfile(b)
	require.NoError(t, err)
	p, err = profile.Parse(b)
	require.NoError(t, err)

	prof2, err := parcaprofile.FromPprof(ctx, log.NewNopLogger(), l, p, 0, false)
	require.NoError(t, err)
	require.NoError(t, app2.AppendFlat(ctx, prof2))

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

//func TestSliceSeriesSet(t *testing.T) {
//	ss := SliceSeriesSet{
//		series: []Series{},
//		i:      -1,
//	}
//
//	for i := 0; i < 100; i++ {
//		ss.series = append(ss.series, &MemSeries{
//			id:      uint64(i),
//			minTime: 0,
//			maxTime: int64(i),
//		})
//	}
//
//	// Iterate over all series
//	for i := 0; i < 100; i++ {
//		require.True(t, ss.Next())
//		require.Equal(t, &MemSeries{id: uint64(i), maxTime: int64(i)}, ss.At())
//	}
//	require.NoError(t, ss.Err())
//	require.False(t, ss.Next())
//}
