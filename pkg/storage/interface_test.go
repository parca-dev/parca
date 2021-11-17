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
	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestCopyInstantProfileTree(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"compyinstantprofiletree",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	profileTree, err := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	profileTreeCopy := CopyInstantProfileTree(profileTree)

	require.Equal(t, profileTree, profileTreeCopy)
}

func TestProfileTreeValueNode_Key(t *testing.T) {
	testcases := []struct {
		node     ProfileTreeValueNode
		location uuid.UUID
		key      *ProfileTreeValueNodeKey
	}{{
		node:     ProfileTreeValueNode{},
		location: uuid.Nil, // root
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000000",
		},
	}, {
		node:     ProfileTreeValueNode{},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000001",
		},
	}, {
		node:     ProfileTreeValueNode{},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000002",
		},
	}, {
		node:     ProfileTreeValueNode{Value: 123}, // Value doesn't matter
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000001",
		},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar"}}},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000001",
			labels:   `"foo"["bar"]`,
		},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar", "baz"}}},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000001",
			labels:   `"foo"["bar" "baz"]`,
		},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar", "baz"}, "a": {"b"}}},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location: "00000000-0000-0000-0000-000000000001",
			labels:   `"a"["b"]"foo"["bar" "baz"]`,
		},
	}, {
		node: ProfileTreeValueNode{
			Label:    map[string][]string{"foo": {"bar"}},
			NumLabel: map[string][]int64{"foo": {123}},
			NumUnit:  map[string][]string{"foo": {"count"}},
		},
		location: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
		key: &ProfileTreeValueNodeKey{
			location:  "00000000-0000-0000-0000-000000000001",
			labels:    `"foo"["bar"]`,
			numlabels: `"foo"[7b][636f756e74]`,
		},
	}}
	for _, tc := range testcases {
		tc.node.Key(tc.location)
		require.Equal(t, tc.key, tc.node.key)
	}
}

func TestScaledInstantProfile(t *testing.T) {
	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))
	pt.Insert(makeSample(1, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000005"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))
	pt.Insert(makeSample(3, []uuid.UUID{
		uuid.MustParse("00000000-0000-0000-0000-000000000004"),
		uuid.MustParse("00000000-0000-0000-0000-000000000003"),
		uuid.MustParse("00000000-0000-0000-0000-000000000002"),
		uuid.MustParse("00000000-0000-0000-0000-000000000001"),
	}))

	p := &Profile{
		Tree: pt,
	}

	sp := NewScaledInstantProfile(p, -1)
	scaledTree := CopyInstantProfileTree(sp.ProfileTree())
	require.Equal(t, &ProfileTree{
		Roots: &ProfileTreeRootNode{
			ProfileTreeNode: &ProfileTreeNode{
				// Roots always have the LocationID 0.
				locationID: uuid.MustParse("00000000-0000-0000-0000-000000000000"),
				Children: []*ProfileTreeNode{{
					locationID: uuid.MustParse("00000000-0000-0000-0000-000000000001"),
					Children: []*ProfileTreeNode{{
						locationID: uuid.MustParse("00000000-0000-0000-0000-000000000002"),
						flatValues: []*ProfileTreeValueNode{{
							Value: -2,
							key: &ProfileTreeValueNodeKey{
								location: "00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000",
							},
						}},
						Children: []*ProfileTreeNode{{
							locationID: uuid.MustParse("00000000-0000-0000-0000-000000000003"),
							Children: []*ProfileTreeNode{{
								locationID: uuid.MustParse("00000000-0000-0000-0000-000000000004"),
								flatValues: []*ProfileTreeValueNode{{
									key: &ProfileTreeValueNodeKey{
										location: "00000000-0000-0000-0000-000000000004|00000000-0000-0000-0000-000000000003|00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000",
									},
									Value: -3,
								}},
							}, {
								locationID: uuid.MustParse("00000000-0000-0000-0000-000000000005"),
								flatValues: []*ProfileTreeValueNode{{
									key: &ProfileTreeValueNodeKey{
										location: "00000000-0000-0000-0000-000000000005|00000000-0000-0000-0000-000000000003|00000000-0000-0000-0000-000000000002|00000000-0000-0000-0000-000000000001|00000000-0000-0000-0000-000000000000",
									},
									Value: -1,
								}},
							}},
						}},
					}},
				}},
			},
		},
	}, scaledTree)
}

func TestSliceProfileSeriesIterator(t *testing.T) {
	it := &SliceProfileSeriesIterator{
		i:       -1,
		samples: []InstantProfile{&Profile{}},
	}

	require.True(t, it.Next())
	require.False(t, it.Next())
}
