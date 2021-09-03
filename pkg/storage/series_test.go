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

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestMemSeriesTree(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	// Note: These keys are not unique per location.
	// For this test they simply seem to be.
	k0 := ProfileTreeValueNodeKey{location: "0"}
	k1 := ProfileTreeValueNodeKey{location: "1|0"}
	k2 := ProfileTreeValueNodeKey{location: "2|1|0"}
	k3 := ProfileTreeValueNodeKey{location: "3|1|0"}
	k4 := ProfileTreeValueNodeKey{location: "4|1|0", labels: `"foo"["bar" "baz"]`, numlabels: `"foo"[1 2][6279746573 6f626a65637473]`}

	s11 := makeSample(1, []uint64{2, 1})

	s12 := makeSample(2, []uint64{4, 1})
	s12.Label = label
	s12.NumLabel = numLabel
	s12.NumUnit = numUnit

	s := NewMemSeries(0, labels.FromStrings("a", "b"), func(int64) {})

	pt1 := NewProfileTree()
	pt1.Insert(s11)
	pt1.Insert(s12)
	err := s.seriesTree.Insert(0, pt1)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1), s.flatValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0])

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(3), s.cumulativeValues[k0][0])
	require.Equal(t, chunkenc.FromValuesXOR(3), s.cumulativeValues[k1][0])
	require.Equal(t, chunkenc.FromValuesXOR(1), s.cumulativeValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4][0])

	require.Len(t, s.labels, 1)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]string{k4: label}, s.labels)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]int64{k4: numLabel}, s.numLabels)
	require.Equal(t, map[ProfileTreeValueNodeKey]map[string][]string{k4: numUnit}, s.numUnits)

	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			keys:       []ProfileTreeValueNodeKey{{location: "0"}},
			LocationID: 0, // root
			Children: []*MemSeriesTreeNode{{
				keys:       []ProfileTreeValueNodeKey{{location: "1|0"}},
				LocationID: 1,
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: 2,
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: 4,
				}},
			}},
		},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one

	s3 := makeSample(3, []uint64{2, 1})

	pt2 := NewProfileTree()
	pt2.Insert(s3)
	err = s.seriesTree.Insert(1, pt2)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0]) // sparse - nothing added

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(3, 3), s.cumulativeValues[k0][0])
	require.Equal(t, chunkenc.FromValuesXOR(3, 3), s.cumulativeValues[k1][0])
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.cumulativeValues[k2][0])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4][0]) // sparse - nothing added

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			keys:       []ProfileTreeValueNodeKey{{location: "0"}},
			LocationID: 0, // root
			Children: []*MemSeriesTreeNode{{
				keys:       []ProfileTreeValueNodeKey{{location: "1|0"}},
				LocationID: 1,
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: 2,
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: 4,
				}},
			}},
		},
	}, s.seriesTree)

	// Merging another profileTree onto the existing one with one new Location
	s4 := makeSample(4, []uint64{3, 1})

	pt3 := NewProfileTree()
	pt3.Insert(s4)
	err = s.seriesTree.Insert(2, pt3)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 3)
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2][0])   // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4][0])      // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.flatValues[k3][0]) // new

	require.Len(t, s.cumulativeValues, 5)

	require.Equal(t, chunkenc.FromValuesXOR(3, 3, 4), s.cumulativeValues[k0][0])
	require.Equal(t, chunkenc.FromValuesXOR(3, 3, 4), s.cumulativeValues[k1][0])
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.cumulativeValues[k2][0])   // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4][0])      // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.cumulativeValues[k3][0]) // new

	// The tree itself didn't change by adding more values but no new locations.
	require.Equal(t, &MemSeriesTree{
		s: s,
		Roots: &MemSeriesTreeNode{
			keys:       []ProfileTreeValueNodeKey{{location: "0"}},
			LocationID: 0, // root
			Children: []*MemSeriesTreeNode{{
				keys:       []ProfileTreeValueNodeKey{{location: "1|0"}},
				LocationID: 1,
				Children: []*MemSeriesTreeNode{{
					keys:       []ProfileTreeValueNodeKey{k2},
					LocationID: 2,
				}, {
					keys:       []ProfileTreeValueNodeKey{k3},
					LocationID: 3,
				}, {
					keys:       []ProfileTreeValueNodeKey{k4},
					LocationID: 4,
				}},
			}},
		},
	}, s.seriesTree)
}
