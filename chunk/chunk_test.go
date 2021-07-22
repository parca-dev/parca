package chunk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendAtZero(t *testing.T) {
	c := NewFakeChunk()
	c.AppendAt(0, 2)

	require.Equal(t, []int64{2}, c.Values())
}

func TestAppendAt(t *testing.T) {
	c := NewFakeChunk()
	c.AppendAt(1, 2)

	require.Equal(t, []int64{0, 2}, c.Values())
}
