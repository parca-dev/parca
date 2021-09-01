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

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
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
		Roots: &ProfileTreeNode{
			// Roots always have the LocationID 0.
			locationID: 0,
			cumulativeValues: []*ProfileTreeValueNode{{
				key:   &ProfileTreeValueNodeKey{location: "0"},
				Value: 6,
			}},
			Children: []*ProfileTreeNode{{
				locationID: 1,
				cumulativeValues: []*ProfileTreeValueNode{{
					key:   &ProfileTreeValueNodeKey{location: "1|0"},
					Value: 6,
				}},
				Children: []*ProfileTreeNode{{
					locationID: 2,
					cumulativeValues: []*ProfileTreeValueNode{{
						key:   &ProfileTreeValueNodeKey{location: "2|1|0"},
						Value: 6,
					}},
					flatValues: []*ProfileTreeValueNode{{
						key:   &ProfileTreeValueNodeKey{location: "2|1|0"},
						Value: 2,
					}},
					Children: []*ProfileTreeNode{{
						locationID: 3,
						cumulativeValues: []*ProfileTreeValueNode{{
							key:   &ProfileTreeValueNodeKey{location: "3|2|1|0"},
							Value: 4,
						}},
						Children: []*ProfileTreeNode{{
							locationID: 4,
							cumulativeValues: []*ProfileTreeValueNode{{
								key:      &ProfileTreeValueNodeKey{location: "4|3|2|1|0", labels: `"foo"["bar" "baz"]`, numlabels: `"foo"[1 2][6279746573 6f626a65637473]`},
								Value:    3,
								Label:    label,
								NumLabel: numLabel,
								NumUnit:  numUnit,
							}},
							flatValues: []*ProfileTreeValueNode{{
								key:      &ProfileTreeValueNodeKey{location: "4|3|2|1|0", labels: `"foo"["bar" "baz"]`, numlabels: `"foo"[1 2][6279746573 6f626a65637473]`},
								Value:    3,
								Label:    label,
								NumLabel: numLabel,
								NumUnit:  numUnit,
							}},
						}, {
							locationID: 5,
							cumulativeValues: []*ProfileTreeValueNode{{
								key:   &ProfileTreeValueNodeKey{location: "5|3|2|1|0"},
								Value: 1,
							}},
							flatValues: []*ProfileTreeValueNode{{
								key:   &ProfileTreeValueNodeKey{location: "5|3|2|1|0"},
								Value: 1,
							}},
						}},
					}},
				}},
			}}},
	}, pt)
}

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

	s := NewMemSeries(labels.FromStrings("a", "b"), 0)

	pt1 := NewProfileTree()
	pt1.Insert(s11)
	pt1.Insert(s12)
	err := s.seriesTree.Insert(0, pt1)
	require.NoError(t, err)

	require.Len(t, s.flatValues, 2)
	require.Equal(t, chunkenc.FromValuesXOR(1), s.flatValues[k2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4])

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(3), s.cumulativeValues[k0])
	require.Equal(t, chunkenc.FromValuesXOR(3), s.cumulativeValues[k1])
	require.Equal(t, chunkenc.FromValuesXOR(1), s.cumulativeValues[k2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4])

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
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4]) // sparse - nothing added

	require.Len(t, s.cumulativeValues, 4)
	require.Equal(t, chunkenc.FromValuesXOR(3, 3), s.cumulativeValues[k0])
	require.Equal(t, chunkenc.FromValuesXOR(3, 3), s.cumulativeValues[k1])
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.cumulativeValues[k2])
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4]) // sparse - nothing added

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
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.flatValues[k2])   // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.flatValues[k4])      // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.flatValues[k3]) // new

	require.Len(t, s.cumulativeValues, 5)

	require.Equal(t, chunkenc.FromValuesXOR(3, 3, 4), s.cumulativeValues[k0])
	require.Equal(t, chunkenc.FromValuesXOR(3, 3, 4), s.cumulativeValues[k1])
	require.Equal(t, chunkenc.FromValuesXOR(1, 3), s.cumulativeValues[k2])   // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXOR(2), s.cumulativeValues[k4])      // sparse - nothing added
	require.Equal(t, chunkenc.FromValuesXORAt(2, 4), s.cumulativeValues[k3]) // new

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

