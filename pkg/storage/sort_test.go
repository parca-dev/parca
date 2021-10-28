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

	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
)

func makeSample(value int64, locationIds []uuid.UUID) *Sample {
	s := &Sample{
		Value: value,
	}

	for _, id := range locationIds {
		s.Location = append(s.Location, &metastore.Location{ID: id})
	}

	return s
}

func Test_SortSamples_EdgeCases(t *testing.T) {

	tests := map[string]struct {
		samples []*Sample
	}{
		"empty first": {
			samples: []*Sample{
				makeSample(1, []uuid.UUID{}),
				makeSample(1, []uuid.UUID{
					uuid.MustParse("00000000-0000-0000-0000-000000000006"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}),
			},
		},
		"empty second": {
			samples: []*Sample{
				makeSample(1, []uuid.UUID{
					uuid.MustParse("00000000-0000-0000-0000-000000000006"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
				}),
				makeSample(1, []uuid.UUID{}),
			},
		},
	}

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			sortSamples(test.samples)

			require.Equal(t,
				[]*Sample{
					makeSample(1, []uuid.UUID{}),
					makeSample(1, []uuid.UUID{
						uuid.MustParse("00000000-0000-0000-0000-000000000006"),
						uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					}),
				},
				test.samples,
			)
		})
	}
}

func TestSortSamples(t *testing.T) {
	samples := []*Sample{
		makeSample(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000006"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}),
		makeSample(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000005"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}),
		makeSample(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		}),
	}

	sortSamples(samples)

	require.Equal(t,
		[]*Sample{
			makeSample(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			}),
			makeSample(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000005"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			}),
			makeSample(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000006"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			}),
		},
		samples,
	)
}
