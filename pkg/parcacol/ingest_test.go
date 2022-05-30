package parcacol

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/metastore"
)

func TestMakeStacktraceKey(t *testing.T) {
	g := metastore.NewLinearUUIDGenerator()

	s := &SampleNormalizer{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string]string{"foo": "bar", "bar": "baz"},
		NumLabel: map[string]int64{"foo": 1},
		NumUnit:  map[string]string{"foo": "cpu"},
	}

	k := []byte(MakeStacktraceKey(s))

	require.Len(t, k, 94)

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
		[]byte(`"bar":"baz""foo":"bar"`),
		k[50:72],
	)

	require.Equal(t,
		[]byte{
			'"', 'f', 'o', 'o', '"',
			':', '{',
			'"', 'c', 'p', 'u', '"',
			':',
			0, 0, 0, 0, 0, 0, 0, 1,
			'}',
		},
		k[72:],
	)
}

func BenchmarkMakeStacktraceKey(b *testing.B) {
	g := metastore.NewLinearUUIDGenerator()
	s := &SampleNormalizer{
		Location: []*metastore.Location{{ID: g.New()}, {ID: g.New()}, {ID: g.New()}},
		Label:    map[string]string{"foo": "bar"},
		NumLabel: map[string]int64{"foo": 1},
		NumUnit:  map[string]string{"foo": "cpu"},
	}

	b.ReportAllocs()
	b.StartTimer()

	for i := 0; i < b.N; i++ {
		_ = MakeStacktraceKey(s)
	}
}