func TestMemSeriesIterator(t *testing.T) {
	var (
		label    = map[string][]string{"foo": {"bar", "baz"}}
		numLabel = map[string][]int64{"foo": {1, 2}}
		numUnit  = map[string][]string{"foo": {"bytes", "objects"}}
	)

	s := NewMemSeries(labels.FromStrings("a", "b"), 0)

	s.timestamps = chunkenc.FromValuesDelta(1, 2)
	s.durations = chunkenc.FromValuesRLE(time.Second.Nanoseconds(), 2)
	s.periods = chunkenc.FromValuesRLE(100, 2)

	{
		pt := NewProfileTree()
		{
			s := makeSample(1, []uint64{2, 1})
			pt.Insert(s)
		}
		{
			s := makeSample(2, []uint64{4, 1})
			s.Label = label
			s.NumLabel = numLabel
			s.NumUnit = numUnit
			pt.Insert(s)
		}

		err := s.appendTree(pt)
		s.numSamples++
		require.NoError(t, err)
	}
	{
		pt := NewProfileTree()
		{
			pt.Insert(makeSample(3, []uint64{3, 1}))
		}
		{
			pt.Insert(makeSample(4, []uint64{4, 1}))
		}

		err := s.appendTree(pt)
		s.numSamples++
		require.NoError(t, err)
	}
	it := s.Iterator()

	// First iteration
	{
		require.True(t, it.Next())
		require.NoError(t, it.Err())
		instantProfile := it.At()
		require.Equal(t, InstantProfileMeta{
			Timestamp: 1,
			Duration:  time.Second.Nanoseconds(),
			Period:    100,
		}, instantProfile.ProfileMeta())

		expected := []struct {
			LocationID       uint64
			CumulativeValues []*ProfileTreeValueNode
			FlatValues       []*ProfileTreeValueNode
		}{
			{
				LocationID:       0,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       2,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 1}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 1}},
			},
			{
				LocationID:       3,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0}},
			},
			{
				LocationID:       4,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 2, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 0}},
			},
		}

		i := 0
		err := WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
			require.Equal(t, expected[i].LocationID, n.LocationID())
			require.Equal(t, expected[i].CumulativeValues, n.CumulativeValues())
			require.Equal(t, expected[i].FlatValues, n.FlatValues())
			i++
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 5, i) // seen 5 nodes
	}

	// Second iteration
	{
		require.True(t, it.Next())
		require.NoError(t, it.Err())
		instantProfile := it.At()
		require.Equal(t, InstantProfileMeta{
			Timestamp: 2,
			Duration:  time.Second.Nanoseconds(),
			Period:    100,
		}, instantProfile.ProfileMeta())

		expected := []struct {
			LocationID       uint64
			CumulativeValues []*ProfileTreeValueNode
			FlatValues       []*ProfileTreeValueNode
		}{
			{
				LocationID:       0,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 7}},
			},
			{
				LocationID:       1,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 7}},
			},
			{
				LocationID:       2,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0}},
			},
			{
				LocationID:       3,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 3}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 3}},
			},
			{
				LocationID:       4,
				CumulativeValues: []*ProfileTreeValueNode{{Value: 0, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 4}},
				FlatValues:       []*ProfileTreeValueNode{{Value: 0, Label: label, NumLabel: numLabel, NumUnit: numUnit}, {Value: 4}},
			},
		}

		i := 0
		err := WalkProfileTree(instantProfile.ProfileTree(), func(n InstantProfileTreeNode) error {
			require.Equal(t, expected[i].LocationID, n.LocationID())
			require.Equal(t, expected[i].CumulativeValues, n.CumulativeValues())
			require.Equal(t, expected[i].FlatValues, n.FlatValues())
			i++
			return nil
		})
		require.NoError(t, err)
		require.Equal(t, 5, i)
	}

	// No more iterations
	require.False(t, it.Next())
}

