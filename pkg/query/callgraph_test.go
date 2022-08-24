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
	"strconv"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
)

func TestGenerateCallgraph(t *testing.T) {
	ctx := context.Background()

	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(fileContent))
	tracer := trace.NewNoopTracerProvider().Tracer("")

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		tracer,
	)
	metastore := metastore.NewInProcessClient(l)
	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]struct{}{}, p, false)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	res, err := GenerateCallgraph(ctx, symbolizedProfile)
	require.NoError(t, err)
	require.NotNil(t, res)

	require.Equal(t, int64(310797348), res.Cumulative, "Root cummulative value mismatch")

	/*
		Validate the result for this stacktrace:

			runtime/pprof.(*protobuf).varint
			runtime/pprof.(*protobuf).uint64
			runtime/pprof.(*protobuf).int64 (inline)
			runtime/pprof.(*protobuf).int64Opt (inline)
			runtime/pprof.(*profileBuilder).emitLocation
			runtime/pprof.(*profileBuilder).appendLocsForStack
			runtime/pprof.(*profileBuilder).build
			runtime/pprof.profileWriter

	*/

	visited := make(map[string]*pb.CallgraphNode, 0)
	requiredNodes := make([]*pb.CallgraphNode, 6)
	for _, node := range res.GetNodes() {
		name := node.Meta.Function.Name
		// Validate duplicate nodes
		if visited[name] != nil {
			require.Fail(t, "Duplicate node found:"+name, node.Id)
		} else {
			visited[name] = node
		}

		// find the required nodes
		if name == "runtime/pprof.profileWriter" {
			require.Equal(t, int64(2308419), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[0] = node
		}
		if name == "runtime/pprof.(*profileBuilder).build" {
			require.Equal(t, int64(2479889), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[1] = node
		}
		if name == "runtime/pprof.(*profileBuilder).appendLocsForStack" {
			require.Equal(t, int64(132520050), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[2] = node
		}
		if name == "runtime/pprof.(*profileBuilder).emitLocation" {
			require.Equal(t, int64(14095085), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[3] = node
		}
		if name == "runtime/pprof.(*protobuf).uint64" {
			require.Equal(t, int64(330616), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[4] = node
		}
		if name == "runtime/pprof.(*protobuf).varint" {
			require.Equal(t, int64(399569), node.Cumulative, "Node cummulative mismatch for "+name)
			requiredNodes[5] = node
		}
	}

	// Validate all the required nodes are there
	for i := 0; i < len(requiredNodes); i++ {
		require.NotNil(t, requiredNodes[i], "Required node not found, index: "+strconv.Itoa(i))
	}

	edges := res.GetEdges()

	// Validate all the required edges are there
	foundEdges := 0

	for _, edge := range edges {
		if edge.GetSource() == requiredNodes[0].GetId() && edge.GetTarget() == requiredNodes[1].GetId() {
			require.Equal(t, int64(1756541), edge.Cumulative, "Edge cumulative mismatch for 0 -> 1")
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[1].GetId() && edge.GetTarget() == requiredNodes[2].GetId() {
			require.Equal(t, int64(1553356), edge.Cumulative, "Edge cumulative mismatch for 1 -> 2")
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[2].GetId() && edge.GetTarget() == requiredNodes[3].GetId() {
			require.Equal(t, int64(13353318), edge.Cumulative, "Edge cumulative mismatch for 2 -> 3")
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[3].GetId() && edge.GetTarget() == requiredNodes[4].GetId() {
			require.Equal(t, int64(114140), edge.Cumulative, "Edge cumulative mismatch for 3 -> 4")
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[4].GetId() && edge.GetTarget() == requiredNodes[5].GetId() {
			require.Equal(t, int64(330616), edge.Cumulative, "Edge cumulative mismatch for 4 -> 5")
			foundEdges++
		}
	}

	require.Equal(t, 5, foundEdges)
}

func TestPruneCallgraph(t *testing.T) {
	/*
			  C - D - E
		     /          \
		A - B  - F - H - I - J
		          \
		           G
	*/
	graph := &pb.Callgraph{
		Nodes: []*pb.CallgraphNode{
			{Id: "A", Cumulative: int64(100)},
			{Id: "B", Cumulative: int64(200)},
			{Id: "C", Cumulative: int64(1)},
			{Id: "D", Cumulative: int64(1)},
			{Id: "E", Cumulative: int64(100)},
			{Id: "F", Cumulative: int64(100)},
			{Id: "G", Cumulative: int64(300)},
			{Id: "H", Cumulative: int64(100)},
			{Id: "I", Cumulative: int64(1)},
			{Id: "J", Cumulative: int64(100)},
		},
		Edges: []*pb.CallgraphEdge{
			{Id: "1", Source: "A", Target: "B", Cumulative: 1},
			{Id: "2", Source: "B", Target: "C", Cumulative: 1},
			{Id: "3", Source: "C", Target: "D", Cumulative: 1},
			{Id: "4", Source: "D", Target: "E", Cumulative: 1},
			{Id: "5", Source: "B", Target: "F", Cumulative: 1},
			{Id: "6", Source: "F", Target: "H", Cumulative: 1},
			{Id: "7", Source: "F", Target: "G", Cumulative: 1},
			{Id: "8", Source: "H", Target: "I", Cumulative: 1},
			{Id: "9", Source: "E", Target: "I", Cumulative: 1},
			{Id: "10", Source: "I", Target: "J", Cumulative: 1},
		},
		Cumulative: 1000,
	}
	prunedGraph := pruneGraph(graph)

	/* Validate the pruned graph:

		   - - E - -
	     /          \
	A - B  - F - H - I - J
	          \
	           G
	*/

	require.Equal(t, 8, len(prunedGraph.GetNodes()), "Number of nodes mismatch")
	require.Equal(t, 8, len(prunedGraph.GetEdges()), "Number of edges mismatch")
	for _, node := range prunedGraph.GetNodes() {
		require.False(t, node.GetId() == "C", "Node C is not pruned")
		require.False(t, node.GetId() == "D", "Node D is not pruned")
	}
	for _, edge := range prunedGraph.GetEdges() {
		require.False(t, edge.GetSource() == "C" && edge.GetTarget() == "D", "Edge C -> D is not pruned")
		require.False(t, edge.GetSource() == "B" && edge.GetTarget() == "C", "Edge B -> C is not pruned")
		if edge.GetSource() == "B" && edge.GetTarget() == "E" {
			require.True(t, edge.IsCollapsed, "Edge B -> F is not marked as collapsed")
		}
	}
}
