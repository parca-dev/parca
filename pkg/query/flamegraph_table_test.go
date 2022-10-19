package query

import (
	"context"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlamegraphTable(t *testing.T) {
	ctx := context.Background()
	var err error

	l := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)

	metastore := metastore.NewInProcessClient(l)

	mres, err := metastore.GetOrCreateMappings(ctx, &metastorepb.GetOrCreateMappingsRequest{
		Mappings: []*metastorepb.Mapping{{
			File: "a",
		}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := metastore.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{{
			Name: "1",
		}, {
			Name: "2",
		}, {
			Name: "3",
		}, {
			Name: "4",
		}, {
			Name: "5",
		}},
	})
	require.NoError(t, err)
	f1 := fres.Functions[0]
	f2 := fres.Functions[1]
	f3 := fres.Functions[2]
	f4 := fres.Functions[3]
	f5 := fres.Functions[4]

	lres, err := metastore.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f1.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f2.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f3.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f4.Id,
			}},
		}, {
			MappingId: m.Id,
			Lines: []*metastorepb.Line{{
				FunctionId: f5.Id,
			}},
		}},
	})
	require.NoError(t, err)
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]
	l3 := lres.Locations[2]
	l4 := lres.Locations[3]
	l5 := lres.Locations[4]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}, {
			LocationIds: []string{l5.Id, l3.Id, l2.Id, l1.Id},
		}, {
			LocationIds: []string{l4.Id, l3.Id, l2.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	s1 := sres.Stacktraces[0]
	s2 := sres.Stacktraces[1]
	s3 := sres.Stacktraces[2]

	tracer := trace.NewNoopTracerProvider().Tracer("")

	p, err := parcacol.NewArrowToProfileConverter(tracer, metastore).SymbolizeNormalizedProfile(ctx, &parcaprofile.NormalizedProfile{
		Samples: []*parcaprofile.NormalizedSample{{
			StacktraceID: s1.Id,
			Value:        2,
		}, {
			StacktraceID: s2.Id,
			Value:        1,
		}, {
			StacktraceID: s3.Id,
			Value:        3,
		}},
	})
	require.NoError(t, err)

	fg, err := GenerateFlamegraphTable(ctx, tracer, p)
	require.NoError(t, err)

	require.Equal(t, int32(5), fg.Height)
	require.Equal(t, int64(6), fg.Total)

	// Check if tables and thus deduplication was correct and deterministic

	require.Equal(t, []string{"a", "", "1", "2", "3", "5", "4"}, fg.StringTable)
	require.Equal(t, []*metastorepb.Location{
		{MappingIndex: 0, Lines: []*metastorepb.Line{{FunctionIndex: 0}}},
		{MappingIndex: 0, Lines: []*metastorepb.Line{{FunctionIndex: 1}}},
		{MappingIndex: 0, Lines: []*metastorepb.Line{{FunctionIndex: 2}}},
		{MappingIndex: 0, Lines: []*metastorepb.Line{{FunctionIndex: 3}}},
		{MappingIndex: 0, Lines: []*metastorepb.Line{{FunctionIndex: 4}}},
	}, fg.Locations)
	require.Equal(t, []*metastorepb.Mapping{
		{BuildIdStringIndex: 1, FileStringIndex: 0},
	}, fg.Mapping)
	require.Equal(t, []*metastorepb.Function{
		{NameStringIndex: 2, SystemNameStringIndex: 1, FilenameStringIndex: 1},
		{NameStringIndex: 3, SystemNameStringIndex: 1, FilenameStringIndex: 1},
		{NameStringIndex: 4, SystemNameStringIndex: 1, FilenameStringIndex: 1},
		{NameStringIndex: 5, SystemNameStringIndex: 1, FilenameStringIndex: 1},
		{NameStringIndex: 6, SystemNameStringIndex: 1, FilenameStringIndex: 1},
	}, fg.Function)

	// Check the recursive flamegraph that references the tables above.

	expected := &pb.FlamegraphRootNode{
		Cumulative: 6,
		Children: []*pb.FlamegraphNode{{
			Cumulative: 6,
			Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 0},
			Children: []*pb.FlamegraphNode{{
				Cumulative: 6,
				Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 1},
				Children: []*pb.FlamegraphNode{{
					Cumulative: 4,
					Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 2},
					Children: []*pb.FlamegraphNode{{
						Cumulative: 1,
						Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 3},
					}, {
						Cumulative: 3,
						Meta:       &pb.FlamegraphNodeMeta{LocationIndex: 4},
					}},
				}},
			}},
		}},
	}
	require.True(t, proto.Equal(expected, fg.Root))
}
