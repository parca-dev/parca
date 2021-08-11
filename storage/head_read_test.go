package storage

import (
	"testing"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestHeadIndexReader_Postings(t *testing.T) {
	ir := headIndexReader{head: NewHead()}
	ir.head.postings.Add(1, labels.Labels{{"foo", "bar"}, {"container", "test1"}})
	ir.head.postings.Add(2, labels.Labels{{"foo", "bar"}, {"container", "test2"}})
	ir.head.postings.Add(3, labels.Labels{{"foo", "baz"}, {"container", "test3"}})

	bm, err := ir.Postings("foo", "bar")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2}, bm.ToArray())

	bm, err = ir.Postings("foo", "bar", "baz")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, bm.ToArray())
}
