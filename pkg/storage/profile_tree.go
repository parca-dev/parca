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
	"bytes"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type ProfileTree struct {
	Roots *ProfileTreeRootNode
}

func NewProfileTree() *ProfileTree {
	return &ProfileTree{
		Roots: &ProfileTreeRootNode{
			ProfileTreeNode: &ProfileTreeNode{},
		},
	}
}

func ProfileTreeFromPprof(ctx context.Context, l log.Logger, s metastore.ProfileMetaStore, p *profile.Profile, sampleIndex int) (*ProfileTree, error) {
	pn := &profileNormalizer{
		logger:    l,
		metaStore: s,

		samples: make(map[stacktraceKey]*Sample, len(p.Sample)),

		// Profile-specific hash tables for each profile inserted.
		locationsByID: make(map[uint64]*metastore.Location, len(p.Location)),
		functionsByID: make(map[uint64]*metastore.Function, len(p.Function)),
		mappingsByID:  make(map[uint64]mapInfo, len(p.Mapping)),
	}

	samples := make([]*Sample, 0, len(p.Sample))
	for _, s := range p.Sample {
		if !isZeroSample(s) {
			sa, isNew, err := pn.mapSample(ctx, s, sampleIndex)
			if err != nil {
				return nil, err
			}
			if isNew {
				samples = append(samples, sa)
			}
		}
	}

	profileTree := NewProfileTree()
	for _, s := range samples {
		profileTree.Insert(s)
	}

	return profileTree, nil
}

func (t *ProfileTree) RootCumulativeValue() int64 {
	return t.Roots.CumulativeValue
}

func (t *ProfileTree) Iterator() InstantProfileTreeIterator {
	return NewProfileTreeIterator(t)
}

func uuidCompare(a, b uuid.UUID) int {
	ab := [16]byte(a)
	bb := [16]byte(b)
	return bytes.Compare(ab[:], bb[:])
}

func (t *ProfileTree) Insert(sample *Sample) {
	cur := t.Roots.ProfileTreeNode
	locations := sample.Location

	locationIDs := make([]uuid.UUID, 0, len(sample.Location)+1)
	for _, l := range sample.Location {
		locationIDs = append(locationIDs, l.ID)
	}
	locationIDs = append(locationIDs, uuid.UUID{}) // add the root

	for i := len(locations) - 1; i >= 0; i-- {
		nextId := locations[i].ID

		var child *ProfileTreeNode

		// Binary search for child in list. If it exists continue to use the existing one.
		index := sort.Search(len(cur.Children), func(i int) bool {
			cmp := uuidCompare(cur.Children[i].LocationID(), nextId)
			return cmp == 0 || cmp == 1
		})
		if index < len(cur.Children) && cur.Children[index].LocationID() == nextId {
			// Child with this ID already exists.
			child = cur.Children[index]
		} else {
			// No child with ID exists, but it should be inserted at `index`.
			newChildren := make([]*ProfileTreeNode, len(cur.Children)+1)
			copy(newChildren, cur.Children[:index])
			child = &ProfileTreeNode{
				locationID: nextId,
			}
			newChildren[index] = child
			copy(newChildren[index+1:], cur.Children[index:])
			cur.Children = newChildren
		}

		cur = child
	}

	if cur.flatValues == nil {
		cur.flatValues = []*ProfileTreeValueNode{{}}
	}
	cur.flatValues[0].Value += sample.Value
	// TODO: We probably need to merge labels, numLabels and numUnits
	cur.flatValues[0].Label = sample.Label
	cur.flatValues[0].NumLabel = sample.NumLabel
	cur.flatValues[0].NumUnit = sample.NumUnit

	t.Roots.CumulativeValue += sample.Value

	for _, fv := range cur.flatValues {
		fv.Key(locationIDs...) //populate the keys
	}
}

// ProfileTreeRootNode is just like a ProfileTreeNode except it can
// additionally hold the cumulative value of the root as an optimization.
type ProfileTreeRootNode struct {
	CumulativeValue int64
	*ProfileTreeNode
}

type ProfileTreeNode struct {
	locationID     uuid.UUID
	flatValues     []*ProfileTreeValueNode
	flatDiffValues []*ProfileTreeValueNode
	Children       []*ProfileTreeNode
}

func (n *ProfileTreeNode) LocationID() uuid.UUID {
	return n.locationID
}

func (n *ProfileTreeNode) FlatDiffValues() []*ProfileTreeValueNode {
	return n.flatDiffValues
}

func (n *ProfileTreeNode) FlatValues() []*ProfileTreeValueNode {
	return n.flatValues
}

type ProfileTreeValueNode struct {
	key *ProfileTreeValueNodeKey

	Value    int64
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type ProfileTreeValueNodeKey struct {
	location  string
	labels    string
	numlabels string
}

func (k *ProfileTreeValueNodeKey) Equals(o ProfileTreeValueNodeKey) bool {
	if k.location != o.location {
		return false
	}
	if k.labels != o.labels {
		return false
	}
	if k.numlabels != o.numlabels {
		return false
	}
	return true
}

func (n *ProfileTreeValueNode) Key(locationIDs ...uuid.UUID) {
	if n.key != nil {
		return
	}

	ids := make([]string, len(locationIDs))
	for i, l := range locationIDs {
		ids[i] = l.String()
	}

	labels := make([]string, 0, len(n.Label))
	for k, v := range n.Label {
		labels = append(labels, fmt.Sprintf("%q%q", k, v))
	}
	sort.Strings(labels)

	numlabels := make([]string, 0, len(n.NumLabel))
	for k, v := range n.NumLabel {
		numlabels = append(numlabels, fmt.Sprintf("%q%x%x", k, v, n.NumUnit[k]))
	}
	sort.Strings(numlabels)

	n.key = &ProfileTreeValueNodeKey{
		location:  strings.Join(ids, "|"),
		labels:    strings.Join(labels, ""),
		numlabels: strings.Join(numlabels, ""),
	}
}

func isZeroSample(s *profile.Sample) bool {
	for _, v := range s.Value {
		if v != 0 {
			return false
		}
	}
	return true
}
