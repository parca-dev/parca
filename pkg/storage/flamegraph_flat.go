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

func (fp *FlatProfile) ProfileMeta() InstantProfileMeta {
	return fp.Meta
}

func (fp *FlatProfile) Samples() map[string]*Sample {
	return fp.samples
}

func GenerateFlamegraphFlat(ctx context.Context, tracer trace.Tracer, metaStore metastore.ProfileMetaStore, p InstantFlatProfile) (*pb.Flamegraph, error) {
	rootNode := &pb.FlamegraphNode{}
	current := rootNode

	samples := p.Samples()

	sampleUUIDs := make([][]byte, 0, len(samples))
	for id := range samples {
		sampleUUIDs = append(sampleUUIDs, []byte(id))
	}

	sampleMap, err := metaStore.GetStacktraceByIDs(ctx, sampleUUIDs...)
	if err != nil {
		return nil, err
	}

	locationUUIDSeen := map[string]struct{}{}
	locationUUIDs := [][]byte{}
	for _, s := range sampleMap {
		for _, id := range s.GetLocationIds() {
			if _, seen := locationUUIDSeen[string(id)]; !seen {
				locationUUIDSeen[string(id)] = struct{}{}
				locationUUIDs = append(locationUUIDs, id)
			}
		}
	}

	// Get the full locations for the location UUIDs
	locationsMap, err := metastore.GetLocationsByIDs(ctx, metaStore, locationUUIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	var height int32

	for k, s := range samples {
		locations := sampleMap[k].GetLocationIds()
		if int32(len(locations)) > height {
			height = int32(len(locations))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(locations) - 1; i >= 0; i-- {
			location := locationsMap[string(locations[i])] // use the fully populated location

			nodes := locationToTreeNodes(location)
			for j := len(nodes) - 1; j >= 0; j-- {
				node := nodes[j]

				index := sort.Search(len(current.GetChildren()), func(i int) bool {
					cmp := bytes.Compare(current.GetChildren()[i].GetMeta().GetLocation().GetId(), node.GetMeta().GetLocation().GetId())
					return cmp == 0 || cmp == 1
				})

				if index < len(current.GetChildren()) && bytes.Equal(
					current.GetChildren()[index].GetMeta().GetLocation().GetId(),
					node.GetMeta().GetLocation().GetId(),
				) {
					// Insert onto existing node
					current = current.Children[index]
					current.Cumulative += s.Value
					current.Diff += s.DiffValue
				} else {
					// Insert new node
					node.Cumulative += s.Value
					node.Diff += s.DiffValue

					newChildren := make([]*pb.FlamegraphNode, len(current.Children)+1)
					copy(newChildren, current.Children[:index])

					newChildren[index] = node
					copy(newChildren[index+1:], current.Children[index:])
					current.Children = newChildren

					current = node

					// There is a case where locationToTreeNodes returns the node pointing to its parent,
					// resulting in an endless loop. We remove all possible children and add them later ourselves.
					current.Children = nil
				}
			}
		}

		// Sum up the value to the cumulative value of the root
		rootNode.Cumulative += s.Value
		rootNode.Diff += s.DiffValue

		// For next sample start at the root again
		current = rootNode
	}

	flamegraph := &pb.Flamegraph{Root: &pb.FlamegraphRootNode{}}
	flamegraph.Total = rootNode.Cumulative
	flamegraph.Height = height + 1 // add one for the root
	flamegraph.Root.Cumulative = rootNode.Cumulative
	flamegraph.Root.Diff = rootNode.Diff
	flamegraph.Root.Children = rootNode.Children

	return aggregateByFunction(flamegraph), nil
}
