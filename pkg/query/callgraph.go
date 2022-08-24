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
	"strconv"

	"github.com/google/uuid"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	NodeCutOffFraction = 0.005
)

func GenerateCallgraph(ctx context.Context, p *profile.Profile) (*querypb.Callgraph, error) {
	nodesMap := make(map[string]*querypb.CallgraphNode)
	nodes := make([]*querypb.CallgraphNode, 0)
	edges := make([]*querypb.CallgraphEdge, 0)
	edgesMap := make(map[string]*querypb.CallgraphEdge)
	cummValue := int64(0)

	for _, s := range p.Samples {
		cummValue += s.Value
		var prevNode *querypb.CallgraphNode = nil
		for _, location := range s.Locations {
			locationNodes := locationToCallgraphNodes(location)
			for _, n := range locationNodes {
				key := getNodeKey(n)
				if _, exists := nodesMap[key]; !exists {
					nodesMap[key] = n
					nodes = append(nodes, n)
				}
				currentNode := nodesMap[key]
				currentNode.Cumulative += s.Value
				currentNodeId := currentNode.Id

				if prevNode != nil {
					key := currentNodeId + " -> " + prevNode.Id
					if _, exists := edgesMap[key]; !exists {
						edge := &querypb.CallgraphEdge{
							Id:         uuid.New().String(),
							Source:     currentNodeId,
							Target:     prevNode.Id,
							Cumulative: s.Value,
						}
						edges = append(edges, edge)
						edgesMap[key] = edge
					} else {
						edgesMap[key].Cumulative += s.Value
					}
				}
				prevNode = currentNode
			}
		}
	}
	return pruneGraph(&querypb.Callgraph{Nodes: nodes, Edges: edges, Cumulative: cummValue}), nil
}

func getNodeKey(node *querypb.CallgraphNode) string {
	if node.Meta.Function == nil {
		return node.Meta.Location.Id
	}

	return node.Meta.Function.Name
}

// locationToCallgraphNodes converts a location to its tree nodes, if the location
// has multiple inlined functions it creates multiple nodes for each inlined
// function.
func locationToCallgraphNodes(location *profile.Location) []*querypb.CallgraphNode {
	if len(location.Lines) > 0 {
		return linesToCallgraphNodes(
			location,
			location.Mapping,
			location.Lines,
		)
	}

	var mappingID string
	if location.Mapping != nil {
		mappingID = location.Mapping.Id
	}
	return []*querypb.CallgraphNode{{
		Id: location.ID,
		Meta: &querypb.CallgraphNodeMeta{
			Location: &pb.Location{
				Id:        location.ID,
				MappingId: mappingID,
				Address:   location.Address,
				IsFolded:  location.IsFolded,
			},
			Mapping: location.Mapping,
		},
	}}
}

// linesToTreeNodes turns inlined `lines` into a stack of TreeNode items and
// returns the slice of items in order from outer-most to inner-most.
func linesToCallgraphNodes(
	location *profile.Location,
	mapping *pb.Mapping,
	lines []profile.LocationLine,
) []*querypb.CallgraphNode {
	if len(lines) == 0 {
		return nil
	}

	res := make([]*querypb.CallgraphNode, len(lines))

	// Same as locations, lines are in order from deepest to highest in the
	// stack. Therefore we start with the innermost, and work ourselves
	// outwards. We want the result to be from higest to deepest to be inserted
	// into our callgraph at our "current" position that's calling
	// linesToTreeNodes.
	for i := 0; i < len(lines); i++ {
		node := lineToGraphNode(
			location,
			mapping,
			lines[i],
		)
		res[len(lines)-1-i] = node
	}

	return res
}

func lineToGraphNode(
	location *profile.Location,
	mapping *pb.Mapping,
	line profile.LocationLine,
) *querypb.CallgraphNode {
	var mappingID string
	if mapping != nil {
		mappingID = mapping.Id
	}

	return &querypb.CallgraphNode{
		// Appending the line number to the location ID to make the node ID unique.
		Id: location.ID + "_" + strconv.FormatInt(line.Line, 10),
		Meta: &querypb.CallgraphNodeMeta{
			Location: &pb.Location{
				Id:        location.ID,
				MappingId: mappingID,
				Address:   location.Address,
				IsFolded:  location.IsFolded,
			},
			Function: line.Function,
			Line: &pb.Line{
				FunctionId: line.Function.Id,
				Line:       line.Line,
			},
			Mapping: mapping,
		},
	}
}

