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

	"github.com/stretchr/testify/require"
)

func TestProfileTreeInsert(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	pt := NewProfileTree()

	s1 := makeSample(2, []uint64{2, 1})
	pt.Insert(s1)

	s2 := makeSample(1, []uint64{5, 3, 2, 1})
	pt.Insert(s2)

	s3 := makeSample(3, []uint64{4, 3, 2, 1})
	s3.Label = label
	s3.NumLabel = numLabel
	s3.NumUnit = numUnit
	pt.Insert(s3)

	require.Equal(t, &ProfileTree{
		Roots: &ProfileTreeRootNode{
			CumulativeValue: 6,
			ProfileTreeNode: &ProfileTreeNode{
				// Roots always have the LocationID 0.
				locationID: 0,
				Children: []*ProfileTreeNode{{
					locationID: 1,
					Children: []*ProfileTreeNode{{
						locationID: 2,
						flatValues: []*ProfileTreeValueNode{{
							key:   &ProfileTreeValueNodeKey{location: "2|1|0"},
							Value: 2,
						}},
						Children: []*ProfileTreeNode{{
							locationID: 3,
							Children: []*ProfileTreeNode{{
								locationID: 4,
								flatValues: []*ProfileTreeValueNode{{
									key:      &ProfileTreeValueNodeKey{location: "4|3|2|1|0", labels: `"foo"["bar" "baz"]`, numlabels: `"foo"[1 2][6279746573 6f626a65637473]`},
									Value:    3,
									Label:    label,
									NumLabel: numLabel,
									NumUnit:  numUnit,
								}},
							}, {
								locationID: 5,
								flatValues: []*ProfileTreeValueNode{{
									key:   &ProfileTreeValueNodeKey{location: "5|3|2|1|0"},
									Value: 1,
								}},
							}},
						}},
					}},
				}},
			},
		},
	}, pt)
}

func TestKeysMap(t *testing.T) {
	m := map[ProfileTreeValueNodeKey]bool{}

	m[ProfileTreeValueNodeKey{location: "0"}] = true
	m[ProfileTreeValueNodeKey{location: "1"}] = true

	if _, ok := m[ProfileTreeValueNodeKey{location: "0"}]; !ok {
		t.Fail()
	}

	m[ProfileTreeValueNodeKey{location: "0", labels: `"foo"["bar"]`}] = true

	if _, ok := m[ProfileTreeValueNodeKey{location: "0"}]; !ok {
		t.Fail()
	}
	if _, ok := m[ProfileTreeValueNodeKey{location: "0", labels: `"foo"["bar"]`}]; !ok {
		t.Fail()
	}
	if _, ok := m[ProfileTreeValueNodeKey{location: "0", labels: `"foo"["baz"]`}]; ok {
		t.Fail()
	}
}
