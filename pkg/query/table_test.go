// Copyright 2023 The Parca Authors
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

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateTable(t *testing.T) {
	ctx := context.Background()
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	reg := prometheus.NewRegistry()
	counter := promauto.With(reg).NewCounter(prometheus.CounterOpts{
		Name: "parca_test_counter",
		Help: "parca_test_counter",
	})

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
	normalizer := parcacol.NewNormalizer(metastore, true, counter)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", map[string]string{}, p, true, nil)
	require.NoError(t, err)

	tracer := trace.NewNoopTracerProvider().Tracer("")
	symbolizedProfile, err := parcacol.NewProfileSymbolizer(tracer, metastore).SymbolizeNormalizedProfile(ctx, profiles[0])
	require.NoError(t, err)

	np, err := OldProfileToArrowProfile(symbolizedProfile)
	require.NoError(t, err)

	rec, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, np)
	require.NoError(t, err)
	defer rec.Release()

	require.NotNil(t, rec)
	require.NotNil(t, cumulative)

	require.Equal(t, int64(310797348), cumulative)
	// require.Equal(t, 899, rec.NumRows())

	mappingFileColumn := rec.Column(rec.Schema().FieldIndices(TableFieldMappingFile)[0]).(*array.Dictionary)
	mappingFileColumnDict := mappingFileColumn.Dictionary().(*array.String)
	mappingBuildIDColumn := rec.Column(rec.Schema().FieldIndices(TableFieldMappingBuildID)[0]).(*array.Dictionary)
	locationAddressColumn := rec.Column(rec.Schema().FieldIndices(TableFieldLocationAddress)[0]).(*array.Uint64)
	locationLineColumn := rec.Column(rec.Schema().FieldIndices(TableFieldLocationLine)[0]).(*array.Int64)
	functionStartLineColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionStartLine)[0]).(*array.Int64)
	functionNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionName)[0]).(*array.Dictionary)
	functionNameColumnDict := functionNameColumn.Dictionary().(*array.String)
	functionSystemNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionSystemName)[0]).(*array.Dictionary)
	functionSystemNameColumnDict := functionSystemNameColumn.Dictionary().(*array.String)
	functionFileNameColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFunctionFileName)[0]).(*array.Dictionary)
	functionFileNameColumnDict := functionFileNameColumn.Dictionary().(*array.String)
	cumulativeColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCumulative)[0]).(*array.Int64)
	cumulativeDiffColumn := rec.Column(rec.Schema().FieldIndices(TableFieldCumulativeDiff)[0]).(*array.Int64)
	flatColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFlat)[0]).(*array.Int64)
	flatDiffColumn := rec.Column(rec.Schema().FieldIndices(TableFieldFlatDiff)[0]).(*array.Int64)

	found := false
	for i := 0; i < int(rec.NumRows()); i++ {
		if locationAddressColumn.Value(i) == uint64(7578561) {
			// mapping
			require.Equal(t, "/bin/operator", mappingFileColumnDict.Value(mappingFileColumn.GetValueIndex(i)))
			require.True(t, mappingBuildIDColumn.IsNull(i))
			// location
			// address is already checked above
			require.Equal(t, int64(107), locationLineColumn.Value(i))
			// function
			require.Equal(t, int64(0), functionStartLineColumn.Value(i))
			require.Equal(t,
				"encoding/json.Unmarshal",
				functionNameColumnDict.Value(functionNameColumn.GetValueIndex(i)),
			)
			require.Equal(t,
				"encoding/json.Unmarshal",
				functionSystemNameColumnDict.Value(functionSystemNameColumn.GetValueIndex(i)),
			)
			require.Equal(t,
				"/opt/hostedtoolcache/go/1.14.10/x64/src/encoding/json/decode.go",
				functionFileNameColumnDict.Value(functionFileNameColumn.GetValueIndex(i)),
			)
			// values
			require.Equal(t, int64(3135531), cumulativeColumn.Value(i))
			require.Equal(t, int64(1251322), flatColumn.Value(i))
			// diff
			require.Equal(t, int64(0), cumulativeDiffColumn.Value(i))
			require.Equal(t, int64(0), flatDiffColumn.Value(i))

			found = true
		}
	}

	require.Truef(t, found, "expected to find the specific function")
}

