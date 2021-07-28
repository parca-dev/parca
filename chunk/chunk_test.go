package chunk

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAppendAtZero(t *testing.T) {
	c := NewFakeChunk()
	c.AppendAt(0, 2)

	it := c.Iterator()
	require.True(t, it.Next())
	require.Equal(t, int64(2), it.At())
}

func TestAppendAt(t *testing.T) {
	c := NewFakeChunk()
	c.AppendAt(1, 2)

	it := c.Iterator()
	require.True(t, it.Next())
	require.Equal(t, int64(0), it.At())
	require.True(t, it.Next())
	require.Equal(t, int64(2), it.At())
}
