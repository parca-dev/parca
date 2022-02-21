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
	"context"
	"testing"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/profile"
)

func TestMemRootSeries_Iterator(t *testing.T) {
	ctx := context.Background()
	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {}, newHeadChunkPool())

	app, err := s.Appender()
	require.NoError(t, err)

	var i int64
	for i = 1; i < 500; i++ {
		p := &profile.FlatProfile{
			Meta: profile.InstantProfileMeta{
				Timestamp: i,
				Duration:  time.Second.Nanoseconds(),
				Period:    time.Second.Nanoseconds(),
			},
			FlatSamples: map[string]*profile.Sample{
				"": {Value: i},
			},
		}
		err = app.AppendFlat(ctx, p)
		require.NoError(t, err)
	}

	// Query subset of timestamps
	{
		it := (&MemRootSeries{s: s, mint: 74, maxt: 420}).Iterator()

		seen := int64(75)
		for it.Next() {
			p := it.At()
			require.Equal(t, seen, p.ProfileMeta().Timestamp)
			for _, sample := range p.Samples() {
				require.Equal(t, seen, sample.Value)
			}
			seen++
		}

		require.NoError(t, it.Err())
		require.Equal(t, int64(419), it.At().ProfileMeta().Timestamp)
	}

	// Query everything
	{
		it := (&MemRootSeries{s: s, mint: 0, maxt: 1000}).Iterator()

		seen := int64(1)
		for it.Next() {
			p := it.At()
			require.Equal(t, seen, p.ProfileMeta().Timestamp)
			seen++
		}

		require.Equal(t, int64(499), it.At().ProfileMeta().Timestamp)
	}
}
