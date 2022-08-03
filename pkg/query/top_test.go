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
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateTopTable(t *testing.T) {
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
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]struct{}{}, p, false)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")
	symbolizedProfile, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	res, err := GenerateTopTable(ctx, symbolizedProfile)
	require.NoError(t, err)

	require.Equal(t, int32(1886), res.Total)
	require.Equal(t, int32(899), res.Reported)
	require.Len(t, res.List, 899)

	found := false
	for _, node := range res.GetList() {
		if node.GetMeta().GetMapping().GetFile() == "/bin/operator" &&
			node.GetMeta().GetFunction().GetName() == "encoding/json.Unmarshal" {
			require.Equal(t, int64(3135531), node.GetCumulative())
			// TODO(metalmatze): This isn't fully deterministic yet, thus some assertions are commented.
			// require.Equal(t, int64(32773), node.GetFlat())

			// require.Equal(t, uint64(7578336), node.GetMeta().GetLocation().GetAddress())
			require.Equal(t, false, node.GetMeta().GetLocation().GetIsFolded())
			require.Equal(t, uint64(4194304), node.GetMeta().GetMapping().GetStart())
			require.Equal(t, uint64(23252992), node.GetMeta().GetMapping().GetLimit())
			require.Equal(t, uint64(0), node.GetMeta().GetMapping().GetOffset())
			require.Equal(t, "/bin/operator", node.GetMeta().GetMapping().GetFile())
			require.Equal(t, "", node.GetMeta().GetMapping().GetBuildId())
			require.Equal(t, true, node.GetMeta().GetMapping().GetHasFunctions())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasFilenames())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasLineNumbers())
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasInlineFrames())

			require.Equal(t, int64(0), node.GetMeta().GetFunction().GetStartLine())
			require.Equal(t, "encoding/json.Unmarshal", node.GetMeta().GetFunction().GetName())
			require.Equal(t, "encoding/json.Unmarshal", node.GetMeta().GetFunction().GetSystemName())
			// require.Equal(t, int64(101), node.GetMeta().GetLine().GetLine())
			found = true
			break
		}
	}
	require.Truef(t, found, "expected to find the specific function")
}

func TestGenerateTopTableAggregateFlat(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	metastore := metastore.NewInProcessClient(metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	))

	lres, err := metastore.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			Address: 0x1,
		}, {
			Address: 0x2,
		}, {
			Address: 0x3,
		}, {
			Address: 0x4,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 4, len(lres.Locations))
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]
	l3 := lres.Locations[2]
	l4 := lres.Locations[3]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l1.Id, l2.Id},
		}, {
			LocationIds: []string{l1.Id, l3.Id},
		}, {
			LocationIds: []string{l1.Id, l4.Id},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(sres.Stacktraces))
	st1 := sres.Stacktraces[0]
	st2 := sres.Stacktraces[1]
	st3 := sres.Stacktraces[2]

	p, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, &profile.NormalizedProfile{
		Samples: []*profile.NormalizedSample{{
			StacktraceID: st1.Id,
			Value:        1,
		}, {
			StacktraceID: st2.Id,
			Value:        1,
		}, {
			StacktraceID: st3.Id,
			Value:        1,
		}},
	})
	require.NoError(t, err)

	top, err := GenerateTopTable(ctx, p)
	require.NoError(t, err)

	require.Equal(t, 4, len(top.List))
	require.Equal(t, uint64(0x1), top.List[0].Meta.Location.Address)
	require.Equal(t, int64(3), top.List[0].Cumulative)
	require.Equal(t, int64(3), top.List[0].Flat)
	require.Equal(t, uint64(0x2), top.List[1].Meta.Location.Address)
	require.Equal(t, int64(1), top.List[1].Cumulative)
	require.Equal(t, int64(0), top.List[1].Flat)
	require.Equal(t, uint64(0x3), top.List[2].Meta.Location.Address)
	require.Equal(t, int64(1), top.List[2].Cumulative)
	require.Equal(t, int64(0), top.List[2].Flat)
	require.Equal(t, uint64(0x4), top.List[3].Meta.Location.Address)
	require.Equal(t, int64(1), top.List[3].Cumulative)
	require.Equal(t, int64(0), top.List[3].Flat)
}

