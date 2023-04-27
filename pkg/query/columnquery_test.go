// Copyright 2022-2023 The Parca Authors
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
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/binary"
	"io"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/apache/arrow/go/v12/arrow/memory"
	"github.com/cespare/xxhash"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	columnstore "github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	metastorepb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/profilestore"
)

func getShareServerConn(t Testing) sharepb.ShareServiceClient {
	conn, err := grpc.Dial("api.pprof.me:443", grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	require.NoError(t, err)
	return sharepb.NewShareServiceClient(conn)
}

func TestColumnQueryAPIQueryRangeEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	_, err = colDB.Table("stacktraces", columnstore.NewTableConfig(parcacol.SchemaDefinition()))
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)
	metastore := metastore.NewInProcessClient(m)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
	)
	_, err = api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
		Start: timestamppb.New(timestamp.Time(0)),
		End:   timestamppb.New(timestamp.Time(9223372036854775807)),
	})
	require.ErrorIs(t, err, status.Error(
		codes.NotFound,
		"No data found for the query, try a different query or time range or no data has been written to be queried yet.",
	))
}

type Testing interface {
	require.TestingT
	Helper()
}

func MustReadAllGzip(t Testing, filename string) []byte {
	t.Helper()

	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

func MustDecompressGzip(t Testing, b []byte) []byte {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

func TestColumnQueryAPIQueryRange(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	dir := "./testdata/many/"
	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	for _, f := range files {
		fileContent, err := os.ReadFile(dir + f.Name())
		require.NoError(t, err)

		_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
			Series: []*profilestorepb.RawProfileSeries{{
				Labels: &profilestorepb.LabelSet{
					Labels: []*profilestorepb.Label{
						{
							Name:  "__name__",
							Value: "memory",
						},
						{
							Name:  "job",
							Value: "default",
						},
					},
				},
				Samples: []*profilestorepb.RawSample{{
					RawProfile: fileContent,
				}},
			}},
		})
		require.NoError(t, err)
	}

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
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

func ptrToString(s string) *string {
	return &s
}

func TestColumnQueryAPIQuerySingle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(MustDecompressGzip(t, fileContent)))

	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "memory",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
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

	unfilteredRes, err := api.Query(ctx, &pb.QueryRequest{
		ReportType: pb.QueryRequest_REPORT_TYPE_TOP,
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)

	filteredRes, err := api.Query(ctx, &pb.QueryRequest{
		ReportType: pb.QueryRequest_REPORT_TYPE_TOP,
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: `memory:alloc_objects:count:space:bytes{job="default", __name__="memory"}`,
				Time:  ts,
			},
		},
		FilterQuery: ptrToString("runtime"),
	})
	require.NoError(t, err)
	require.Less(t, len(filteredRes.Report.(*pb.QueryResponse_Top).Top.List), len(unfilteredRes.Report.(*pb.QueryResponse_Top).Top.List), "filtered result should be smaller than unfiltered result")

	testProf := &pprofpb.Profile{}
	err = testProf.UnmarshalVT(MustDecompressGzip(t, res.Report.(*pb.QueryResponse_Pprof).Pprof))
	require.NoError(t, err)
}

