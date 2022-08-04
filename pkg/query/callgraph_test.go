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
	"fmt"
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

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)
	metastore := metastore.NewInProcessClient(l)
	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", p, false)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")
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

	visited := make(map[string]bool, 0)
	requiredNodes := make([]*pb.CallgraphNode, 8)
	for _, node := range res.GetNodes() {

		// Validate duplicate nodes
		if visited[node.GetId()] == true {
			fmt.Printf("Duplicate: %s\n", node.GetId())
			require.Fail(t, "Duplicate node found:"+node.GetName())
		} else {
			visited[node.GetId()] = true
		}

		// find the required nodes
		if node.GetName() == "runtime/pprof.profileWriter" {
			requiredNodes[0] = node
		}
		if node.GetName() == "runtime/pprof.(*profileBuilder).build" {
			requiredNodes[1] = node
		}
		if node.GetName() == "runtime/pprof.(*profileBuilder).appendLocsForStack" {
			requiredNodes[2] = node
		}
		if node.GetName() == "runtime/pprof.(*profileBuilder).emitLocation" {
			requiredNodes[3] = node
		}
		if node.GetName() == "runtime/pprof.(*protobuf).int64Opt" {
			requiredNodes[4] = node
		}
		if node.GetName() == "runtime/pprof.(*protobuf).int64" {
			requiredNodes[5] = node
		}
		if node.GetName() == "runtime/pprof.(*protobuf).uint64" {
			requiredNodes[6] = node
		}
		if node.GetName() == "runtime/pprof.(*protobuf).varint" {
			requiredNodes[7] = node
		}
	}

	// Validate all the required nodes are there
	for i := 0; i < len(requiredNodes); i++ {
		require.NotNil(t, requiredNodes[i])
	}

	edges := res.GetEdges()

	// Validate all the required edges are there
	for i := 0; i < len(requiredNodes)-1; i++ {
		found := false

		for _, edge := range edges {
			if edge.GetSource() == requiredNodes[i].GetId() && edge.GetTarget() == requiredNodes[i+1].GetId() {
				found = true
				break
			}
		}
		require.True(t, found)
	}
}
