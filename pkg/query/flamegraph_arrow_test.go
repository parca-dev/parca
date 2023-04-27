package query

import (
	"context"
	"testing"

	"github.com/apache/arrow/go/v10/arrow/array"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlamegraphArrow(t *testing.T) {
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
		Mappings: []*metastorepb.Mapping{{Start: 1, Limit: 1, Offset: 0x1234, File: "a", BuildId: "aID"}},
	})
	require.NoError(t, err)
	m := mres.Mappings[0]

	fres, err := metastore.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{
			{Name: "1", SystemName: "1", Filename: "1", StartLine: 1},
			{Name: "2", SystemName: "2", Filename: "2", StartLine: 2},
			{Name: "3", SystemName: "3", Filename: "3", StartLine: 3},
			{Name: "4", SystemName: "4", Filename: "4", StartLine: 4},
			{Name: "5", SystemName: "5", Filename: "5", StartLine: 5},
		},
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
			Address:   0xa1,
			Lines: []*metastorepb.Line{{
				FunctionId: f1.Id,
				Line:       1,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa2,
			Lines: []*metastorepb.Line{{
				FunctionId: f2.Id,
				Line:       2,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa3,
			Lines: []*metastorepb.Line{{
				FunctionId: f3.Id,
				Line:       3,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa4,
			Lines: []*metastorepb.Line{{
				FunctionId: f4.Id,
				Line:       4,
			}},
		}, {
			MappingId: m.Id,
			Address:   0xa5,
			Lines: []*metastorepb.Line{{
				FunctionId: f5.Id,
				Line:       5,
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

	fa, err := GenerateFlamegraphArrow(ctx, tracer, p, 0)
	require.NoError(t, err)

	require.Equal(t, int64(10), fa.NumRows())
	require.Equal(t, int64(15), fa.NumCols())

	require.Equal(t,
		[]uint64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingStart)[0]).(*array.Uint64).Uint64Values(),
	)
	require.Equal(t,
		[]uint64{1, 1, 1, 1, 1, 1, 1, 1, 1, 1},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingLimit)[0]).(*array.Uint64).Uint64Values(),
	)
	require.Equal(t,
		[]uint64{0x1234, 0x1234, 0x1234, 0x1234, 0x1234, 0x1234, 0x1234, 0x1234, 0x1234, 0x1234},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingOffset)[0]).(*array.Uint64).Uint64Values(),
	)

	mappingFilesDict := fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingFile)[0]).(*array.Dictionary)
	mappingFilesString := mappingFilesDict.Dictionary().(*array.String)
	mappingFiles := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		mappingFiles[i] = mappingFilesString.Value(mappingFilesDict.GetValueIndex(i))
	}
	require.Equal(t, []string{"a", "a", "a", "a", "a", "a", "a", "a", "a", "a"}, mappingFiles)

	mappingBuildIDDict := fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingBuildID)[0]).(*array.Dictionary)
	mappingBuildIDString := mappingBuildIDDict.Dictionary().(*array.String)
	mappingBuildID := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		mappingBuildID[i] = mappingBuildIDString.Value(mappingBuildIDDict.GetValueIndex(i))
	}
	require.Equal(t, []string{"aID", "aID", "aID", "aID", "aID", "aID", "aID", "aID", "aID", "aID"}, mappingBuildID)

	require.Equal(t,
		[]uint64{0xa1, 0xa2, 0xa1, 0xa2, 0xa3, 0xa5, 0xa1, 0xa2, 0xa3, 0xa4},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldLocationAddress)[0]).(*array.Uint64).Uint64Values(),
	)

	require.Equal(t,
		[]int64{1, 2, 1, 2, 3, 5, 1, 2, 3, 4},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldLocationLine)[0]).(*array.Int64).Int64Values(),
	)

	functionNames := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		functionNames[i] = fa.Column(fa.Schema().FieldIndices(flamegraphFieldFunctionName)[0]).(*array.String).Value(i)
	}
	require.Equal(t,
		[]string{"1", "2", "1", "2", "3", "5", "1", "2", "3", "4"},
		functionNames,
	)

	// TODO: Finish testing functions

	require.Equal(t,
		[]int64{2, 2, 1, 1, 1, 1, 3, 3, 3, 3},
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldCumulative)[0]).(*array.Int64).Int64Values(),
	)
	require.Equal(t,
		10,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldDiff)[0]).(*array.Int64).NullN(),
	)
}
