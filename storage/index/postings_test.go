package index

import (
	"testing"

	"github.com/dgraph-io/sroar"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestMemPostings(t *testing.T) {
	p := NewMemPostings()
	p.Add(42, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test1"}})
	p.Add(123, labels.Labels{{Name: "foo", Value: "bar"}, {Name: "container", Value: "test2"}})

	var empty []uint64
	require.Equal(t, []uint64{42, 123}, p.Get("foo", "bar").ToArray())
	require.Equal(t, []uint64{42}, p.Get("container", "test1").ToArray())
	require.Equal(t, []uint64{123}, p.Get("container", "test2").ToArray())
	require.Equal(t, empty, p.Get("container", "test3").ToArray())

	require.ElementsMatch(t, []string{"foo", "container"}, p.LabelNames())
	require.ElementsMatch(t, []string{"test1", "test2"}, p.LabelValues("container"))
}

func TestBitmap(t *testing.T) {
	b1 := sroar.NewBitmap()
	b1.Set(123)
	require.Equal(t, []uint64{123}, b1.ToArray())

	b2 := sroar.NewBitmap()
	b2.Set(42)
	require.Equal(t, []uint64{42}, b2.ToArray())

	b3 := b1.Clone() // we would be mutating b1 so instead clone as b3
	b3.Or(b2)        // all data in b1 OR b2 (union)
	require.Equal(t, []uint64{42, 123}, b3.ToArray())

	b4 := sroar.NewBitmap()
	b4.SetMany([]uint64{123, 66})
	b4.And(b1) // all data in b4 AND b1 (intersection)
	require.Equal(t, []uint64{123}, b4.ToArray())
}