func TestColumnQueryAPIQueryFgprof(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	fileContent, err := os.ReadFile("testdata/fgprof.pb.gz")
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "fgprof",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
	)

	res, err := api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `fgprof:samples:count:wallclock:nanoseconds:delta`,
		Start: timestamppb.New(timestamp.Time(0)),
		End:   timestamppb.New(timestamp.Time(9223372036854775807)),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 1, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 1, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQueryDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)
	metastore := metastore.NewInProcessClient(m)

	fres, err := m.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{{
			Name: "testFunc",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(fres.Functions))
	f1 := fres.Functions[0]

	fres, err = m.GetOrCreateFunctions(ctx, &metastorepb.GetOrCreateFunctionsRequest{
		Functions: []*metastorepb.Function{{
			// Intentionally doing this again using the same name as f1 to simulate
			// what would happen when the two profiles are written separately.
			Name: "testFunc",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(fres.Functions))
	f2 := fres.Functions[0]

	lres, err := m.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			Address: 0x1,
			Lines: []*metastorepb.Line{{
				Line:       1,
				FunctionId: f1.Id,
			}},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres.Locations))
	loc1 := lres.Locations[0]

	sres, err := m.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{loc1.Id},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(sres.Stacktraces))
	st1 := sres.Stacktraces[0]

	lres, err = m.GetOrCreateLocations(ctx, &metastorepb.GetOrCreateLocationsRequest{
		Locations: []*metastorepb.Location{{
			Address: 0x2,
			Lines: []*metastorepb.Line{{
				Line:       2,
				FunctionId: f2.Id,
			}},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres.Locations))
	loc2 := lres.Locations[0]

	sres, err = m.GetOrCreateStacktraces(ctx, &metastorepb.GetOrCreateStacktracesRequest{
		Stacktraces: []*metastorepb.Stacktrace{{
			LocationIds: []string{loc2.Id},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(sres.Stacktraces))
	st2 := sres.Stacktraces[0]

	ingester := parcacol.NewNormalizedIngester(
		logger,
		table,
		schema,
		&sync.Pool{
			New: func() any {
				return new(bytes.Buffer)
			},
		},
		[]string{"job"},
		nil, nil,
	)

	require.NoError(t, ingester.Ingest(ctx, []parcacol.Series{{
		Labels: map[string]string{"job": "default"},
		Samples: [][]*profile.NormalizedProfile{{{
			Meta: profile.Meta{
				Name:       "memory",
				PeriodType: profile.ValueType{Type: "space", Unit: "bytes"},
				SampleType: profile.ValueType{Type: "alloc_objects", Unit: "count"},
				Timestamp:  1,
			},
			Samples: []*profile.NormalizedSample{{
				StacktraceID: st1.Id,
				Value:        1,
			}},
		}}},
	}}))

	require.NoError(t, ingester.Ingest(ctx, []parcacol.Series{{
		Labels: map[string]string{"job": "default"},
		Samples: [][]*profile.NormalizedProfile{{{
			Meta: profile.Meta{
				Name:       "memory",
				PeriodType: profile.ValueType{Type: "space", Unit: "bytes"},
				SampleType: profile.ValueType{Type: "alloc_objects", Unit: "count"},
				Timestamp:  2,
			},
			Samples: []*profile.NormalizedSample{{
				StacktraceID: st2.Id,
				Value:        2,
			}},
		}}},
	}}))

	_, err = m.Stacktraces(ctx, &metastorepb.StacktracesRequest{
		StacktraceIds: []string{st1.Id, st2.Id},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
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

	testProf := &pprofpb.Profile{}
	err = testProf.UnmarshalVT(MustDecompressGzip(t, res.Report.(*pb.QueryResponse_Pprof).Pprof))
	require.NoError(t, err)
	require.Equal(t, 2, len(testProf.Sample))
	require.Equal(t, []int64{2}, testProf.Sample[0].Value)
	require.Equal(t, []int64{-1}, testProf.Sample[1].Value)
}

func TestColumnQueryAPITypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	fileContent, err := os.ReadFile("testdata/alloc_space_delta.pb.gz")
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "memory",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	require.NoError(t, err)

	require.NoError(t, table.EnsureCompaction())

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
	)
	res, err := api.ProfileTypes(ctx, &pb.ProfileTypesRequest{})
	require.NoError(t, err)

	/* res returned by profile type on arm machine did not have same ordering
	on `SampleType: "inuse_objects"` and `inuse_space`. Due to which test
	was quite flaky and failing. So instead of testing for exact structure of
	the proto message, comparing by proto size of the messages.
	*/
	require.Equal(t, proto.Size(&pb.ProfileTypesResponse{Types: []*pb.ProfileType{
		{Name: "memory", SampleType: "alloc_objects", SampleUnit: "count", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "alloc_space", SampleUnit: "bytes", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "inuse_objects", SampleUnit: "count", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
		{Name: "memory", SampleType: "inuse_space", SampleUnit: "bytes", PeriodType: "space", PeriodUnit: "bytes", Delta: true},
	}}), proto.Size(res))
}

func TestColumnQueryAPILabelNames(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "memory",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
	)
	res, err := api.Labels(ctx, &pb.LabelsRequest{})
	require.NoError(t, err)

	require.Equal(t, []string{
		"job",
	}, res.LabelNames)
}

func TestColumnQueryAPILabelValues(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	metastore := metastore.NewInProcessClient(m)
	store := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		true,
	)

	_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{
				Labels: []*profilestorepb.Label{
					{
						Name:  "__name__",
						Value: "memory",
					},
					{
						Name:  "job",
						Value: "default",
					},
				},
			},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	require.NoError(t, err)

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				memory.DefaultAllocator,
				colDB.TableProvider(),
			),
			"stacktraces",
			metastore,
		),
	)
	res, err := api.Values(ctx, &pb.ValuesRequest{
		LabelName: "job",
	})
	require.NoError(t, err)

	require.Equal(t, []string{
		"default",
	}, res.LabelValues)
}

func BenchmarkQuery(b *testing.B) {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	p, err := pprofprofile.ParseData(fileContent)
	require.NoError(b, err)

	sp, err := PprofToSymbolizedProfile(profile.Meta{}, p, 0)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = RenderReport(ctx, tracer, sp, pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW, 0, 0, NewTableConverterPool())
	}
}

