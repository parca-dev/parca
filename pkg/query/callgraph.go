package query

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
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
	return &querypb.Callgraph{Nodes: nodes, Edges: edges, Cumulative: cummValue}, nil
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
