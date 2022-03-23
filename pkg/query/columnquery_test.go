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
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

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
	table := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
		),
		logger,
	)
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
	table := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
		),
		logger,
	)
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

func TestColumnQueryAPILabelNames(t *testing.T) {
	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
		),
		logger,
	)
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
	table := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(
			parcacol.Schema(),
			8196,
			64*1024*1024,
		),
		logger,
	)
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
