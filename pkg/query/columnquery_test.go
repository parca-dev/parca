// Copyright 2022 The Parca Authors
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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow"
	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/polarsignals/arcticdb/query/logicalplan"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestColumnQueryAPIQueryRange(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	dir := "./testdata/many/"
	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	for _, f := range files {
		fileContent, err := ioutil.ReadFile(dir + f.Name())
		require.NoError(t, err)
		p, err := profile.Parse(bytes.NewBuffer(fileContent))
		require.NoError(t, err)
		profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
		require.NoError(t, err)
		_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
			Name:  "__name__",
			Value: "memory",
		}, {
			Name:  "job",
			Value: "default",
		}}, profiles[0])
		require.NoError(t, err)
	}

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	res, err := api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
		Start: timestamppb.New(timestamp.Time(0)),
		End:   timestamppb.New(timestamp.Time(9223372036854775807)),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 1, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 10, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQuery(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 4, len(profiles))
	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}, {
		Name:  "job",
		Value: "default",
	}}, profiles[0])
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	ts := timestamppb.New(timestamp.Time(p.TimeNanos / time.Millisecond.Nanoseconds()))
	res, err := api.Query(ctx, &pb.QueryRequest{
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int32(33), res.Report.(*pb.QueryResponse_Flamegraph).Flamegraph.Height)

	res, err = api.Query(ctx, &pb.QueryRequest{
		ReportType: pb.QueryRequest_REPORT_TYPE_PPROF,
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)

	_, err = profile.ParseData(res.Report.(*pb.QueryResponse_Pprof).Pprof)
	require.NoError(t, err)
}

func TestColumnQueryAPIQueryPprofLabels(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/cpu_labels.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(profiles))
	for _, profile := range profiles {
		_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
			Name:  "__name__",
			Value: "process_cpu",
		}, {
			Name:  "job",
			Value: "default",
		}}, profile)
		require.NoError(t, err)
	}

	engine := query.NewEngine(
		memory.DefaultAllocator,
		colDB.TableProvider(),
	)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		engine,
		"stacktraces",
	)

	err = engine.ScanTable("stacktraces").
		Project(
			logicalplan.Col("sample_unit"),
			logicalplan.Col("profile_labels.grpcmethod"),
			logicalplan.Col("timestamp"),
			logicalplan.Col("value"),
		).
		Filter(
			logicalplan.And(
				logicalplan.Col("profile_labels.grpcmethod").Eq(logicalplan.Literal("Query")),
			),
		).
		Execute(func(r arrow.Record) error {
			fmt.Println(r)
			return nil
		})
	require.NoError(t, err)

	ts := timestamppb.New(timestamp.Time(p.TimeNanos / time.Millisecond.Nanoseconds()))
	res, err := api.Query(ctx, &pb.QueryRequest{
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: `process_cpu:cpu:nanoseconds:cpu:nanoseconds:delta{profile_label_grpcmethod="Query"}`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)
	require.Equal(t, int32(15), res.Report.(*pb.QueryResponse_Flamegraph).Flamegraph.Height)
}

