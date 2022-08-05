// Copyright 2022 The Parca Authors
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
