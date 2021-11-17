package storage

import (
	"testing"

	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
)

func TestMakeStacktraceKey(t *testing.T) {
	g := NewLinearUUIDGenerator()

	s := &Sample{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string][]string{"foo": {"bar", "baz"}, "bar": {"baz"}},
		NumLabel: map[string][]int64{"foo": {0, 1}},
		NumUnit:  map[string][]string{"foo": {"cpu", "memory"}},
	}

	k := []byte(makeStacktraceKey(s))

	require.Len(t, k, 119)

	require.Equal(t,
		[]byte{
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1,
			'|',
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2,
			'|',
			0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x3,
		},
		k[0:50],
	)

	require.Equal(t,
		[]byte(`"bar"["baz"]"foo"["bar" "baz"]`),
		k[50:80],
	)

	require.Equal(t,
		[]byte{
			'"', 'f', 'o', 'o', '"',
			'[',
			0, 0, 0, 0, 0, 0, 0, 0,
			0, 0, 0, 0, 0, 0, 0, 1,
			']',
			'[', '"', 'c', 'p', 'u', '"', ' ', '"', 'm', 'e', 'm', 'o', 'r', 'y', '"', ']',
		},
		k[80:],
	)
}

func BenchmarkMakeStacktraceKey(b *testing.B) {
	g := NewLinearUUIDGenerator()
	s := &Sample{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string][]string{"foo": {"bar", "baz"}},
		NumLabel: map[string][]int64{"foo": {0, 1}},
		NumUnit:  map[string][]string{"foo": {"cpu", "memory"}},
	}

	b.ReportAllocs()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = makeStacktraceKey(s)
	}
}
