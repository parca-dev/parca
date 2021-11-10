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
	"context"
	"testing"

	"github.com/google/uuid"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

var (
	id1  = "00000000-0000-0000-0000-000000000001"
	id2  = "00000000-0000-0000-0000-000000000002"
	id3  = "00000000-0000-0000-0000-000000000003"
	id4  = "00000000-0000-0000-0000-000000000004"
	id5  = "00000000-0000-0000-0000-000000000005"
	ida1 = "00000000-0000-0000-0000-0000000000a1"
	idf1 = "00000000-0000-0000-0000-0000000000f1"
	idf2 = "00000000-0000-0000-0000-0000000000f2"
	idf3 = "00000000-0000-0000-0000-0000000000f3"
	idf4 = "00000000-0000-0000-0000-0000000000f4"
	idf5 = "00000000-0000-0000-0000-0000000000f5"

	uuid1  = uuid.MustParse(id1)
	uuid2  = uuid.MustParse(id2)
	uuid3  = uuid.MustParse(id3)
	uuid4  = uuid.MustParse(id4)
	uuid5  = uuid.MustParse(id5)
	uuida1 = uuid.MustParse(ida1)
	uuidf1 = uuid.MustParse(idf1)
	uuidf2 = uuid.MustParse(idf2)
	uuidf3 = uuid.MustParse(idf3)
	uuidf4 = uuid.MustParse(idf4)
	uuidf5 = uuid.MustParse(idf5)
)

func TestGenerateFlamegraphFlat(t *testing.T) {
	mapping := &metastore.Mapping{ID: uuida1}
	locations := &fakeLocations{m: map[uuid.UUID]*metastore.Location{
		uuid1: {
			ID:      uuid1,
			Mapping: mapping,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf1, FunctionKey: metastore.FunctionKey{Name: "1"}},
			}},
		},
		uuid2: {
			ID:      uuid2,
			Mapping: mapping,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf2, FunctionKey: metastore.FunctionKey{Name: "2"}},
			}},
		},
		uuid3: {
			ID:      uuid3,
			Mapping: mapping,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf3, FunctionKey: metastore.FunctionKey{Name: "3"}},
			}},
		},
		uuid4: {
			ID:      uuid4,
			Mapping: mapping,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf4, FunctionKey: metastore.FunctionKey{Name: "4"}},
			}},
		},
		uuid5: {
			ID:      uuid5,
			Mapping: mapping,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf5, FunctionKey: metastore.FunctionKey{Name: "5"}},
			}},
		},
	}}

	s0 := makeSample(2, []uuid.UUID{uuid2, uuid1})
	s1 := makeSample(1, []uuid.UUID{uuid5, uuid3, uuid2, uuid1})
	s2 := makeSample(3, []uuid.UUID{uuid4, uuid3, uuid2, uuid1})

	fp := &FlatProfile{
		Meta:    InstantProfileMeta{},
		samples: []*Sample{s0, s1, s2},
	}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	fg, err := GenerateFlamegraphFlat(ctx, tracer, locations, fp)
	require.NoError(t, err)

	require.Equal(t, &pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{Id: idf1, Name: "1"},
				Line:     &pb.Line{LocationId: id1, FunctionId: idf1},
				Location: &pb.Location{Id: id1, MappingId: ida1},
				Mapping:  &pb.Mapping{Id: ida1},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{Id: idf2, Name: "2"},
					Line:     &pb.Line{LocationId: id2, FunctionId: idf2},
					Location: &pb.Location{Id: id2, MappingId: ida1},
					Mapping:  &pb.Mapping{Id: ida1},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{Id: idf3, Name: "3"},
						Line:     &pb.Line{LocationId: id3, FunctionId: idf3},
						Location: &pb.Location{Id: id3, MappingId: ida1},
						Mapping:  &pb.Mapping{Id: ida1},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{Id: idf4, Name: "4"},
							Line:     &pb.Line{LocationId: id4, FunctionId: idf4},
							Location: &pb.Location{Id: id4, MappingId: ida1},
							Mapping:  &pb.Mapping{Id: ida1},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{Id: idf5, Name: "5"},
							Line:     &pb.Line{LocationId: id5, FunctionId: idf5},
							Location: &pb.Location{Id: id5, MappingId: ida1},
							Mapping:  &pb.Mapping{Id: ida1},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}}, fg)
}

func TestGenerateInlinedFunctionFlamegraphFlat(t *testing.T) {
	m := &metastore.Mapping{ID: uuida1}
	l := &fakeLocations{m: map[uuid.UUID]*metastore.Location{
		uuid1: {
			ID:      uuid1,
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf1, FunctionKey: metastore.FunctionKey{Name: "1"}},
			}},
		},
		uuid2: {
			ID:      uuid2,
			Mapping: m,
			Lines: []metastore.LocationLine{{
				Function: &metastore.Function{ID: uuidf3, FunctionKey: metastore.FunctionKey{Name: "3"}},
			}, {
				Function: &metastore.Function{ID: uuidf2, FunctionKey: metastore.FunctionKey{Name: "2"}},
			}},
		},
	}}

	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	fp := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: []*Sample{
			makeSample(2, []uuid.UUID{uuid2, uuid1}),
		},
	}

	fg, err := GenerateFlamegraphFlat(ctx, tracer, l, fp)
	require.NoError(t, err)
	require.Equal(t, &pb.Flamegraph{Height: 3, Total: 2, Root: &pb.FlamegraphRootNode{
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 2,
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{Id: idf1, Name: "1"},
				Line:     &pb.Line{LocationId: id1, FunctionId: idf1},
				Location: &pb.Location{Id: id1, MappingId: ida1},
				Mapping:  &pb.Mapping{Id: ida1},
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 2,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{Id: idf2, Name: "2"},
					Line:     &pb.Line{LocationId: id2, FunctionId: idf2},
					Location: &pb.Location{Id: id2, MappingId: ida1},
					Mapping:  &pb.Mapping{Id: ida1},
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 2,
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{Id: idf3, Name: "3"},
						Line:     &pb.Line{LocationId: id2, FunctionId: idf3},
						Location: &pb.Location{Id: id2, MappingId: ida1},
						Mapping:  &pb.Mapping{Id: ida1},
					},
				}},
			}},
		}},
	}}, fg)
}
