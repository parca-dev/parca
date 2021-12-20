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
	"github.com/stretchr/testify/require"
)

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
	k1 := makeStacktraceKey(s1)
	k2 := makeStacktraceKey(s2)
	k3 := makeStacktraceKey(s3)

	p := &FlatProfile{
		samples: map[string]*Sample{
			string(k1): s1,
			string(k2): s2,
			string(k3): s3,
		},
	}

	sp := NewScaledInstantProfile(p, -1)

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
		samples: []InstantProfile{&FlatProfile{}},
	}

	require.True(t, it.Next())
	require.False(t, it.Next())
}
