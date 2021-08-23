package chunkenc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeltaNonZeroFirstValue(t *testing.T) {
	c := NewDeltaChunk()
	app, err := c.Appender()
	require.NoError(t, err)

	app.Append(3)
	app.Append(5)
	app.Append(7)

	it := c.Iterator(nil)
	require.True(t, it.Next())
	require.Equal(t, int64(3), it.At())
	require.True(t, it.Next())
	require.Equal(t, int64(5), it.At())
	require.True(t, it.Next())
	require.Equal(t, int64(7), it.At())
	require.False(t, it.Next())
}
