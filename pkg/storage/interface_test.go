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
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
)

func makeSample(value int64, locationIds []uuid.UUID) *profile.Sample {
	s := &profile.Sample{
		Value: value,
	}

	for _, id := range locationIds {
		s.Location = append(s.Location, &metastore.Location{ID: id})
	}

	return s
}

func TestScaledInstantProfile(t *testing.T) {
	s1 := makeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	s2 := makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000005"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	s3 := makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	})
	k1 := profile.MakeStacktraceKey(s1)
	k2 := profile.MakeStacktraceKey(s2)
	k3 := profile.MakeStacktraceKey(s3)

	p := &profile.FlatProfile{
		FlatSamples: map[string]*profile.Sample{
			string(k1): s1,
			string(k2): s2,
			string(k3): s3,
		},
	}

	sp := profile.NewScaledInstantProfile(p, -1)

	expected := map[string]int64{
		string(k1): -2,
		string(k2): -1,
		string(k3): -3,
	}
	for k, s := range sp.Samples() {
		require.Equal(t, expected[k], s.Value)
	}
}

func TestSliceProfileSeriesIterator(t *testing.T) {
	it := &SliceProfileSeriesIterator{
		i:       -1,
		samples: []profile.InstantProfile{&profile.FlatProfile{}},
	}

	require.True(t, it.Next())
	require.False(t, it.Next())
}
