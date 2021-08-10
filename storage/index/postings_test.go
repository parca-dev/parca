package index

import (
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestMemPostings(t *testing.T) {
	p := NewMemPostings()
	p.Add(42, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test1"}})
	p.Add(123, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test2"}})

	seen := make([]uint64, 0, 2)
	it := p.Get("foo", "bar")
	for it.Next() {
		seen = append(seen, it.At())
	}
	require.NoError(t, it.Err())
	require.Equal(t, []uint64{42, 123}, seen)

	seen = make([]uint64, 0, 1)
	it = p.Get("container", "test1")
	for it.Next() {
		seen = append(seen, it.At())
	}
	require.NoError(t, it.Err())
	require.Equal(t, []uint64{42}, seen)
}
