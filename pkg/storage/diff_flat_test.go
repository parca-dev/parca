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
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestDiffProfileSimple(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	s1 := parcaprofile.MakeSample(3, []uuid.UUID{uuid2, uuid1})
	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")

	p1 := &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k1[:]): s1,
		},
	}

	s2 := parcaprofile.MakeSample(1, []uuid.UUID{uuid3, uuid1})
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")

	p2 := &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k2[:]): s2,
		},
	}

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, parcaprofile.InstantProfileMeta{
		PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
	}, dp.ProfileMeta())

	diffed := dp.Samples()
	require.Len(t, diffed, 1)

	require.Equal(t, &parcaprofile.Sample{
		Value:     1,
		DiffValue: 0,
		Location:  []*metastore.Location{{ID: uuid3}, {ID: uuid1}},
	}, diffed[string(k2[:])])
}

func TestDiffProfileDeep(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid6 := uuid.MustParse("00000000-0000-0000-0000-000000000006")

	s0 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid3, uuid2})
	s1 := parcaprofile.MakeSample(3, []uuid.UUID{uuid6, uuid2})
	s2 := parcaprofile.MakeSample(3, []uuid.UUID{uuid2, uuid3})
	s3 := parcaprofile.MakeSample(3, []uuid.UUID{uuid1, uuid3})

	k0 := uuid.MustParse("00000000-0000-0000-0000-0000000000e0")
	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")
	k3 := uuid.MustParse("00000000-0000-0000-0000-0000000000e3")

	p1 := &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k0[:]): s0,
			string(k1[:]): s1,
			string(k2[:]): s2,
			string(k3[:]): s3,
		},
	}

	s4 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid2, uuid2})
	s5 := parcaprofile.MakeSample(5, []uuid.UUID{uuid2, uuid3})

	k4 := uuid.MustParse("00000000-0000-0000-0000-0000000000e4")

	p2 := &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k4[:]): s4,
			string(k2[:]): s5,
		},
	}

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, parcaprofile.InstantProfileMeta{
		PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
	}, dp.ProfileMeta())

	diffed := dp.Samples()
	require.Len(t, diffed, 2)

	require.Equal(t, &parcaprofile.Sample{
		Value:     3,
		DiffValue: 0,
		Location:  []*metastore.Location{{ID: uuid3}, {ID: uuid2}, {ID: uuid2}},
	}, diffed[string(k4[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:     5,
		DiffValue: 2,
		Location:  []*metastore.Location{{ID: uuid2}, {ID: uuid3}},
	}, diffed[string(k2[:])])
}

// go test -bench=BenchmarkFlatDiff -count=3 ./pkg/storage | tee ./pkg/storage/benchmark/diff-flat.txt

func BenchmarkFlatDiff(b *testing.B) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(b, err)
	p2, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	require.NoError(b, err)
	b.Cleanup(func() {
		l.Close()
	})
	profile1, err := parcaprofile.FromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(b, err)
	profile2, err := parcaprofile.FromPprof(ctx, log.NewNopLogger(), l, p2, 0)
	require.NoError(b, err)

	b.Run("simple", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			d, err := NewDiffProfile(profile1, profile2)
			require.NoError(b, err)
			parcaprofile.CopyInstantProfile(d)
		}
	})
}
