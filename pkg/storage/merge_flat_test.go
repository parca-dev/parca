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
	"fmt"
	"math"
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

func TestMergeFlatProfileSimple(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")

	s1 := parcaprofile.MakeSample(2, []uuid.UUID{uuid2, uuid1})
	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")

	p1 := &parcaprofile.FlatProfile{
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

	p2 := &parcaprofile.FlatProfile{
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

	mp, err := MergeProfiles(p1, p2)
	require.NoError(t, err)
	require.Equal(t, parcaprofile.InstantProfileMeta{
		PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	merged := mp.Samples()
	require.Len(t, merged, 2)

	require.Equal(t, &parcaprofile.Sample{
		Value:    2,
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid1}},
	}, merged[string(k1[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid1}},
	}, merged[string(k2[:])])
}

func TestMergeFlatProfileDeep(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid6 := uuid.MustParse("00000000-0000-0000-0000-000000000006")

	s1 := parcaprofile.MakeSample(3, []uuid.UUID{uuid1, uuid3})
	s2 := parcaprofile.MakeSample(3, []uuid.UUID{uuid2, uuid3})
	s3 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid3, uuid2})
	s4 := parcaprofile.MakeSample(3, []uuid.UUID{uuid6, uuid2})

	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")
	k3 := uuid.MustParse("00000000-0000-0000-0000-0000000000e3")
	k4 := uuid.MustParse("00000000-0000-0000-0000-0000000000e4")

	p1 := &parcaprofile.FlatProfile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k1[:]): s1,
			string(k2[:]): s2,
			string(k3[:]): s3,
			string(k4[:]): s4,
		},
	}

	s5 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid2, uuid2})
	k5 := uuid.MustParse("00000000-0000-0000-0000-0000000000e5")

	p2 := &parcaprofile.FlatProfile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k5[:]): s5,
		},
	}

	mp, err := MergeProfiles(p1, p2)
	require.NoError(t, err)
	require.Equal(t, parcaprofile.InstantProfileMeta{
		PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	merged := mp.Samples()

	require.Len(t, merged, 5)

	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid1}, {ID: uuid3}},
	}, merged[string(k1[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid3}},
	}, merged[string(k2[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid3}, {ID: uuid2}},
	}, merged[string(k3[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid2}},
	}, merged[string(k4[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid2}, {ID: uuid2}},
	}, merged[string(k5[:])])
}

