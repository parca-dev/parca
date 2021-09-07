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

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestMergeMemSeriesConsistency(t *testing.T) {
	ctx := context.Background()
	s, err := metastore.NewInMemorySQLiteProfileMetaStore(
		trace.NewNoopTracerProvider().Tracer(""),
		"memseriesconsistency",
	)
	t.Cleanup(func() {
		s.Close()
	})
	require.NoError(t, err)
	f, err := os.Open("./testdata/profile1.pb.gz")
	require.NoError(t, err)
	pprof1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	p := ProfileFromPprof(ctx, log.NewNopLogger(), s, pprof1, 0)

	db := OpenDB(prometheus.NewRegistry(), nil)

	app, err := db.Appender(ctx, labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
	})
	require.NoError(t, err)

	n := 1024
	for j := 0; j < n; j++ {
		p.Meta.Timestamp = int64(j + 1)
		err = app.Append(p)
		require.NoError(t, err)
	}

	set := db.Querier(
		ctx,
		int64(0),
		int64(n),
	).Select(nil, &labels.Matcher{
		Type:  labels.MatchEqual,
		Name:  "__name__",
		Value: "allocs",
	})

	p1, err := MergeSeriesSetProfiles(ctx, set)
	require.NoError(t, err)

	set = db.Querier(
		ctx,
		int64(0),
		int64(n),
	).Select(&SelectHints{
		Start: int64(0),
		End:   int64(n),
		Merge: true,
	}, &labels.Matcher{
		Type:  labels.MatchEqual,
		Name:  "__name__",
		Value: "allocs",
	})
	p2, err := MergeSeriesSetProfiles(ctx, set)
	require.NoError(t, err)

	require.Equal(t, p1, p2)
}

func TestMemMergeSeriesTree(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s11 := makeSample(1, []uint64{2, 1})

	s12 := makeSample(2, []uint64{4, 1})
	s12.Label = label
	s12.NumLabel = numLabel
	s12.NumUnit = numUnit

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {})

	pt1 := NewProfileTree()
	pt1.Insert(s11)
	pt1.Insert(s12)

	app, err := s.Appender()
	require.NoError(t, err)

	err = app.Append(&Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			Timestamp: 1,
		},
	})
	require.NoError(t, err)
	err = app.Append(&Profile{
		Tree: pt1,
		Meta: InstantProfileMeta{
			Timestamp: 2,
		},
	})
	require.NoError(t, err)

	ms := &MemMergeSeries{
		s:    s,
		mint: 0,
		maxt: 2,
	}
	it := ms.Iterator()
	require.True(t, it.Next())
	p := CopyInstantProfile(it.At())

	require.Equal(t, &Profile{
		Meta: InstantProfileMeta{
			Timestamp: 1,
		},
		Tree: &ProfileTree{
			Roots: &ProfileTreeNode{
				cumulativeValues: []*ProfileTreeValueNode{{
					Value: 6,
				}},
				Children: []*ProfileTreeNode{{
					locationID: 1,
					cumulativeValues: []*ProfileTreeValueNode{{
						Value: 6,
					}},
					Children: []*ProfileTreeNode{{
						locationID: 2,
						flatValues: []*ProfileTreeValueNode{{
							Value: 2,
						}},
						cumulativeValues: []*ProfileTreeValueNode{{
							Value: 2,
						}},
					}, {
						locationID: 4,
						flatValues: []*ProfileTreeValueNode{{
							Value:    4,
							Label:    label,
							NumLabel: numLabel,
							NumUnit:  numUnit,
						}},
						cumulativeValues: []*ProfileTreeValueNode{{
							Value:    4,
							Label:    label,
							NumLabel: numLabel,
							NumUnit:  numUnit,
						}},
					}},
				}},
			},
		},
	}, p)
}