func TestGenerateDiffTopTable(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	p1 := &pprofpb.Profile{}
	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	require.NoError(t, p1.UnmarshalVT(fileContent))

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)
	metastore := metastore.NewInProcessClient(l)
	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]struct{}{}, p1, false)
	require.NoError(t, err)

	p2 := profiles[0]

	// The highest unique sample value is 31024846
	// which we use for testing with a unique sample
	const testValue = 31024846

	found := false
	for _, sample := range p2.Samples {
		if sample.Value == testValue {
			sample.DiffValue = -testValue
			found = true
		}
	}
	require.Truef(t, found, "expected to find the specific sample")

	tracer := trace.NewNoopTracerProvider().Tracer("")
	p, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	res, err := GenerateTopTable(ctx, p)
	require.NoError(t, err)

	found = false
	for _, node := range res.List {
		if node.Diff == -testValue {
			found = true
		}
	}
	require.Truef(t, found, "Expected to find our test diff value in top nodes")
}

func TestAggregateTopByFunction(t *testing.T) {
	t.Parallel()

	id1 := "1"
	id2 := "2"
	id3 := "3"

	testcases := []struct {
		name   string
		input  *pb.Top
		output *pb.Top
	}{{
		name: "Empty",
		input: &pb.Top{
			Total: 0,
			List:  []*pb.TopNode{},
		},
		output: &pb.Top{
			Total:    0,
			Reported: 0,
			List:     []*pb.TopNode{},
		},
	}, {
		name: "NoMeta",
		input: &pb.Top{
			Total: 2,
			List: []*pb.TopNode{
				{Meta: nil, Cumulative: 1, Flat: 1},
				{Meta: nil, Cumulative: 2, Flat: 2},
			},
		},
		output: &pb.Top{
			Total:    2,
			Reported: 0,
			List:     []*pb.TopNode{},
		},
	}, {
		name: "UniqueAddress",
		input: &pb.Top{
			Total: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id3, Address: 3},
					},
					Cumulative: 1,
					Flat:       1,
				},
			},
		},
		output: &pb.Top{
			Total:    2,
			Reported: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id3, Address: 3},
					},
					Cumulative: 1,
					Flat:       1,
				},
			},
		},
	}, {
		name: "UniqueFunction",
		input: &pb.Top{
			Total: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
						Function: &metastorev1alpha1.Function{Id: id2, Name: "func2"},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id3, Address: 3},
						Function: &metastorev1alpha1.Function{Id: id3, Name: "func3"},
					},
					Cumulative: 1,
					Flat:       1,
				},
			},
		},
		output: &pb.Top{
			Total:    2,
			Reported: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
						Function: &metastorev1alpha1.Function{Id: id2, Name: "func2"},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id3, Address: 3},
						Function: &metastorev1alpha1.Function{Id: id3, Name: "func3"},
					},
					Cumulative: 1,
					Flat:       1,
				},
			},
		},
	}, {
		name: "AggregateAddress",
		input: &pb.Top{
			Total: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
					},
					Cumulative: 1,
					Flat:       1,
				},
			},
		},
		output: &pb.Top{
			Total:    2,
			Reported: 1,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
					},
					Cumulative: 2,
					Flat:       2,
				},
			},
		},
	}, {
		name: "AggregateFunction",
		input: &pb.Top{
			Total: 2,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
						Function: &metastorev1alpha1.Function{Id: id2, Name: "func2"},
					},
					Cumulative: 1,
					Flat:       1,
				},
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
						Function: &metastorev1alpha1.Function{Id: id2, Name: "func2"},
					},
					Cumulative: 2,
					Flat:       2,
				},
			},
		},
		output: &pb.Top{
			Total:    2,
			Reported: 1,
			List: []*pb.TopNode{
				{
					Meta: &pb.TopNodeMeta{
						Mapping:  &metastorev1alpha1.Mapping{Id: id1},
						Location: &metastorev1alpha1.Location{Id: id2, Address: 2},
						Function: &metastorev1alpha1.Function{Id: id2, Name: "func2"},
					},
					Cumulative: 3,
					Flat:       3,
				},
			},
		},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.output, aggregateTopByFunction(tc.input))
		})
	}
}
