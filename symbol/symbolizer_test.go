package symbol

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestSymbolizer(t *testing.T) {
	expResp := testResponse()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		err := json.NewEncoder(w).Encode(expResp)
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer ts.Close()

	s := NewSymbolizer(NewSymbolServerClient(ts.URL))
	m := &profile.Mapping{
		ID:      uint64(1),
		Start:   0x400000,
		BuildID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}
	p := &profile.Profile{
		Location: []*profile.Location{{
			Mapping: m,
			Address: 0x463781,
		}},
		Mapping: []*profile.Mapping{m},
	}

	err := s.Symbolize(context.Background(), p)
	require.NoError(t, err)

	require.Equal(t, 1, len(p.Location))
	require.Equal(t, 1, len(p.Mapping))

	lines := p.Location[0].Line
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(27), lines[0].Line)
	require.Equal(t, "main.iterate", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(10), lines[2].Line)
	require.Equal(t, "main.main", lines[2].Function.Name)
}
