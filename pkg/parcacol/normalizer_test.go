package parcacol

import (
	"testing"

	"github.com/stretchr/testify/require"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func TestLabelsFromSample(t *testing.T) {
	cases := []struct {
		name            string
		takenLabels     map[string]struct{}
		stringTable     []string
		samples         []*pprofpb.Label
		resultLabels    map[string]string
		resultNumLabels map[string]int64
	}{{
		name: "descending order",
		takenLabels: map[string]struct{}{
			"foo": {},
		},
		stringTable: []string{"", "foo", "bar", "exported_foo", "baz"},
		samples: []*pprofpb.Label{{
			Key: 1,
			Str: 2,
		}, {
			Key: 3,
			Str: 4,
		}},
		resultLabels: map[string]string{
			"exported_foo":          "baz",
			"exported_exported_foo": "bar",
		},
		resultNumLabels: map[string]int64{},
	}, {
		name: "ascending order",
		takenLabels: map[string]struct{}{
			"a": {},
		},
		stringTable: []string{"", "a", "bar", "exported_a", "baz"},
		samples: []*pprofpb.Label{{
			Key: 1,
			Str: 2,
		}, {
			Key: 3,
			Str: 4,
		}},
		resultLabels: map[string]string{
			"exported_a":          "bar",
			"exported_exported_a": "baz",
		},
		resultNumLabels: map[string]int64{},
	}}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			labels, numLabels := labelsFromSample(c.takenLabels, c.stringTable, c.samples)
			require.Equal(t, c.resultLabels, labels)
			require.Equal(t, c.resultNumLabels, numLabels)
		})
	}
}