func TestMergeFlatProfile(t *testing.T) {
	uuid1 := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	uuid2 := uuid.MustParse("00000000-0000-0000-0000-000000000002")
	uuid3 := uuid.MustParse("00000000-0000-0000-0000-000000000003")
	uuid4 := uuid.MustParse("00000000-0000-0000-0000-000000000004")
	uuid5 := uuid.MustParse("00000000-0000-0000-0000-000000000005")
	uuid6 := uuid.MustParse("00000000-0000-0000-0000-000000000006")

	s1 := parcaprofile.MakeSample(2, []uuid.UUID{uuid2, uuid1})
	s2 := parcaprofile.MakeSample(1, []uuid.UUID{uuid6, uuid3, uuid2, uuid1})
	s3 := parcaprofile.MakeSample(3, []uuid.UUID{uuid4, uuid3, uuid2, uuid1})
	s4 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid3, uuid2})
	s5 := parcaprofile.MakeSample(3, []uuid.UUID{uuid6, uuid2})

	k1 := uuid.MustParse("00000000-0000-0000-0000-0000000000e1")
	k2 := uuid.MustParse("00000000-0000-0000-0000-0000000000e2")
	k3 := uuid.MustParse("00000000-0000-0000-0000-0000000000e3")
	k4 := uuid.MustParse("00000000-0000-0000-0000-0000000000e4")
	k5 := uuid.MustParse("00000000-0000-0000-0000-0000000000e5")

	p1 := &parcaprofile.FlatProfile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k1[:]): s1,
			string(k2[:]): s2,
			string(k3[:]): s3,
			string(k4[:]): s4,
			string(k5[:]): s5,
		},
	}

	s6 := parcaprofile.MakeSample(1, []uuid.UUID{uuid5, uuid3, uuid2, uuid1})
	s7 := parcaprofile.MakeSample(3, []uuid.UUID{uuid3, uuid2, uuid2})

	k6 := uuid.MustParse("00000000-0000-0000-0000-0000000000e6")
	k7 := uuid.MustParse("00000000-0000-0000-0000-0000000000e7")

	p2 := &parcaprofile.FlatProfile{
		Meta: parcaprofile.InstantProfileMeta{
			PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			string(k1[:]): s1,
			string(k3[:]): s3,
			string(k6[:]): s6,
			string(k7[:]): s7,
		},
	}

	mp, err := MergeProfiles(p1, p2)
	require.NoError(t, err)
	require.Equal(t, parcaprofile.InstantProfileMeta{
		PeriodType: parcaprofile.ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: parcaprofile.ValueType{Type: "numSamples", Unit: "count"},
		Timestamp:  1,
		Duration:   int64(time.Second * 20),
		Period:     100,
	}, mp.ProfileMeta())

	merged := mp.Samples()
	require.Len(t, merged, 7)

	require.Equal(t, &parcaprofile.Sample{
		Value:    4, // 2 + 2
		Location: []*metastore.Location{{ID: uuid2}, {ID: uuid1}},
	}, merged[string(k1[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k2[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    6, // 3 + 3
		Location: []*metastore.Location{{ID: uuid4}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k3[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid3}, {ID: uuid2}},
	}, merged[string(k4[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid6}, {ID: uuid2}},
	}, merged[string(k5[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    1,
		Location: []*metastore.Location{{ID: uuid5}, {ID: uuid3}, {ID: uuid2}, {ID: uuid1}},
	}, merged[string(k6[:])])
	require.Equal(t, &parcaprofile.Sample{
		Value:    3,
		Location: []*metastore.Location{{ID: uuid3}, {ID: uuid2}, {ID: uuid2}},
	}, merged[string(k7[:])])
}

func TestMergeSingleFlat(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)

	m, err := MergeProfiles(prof)
	require.NoError(t, err)
	require.Len(t, m.Samples(), 32)
}

func TestMergeManyFlat(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p, 0)
	require.NoError(t, err)

	num := 1000
	profiles := make([]parcaprofile.InstantProfile, 0, 1000)
	for i := 0; i < num; i++ {
		profiles = append(profiles, prof)
	}

	m, err := MergeProfiles(profiles...)
	require.NoError(t, err)
	parcaprofile.CopyInstantFlatProfile(m)
}

func BenchmarkFlatMerge(b *testing.B) {
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

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"treemerge",
	)
	b.Cleanup(func() {
		l.Close()
	})
	require.NoError(b, err)
	profile1, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(b, err)
	profile2, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p2, 0)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()
	m, err := MergeProfiles(profile1, profile2)
	require.NoError(b, err)
	parcaprofile.CopyInstantFlatProfile(m)
}

func BenchmarkMergeFlatMany(b *testing.B) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	registry := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			f, err := os.Open("testdata/profile1.pb.gz")
			require.NoError(b, err)
			p, err := profile.Parse(f)
			require.NoError(b, err)
			require.NoError(b, f.Close())

			l := metastore.NewBadgerMetastore(
				logger,
				registry,
				tracer,
				metastore.NewRandomUUIDGenerator(),
			)
			defer func() {
				l.Close()
			}()

			prof, err := parcaprofile.FlatProfileFromPprof(ctx, logger, l, p, 0)
			require.NoError(b, err)

			profiles := make([]parcaprofile.InstantProfile, 0, n)
			for i := 0; i < n; i++ {
				profiles = append(profiles, prof)
			}

			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				m, err := MergeProfiles(profiles...)
				require.NoError(b, err)
				parcaprofile.CopyInstantFlatProfile(m)
			}
		})
	}
}
