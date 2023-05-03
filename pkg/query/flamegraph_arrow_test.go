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

	// Create a list of all rows for humans.
	rows := []struct {
		MappingStart       uint64
		MappingLimit       uint64
		MappingOffset      uint64
		MappingFile        string
		MappingBuildID     string
		LocationAddress    uint64
		LocationFolded     bool
		LocationLine       int64
		FunctionStartLine  int64
		FunctionName       string
		FunctionSystemName string
		FunctionFilename   string
		Children           []uint32
		Cumulative         int64
	}{
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 2},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 2},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 1},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 1},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 1},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa5, LocationFolded: false, LocationLine: 5, FunctionStartLine: 5, FunctionName: "5", FunctionSystemName: "5", FunctionFilename: "5", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 1},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa1, LocationFolded: false, LocationLine: 1, FunctionStartLine: 1, FunctionName: "1", FunctionSystemName: "1", FunctionFilename: "1", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 3},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa2, LocationFolded: false, LocationLine: 2, FunctionStartLine: 2, FunctionName: "2", FunctionSystemName: "2", FunctionFilename: "2", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 3},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa3, LocationFolded: false, LocationLine: 3, FunctionStartLine: 3, FunctionName: "3", FunctionSystemName: "3", FunctionFilename: "3", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 3},
		{MappingStart: 1, MappingLimit: 1, MappingOffset: 0x1234, MappingFile: "a", MappingBuildID: "aID", LocationAddress: 0xa4, LocationFolded: false, LocationLine: 4, FunctionStartLine: 4, FunctionName: "4", FunctionSystemName: "4", FunctionFilename: "4", Children: []uint32{1, 2, 3, 4, 5}, Cumulative: 3},
	}

	// Convert the rows to columns for easier access when testing below.
	columns := struct {
		mappingStart        []uint64
		mappingLimit        []uint64
		mappingOffset       []uint64
		mappingFiles        []string
		mappingBuildIDs     []string
		locationAddresses   []uint64
		locationFolded      []bool
		locationLines       []int64
		functionStartLines  []int64
		functionNames       []string
		functionSystemNames []string
		functionFileNames   []string
		children            [][]uint32
		cumulative          []int64
	}{}
	for _, row := range rows {
		columns.mappingStart = append(columns.mappingStart, row.MappingStart)
		columns.mappingLimit = append(columns.mappingLimit, row.MappingLimit)
		columns.mappingOffset = append(columns.mappingOffset, row.MappingOffset)
		columns.mappingFiles = append(columns.mappingFiles, row.MappingFile)
		columns.mappingBuildIDs = append(columns.mappingBuildIDs, row.MappingBuildID)
		columns.locationAddresses = append(columns.locationAddresses, row.LocationAddress)
		columns.locationFolded = append(columns.locationFolded, row.LocationFolded)
		columns.locationLines = append(columns.locationLines, row.LocationLine)
		columns.functionStartLines = append(columns.functionStartLines, row.FunctionStartLine)
		columns.functionNames = append(columns.functionNames, row.FunctionName)
		columns.functionSystemNames = append(columns.functionSystemNames, row.FunctionSystemName)
		columns.functionFileNames = append(columns.functionFileNames, row.FunctionFilename)
		columns.children = append(columns.children, row.Children)
		columns.cumulative = append(columns.cumulative, row.Cumulative)
	}

	require.Equal(t,
		columns.mappingStart,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingStart)[0]).(*array.Uint64).Uint64Values(),
	)
	require.Equal(t,
		columns.mappingLimit,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingLimit)[0]).(*array.Uint64).Uint64Values(),
	)
	require.Equal(t,
		columns.mappingOffset,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingOffset)[0]).(*array.Uint64).Uint64Values(),
	)

	mappingFilesDict := fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingFile)[0]).(*array.Dictionary)
	mappingFilesString := mappingFilesDict.Dictionary().(*array.String)
	mappingFiles := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		mappingFiles[i] = mappingFilesString.Value(mappingFilesDict.GetValueIndex(i))
	}
	require.Equal(t, columns.mappingFiles, mappingFiles)

	mappingBuildIDDict := fa.Column(fa.Schema().FieldIndices(flamegraphFieldMappingBuildID)[0]).(*array.Dictionary)
	mappingBuildIDString := mappingBuildIDDict.Dictionary().(*array.String)
	mappingBuildID := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		mappingBuildID[i] = mappingBuildIDString.Value(mappingBuildIDDict.GetValueIndex(i))
	}
	require.Equal(t, columns.mappingBuildIDs, mappingBuildID)

	require.Equal(t,
		columns.locationAddresses,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldLocationAddress)[0]).(*array.Uint64).Uint64Values(),
	)

	locationFolded := make([]bool, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		locationFolded[i] = fa.Column(fa.Schema().FieldIndices(flamegraphFieldLocationFolded)[0]).(*array.Boolean).Value(i)
	}
	require.Equal(t, columns.locationFolded, locationFolded)

	require.Equal(t,
		columns.locationLines,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldLocationLine)[0]).(*array.Int64).Int64Values(),
	)

	require.Equal(t,
		columns.functionStartLines,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldFunctionStartLine)[0]).(*array.Int64).Int64Values(),
	)

	functionNames := make([]string, fa.NumRows())
	functionSystemNames := make([]string, fa.NumRows())
	functionFileNames := make([]string, fa.NumRows())
	for i := 0; i < int(fa.NumRows()); i++ {
		functionNames[i] = fa.Column(fa.Schema().FieldIndices(flamegraphFieldFunctionName)[0]).(*array.String).Value(i)
		functionSystemNames[i] = fa.Column(fa.Schema().FieldIndices(flamegraphFieldFunctionSystemName)[0]).(*array.String).Value(i)
		functionFileNames[i] = fa.Column(fa.Schema().FieldIndices(flamegraphFieldFunctionFileName)[0]).(*array.String).Value(i)
	}
	require.Equal(t, columns.functionNames, functionNames)
	require.Equal(t, columns.functionSystemNames, functionSystemNames)
	require.Equal(t, columns.functionFileNames, functionFileNames)

	require.Equal(t,
		columns.cumulative,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldCumulative)[0]).(*array.Int64).Int64Values(),
	)
	require.Equal(t,
		10,
		fa.Column(fa.Schema().FieldIndices(flamegraphFieldDiff)[0]).(*array.Int64).NullN(),
	)
}
