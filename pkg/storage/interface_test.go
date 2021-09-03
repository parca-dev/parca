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
	"github.com/stretchr/testify/require"
)

func TestCopyInstantProfileTree(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemoryProfileMetaStore("compyinstantprofiletree")
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	profileTree := ProfileTreeFromPprof(ctx, log.NewNopLogger(), l, p1, 0)

	profileTreeCopy := CopyInstantProfileTree(profileTree)

	require.Equal(t, profileTree, profileTreeCopy)
}

func TestProfileTreeValueNode_Key(t *testing.T) {
	testcases := []struct {
		node     ProfileTreeValueNode
		location uint64
		key      *ProfileTreeValueNodeKey
	}{{
		node:     ProfileTreeValueNode{},
		location: 0, // root
		key:      &ProfileTreeValueNodeKey{location: "0"},
	}, {
		node:     ProfileTreeValueNode{},
		location: 1,
		key:      &ProfileTreeValueNodeKey{location: "1"},
	}, {
		node:     ProfileTreeValueNode{},
		location: 2,
		key:      &ProfileTreeValueNodeKey{location: "2"},
	}, {
		node:     ProfileTreeValueNode{Value: 123}, // Value doesn't matter
		location: 1,
		key:      &ProfileTreeValueNodeKey{location: "1"},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar"}}},
		location: 1,
		key:      &ProfileTreeValueNodeKey{location: "1", labels: `"foo"["bar"]`},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar", "baz"}}},
		location: 1,
		key:      &ProfileTreeValueNodeKey{location: "1", labels: `"foo"["bar" "baz"]`},
	}, {
		node:     ProfileTreeValueNode{Label: map[string][]string{"foo": {"bar", "baz"}, "a": {"b"}}},
		location: 1,
		key:      &ProfileTreeValueNodeKey{location: "1", labels: `"a"["b"]"foo"["bar" "baz"]`},
	}, {
		node: ProfileTreeValueNode{
			Label:    map[string][]string{"foo": {"bar"}},
			NumLabel: map[string][]int64{"foo": {123}},
			NumUnit:  map[string][]string{"foo": {"count"}},
		},
		location: 1,
		key: &ProfileTreeValueNodeKey{
			location:  "1",
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
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))

	p := &Profile{
		Tree: pt,
	}

	sp := NewScaledInstantProfile(p, -1)
	scaledTree := CopyInstantProfileTree(sp.ProfileTree())
	require.Equal(t, &ProfileTree{
		Roots: &ProfileTreeNode{
			cumulativeValues: []*ProfileTreeValueNode{{
				key: &ProfileTreeValueNodeKey{
					location: "0",
				},
				Value: -6,
			}},
			// Roots always have the LocationID 0.
			locationID: 0,
			Children: []*ProfileTreeNode{{
				locationID: 1,
				cumulativeValues: []*ProfileTreeValueNode{{
					key: &ProfileTreeValueNodeKey{
						location: "1|0",
					},
					Value: -6,
				}},
				Children: []*ProfileTreeNode{{
					locationID: 2,
					cumulativeValues: []*ProfileTreeValueNode{{
						Value: -6,
						key: &ProfileTreeValueNodeKey{
							location: "2|1|0",
						},
					}},
					flatValues: []*ProfileTreeValueNode{{
						Value: -2,
						key: &ProfileTreeValueNodeKey{
							location: "2|1|0",
						},
					}},
					Children: []*ProfileTreeNode{{
						locationID: 3,
						cumulativeValues: []*ProfileTreeValueNode{{
							key: &ProfileTreeValueNodeKey{
								location: "3|2|1|0",
							},
							Value: -4,
						}},
						Children: []*ProfileTreeNode{{
							locationID: 4,
							cumulativeValues: []*ProfileTreeValueNode{{
								key: &ProfileTreeValueNodeKey{
									location: "4|3|2|1|0",
								},
								Value: -3,
							}},
							flatValues: []*ProfileTreeValueNode{{
								key: &ProfileTreeValueNodeKey{
									location: "4|3|2|1|0",
								},
								Value: -3,
							}},
						}, {
							locationID: 5,
							cumulativeValues: []*ProfileTreeValueNode{{
								key: &ProfileTreeValueNodeKey{
									location: "5|3|2|1|0",
								},
								Value: -1,
							}},
							flatValues: []*ProfileTreeValueNode{{
								key: &ProfileTreeValueNodeKey{
									location: "5|3|2|1|0",
								},
								Value: -1,
							}},
						}},
					}},
				}},
			}}},
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
