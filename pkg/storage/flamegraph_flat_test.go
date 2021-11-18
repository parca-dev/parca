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
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
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
	l := metastore.NewBadgerMetastore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		NewLinearUUIDGenerator(),
	)

	m := &metastore.Mapping{
		File: "a",
	}
	m.ID, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "1",
		},
	}
	f1.ID, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "2",
		},
	}
	f2.ID, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "3",
		},
	}
	f3.ID, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	f4 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "4",
		},
	}
	f4.ID, err = l.CreateFunction(ctx, f4)
	require.NoError(t, err)

	f5 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "5",
		},
	}
	f5.ID, err = l.CreateFunction(ctx, f5)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1.ID, err = l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f2,
			},
		},
	}
	l2.ID, err = l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	l3 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f3,
			},
		},
	}
	l3.ID, err = l.CreateLocation(ctx, l3)
	require.NoError(t, err)

	l4 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f4,
			},
		},
	}
	l4.ID, err = l.CreateLocation(ctx, l4)
	require.NoError(t, err)

	l5 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f5,
			},
		},
	}
	l5.ID, err = l.CreateLocation(ctx, l5)
	require.NoError(t, err)

	s0 := makeSample(2, []uuid.UUID{l2.ID, l1.ID})
	s1 := makeSample(1, []uuid.UUID{l5.ID, l3.ID, l2.ID, l1.ID})
	s2 := makeSample(3, []uuid.UUID{l4.ID, l3.ID, l2.ID, l1.ID})

	k0 := makeStacktraceKey(s0)
	k1 := makeStacktraceKey(s1)
	k2 := makeStacktraceKey(s2)

	fp := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: map[string]*Sample{
			string(k0): s0,
			string(k1): s1,
			string(k2): s2,
		},
	}

	tracer := trace.NewNoopTracerProvider().Tracer("")

	fg, err := GenerateFlamegraphFlat(ctx, tracer, l, fp)
	require.NoError(t, err)

	require.Equal(t, &pb.Flamegraph{Height: 5, Total: 6, Root: &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{Id: f1.ID.String(), Name: "1"},
				Line:     &pb.Line{LocationId: l1.ID.String(), FunctionId: f1.ID.String()},
				Location: &pb.Location{Id: l1.ID.String(), MappingId: m.ID.String()},
				Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
			},
			Cumulative: 6,
			Children: []*pb.FlamegraphNode{{
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{Id: f2.ID.String(), Name: "2"},
					Line:     &pb.Line{LocationId: l2.ID.String(), FunctionId: f2.ID.String()},
					Location: &pb.Location{Id: l2.ID.String(), MappingId: m.ID.String()},
					Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
				},
				Cumulative: 6,
				Children: []*pb.FlamegraphNode{{
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{Id: f3.ID.String(), Name: "3"},
						Line:     &pb.Line{LocationId: l3.ID.String(), FunctionId: f3.ID.String()},
						Location: &pb.Location{Id: l3.ID.String(), MappingId: m.ID.String()},
						Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
					},
					Cumulative: 4,
					Children: []*pb.FlamegraphNode{{
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{Id: f4.ID.String(), Name: "4"},
							Line:     &pb.Line{LocationId: l4.ID.String(), FunctionId: f4.ID.String()},
							Location: &pb.Location{Id: l4.ID.String(), MappingId: m.ID.String()},
							Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
						},
						Cumulative: 3,
					}, {
						Meta: &pb.FlamegraphNodeMeta{
							Function: &pb.Function{Id: f5.ID.String(), Name: "5"},
							Line:     &pb.Line{LocationId: l5.ID.String(), FunctionId: f5.ID.String()},
							Location: &pb.Location{Id: l5.ID.String(), MappingId: m.ID.String()},
							Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
						},
						Cumulative: 1,
					}},
				}},
			}},
		}},
	}}, fg)
}

func TestGenerateInlinedFunctionFlamegraphFlat(t *testing.T) {
	ctx := context.Background()
	var err error
	l := metastore.NewBadgerMetastore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)

	m := &metastore.Mapping{
		File: "a",
	}
	m.ID, err = l.CreateMapping(ctx, m)
	require.NoError(t, err)

	f1 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "1",
		},
	}
	f1.ID, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "2",
		},
	}
	f2.ID, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	f3 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "3",
		},
	}
	f3.ID, err = l.CreateFunction(ctx, f3)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Mapping: m,
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1.ID, err = l.CreateLocation(ctx, l1)
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
	l2.ID, err = l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")

	s0 := makeSample(2, []uuid.UUID{l2.ID, l1.ID})
	k0 := makeStacktraceKey(s0)

	fp := &FlatProfile{
		Meta: InstantProfileMeta{},
		samples: map[string]*Sample{
			string(k0): s0,
		},
	}

	fg, err := GenerateFlamegraphFlat(ctx, tracer, l, fp)
	require.NoError(t, err)
	require.Equal(t, &pb.Flamegraph{Height: 3, Total: 2, Root: &pb.FlamegraphRootNode{
		Cumulative: 2,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 2,
			Meta: &pb.FlamegraphNodeMeta{
				Function: &pb.Function{Id: f1.ID.String(), Name: "1"},
				Line:     &pb.Line{LocationId: l1.ID.String(), FunctionId: f1.ID.String()},
				Location: &pb.Location{Id: l1.ID.String(), MappingId: m.ID.String()},
				Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
			},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 2,
				Meta: &pb.FlamegraphNodeMeta{
					Function: &pb.Function{Id: f2.ID.String(), Name: "2"},
					Line:     &pb.Line{LocationId: l2.ID.String(), FunctionId: f2.ID.String()},
					Location: &pb.Location{Id: l2.ID.String(), MappingId: m.ID.String()},
					Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
				},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 2,
					Meta: &pb.FlamegraphNodeMeta{
						Function: &pb.Function{Id: f3.ID.String(), Name: "3"},
						Line:     &pb.Line{LocationId: l2.ID.String(), FunctionId: f3.ID.String()},
						Location: &pb.Location{Id: l2.ID.String(), MappingId: m.ID.String()},
						Mapping:  &pb.Mapping{Id: m.ID.String(), File: "a"},
					},
				}},
			}},
		}},
	}}, fg)
}

func TestGenerateFlamegraphFromFlatProfile(t *testing.T) {
	tracer := trace.NewNoopTracerProvider().Tracer("")
	reg := prometheus.NewRegistry()

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		reg,
		tracer,
		"generateflamegraphfrominstantprofile",
	)
	require.NoError(t, err)
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
