package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunk"
	"github.com/stretchr/testify/require"
)

func TestChunk(t *testing.T) {
	os.Remove("result-profile1.pb.gz")
	os.Remove("result-profile2.pb.gz")

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(t, err)
	p2, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	c := &Series{chunk: &chunk.Chunk{}}
	require.NoError(t, c.Append(p1))
	require.NoError(t, c.Append(p2))

	it := c.Iterator()

	require.Equal(t, 2, len(it.data.Timestamps))
	require.Equal(t, 2, len(it.data.Durations))
	require.Equal(t, 2, len(it.data.Periods))

	f, err = os.Create("result-profile1.pb.gz")
	defer os.Remove("result-profile1.pb.gz")
	require.NoError(t, err)
	require.True(t, it.Next())
	resp1 := it.At()
	require.Equal(t, len(p1.Sample), len(resp1.Sample))
	require.NoError(t, resp1.Write(f))
	require.NoError(t, f.Close())

	f, err = os.Create("result-profile2.pb.gz")
	defer os.Remove("result-profile2.pb.gz")
	require.NoError(t, err)
	require.True(t, it.Next())
	resp2 := it.At()
	require.Equal(t, len(p1.Sample), len(resp1.Sample))
	require.NoError(t, resp2.Write(f))
	require.NoError(t, f.Close())

	require.False(t, it.Next())
}