func TestColumnQueryAPIQueryFgprof(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/fgprof.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	p.TimeNanos = time.Now().UnixNano()
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 2, len(profiles))
	for _, profile := range profiles {
		_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
			Name:  "__name__",
			Value: "fgprof",
		}, {
			Name:  "job",
			Value: "default",
		}}, profile)
		require.NoError(t, err)
	}

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	res, err := api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `fgprof:samples:count::`,
		Start: timestamppb.New(timestamp.Time(0)),
		End:   timestamppb.New(timestamp.Time(9223372036854775807)),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 1, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 1, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQueryDiff(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	f1 := &metastorepb.Function{
		Name: "testFunc",
	}
	f1.Id, err = m.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metastorepb.Function{
		Name: "testFunc",
	}
	f2.Id, err = m.CreateFunction(ctx, f2)
	require.NoError(t, err)

	loc1 := &metastore.Location{
		Address: 0x1,
		Lines: []metastore.LocationLine{{
			Line:     1,
			Function: f1,
		}},
	}
	loc2 := &metastore.Location{
		Address: 0x2,
		Lines: []metastore.LocationLine{{
			Line:     2,
			Function: f2,
		}},
	}

	id1, err := m.CreateLocation(ctx, loc1)
	require.NoError(t, err)
	loc1.ID, err = uuid.FromBytes(id1)
	require.NoError(t, err)

	id2, err := m.CreateLocation(ctx, loc2)
	require.NoError(t, err)
	loc2.ID, err = uuid.FromBytes(id2)
	require.NoError(t, err)

	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}, {
		Name:  "job",
		Value: "default",
	}}, &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			Timestamp: 1,
			SampleType: parcaprofile.ValueType{
				Type: "alloc_objects",
				Unit: "count",
			},
			PeriodType: parcaprofile.ValueType{
				Type: "space",
				Unit: "bytes",
			},
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			"a": {
				Location: []*metastore.Location{loc1},
				Value:    1,
			},
		},
	})
	require.NoError(t, err)
	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}, {
		Name:  "job",
		Value: "default",
	}}, &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			Timestamp: 2,
			SampleType: parcaprofile.ValueType{
				Type: "alloc_objects",
				Unit: "count",
			},
			PeriodType: parcaprofile.ValueType{
				Type: "space",
				Unit: "bytes",
			},
		},
		FlatSamples: map[string]*parcaprofile.Sample{
			"b": {
				Location: []*metastore.Location{loc2},
				Value:    2,
			},
		},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)

	res, err := api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_DIFF,
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				A: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(2)),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	fg := res.Report.(*pb.QueryResponse_Flamegraph).Flamegraph
	require.Equal(t, int32(2), fg.Height)
	require.Equal(t, 1, len(fg.Root.Children))
	require.Equal(t, int64(2), fg.Root.Children[0].Cumulative)
	require.Equal(t, int64(1), fg.Root.Children[0].Diff)

	res, err = api.Query(ctx, &pb.QueryRequest{
		Mode:       pb.QueryRequest_MODE_DIFF,
		ReportType: *pb.QueryRequest_REPORT_TYPE_TOP.Enum(),
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				A: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(2)),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	topList := res.Report.(*pb.QueryResponse_Top).Top.List
	require.Equal(t, 1, len(topList))
	require.Equal(t, int64(2), topList[0].Cumulative)
	require.Equal(t, int64(1), topList[0].Diff)

	res, err = api.Query(ctx, &pb.QueryRequest{
		Mode:       pb.QueryRequest_MODE_DIFF,
		ReportType: *pb.QueryRequest_REPORT_TYPE_PPROF.Enum(),
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				A: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(2)),
						},
					},
				},
			},
		},
	})
	require.NoError(t, err)

	resProf, err := profile.ParseData(res.Report.(*pb.QueryResponse_Pprof).Pprof)
	require.NoError(t, err)
	require.Equal(t, 2, len(resProf.Sample))
	require.Equal(t, []int64{2}, resProf.Sample[0].Value)
	require.Equal(t, []int64{-1}, resProf.Sample[1].Value)
}

func TestColumnQueryAPITypes(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/alloc_space_delta.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 4, len(profiles))
	for _, prof := range profiles {
		_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
			Name:  "__name__",
			Value: "memory",
		}, {
			Name:  "job",
			Value: "default",
		}}, prof)
		require.NoError(t, err)
	}

	table.Sync()

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	res, err := api.ProfileTypes(ctx, &pb.ProfileTypesRequest{})
	require.NoError(t, err)

	require.True(t, proto.Equal(&pb.ProfileTypesResponse{Types: []*pb.ProfileType{
		{Name: "memory", SampleType: "alloc_objects", SampleUnit: "count", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "alloc_space", SampleUnit: "bytes", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "inuse_objects", SampleUnit: "count", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "inuse_space", SampleUnit: "bytes", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
	}}, res))
}

func TestColumnQueryAPILabelNames(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 4, len(profiles))
	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}, {
		Name:  "job",
		Value: "default",
	}}, profiles[0])
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	res, err := api.Labels(ctx, &pb.LabelsRequest{})
	require.NoError(t, err)

	require.Equal(t, []string{
		"job",
	}, res.LabelNames)
}

func TestColumnQueryAPILabelValues(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(
		reg,
		8196,
		64*1024*1024,
	)
	colDB, err := col.DB("parca")
	require.NoError(t, err)
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
		),
		logger,
	)
	require.NoError(t, err)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

	fileContent, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(bytes.NewBuffer(fileContent))
	require.NoError(t, err)
	profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
	require.NoError(t, err)
	require.Equal(t, 4, len(profiles))
	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}, {
		Name:  "job",
		Value: "default",
	}}, profiles[0])
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)
	res, err := api.Values(ctx, &pb.ValuesRequest{
		LabelName: "job",
	})
	require.NoError(t, err)

	require.Equal(t, []string{
		"default",
	}, res.LabelValues)
}
