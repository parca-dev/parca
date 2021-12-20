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
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

// Make entries ~50B in size, to emulate real-world high cardinality.
const (
	postingsBenchSuffix = "aaaaaaaaaabbbbbbbbbbccccccccccdddddddddd"
)

// for i in {1..10}; do go test -bench=BenchmarkHeadQuerier_Select --benchtime=1000x ./pkg/storage >> pkg/storage/benchmark/head-querier-select.txt; done
// benchstat pkg/storage/benchmark/head-querier-select.txt

func BenchmarkHeadQuerier_Select(b *testing.B) {
	ctx := context.Background()
	h := NewHead(nil, trace.NewNoopTracerProvider().Tracer(""), nil)

	numSeries := 1_000_000
	for i := 1; i <= numSeries; i++ {
		app, err := h.Appender(ctx, labels.FromStrings("foo", "bar", "s", fmt.Sprintf("%d%s", i, postingsBenchSuffix)))
		require.NoError(b, err)
		err = app.AppendFlat(ctx, &FlatProfile{
			Meta: InstantProfileMeta{
				Timestamp: int64(i),
			},
		})
		require.NoError(b, err)
	}

	matcher := labels.MustNewMatcher(labels.MatchEqual, "foo", "bar")

	for s := 1; s <= numSeries; s *= 10 {
		b.Run(fmt.Sprintf("%dof%d", s, numSeries), func(b *testing.B) {
			q := h.Querier(ctx, 0, int64(s))

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				ss := q.Select(nil, matcher)
				for ss.Next() {
				}
				require.NoError(b, ss.Err())
			}
		})
	}
}
