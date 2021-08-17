package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

// Make entries ~50B in size, to emulate real-world high cardinality.
const (
	postingsBenchSuffix = "aaaaaaaaaabbbbbbbbbbccccccccccdddddddddd"
)

// for i in {1..10}; do go test -bench=BenchmarkHeadQuerier_Select --benchtime=1000x ./pkg/storage >> pkg/storage/benchmark/head-querier-select.txt; done
// benchstat pkg/storage/benchmark/head-querier-select.txt

func BenchmarkHeadQuerier_Select(b *testing.B) {
	ctx := context.Background()
	h := NewHead()

	numSeries := 10_000
	//numSeries := 1_000_000 // TODO: It's too much yet
	for i := 1; i <= numSeries; i++ {
		app, err := h.Appender(ctx, labels.FromStrings("foo", "bar", "s", fmt.Sprintf("%d%s", i, postingsBenchSuffix)))
		require.NoError(b, err)
		err = app.Append(&Profile{
			Tree: NewProfileTree(),
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
