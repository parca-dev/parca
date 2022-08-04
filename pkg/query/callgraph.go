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

package query

import (
	"context"

	"github.com/google/uuid"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateCallgraph(ctx context.Context, p *profile.Profile) (*pb.Callgraph, error) {
	nodesMap := make(map[string]*pb.CallgraphNode)
	nodes := make([]*pb.CallgraphNode, 0)
	edges := make([]*pb.CallgraphEdge, 0)
	edgesMap := make(map[string]*pb.CallgraphEdge)

	for _, node := range p.Samples {
		var prevNode *pb.CallgraphNode = nil
		for _, location := range node.Locations {
			for _, line := range location.Lines {
				n := nodesMap[line.Function.Name];
				if (n == nil) {
					n = &pb.CallgraphNode{
						Id: line.Function.Id,
						Name: line.Function.Name,
					}
					nodesMap[line.Function.Name] = n
					nodes = append(nodes, n)
				}
				if prevNode != nil {
					key := n.Id + " -> " + prevNode.Id;
					if _, exists := edgesMap[key]; !exists {
						edge := &pb.CallgraphEdge{
							Id: uuid.New().String(),
							Source: n.Id,
							Target: prevNode.Id,
							Visits: 1,
						}
						edges = append(edges, edge)
						edgesMap[key] = edge
					} else {
						edgesMap[key].Visits += 1;
					}
				}
				prevNode = n
			}
		}
	}
	return &pb.Callgraph{Nodes: nodes, Edges: edges}, nil
}
