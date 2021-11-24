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
	"encoding/binary"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	metapb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
)

// Some tests need UUID generation to be predictable, so this generator just
// returns monotonically increasing UUIDs as if the UUID was a 16 byte integer.
type LinearUUIDGenerator struct {
	i uint64
}

// NewLinearUUIDGenerator returns a new LinearUUIDGenerator.
func NewLinearUUIDGenerator() metastore.UUIDGenerator {
	return &LinearUUIDGenerator{}
}

// New returns the next UUID according to the current count.
func (g *LinearUUIDGenerator) New() uuid.UUID {
	g.i++
	buf := make([]byte, 16)
	binary.BigEndian.PutUint64(buf[8:], g.i)
	id, err := uuid.FromBytes(buf)
	if err != nil {
		panic(err)
	}

	return id
}

func TestGenerateFlamegraphFlat(t *testing.T) {
	ctx := context.Background()
	var err error

	// We need UUID generation to be linear for this test to work as UUID are
	// sorted in the Flamegraph result, so predictable UUIDs are necessary for
	// a stable result.
	uuidGenerator := NewLinearUUIDGenerator()

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		uuidGenerator,
	)

	m := &metapb.Mapping{
		File: "a",
	}
	m.Id, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metapb.Function{
		Name: "1",
	}
	f1.Id, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metapb.Function{
		Name: "2",
	}
	f2.Id, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metapb.Function{
		Name: "3",
	}
	f3.Id, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	f4 := &metapb.Function{
		Name: "4",
	}
	f4.Id, err = l.CreateFunction(ctx, f4)
	require.NoError(t, err)

	f5 := &metapb.Function{
		Name: "5",
	}
	f5.Id, err = l.CreateFunction(ctx, f5)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1ID, err := l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l1.ID, err = uuid.FromBytes(l1ID)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f2,
			},
		},
	}
	l2ID, err := l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	l2.ID, err = uuid.FromBytes(l2ID)
	require.NoError(t, err)

	l3 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f3,
			},
		},
	}
	l3ID, err := l.CreateLocation(ctx, l3)
	require.NoError(t, err)

	l3.ID, err = uuid.FromBytes(l3ID)
	require.NoError(t, err)

	l4 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f4,
			},
		},
	}
	l4ID, err := l.CreateLocation(ctx, l4)
	require.NoError(t, err)

	l4.ID, err = uuid.FromBytes(l4ID)
	require.NoError(t, err)

	l5 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f5,
			},
		},
	}
	l5ID, err := l.CreateLocation(ctx, l5)
	require.NoError(t, err)

	l5.ID, err = uuid.FromBytes(l5ID)
	require.NoError(t, err)

	s0 := makeSample(2, []uuid.UUID{l2.ID, l1.ID})
	s1 := makeSample(1, []uuid.UUID{l5.ID, l3.ID, l2.ID, l1.ID})
	s2 := makeSample(3, []uuid.UUID{l4.ID, l3.ID, l2.ID, l1.ID})

	k0 := uuidGenerator.New()
	k1 := uuidGenerator.New()
	k2 := uuidGenerator.New()

	fp := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: map[[16]byte]*Sample{
			k0: s0,
			k1: s1,
			k2: s2,
		},
	}

	tracer := trace.NewNoopTracerProvider().Tracer("")

	fg, err := GenerateFlamegraphFlat(ctx, tracer, l, fp)
	require.NoError(t, err)

	require.True(t, proto.Equal(&pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{Id: f1.Id, Name: "1"},
				Line:     &metapb.Line{FunctionId: f1.Id},
				Location: &metapb.Location{Id: l1.ID[:], MappingId: m.Id},
				Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{Id: f2.Id, Name: "2"},
					Line:     &metapb.Line{FunctionId: f2.Id},
					Location: &metapb.Location{Id: l2.ID[:], MappingId: m.Id},
					Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{Id: f3.Id, Name: "3"},
						Line:     &metapb.Line{FunctionId: f3.Id},
						Location: &metapb.Location{Id: l3.ID[:], MappingId: m.Id},
						Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{Id: f4.Id, Name: "4"},
							Line:     &metapb.Line{FunctionId: f4.Id},
							Location: &metapb.Location{Id: l4.ID[:], MappingId: m.Id},
							Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &metapb.Function{Id: f5.Id, Name: "5"},
							Line:     &metapb.Line{FunctionId: f5.Id},
							Location: &metapb.Location{Id: l5.ID[:], MappingId: m.Id},
							Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}}, fg))
}

func TestGenerateInlinedFunctionFlamegraphFlat(t *testing.T) {
	ctx := context.Background()
	var err error
	uuidGenerator := metastore.NewRandomUUIDGenerator()
	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		uuidGenerator,
	)

	m := &metapb.Mapping{
		File: "a",
	}
	m.Id, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metapb.Function{
		Name: "1",
	}
	f1.Id, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metapb.Function{
		Name: "2",
	}
	f2.Id, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metapb.Function{
		Name: "3",
	}
	f3.Id, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1ID, err := l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l1.ID, err = uuid.FromBytes(l1ID)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f3,
			},
			{
				Function: f2,
			},
		},
	}
	l2ID, err := l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	l2.ID, err = uuid.FromBytes(l2ID)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")

	s0 := makeSample(2, []uuid.UUID{l2.ID, l1.ID})
	k0 := uuidGenerator.New()

	fp := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: map[[16]byte]*Sample{
			k0: s0,
		},
	}

	fg, err := GenerateFlamegraphFlat(ctx, tracer, l, fp)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.Flamegraph{Height: 3, Total: 2, Root: &pb.FlamegraphRootNode{
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 2,
			Meta: &pb.FlamegraphNodeMeta{
				Function: &metapb.Function{Id: f1.Id, Name: "1"},
				Line:     &metapb.Line{FunctionId: f1.Id},
				Location: &metapb.Location{Id: l1.ID[:], MappingId: m.Id},
				Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 2,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &metapb.Function{Id: f2.Id, Name: "2"},
					Line:     &metapb.Line{FunctionId: f2.Id},
					Location: &metapb.Location{Id: l2.ID[:], MappingId: m.Id},
					Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 2,
					Meta: &pb.FlamegraphNodeMeta{
						Function: &metapb.Function{Id: f3.Id, Name: "3"},
						Line:     &metapb.Line{FunctionId: f3.Id},
						Location: &metapb.Location{Id: l2.ID[:], MappingId: m.Id},
						Mapping:  &metapb.Mapping{Id: m.Id, File: "a"},
					},
				}},
			}},
		}},
	}}, fg))
}

func TestGenerateFlamegraphFromFlatProfile(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		l.Close()
	})

	testGenerateFlamegraphFromFlatProfile(t, l)
}

func testGenerateFlamegraphFromFlatProfile(t *testing.T, l metastore.ProfileMetaStore) *pb.Flamegraph {
	ctx := context.Background()

	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	flatProfile, err := FlatProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	fg, err := GenerateFlamegraphFlat(ctx, trace.NewNoopTracerProvider().Tracer(""), l, flatProfile)
	require.NoError(t, err)

	return fg
}
