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
	"fmt"
	"sort"

	"github.com/google/uuid"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"go.opentelemetry.io/otel/trace"
)

type InstantFlatProfile interface {
	ProfileMeta() InstantProfileMeta
	Samples() []*Sample
}

type FlatProfile struct {
	Meta    InstantProfileMeta
	samples []*Sample
}

func (fp *FlatProfile) ProfileMeta() InstantProfileMeta {
	return fp.Meta
}

func (fp *FlatProfile) Samples() []*Sample {
	return fp.samples
}

func GenerateFlamegraphFlat(ctx context.Context, tracer trace.Tracer, locations Locations, p InstantFlatProfile) (*pb.Flamegraph, error) {
	rootNode := &pb.FlamegraphNode{}
	cur := rootNode

	var height int32

	locationUUIDSeen := map[uuid.UUID]struct{}{}
	locationUUIDs := []uuid.UUID{}
	for _, s := range p.Samples() {
		for _, l := range s.Location {
			if _, seen := locationUUIDSeen[l.ID]; !seen {
				locationUUIDSeen[l.ID] = struct{}{}
				locationUUIDs = append(locationUUIDs, l.ID)
			}
		}
	}
	// Get the full locations for the location UUIDs
	locationsMap, err := locations.GetLocationsByIDs(ctx, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	for _, s := range p.Samples() {
		var child *pb.FlamegraphNode

		if int32(len(s.Location)) > height {
			height = int32(len(s.Location))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(s.Location) - 1; i >= 0; i-- {
			location := locationsMap[s.Location[i].ID] // use the fully populated location
			nextID := location.ID

			index := sort.Search(len(cur.Children), func(i int) bool {
				cmp := uuidCompare(uuid.MustParse(cur.Children[i].GetMeta().GetLocation().GetId()), nextID)
				return cmp == 0 || cmp == 1
			})

			if index < len(cur.Children) && uuid.MustParse(cur.Children[index].GetMeta().GetLocation().GetId()) == nextID {
				child = cur.Children[index]
			} else {
				newChildren := make([]*pb.FlamegraphNode, len(cur.Children)+1)
				copy(newChildren, cur.Children[:index])

				nodes := locationToTreeNodes(location)
				for i, n := range nodes {
					if i == 0 {
						// Ignore the first node as we add to it later
						continue
					}
					n.Cumulative += s.Value
				}

				child = nodes[0]
				newChildren[index] = child
				copy(newChildren[index+1:], cur.Children[index:])
				cur.Children = newChildren
			}

			cur = child

			// Add the value to the cumulative value for each node
			cur.Cumulative += s.Value
		}

		// Sum up the value to the cumulative value of the root
		rootNode.Cumulative += s.Value

		// For next sample start at the root again
		cur = rootNode
	}

	flamegraph := &pb.Flamegraph{Root: &pb.FlamegraphRootNode{}}
	flamegraph.Total = rootNode.Cumulative
	flamegraph.Height = height + 1 // add one for the root
	flamegraph.Root.Cumulative = rootNode.Cumulative
	flamegraph.Root.Diff = rootNode.Diff
	flamegraph.Root.Children = rootNode.Children

	return flamegraph, nil
}
