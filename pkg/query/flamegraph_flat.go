// Copyright 2022 The Parca Authors
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

package query

import (
	"context"
	"sort"

	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateFlamegraphFlat(ctx context.Context, tracer trace.Tracer, p *profile.Profile) (*pb.Flamegraph, error) {
	rootNode := &pb.FlamegraphNode{}
	current := rootNode

	var height int32

	for _, s := range p.Samples {
		locations := s.Locations
		if int32(len(locations)) > height {
			height = int32(len(locations))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(locations) - 1; i >= 0; i-- {
			location := locations[i]

			nodes := locationToTreeNodes(location)
			for j := len(nodes) - 1; j >= 0; j-- {
				node := nodes[j]

				index := sort.Search(len(current.Children), func(i int) bool {
					return current.Children[i].Meta.Location.Id >= node.Meta.Location.Id
				})

				if index < len(current.GetChildren()) && current.Children[index].Meta.Location.Id == node.Meta.Location.Id {
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

	flamegraph := &pb.Flamegraph{
		Root: &pb.FlamegraphRootNode{
			Cumulative: rootNode.Cumulative,
			Diff:       rootNode.Diff,
			Children:   rootNode.Children,
		},
		Total:  rootNode.Cumulative,
		Unit:   p.Meta.SampleType.Unit,
		Height: height + 1, // add one for the root
	}

	return aggregateByFunction(flamegraph), nil
}
