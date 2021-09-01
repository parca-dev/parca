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

func TestInMemoryMetaStore(t *testing.T) {
	s := NewInMemoryProfileMetaStore()
	l := &profile.Location{
		ID:      uint64(8),
		Address: uint64(42),
	}
	s.CreateLocation(l)
	require.Equal(t, uint64(1), l.ID)
	_, err := s.GetLocationByID(l.ID)
	require.NoError(t, err)
	_, err = s.GetLocationByKey(MakeLocationKey(l))
	require.NoError(t, err)
}
