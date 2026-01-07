// Copyright 2022-2026 The Parca Authors
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

	pprofprofile "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateTopTable(t *testing.T) {
	ctx := context.Background()

	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		pp,
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	res, cummulative, err := GenerateTopTable(ctx, op)
	require.NoError(t, err)

	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	require.Equal(t, int32(310797348), res.Total)
	require.Equal(t, int32(899), res.Reported)
	require.Equal(t, int64(310797348), cummulative)
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
			require.Equal(t, false, node.GetMeta().GetMapping().GetHasFunctions())
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

	locations := []*pprofprofile.Location{{
		Address: 0x1,
	}, {
		Address: 0x2,
	}, {
		Address: 0x3,
	}, {
		Address: 0x4,
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			Location: locations,
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[0], locations[1]},
				Value:    []int64{1},
			}, {
				Location: []*pprofprofile.Location{locations[0], locations[2]},
				Value:    []int64{1},
			}, {
				Location: []*pprofprofile.Location{locations[0], locations[3]},
				Value:    []int64{1},
			}},
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	op, err := parcacol.NewArrowToProfileConverter(nil, kv.NewKeyMaker()).Convert(ctx, p)
	require.NoError(t, err)

	top, _, err := GenerateTopTable(ctx, op)
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
