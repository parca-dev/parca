package chunkenc

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRLEChunk(t *testing.T) {
	c := NewRLEChunk()

	// check empty chunk
	require.Equal(t, []byte{0, 0, 0, 0}, c.Bytes())
	app, err := c.Appender()
	require.NoError(t, err)

	app.Append(0)
	require.Equal(t, []byte{0, 1, 0, 1, 0, 0, 1, 0}, c.Bytes())
	require.Equal(t, 1, c.NumSamples())
	app.Append(0)
	require.Equal(t, []byte{0, 2, 0, 1, 0, 0, 2, 0}, c.Bytes())
	require.Equal(t, 2, c.NumSamples())

	// Append 1 twice and then 5 more times
	app.Append(1)
	require.Equal(t, []byte{0, 3, 0, 2, 0, 0, 2, 2, 0, 1, 0}, c.Bytes())
	require.Equal(t, 3, c.NumSamples())

	app.Append(1)
	require.Equal(t, []byte{0, 4, 0, 2, 0, 0, 2, 2, 0, 2, 0}, c.Bytes())
	require.Equal(t, 4, c.NumSamples())

	for i := 0; i < 5; i++ {
		app.Append(1)
	}
	require.Equal(t, []byte{0, 9, 0, 2, 0, 0, 2, 2, 0, 7, 0}, c.Bytes())
	require.Equal(t, 9, c.NumSamples())

	// Append 2 twice to test another value
	app.Append(2)
	require.Equal(t, []byte{0, 10, 0, 3, 0, 0, 2, 2, 0, 7, 4, 0, 1, 0}, c.Bytes())
	require.Equal(t, 10, c.NumSamples())
	app.Append(2)
	require.Equal(t, []byte{0, 11, 0, 3, 0, 0, 2, 2, 0, 7, 4, 0, 2, 0}, c.Bytes())
	require.Equal(t, 11, c.NumSamples())

	// Append 3 100x to get a lot of the same values.
	for i := 0; i < 100; i++ {
		app.Append(3)
	}
	require.Equal(t, []byte{0, 111, 0, 4, 0, 0, 2, 2, 0, 7, 4, 0, 2, 6, 0, 100, 0}, c.Bytes())
	require.Equal(t, 111, c.NumSamples())

	// Iterate over the first values manually
	it := c.Iterator(nil)
	it.Next()
	require.Equal(t, int64(0), it.At())
	it.Next()
	require.Equal(t, int64(0), it.At())

	it.Next()
	require.Equal(t, int64(1), it.At())
	it.Next()
	require.Equal(t, int64(1), it.At())

	for i := 0; i < 5; i++ {
		it.Next()
		require.Equal(t, int64(1), it.At())
	}

	it.Next()
	require.Equal(t, int64(2), it.At())
	it.Next()
	require.Equal(t, int64(2), it.At())

	for it.Next() {
		require.NoError(t, it.Err())
		require.Equal(t, int64(3), it.At())
	}

	require.NoError(t, it.Err())
	require.False(t, it.Next())
}
