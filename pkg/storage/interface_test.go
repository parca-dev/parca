package storage

import (
	"os"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
)

func TestCopyInstantProfileTree(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	profileTree := ProfileTreeFromPprof(l, p1, 0)

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