func PprofToSymbolizedProfile(meta profile.Meta, prof *pprofprofile.Profile, index int) (*profile.Profile, error) {
	p := &profile.Profile{
		Meta:    meta,
		Samples: make([]*profile.SymbolizedSample, 0, len(prof.Sample)),
	}
	for i := range prof.Sample {
		if len(prof.Sample[i].Value) <= index {
			return nil, status.Errorf(codes.InvalidArgument, "failed to find samples for profile type")
		}

		locs := make([]*profile.Location, 0, len(prof.Sample[i].Location))
		for _, loc := range prof.Sample[i].Location {
			symLoc := &profile.Location{
				Address: loc.Address,
			}

			if loc.Mapping != nil {
				symLoc.Mapping = &metastorepb.Mapping{
					Start:   loc.Mapping.Start,
					Limit:   loc.Mapping.Limit,
					Offset:  loc.Mapping.Offset,
					File:    loc.Mapping.File,
					BuildId: loc.Mapping.BuildID,
				}
				symLoc.Mapping.Id = metastore.MakeMappingID(symLoc.Mapping)
				symLoc.ID = symLoc.Mapping.Id + "-" + strconv.FormatUint(loc.Address, 16)
			}

			if loc.Line != nil {
				symLoc.Lines = make([]profile.LocationLine, 0, len(loc.Line))
				for _, line := range loc.Line {
					f := &metastorepb.Function{
						StartLine:  line.Function.StartLine,
						Name:       line.Function.Name,
						SystemName: line.Function.SystemName,
						Filename:   line.Function.Filename,
					}
					f.Id = metastore.MakeFunctionID(f)
					symLoc.Lines = append(symLoc.Lines, profile.LocationLine{
						Line:     line.Line,
						Function: f,
					})
				}

				symLoc.ID = makeLocationIDWithLines(symLoc.Lines)
			}

			locs = append(locs, symLoc)
		}

		var labels map[string]string
		if len(prof.Sample[i].Label) > 0 {
			labels = make(map[string]string, len(prof.Sample[i].Label))
			for key, values := range prof.Sample[i].Label {
				if len(values) > 0 {
					labels[key] = values[0]
				}
			}
		}

		p.Samples = append(p.Samples, &profile.SymbolizedSample{
			Locations: locs,
			Value:     prof.Sample[i].Value[index],
			Label:     labels,
		})
	}

	return p, nil
}

// makeLocationIDWithLines returns a key for the location that uniquely
// identifies the location. Locations are uniquely identified by their inlined
// function callstack.
func makeLocationIDWithLines(lines []profile.LocationLine) string {
	size := len(lines) * 16 // 8 bytes for line number and 8 bytes for function start line number

	for _, line := range lines {
		size += len(line.Function.Name) + len(line.Function.SystemName) + len(line.Function.Filename)
	}

	buf := make([]byte, size)
	pos := 0
	for _, line := range lines {
		binary.BigEndian.PutUint64(buf[pos:], uint64(line.Line))
		pos += 8

		binary.BigEndian.PutUint64(buf[pos:], uint64(line.Function.StartLine))
		pos += 8

		copy(buf[pos:], line.Function.Name)
		pos += len(line.Function.Name)

		copy(buf[pos:], line.Function.SystemName)
		pos += len(line.Function.SystemName)

		copy(buf[pos:], line.Function.Filename)
		pos += len(line.Function.Filename)
	}

	return strconv.FormatUint(xxhash.Sum64(buf), 16)
}
