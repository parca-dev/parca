// Copyright 2022-2023 The Parca Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parcacol

import (
	"testing"

	"github.com/stretchr/testify/require"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func TestLabelsFromSample(t *testing.T) {
	cases := []struct {
		name            string
		takenLabels     map[string]string
		stringTable     []string
		samples         []*pprofpb.Label
		resultLabels    map[string]string
		resultNumLabels map[string]int64
	}{{
		name: "descending order",
		takenLabels: map[string]string{
			"foo": "bar",
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
		takenLabels: map[string]string{
			"a": "b",
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
			labels, numLabels := LabelsFromSample(c.takenLabels, c.stringTable, c.samples)
			require.Equal(t, c.resultLabels, labels)
			require.Equal(t, c.resultNumLabels, numLabels)
		})
	}
}

func BenchmarkLabelsFromSample(b *testing.B) {
	var (
		takenLabels = map[string]string{
			"foo": "bar",
		}
		stringTable = []string{"", "foo", "bar", "exported_foo", "baz"}
		samples     = []*pprofpb.Label{{
			Key: 1,
			Str: 2,
		}, {
			Key: 3,
			Str: 4,
		}}
	)
	var (
		resultLabels = map[string]string{
			"exported_foo":          "baz",
			"exported_exported_foo": "bar",
		}
		resultNumLabels = map[string]int64{}
	)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		labels, numLabels := LabelsFromSample(takenLabels, stringTable, samples)
		require.Equal(b, resultLabels, labels)
		require.Equal(b, resultNumLabels, numLabels)
	}
}

func TestSampleKey(t *testing.T) {
	tests := map[string]struct {
		stacktraceID string
		labels       map[string]string
		numLabels    map[string]int64
		want         string
	}{
		"no labels": {
			stacktraceID: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=",
			labels:       nil,
			numLabels:    nil,
			want:         "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=;;",
		},
		"number labels": {
			stacktraceID: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=",
			labels:       nil,
			numLabels: map[string]int64{
				"bytes": 98304,
			},
			want: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=;;bytes=98304;",
		},
		"string and number labels": {
			stacktraceID: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=",
			labels: map[string]string{
				"fizz": "bazz",
			},
			numLabels: map[string]int64{
				"bytes": 98304,
			},
			want: "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=;fizz=bazz;;bytes=98304;",
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			got := sampleKey(tc.stacktraceID, tc.labels, tc.numLabels)
			if tc.want != got {
				t.Errorf("expected %q got %q", tc.want, got)
			}
		})
	}
}

func BenchmarkSampleKey(b *testing.B) {
	var (
		stacktraceID = "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434="
		labels       = map[string]string{
			"fizz": "bazz",
		}
		numLabels = map[string]int64{
			"bytes": 98304,
		}
	)
	want := "2b-t2tYPDARtdf-_FCsqMnUDllVoG8eHx3DGY6B2zsc=/T7hXckRIomziKtDlxDk8ymfFY0eP56PwOsxoERlyGY8=/MdCcinwyzSndRZX6l3oZ5U9JEiNN1OsxiZcvjaWe434=;fizz=bazz;;bytes=98304;"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		got := sampleKey(stacktraceID, labels, numLabels)
		if want != got {
			b.Errorf("expected %q got %q", want, got)
		}
	}
}
