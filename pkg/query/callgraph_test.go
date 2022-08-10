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
			requiredNodes[0] = node
		}
		if name == "runtime/pprof.(*profileBuilder).build" {
			requiredNodes[1] = node
		}
		if name == "runtime/pprof.(*profileBuilder).appendLocsForStack" {
			requiredNodes[2] = node
		}
		if name == "runtime/pprof.(*profileBuilder).emitLocation" {
			requiredNodes[3] = node
		}
		if name == "runtime/pprof.(*protobuf).uint64" {
			requiredNodes[4] = node
		}
		if name == "runtime/pprof.(*protobuf).varint" {
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
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[1].GetId() && edge.GetTarget() == requiredNodes[2].GetId() {
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[2].GetId() && edge.GetTarget() == requiredNodes[3].GetId() {
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[3].GetId() && edge.GetTarget() == requiredNodes[4].GetId() {
			foundEdges++
		}
		if edge.GetSource() == requiredNodes[4].GetId() && edge.GetTarget() == requiredNodes[5].GetId() {
			foundEdges++
		}
	}

	require.Equal(t, 5, foundEdges)
}
