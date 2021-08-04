package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestCopyInstantProfileTree(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree := ProfileTreeFromPprof(l, p1)

	profileTreeCopy := CopyInstantProfileTree(profileTree)

	require.Equal(t, profileTree, profileTreeCopy)
}
