package query

import (
	"context"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestGenerateTopTable(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})
	p, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	res, err := GenerateTopTable(ctx, l, p)
	require.NoError(t, err)

	require.Equal(t, int32(4650), res.Total)
	require.Equal(t, int32(4650), res.Reported)
	require.Len(t, res.List, 4650)

	found := false
	for _, node := range res.GetList() {
		if node.GetMeta().GetFunction().GetName() == "encoding/json.Unmarshal" && node.GetFlat() == 14897 {
			require.Equal(t, int64(14897), node.GetCumulative()) // TODO: This need to be fixed
			require.Equal(t, int64(14897), node.GetFlat())

			require.Equal(t, uint64(7578336), node.GetMeta().GetLocation().GetAddress())
			require.Equal(t, false, node.GetMeta().GetLocation().GetIsFolded())
			require.Equal(t, uint64(4194304), node.GetMeta().GetMapping().GetStart())
			require.Equal(t, uint64(23252992), node.GetMeta().GetMapping().GetLimit())
			require.Equal(t, uint64(0), node.GetMeta().GetMapping().GetOffset())
			require.Equal(t, "/bin/operator", node.GetMeta().GetMapping().GetFile())
			require.Equal(t, "", node.GetMeta().GetMapping().GetBuildId())
			require.Equal(t, true, node.GetMeta().GetMapping().GetHasFunctions())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasFilenames())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasLineNumbers())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasInlineFrames())

			require.Equal(t, int64(0), node.GetMeta().GetFunction().GetStartLine())
			require.Equal(t, "encoding/json.Unmarshal", node.GetMeta().GetFunction().GetName())
			require.Equal(t, "encoding/json.Unmarshal", node.GetMeta().GetFunction().GetSystemName())
			require.Equal(t, int64(100), node.GetMeta().GetLine().GetLine())

			found = true
		}
	}
	require.Truef(t, found, "expected to find the specific function")
}
