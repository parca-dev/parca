package api

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestConsistentFlamegraph(t *testing.T) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	var res []byte

	for i := 0; i < 100; i++ {
		root, err := generateFlamegraphReport(p, "")
		require.NoError(t, err)

		newRes, err := json.Marshal(root)
		require.NoError(t, err)

		// Just for the first iteration.
		if res == nil {
			res = newRes
			continue
		}

		if !bytes.Equal(res, newRes) {
			t.Fatal("Flamegraphs are not generated consistently.")
		}
	}
}
