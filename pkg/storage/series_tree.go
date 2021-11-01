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
	"fmt"

	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
)

type MemSeriesTreeNode struct {
	keys []ProfileTreeValueNodeKey

	LocationID uuid.UUID
	Children   []*MemSeriesTreeNode
}

func (n *MemSeriesTreeNode) addKey(key ProfileTreeValueNodeKey) {
	for _, k := range n.keys {
		if k.Equals(key) {
			return
		}
	}
	n.keys = append(n.keys, key)
}

type MemSeriesTree struct {
	s     *MemSeries
	Roots *MemSeriesTreeNode
}

func (t *MemSeriesTree) Iterator() *MemSeriesTreeIterator {
	return NewMemSeriesTreeIterator(t)
}

func (t *MemSeriesTree) Insert(index uint16, profileTree *ProfileTree) error {
	if t.Roots == nil {
		t.Roots = &MemSeriesTreeNode{}
	}

	pit := profileTree.Iterator()
	sit := t.Iterator()

	for pit.HasMore() {
		if pit.NextChild() {
			profileTreeChild := pit.At()
			pId := profileTreeChild.LocationID()

			done := false
			for {
				if !sit.NextChild() {
					node := sit.Node()
					seriesTreeChild := &MemSeriesTreeNode{
						LocationID: pId,
					}

					for _, n := range profileTreeChild.FlatValues() {
						if n.key == nil {
							n.Key(profileTreeChild.LocationID())
						}

						if len(t.s.flatValues[*n.key]) == 0 {
							// Create the needed amount of chunks based on how many timestamp chunks there are.
							t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
							for i := 0; i < len(t.s.timestamps); i++ {
								t.s.flatValues[*n.key][i] = t.s.chunkPool.GetXOR()
							}
						}
						app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
						if err != nil {
							return fmt.Errorf("failed to open flat appender: %w", err)
						}
						app.AppendAt(index, n.Value)

						// We need to keep track of the node keys.
						seriesTreeChild.addKey(*n.key)

						if len(n.Label) > 0 {
							if t.s.labels[*n.key] == nil {
								t.s.labels[*n.key] = n.Label
							}

							if t.s.numLabels[*n.key] == nil {
								t.s.numLabels[*n.key] = n.NumLabel
							}

							if t.s.numUnits[*n.key] == nil {
								t.s.numUnits[*n.key] = n.NumUnit
							}
						}
					}

					node.Children = append(node.Children, seriesTreeChild)

					pit.StepInto()
					sit.StepInto()
					done = true
					break
				}
				sId := sit.At().LocationID
				cmp := uuidCompare(pId, sId)
				if cmp <= 0 {
					break
				}
			}
			if done {
				continue
			}

			seriesTreeChild := sit.At()
			sId := seriesTreeChild.LocationID

			// The node with the location id in the profile-tree is the same (except Location ID 0 - the root),
			// this means this node present in the series-tree, so we need add the new values to the existing node.
			if pId == sId {
				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					// Even if the location exists.
					// labels can be different and then the key is different, so we need check.
					if len(t.s.flatValues[*n.key]) == 0 {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.flatValues[*n.key][i] = t.s.chunkPool.GetXOR()
						}
					}
					app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
					if err != nil {
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)

					// We need to keep track of the node IDs.
					seriesTreeChild.addKey(*n.key)
				}

				pit.StepInto()
				sit.StepInto()
				continue
			}

			// The node with the location id in the profile-tree is smaller,
			// this means this node is not present yet in the series-tree, so it has to be added at the current child position.
			if uuidCompare(pId, sId) == -1 {
				node := sit.Node()
				childIndex := sit.ChildIndex()
				newChildren := make([]*MemSeriesTreeNode, len(node.Children)+1)
				copy(newChildren, node.Children[:childIndex])
				newChild := &MemSeriesTreeNode{
					LocationID: pId,
				}

				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					if len(t.s.flatValues[*n.key]) == 0 {
						// Create the needed amount of chunks based on how many timestamp chunks there are.
						t.s.flatValues[*n.key] = make([]chunkenc.Chunk, len(t.s.timestamps))
						for i := 0; i < len(t.s.timestamps); i++ {
							t.s.flatValues[*n.key][i] = t.s.chunkPool.GetXOR()
						}
					}
					app, err := t.s.flatValues[*n.key][len(t.s.flatValues[*n.key])-1].Appender()
					if err != nil {
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)

					// We need to keep track of the node keys.
					newChild.addKey(*n.key)
				}

				newChildren[childIndex] = newChild
				copy(newChildren[childIndex+1:], node.Children[childIndex:])
				node.Children = newChildren

				pit.StepInto()
				sit.StepInto()
				continue
			}
		}
		pit.StepUp()
		sit.StepUp()
	}

	return nil
}
