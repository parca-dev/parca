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

func TestLabelNamesFromSamples(t *testing.T) {
	cases := []struct {
		name         string
		takenLabels  map[string]string
		stringTable  []string
		samples      []*pprofpb.Sample
		allLabels    map[string]struct{}
		allNumLabels map[string]struct{}
	}{
		{
			name: "colliding labels in descending order",
			takenLabels: map[string]string{
				"instance": "127.0.0.1:6060",
			},
			stringTable: []string{"", "instance", "17", "method", "GET"},
			samples: []*pprofpb.Sample{
				{
					Label: []*pprofpb.Label{{
						Key: 3,
						Str: 4,
					}, {
						Key: 1,
						Str: 2,
					}},
				},
			},
			allLabels: map[string]struct{}{
				"exported_instance": {},
				"method":            {},
			},
			allNumLabels: map[string]struct{}{},
		},
		{
			name: "colliding labels in ascending order",
			takenLabels: map[string]string{
				"instance": "127.0.0.1:6060",
			},
			stringTable: []string{"", "instance", "17", "method", "GET"},
			samples: []*pprofpb.Sample{
				{
					Label: []*pprofpb.Label{{
						Key: 1,
						Str: 2,
					}, {
						Key: 3,
						Str: 4,
					}},
				},
			},
			allLabels: map[string]struct{}{
				"exported_instance": {},
				"method":            {},
			},
			allNumLabels: map[string]struct{}{},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			allLabels := map[string]struct{}{}
			allNumLabels := map[string]struct{}{}
			LabelNamesFromSamples(c.takenLabels, c.stringTable, c.samples, allLabels, allNumLabels)
			require.Equal(t, c.allLabels, allLabels)
			require.Equal(t, c.allNumLabels, allNumLabels)
		})
	}
}

func TestLabelsFromSample(t *testing.T) {
	cases := []struct {
		name            string
		takenLabels     map[string]string
		stringTable     []string
		labels          []*pprofpb.Label
		resultLabels    map[string]string
		resultNumLabels map[string]int64
	}{
		{
			name: "colliding labels in descending order",
			takenLabels: map[string]string{
				"instance": "127.0.0.1:6060",
			},
			stringTable: []string{"", "instance", "17", "method", "GET"},
			labels: []*pprofpb.Label{{
				Key: 3,
				Str: 4,
			}, {
				Key: 1,
				Str: 2,
			}},
			resultLabels: map[string]string{
				"exported_instance": "17",
				"method":            "GET",
			},
			resultNumLabels: map[string]int64{},
		},
		{
			name: "colliding labels in ascending order",
			takenLabels: map[string]string{
				"instance": "127.0.0.1:6060",
			},
			stringTable: []string{"", "instance", "17", "method", "GET"},
			labels: []*pprofpb.Label{{
				Key: 1,
				Str: 2,
			}, {
				Key: 3,
				Str: 4,
			}},
			resultLabels: map[string]string{
				"exported_instance": "17",
				"method":            "GET",
			},
			resultNumLabels: map[string]int64{},
		},
	}

	for _, c := range cases {
		t.Run("", func(t *testing.T) {
			labels, numLabels := LabelsFromSample(c.takenLabels, c.stringTable, c.labels)
			require.Equal(t, c.resultLabels, labels)
			require.Equal(t, c.resultNumLabels, numLabels)
		})
	}
}
