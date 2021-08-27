package storage

import (
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
