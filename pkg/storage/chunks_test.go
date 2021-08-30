package storage

import (
	"math"
	"testing"

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/stretchr/testify/require"
)

func TestMultiChunks(t *testing.T) {
	var chks []chunkenc.Chunk
	var app chunkenc.Appender
	for i := int64(0); i < 1_000; i++ {
		if i%120 == 0 {
			c := chunkenc.NewDeltaChunk()
			chks = append(chks, c)
			app, _ = c.Appender()
		}
		app.Append(i)
	}

	require.Len(t, chks, 9) // ceil(1_000/120)

	var it MultiChunkIterator
	it = &multiChunksIterator{chunks: chks}

	seen := int64(0)
	for it.Next() {
		require.Equal(t, seen, it.At())
		seen++
	}

	require.NoError(t, it.Err())
	require.Equal(t, int64(1_000), seen)
}

func TestTimestampChunks_indexRange(t *testing.T) {
	var tcs timestampChunks
	tcs = append(tcs, timestampChunk{minTime: 0, maxTime: 20})

	// within the chunk minTime+5 and maxTime-5
	start, end := tcs.indexRange(5, 15)
	require.Equal(t, 0, start)
	require.Equal(t, 1, end)

	// from outside the chunk to outside the chunk. minTime-5 and maxTime+5
	start, end = tcs.indexRange(-5, 25)
	require.Equal(t, 0, start)
	require.Equal(t, 1, end)

	// both mint and maxt are fully before the first timestamp.
	start, end = tcs.indexRange(-15, -5)
	require.Equal(t, 0, start)
	require.Equal(t, 0, end)

	// both mint and maxt are fully after the first timestamp.
	start, end = tcs.indexRange(25, 30)
	require.Equal(t, 1, start)
	require.Equal(t, 1, end)

	for i := 20; i < 1_000; i++ {
		if i%20 == 0 {
			tcs = append(tcs, timestampChunk{
				minTime: int64(i),
				chunk:   chunkenc.NewDeltaChunk(),
			})
		}
		tcs[int(math.Floor(float64(i/20)))].maxTime = int64(i)
	}

	start, end = tcs.indexRange(123, 256)
	require.Equal(t, 6, start)
	require.Equal(t, 13, end)
}
