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
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestDiffProfileSimple(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	p1 := &Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	p2 := &Profile{
		Tree: pt2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
	}, dp.ProfileMeta())

	var res []sample
	err = WalkProfileTree(dp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:       n.LocationID(),
			flat:     n.FlatValues(),
			flatDiff: n.FlatDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		},
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		},
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			flat: []*ProfileTreeValueNode{{
				Value: int64(1),
				key: &ProfileTreeValueNodeKey{
					location: "00000000-0000-0000-0000-000000000003|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000",
				},
			}},
			flatDiff: []*ProfileTreeValueNode{{
				Value: int64(1),
			}},
		},
	}, res)
}

func TestDiffProfileDeep(t *testing.T) {
	pt1 := NewProfileTree()
	pt1.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	}))
	pt1.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000006"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	}))
	pt1.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
	}))
	pt1.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
	}))

	p1 := &Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	pt2 := NewProfileTree()
	pt2.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
	}))

	p2 := &Profile{
		Tree: pt2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	dp, err := NewDiffProfile(p1, p2)
	require.NoError(t, err)
	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
	}, dp.ProfileMeta())

	res := []sample{}
	err = WalkProfileTree(dp.ProfileTree(), func(n InstantProfileTreeNode) error {
		res = append(res, sample{
			id:       n.LocationID(),
			flat:     n.FlatValues(),
			flatDiff: n.FlatDiffValues(),
		})
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []sample{
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
		},
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		},
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		},
		{
			id: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
			flat: []*ProfileTreeValueNode{{
				Value: int64(3),
				key: &ProfileTreeValueNodeKey{
					location: "00000000-0000-0000-0000-000000000003|00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000000",
				},
			}},
			flatDiff: []*ProfileTreeValueNode{{
				Value: int64(3),
			}},
		},
	}, res)
}

func BenchmarkDiff(b *testing.B) {
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
	profileTree1, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(b, err)
	profileTree2, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p2, 0)
	require.NoError(b, err)

	prof1 := &Profile{
		Tree: profileTree1,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	prof2 := &Profile{
		Tree: profileTree2,
		Meta: InstantProfileMeta{
			PeriodType: ValueType{Type: "cpu", Unit: "cycles"},
			SampleType: ValueType{Type: "samples", Unit: "count"},
			Timestamp:  1,
			Duration:   int64(time.Second * 10),
			Period:     100,
		},
	}

	b.Run("simple", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			d, err := NewDiffProfile(prof1, prof2)
			require.NoError(b, err)
			CopyInstantProfileTree(d.ProfileTree())
		}
	})
}
