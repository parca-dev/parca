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

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"go.opentelemetry.io/otel/trace"
)

type InstantFlatProfile interface {
	ProfileMeta() InstantProfileMeta
	Samples() map[string]*Sample
}

type FlatProfile struct {
	Meta    InstantProfileMeta
	samples map[string]*Sample
}

func (fp *FlatProfile) ProfileTree() InstantProfileTree {
	panic("won't be implement - use Profile instead")
}

func (fp *FlatProfile) ProfileMeta() InstantProfileMeta {
	return fp.Meta
}

func (fp *FlatProfile) Samples() map[string]*Sample {
	return fp.samples
}

func GenerateFlamegraphFlat(ctx context.Context, tracer trace.Tracer, metaStore metastore.ProfileMetaStore, p InstantFlatProfile) (*pb.Flamegraph, error) {
	rootNode := &pb.FlamegraphNode{}
	cur := rootNode

	var height int32

	locationUUIDSeen := map[string]struct{}{}
	locationUUIDs := [][]byte{}
	for _, s := range p.Samples() {
		for _, l := range s.Location {
			if _, seen := locationUUIDSeen[string(l.ID[:])]; !seen {
				locationUUIDSeen[string(l.ID[:])] = struct{}{}
				locationUUIDs = append(locationUUIDs, l.ID[:])
			}
		}
	}
	// Get the full locations for the location UUIDs
	locationsMap, err := metastore.GetLocationsByIDs(ctx, metaStore, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	for _, s := range p.Samples() {
		if int32(len(s.Location)) > height {
			height = int32(len(s.Location))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(s.Location) - 1; i >= 0; i-- {
			location := locationsMap[string(s.Location[i].ID[:])] // use the fully populated location
			nextID := location.ID

			index := sort.Search(len(cur.Children), func(i int) bool {
				cmp := bytes.Compare(cur.Children[i].GetMeta().GetLocation().GetId(), nextID[:])
				return cmp == 0 || cmp == 1
			})

			if index < len(cur.Children) && bytes.Equal(cur.Children[index].GetMeta().GetLocation().GetId(), nextID[:]) {
				// The node already exists in the flamegraph, we can simply add the value and diff value and continue with this child's children next.
				cur = cur.Children[index]
				cur.Cumulative += s.Value
				cur.Diff += s.DiffValue
			} else {
				nodes := locationToTreeNodes(location)
				if len(nodes) > 1 {
					// There are multiple inlined nodes.
					// We iterate over these nodes, adding value and diff value,
					// and make the first node the leaf whereas the last is the root of this subtree.
					// In the end we insert the last node (sub tree root node) as child to the cur children.
					// We set the leaf node (first node) as cur to continue inserting the next nodes with it.
					for i := len(nodes) - 1; i >= 0; i-- {
						node := nodes[i]
						node.Cumulative += s.Value
						node.Diff += s.DiffValue
						if i > 0 {
							node.Children = []*pb.FlamegraphNode{nodes[i-1]}
						} else {
							node.Children = nil
						}
					}

					newChildren := make([]*pb.FlamegraphNode, len(cur.Children)+1)
					copy(newChildren, cur.Children[:index])
					newChildren[index] = nodes[len(nodes)-1] // Add the root of these nodes to the current children
					copy(newChildren[index+1:], cur.Children[index:])
					cur.Children = newChildren

					cur = nodes[0] // Continue with the leaf of these nodes
				} else {
					// There is only one node to insert and the node hasn't been in the flame graph before.
					// The value and diff value are added to the node, and
					// we insert the node to the correct index within the node's current children.
					node := nodes[0]
					node.Cumulative += s.Value
					node.Diff += s.DiffValue

					newChildren := make([]*pb.FlamegraphNode, len(cur.Children)+1)
					copy(newChildren, cur.Children[:index])

					newChildren[index] = node
					copy(newChildren[index+1:], cur.Children[index:])
					cur.Children = newChildren

					cur = node
				}
			}
		}

		// Sum up the value to the cumulative value of the root
		rootNode.Cumulative += s.Value
		rootNode.Diff += s.DiffValue

		// For next sample start at the root again
		cur = rootNode
	}

	flamegraph := &pb.Flamegraph{Root: &pb.FlamegraphRootNode{}}
	flamegraph.Total = rootNode.Cumulative
	flamegraph.Height = height + 1 // add one for the root
	flamegraph.Root.Cumulative = rootNode.Cumulative
	flamegraph.Root.Diff = rootNode.Diff
	flamegraph.Root.Children = rootNode.Children

	return aggregateByFunction(flamegraph), nil
}
