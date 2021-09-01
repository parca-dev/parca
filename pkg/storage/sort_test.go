// Copyright 2021 The Parca Authors
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

package storage

import (
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func makeSample(value int64, locationIds []uint64) *profile.Sample {
	s := &profile.Sample{
		Value: []int64{value},
	}

	for _, id := range locationIds {
		s.Location = append(s.Location, &profile.Location{ID: id})
	}

	return s
}

func Test_SortSamples_EdgeCases(t *testing.T) {

	tests := map[string]struct {
		samples []*profile.Sample
	}{
		"empty first": {
			samples: []*profile.Sample{
				makeSample(1, []uint64{}),
				makeSample(1, []uint64{6, 3, 1, 2}),
			},
		},
		"empty second": {
			samples: []*profile.Sample{
				makeSample(1, []uint64{6, 3, 1, 2}),
				makeSample(1, []uint64{}),
			},
		},
	}

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sortSamples(test.samples)

			require.Equal(t,
				[]*profile.Sample{
					makeSample(1, []uint64{}),
					makeSample(1, []uint64{6, 3, 1, 2}),
				},
				test.samples,
			)
		})
	}
}

func TestSortSamples(t *testing.T) {
	samples := []*profile.Sample{
		makeSample(1, []uint64{6, 3, 1}),
		makeSample(1, []uint64{5, 3, 1}),
		makeSample(1, []uint64{3, 1}),
	}

	sortSamples(samples)

	require.Equal(t,
		[]*profile.Sample{
			makeSample(1, []uint64{3, 1}),
			makeSample(1, []uint64{5, 3, 1}),
			makeSample(1, []uint64{6, 3, 1}),
		},
		samples,
	)
}