func TestIteratorConsistency(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	profile := ProfileFromPprof(l, p1, 0)
	require.NoError(t, app.Append(profile))

	profileTree := profile.Tree

	res1 := []uint64{}
	err = WalkProfileTree(profileTree, func(n InstantProfileTreeNode) error {
		res1 = append(res1, n.LocationID())
		return nil
	})
	require.NoError(t, err)

	sit := s.Iterator()
	require.True(t, sit.Next())
	require.NoError(t, sit.Err())

	res2 := []uint64{}
	err = WalkProfileTree(sit.At().ProfileTree(), func(n InstantProfileTreeNode) error {
		res2 = append(res2, n.LocationID())
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, res1, res2)
}

func TestRealInsert(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	profile := ProfileFromPprof(l, p, 0)
	require.NoError(t, app.Append(profile))
	require.Equal(t, len(p.Location), len(l.locations))
}

func TestRealInserts(t *testing.T) {
	os.Remove("result-profile1.pb.gz")
	os.Remove("result-profile2.pb.gz")

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	f, err = os.Open("testdata/profile2.pb.gz")
	require.NoError(t, err)
	p2, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l := NewInMemoryProfileMetaStore()
	s := NewMemSeries(labels.Labels{{Name: "test_name", Value: "test_value"}}, 1)
	require.NoError(t, err)
	app, err := s.Appender()
	require.NoError(t, err)
	prof1 := ProfileFromPprof(l, p1, 0)
	prof2 := ProfileFromPprof(l, p2, 0)
	require.NoError(t, app.Append(prof1))
	require.NoError(t, app.Append(prof2))

	it := s.Iterator()
	require.True(t, it.Next())
	require.Equal(t, int64(1626013307085), it.At().ProfileMeta().Timestamp)
	require.True(t, it.Next())
	require.Equal(t, int64(1626014267084), it.At().ProfileMeta().Timestamp)
	require.False(t, it.Next())
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

func TestGetIndexRange(t *testing.T) {
	c := chunkenc.FromValuesDelta(2, 4, 6, 7, 8)

	ts, startIndex, endIndex, err := getIndexRange(c.Iterator(nil), 1, 9)
	require.NoError(t, err)
	require.Equal(t, int64(2), ts)
	require.Equal(t, 0, startIndex)
	require.Equal(t, 5, endIndex)

	ts, startIndex, endIndex, err = getIndexRange(c.Iterator(nil), 2, 9)
	require.NoError(t, err)
	require.Equal(t, int64(2), ts)
	require.Equal(t, 0, startIndex)
	require.Equal(t, 5, endIndex)

	ts, startIndex, endIndex, err = getIndexRange(c.Iterator(nil), 3, 6)
	require.NoError(t, err)
	require.Equal(t, int64(4), ts)
	require.Equal(t, 1, startIndex)
	require.Equal(t, 3, endIndex)

	ts, startIndex, endIndex, err = getIndexRange(c.Iterator(nil), 3, 7)
	require.NoError(t, err)
	require.Equal(t, int64(4), ts)
	require.Equal(t, 1, startIndex)
	require.Equal(t, 4, endIndex)

	ts, startIndex, endIndex, err = getIndexRange(c.Iterator(nil), 3, 8)
	require.NoError(t, err)
	require.Equal(t, int64(4), ts)
	require.Equal(t, 1, startIndex)
	require.Equal(t, 5, endIndex)

	ts, startIndex, endIndex, err = getIndexRange(c.Iterator(nil), 3, 9)
	require.NoError(t, err)
	require.Equal(t, int64(4), ts)
	require.Equal(t, 1, startIndex)
	require.Equal(t, 5, endIndex)
}

func TestIteratorRangeSum(t *testing.T) {
	c := chunkenc.FromValuesDelta(2, 4, 6, 7, 8)
	_, startIndex, endIndex, err := getIndexRange(c.Iterator(nil), 3, 6)
	require.NoError(t, err)

	sum, err := iteratorRangeSum(c.Iterator(nil), startIndex, endIndex)
	require.NoError(t, err)
	require.Equal(t, int64(10), sum)
}

func TestIteratorRangeMax(t *testing.T) {
	c := chunkenc.FromValuesDelta(10, 4, 12, 7, 8)
	max, err := iteratorRangeMax(c.Iterator(nil), 1, 5)
	require.NoError(t, err)
	require.Equal(t, int64(12), max)
}

func TestMergeMemSeriesConsistency(t *testing.T) {
	s := NewInMemoryProfileMetaStore()
	f, err := os.Open("./testdata/profile1.pb.gz")
	require.NoError(t, err)
	pprof1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	p := ProfileFromPprof(s, pprof1, 0)

	ctx := context.Background()
	db := OpenDB(prometheus.NewRegistry())

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

	s := NewMemSeries(labels.FromStrings("a", "b"), 0)

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
		MemSeries: s,
		mint:      0,
		maxt:      2,
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
