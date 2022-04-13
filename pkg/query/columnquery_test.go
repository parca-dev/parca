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
	"io/ioutil"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
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
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
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
		Query: `{job="default"}`,
		Start: timestamppb.New(timestamp.Time(0)),
		End:   timestamppb.New(timestamp.Time(9223372036854775807)),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 2, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 10, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQuery(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
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
				Query: `{job="default"}`,
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
				Query: `{job="default"}`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)

	_, err = profile.ParseData(res.Report.(*pb.QueryResponse_Pprof).Pprof)
	require.NoError(t, err)
}

func TestColumnQueryAPIQueryDiff(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
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
		Name:  "job",
		Value: "default",
	}}, &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			Timestamp: 1,
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
		Name:  "job",
		Value: "default",
	}}, &parcaprofile.Profile{
		Meta: parcaprofile.InstantProfileMeta{
			Timestamp: 2,
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
							Query: `{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `{job="default"}`,
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
							Query: `{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `{job="default"}`,
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
							Query: `{job="default"}`,
							Time:  timestamppb.New(timestamp.Time(1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: `{job="default"}`,
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

func TestColumnQueryAPILabelNames(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
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
		"__name__",
		"job",
	}, res.LabelNames)
}

func TestColumnQueryAPILabelValues(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
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
