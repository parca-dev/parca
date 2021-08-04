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
	s, err := NewMemSeries(l)
	require.NoError(t, err)
	require.NoError(t, s.Append(p1))

	profileTree, err := s.prepareSamplesForInsert(p1)
	require.NoError(t, err)

	profileTreeCopy := CopyInstantProfileTree(profileTree)

	require.Equal(t, profileTree, profileTreeCopy)
}