func prunableNodes(nodes []*querypb.CallgraphNode, c int64) []*querypb.CallgraphNode {
	if len(nodes) == 0 {
		return nodes
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Cumulative > nodes[j].Cumulative
	})
	i := 0
	cutoffValue := (float64(c) * NodeCutOffFraction)
	for ; i < len(nodes); i++ {
		if float64(nodes[i].Cumulative) < cutoffValue {
			break
		}
	}
	return nodes[i:]
}

func pruneGraph(graph *querypb.Callgraph) *querypb.Callgraph {
	prunableNodes := prunableNodes(graph.Nodes, graph.Cumulative)
	finalNodes := make([]*querypb.CallgraphNode, 0)
	finalEdges := make([]*querypb.CallgraphEdge, 0)
	edgesMap := make(map[string]*querypb.CallgraphEdge)
	incomingEdges := make(map[string][]*querypb.CallgraphEdge)
	outgoingEdges := make(map[string][]*querypb.CallgraphEdge)
	prunableNodesMap := make(map[string]bool)
	for _, edge := range graph.Edges {
		if incomingEdges[edge.Target] == nil {
			incomingEdges[edge.Target] = []*querypb.CallgraphEdge{}
		}
		incomingEdges[edge.Target] = append(incomingEdges[edge.Target], edge)

		if outgoingEdges[edge.Source] == nil {
			outgoingEdges[edge.Source] = []*querypb.CallgraphEdge{}
		}
		outgoingEdges[edge.Source] = append(outgoingEdges[edge.Source], edge)
		edgesMap[edge.Id] = edge
	}

	for _, node := range prunableNodes {
		prunableNodesMap[node.Id] = true
	}

	nodesToRemove := make(map[string]bool, 0)
	edgesToRemove := make(map[string]bool, 0)
	edgesToCreate := make([]*querypb.CallgraphEdge, 0)

	// Validate the eligibility of each prunableNode
	for _, node := range prunableNodes {
		if !prunableNodesMap[node.Id] {
			continue
		}
		if len(incomingEdges[node.Id]) > 1 {
			// Cannot prune nodes with multiple incoming edges.
			continue
		}
		if len(incomingEdges[node.Id]) == 0 || len(outgoingEdges[node.Id]) == 0 {
			// Cannot prune leaf nodes.
			continue
		}
		nodesToRemove[node.Id] = true
	}

	// Remove nodes and identify edges to patch
	for _, node := range graph.Nodes {
		if nodesToRemove[node.Id] {
			// patch the edges from its parent to child nodes
			parentNodeId, cummValue, incomingEdgesToRemove := findAValidParent(node, incomingEdges, outgoingEdges, nodesToRemove)
			for _, edge := range outgoingEdges[node.Id] {
				if nodesToRemove[edge.Target] {
					// Skipping the edge creation as this will be patched by the downstream node that is being removed.
					continue
				}
				newEdge := &querypb.CallgraphEdge{Id: uuid.New().String(), Source: parentNodeId, Target: edge.Target, Cumulative: cummValue, IsCollapsed: true}
				edgesToCreate = append(edgesToCreate, newEdge)
			}
			for _, outgoingEdge := range append(outgoingEdges[node.Id], incomingEdgesToRemove...) {
				edgesToRemove[outgoingEdge.Id] = true
			}

			continue
		} else {
			finalNodes = append(finalNodes, node)
		}
	}

	// Patch the edges and prepare the final list of edges
	for _, edge := range graph.Edges {
		if edgesToRemove[edge.Id] {
			continue
		}
		finalEdges = append(finalEdges, edge)
	}
	finalEdges = append(finalEdges, edgesToCreate...)

	return &querypb.Callgraph{Nodes: finalNodes, Edges: finalEdges, Cumulative: graph.Cumulative}
}

// Traverse the graph and find a valid parent node that is not marked to be deleted.
func findAValidParent(node *querypb.CallgraphNode, incomingEdges map[string][]*querypb.CallgraphEdge, outgoingEdges map[string][]*querypb.CallgraphEdge, nodesToRemove map[string]bool) (string, int64, []*querypb.CallgraphEdge) {
	parent := incomingEdges[node.Id][0].Source
	c := incomingEdges[node.Id][0].Cumulative
	edgesToRemove := []*querypb.CallgraphEdge{incomingEdges[node.Id][0]}
	for nodesToRemove[parent] {
		c += incomingEdges[parent][0].Cumulative
		edgesToRemove = append(edgesToRemove, incomingEdges[parent][0])
		parent = incomingEdges[parent][0].Source

	}
	return parent, c, edgesToRemove
}