func TestGenerateTableAggregateFlat(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	metastore := metastore.NewInProcessClient(metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	))

	mres, err := metastore.GetOrCreateMappings(ctx, &metastorepb.GetOrCreateMappingsRequest{
		Mappings: []*metastorepb.Mapping{{
			Id:      "1",
			Start:   1,
			Limit:   1,
			Offset:  1,
			File:    "1",
			BuildId: "1",
		}},
	})
	require.NoError(t, err)

	lres, err := metastore.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			Address:   0x1,
			MappingId: mres.Mappings[0].Id,
		}, {
			Address:   0x2,
			MappingId: mres.Mappings[0].Id,
		}, {
			Address:   0x3,
			MappingId: mres.Mappings[0].Id,
		}, {
			Address:   0x4,
			MappingId: mres.Mappings[0].Id,
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 4, len(lres.Locations))
	l0 := lres.Locations[0]
	l1 := lres.Locations[1]
	l2 := lres.Locations[2]
	l3 := lres.Locations[3]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{l1.Id, l0.Id},
		}, {
			LocationIds: []string{l2.Id, l0.Id},
		}, {
			LocationIds: []string{l3.Id, l0.Id},
		}, {
			LocationIds: []string{l0.Id},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 4, len(sres.Stacktraces))

	p, err := parcacol.NewProfileSymbolizer(tracer, metastore).SymbolizeNormalizedProfile(ctx, &profile.NormalizedProfile{
		Samples: []*profile.NormalizedSample{{
			StacktraceID: sres.Stacktraces[0].Id,
			Value:        1,
		}, {
			StacktraceID: sres.Stacktraces[1].Id,
			Value:        2,
		}, {
			StacktraceID: sres.Stacktraces[2].Id,
			Value:        3,
		}, {
			StacktraceID: sres.Stacktraces[3].Id,
			Value:        4,
		}},
	})
	require.NoError(t, err)

	np, err := OldProfileToArrowProfile(p)
	require.NoError(t, err)

	rec, cumulative, err := generateTableArrowRecord(ctx, mem, tracer, np)
	require.NoError(t, err)
	defer rec.Release()

	require.Equal(t, int64(4), rec.NumRows())
	require.Equal(t, int64(10), cumulative)

	expectedColumns := tableColumns{
		mappingFile:        []string{"1", "1", "1", "1"},
		mappingBuildID:     []string{"1", "1", "1", "1"},
		locationAddress:    []uint64{2, 1, 3, 4},
		locationLine:       []int64{0, 0, 0, 0},
		functionStartLine:  []int64{0, 0, 0, 0},
		functionName:       []string{"(null)", "(null)", "(null)", "(null)"},
		functionSystemName: []string{"(null)", "(null)", "(null)", "(null)"},
		functionFileName:   []string{"(null)", "(null)", "(null)", "(null)"},
		cumulative:         []int64{1, 10, 2, 3},
		cumulativeDiff:     []int64{0, 0, 0, 0},
		flat:               []int64{1, 4, 2, 3},
		flatDiff:           []int64{0, 0, 0, 0},
	}
	actualColumns := tableRecordToColumns(t, rec)

	require.Equal(t, expectedColumns, actualColumns)
}

type tableColumns struct {
	mappingFile        []string
	mappingBuildID     []string
	locationAddress    []uint64
	locationLine       []int64
	functionStartLine  []int64
	functionName       []string
	functionSystemName []string
	functionFileName   []string
	cumulative         []int64
	cumulativeDiff     []int64
	flat               []int64
	flatDiff           []int64
}

func tableRecordToColumns(t *testing.T, r arrow.Record) tableColumns {
	return tableColumns{
		mappingFile:        extractColumn(t, r, TableFieldMappingFile).([]string),
		mappingBuildID:     extractColumn(t, r, TableFieldMappingBuildID).([]string),
		locationAddress:    extractColumn(t, r, TableFieldLocationAddress).([]uint64),
		locationLine:       extractColumn(t, r, TableFieldLocationLine).([]int64),
		functionStartLine:  extractColumn(t, r, TableFieldFunctionStartLine).([]int64),
		functionName:       extractColumn(t, r, TableFieldFunctionName).([]string),
		functionSystemName: extractColumn(t, r, TableFieldFunctionSystemName).([]string),
		functionFileName:   extractColumn(t, r, TableFieldFunctionFileName).([]string),
		cumulative:         extractColumn(t, r, TableFieldCumulative).([]int64),
		cumulativeDiff:     extractColumn(t, r, TableFieldCumulativeDiff).([]int64),
		flat:               extractColumn(t, r, TableFieldFlat).([]int64),
		flatDiff:           extractColumn(t, r, TableFieldFlatDiff).([]int64),
	}
}
