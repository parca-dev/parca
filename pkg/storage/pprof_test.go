package storage

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestGeneratePprof(t *testing.T) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	p := ProfileFromPprof(l, p1, 0)
	res, err := generatePprof(l, p)
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	require.NoError(t, res.Write(tmpfile))
	require.NoError(t, tmpfile.Close())

	f, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}
