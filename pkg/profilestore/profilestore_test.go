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

package profilestore

import (
	"testing"

	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/columnstore"
	"github.com/stretchr/testify/require"
)

func makeSampleRow(value int64, locationIds []uuid.UUID) *columnstore.SampleRow {
	stacktrace := make([]columnstore.UUID, 0, len(locationIds))
	for _, locationId := range locationIds {
		stacktrace = append(stacktrace, columnstore.UUID(locationId))
	}

	s := &columnstore.SampleRow{
		Value:      value,
		Stacktrace: stacktrace,
	}

	return s
}

func Test_SortSampleRows_EdgeCases(t *testing.T) {

	tests := map[string]struct {
		samples []*columnstore.SampleRow
	}{
		"empty first": {
			samples: []*columnstore.SampleRow{
				makeSampleRow(1, []uuid.UUID{}),
				makeSampleRow(1, []uuid.UUID{
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					uuid.MustParse("00000000-0000-0000-0000-000000000006"),
				}),
			},
		},
		"empty second": {
			samples: []*columnstore.SampleRow{
				makeSampleRow(1, []uuid.UUID{
					uuid.MustParse("00000000-0000-0000-0000-000000000002"),
					uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					uuid.MustParse("00000000-0000-0000-0000-000000000003"),
					uuid.MustParse("00000000-0000-0000-0000-000000000006"),
				}),
				makeSampleRow(1, []uuid.UUID{}),
			},
		},
	}

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			columnstore.SortSampleRows(test.samples)

			require.Equal(t,
				[]*columnstore.SampleRow{
					makeSampleRow(1, []uuid.UUID{}),
					makeSampleRow(1, []uuid.UUID{
						uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						uuid.MustParse("00000000-0000-0000-0000-000000000001"),
						uuid.MustParse("00000000-0000-0000-0000-000000000003"),
						uuid.MustParse("00000000-0000-0000-0000-000000000006"),
					}),
				},
				test.samples,
			)
		})
	}
}

func TestSortSampleRows(t *testing.T) {
	samples := []*columnstore.SampleRow{
		makeSampleRow(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			uuid.MustParse("00000000-0000-0000-0000-000000000006"),
		}),
		makeSampleRow(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			uuid.MustParse("00000000-0000-0000-0000-000000000005"),
		}),
		makeSampleRow(1, []uuid.UUID{
			uuid.MustParse("00000000-0000-0000-0000-000000000001"),
			uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		}),
	}

	columnstore.SortSampleRows(samples)

	require.Equal(t,
		[]*columnstore.SampleRow{
			makeSampleRow(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			}),
			makeSampleRow(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				uuid.MustParse("00000000-0000-0000-0000-000000000005"),
			}),
			makeSampleRow(1, []uuid.UUID{
				uuid.MustParse("00000000-0000-0000-0000-000000000001"),
				uuid.MustParse("00000000-0000-0000-0000-000000000003"),
				uuid.MustParse("00000000-0000-0000-0000-000000000006"),
			}),
		},
		samples,
	)
}
