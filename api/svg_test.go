package api

import (
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestSVGRenderer(t *testing.T) {
	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	r := NewSVGRenderer(log.NewNopLogger(), p, "")
	rec := httptest.NewRecorder()
	tryRender(t, r, rec)

	require.Greater(t, len(rec.Body.Bytes()), 0)
}
