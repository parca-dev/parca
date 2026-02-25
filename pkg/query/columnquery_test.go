// Copyright 2022-2026 The Parca Authors
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
	"io"
	"os"
	"testing"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	pprofprofile "github.com/google/pprof/profile"
	columnstore "github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/ingester"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/normalizer"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/profilestore"
)

func getShareServerConn(t Testing) sharepb.ShareServiceClient {
	conn, err := grpc.NewClient("api.pprof.me:443", grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	require.NoError(t, err)
	return sharepb.NewShareServiceClient(conn)
}

func TestColumnQueryAPIQueryRangeEmpty(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	_, err = colDB.Table("stacktraces", columnstore.NewTableConfig(profile.SchemaDefinition()))
	require.NoError(t, err)

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
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
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	dir := "./testdata/many/"
	files, err := os.ReadDir(dir)
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)
	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
	)
	res, err := api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `memory:alloc_objects:count:space:bytes{job="default"}`,
		Start: timestamppb.New(time.Unix(0, 0)),
		End:   timestamppb.New(time.Unix(0, 9223372036854775807)),
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 1, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 10, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQuerySingle(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)
	ingester := ingester.NewIngester(
		logger,
		table,
	)
	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
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
		Filter: []*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "runtime",
									},
								},
							},
						},
					},
				},
			},
		},
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
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	fileContent, err := os.ReadFile("testdata/fgprof.pb.gz")
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)

	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
	)

	res, err := api.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `fgprof:samples:count:wallclock:nanoseconds:delta`,
		Start: timestamppb.New(time.Unix(0, 0)),
		End:   timestamppb.New(time.Unix(0, 9223372036854775807)),
		SumBy: []string{"job"},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(res.Series))
	require.Equal(t, 1, len(res.Series[0].Labelset.Labels))
	require.Equal(t, 1, len(res.Series[0].Samples))
}

func TestColumnQueryAPIQueryCumulative(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)

	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
	)

	// Load CPU and memory profiles
	fileNames := []string{
		"testdata/alloc_objects.pb.gz",
		"testdata/profile1.pb.gz",
	}
	labelSets := []*profilestorepb.LabelSet{
		{
			Labels: []*profilestorepb.Label{
				{Name: "__name__", Value: "memory"},
				{Name: "job", Value: "default"},
			},
		},
		{
			Labels: []*profilestorepb.Label{
				{Name: "__name__", Value: "cpu"},
				{Name: "job", Value: "default"},
			},
		},
	}
	for i, fileName := range fileNames {
		fileContent, err := os.ReadFile(fileName)
		require.NoError(t, err)

		p := &pprofpb.Profile{}
		require.NoError(t, p.UnmarshalVT(MustDecompressGzip(t, fileContent)))

		_, err = store.WriteRaw(ctx, &profilestorepb.WriteRawRequest{
			Series: []*profilestorepb.RawProfileSeries{{
				Labels: labelSets[i],
				Samples: []*profilestorepb.RawSample{{
					RawProfile: fileContent,
				}},
			}},
		})
		require.NoError(t, err)
	}

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
	)

	// These have been extracted from the profiles above.
	queries := []struct {
		name      string
		query     string
		timeNanos int64
		// expected
		total    int64
		filtered int64
	}{{
		name:      "memory",
		query:     `memory:alloc_objects:count:space:bytes{job="default"}`,
		timeNanos: 1608199718549304626,
		total:     int64(310797348),
		filtered:  int64(0),
	}, {
		name:      "cpu",
		query:     `cpu:samples:count:cpu:nanoseconds:delta{job="default"}`,
		timeNanos: 1626013307085084416,
		total:     int64(48),
		filtered:  int64(0),
	}}

	// Check that the following report type return the same cumulative and filtered values.

	reportTypes := []pb.QueryRequest_ReportType{
		pb.QueryRequest_REPORT_TYPE_TOP,
		pb.QueryRequest_REPORT_TYPE_CALLGRAPH,
		pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_TABLE,
		pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW,
	}

	for _, query := range queries {
		for _, reportType := range reportTypes {
			t.Run(query.name+"-"+pb.QueryRequest_ReportType_name[int32(reportType)], func(t *testing.T) {
				res, err := api.Query(ctx, &pb.QueryRequest{
					ReportType: pb.QueryRequest_REPORT_TYPE_TOP,
					Options: &pb.QueryRequest_Single{
						Single: &pb.SingleProfile{
							Query: query.query,
							Time:  timestamppb.New(timestamp.Time(query.timeNanos / time.Millisecond.Nanoseconds())),
						},
					},
				})
				require.NoError(t, err)
				require.Equal(t, query.total, res.Total)
				require.Equal(t, query.filtered, res.Filtered)
			})
		}
	}
}

func MustCompressGzip(t Testing, p *pprofpb.Profile) []byte {
	t.Helper()

	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	content, err := p.MarshalVT()
	require.NoError(t, err)
	_, err = w.Write(content)
	require.NoError(t, err)
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestColumnQueryAPIQueryDiff(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	p := &pprofpb.Profile{
		StringTable: []string{
			"",
			"testFunc",
			"alloc_objects",
			"count",
			"space",
			"bytes",
		},
		Function: []*pprofpb.Function{{
			Id:   1,
			Name: 1,
		}},
		Location: []*pprofpb.Location{{
			Id:      1,
			Address: 0x1,
			Line: []*pprofpb.Line{{
				Line:       1,
				FunctionId: 1,
			}},
		}, {
			Id:      2,
			Address: 0x2,
			Line: []*pprofpb.Line{{
				Line:       2,
				FunctionId: 1,
			}},
		}},
		SampleType: []*pprofpb.ValueType{{
			Type: 2,
			Unit: 3,
		}},
		PeriodType: &pprofpb.ValueType{
			Type: 4,
			Unit: 5,
		},
		TimeNanos: 1000000,
		Sample: []*pprofpb.Sample{{
			Value:      []int64{1},
			LocationId: []uint64{1},
		}},
	}

	ingester := ingester.NewIngester(
		logger,
		table,
	)
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	r1, err := normalizer.WriteRawRequestToArrowRecord(ctx, mem, &profilestorepb.WriteRawRequest{
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
				RawProfile: MustCompressGzip(t, p),
			}},
		}},
	}, schema)
	require.NoError(t, err)
	require.NoError(t, ingester.Ingest(ctx, r1))
	r1.Release()

	p.Sample = []*pprofpb.Sample{{
		Value:      []int64{2},
		LocationId: []uint64{2},
	}}
	p.TimeNanos = 2000000
	r2, err := normalizer.WriteRawRequestToArrowRecord(ctx, mem, &profilestorepb.WriteRawRequest{
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
				RawProfile: MustCompressGzip(t, p),
			}},
		}},
	}, schema)
	require.NoError(t, err)
	require.NoError(t, ingester.Ingest(ctx, r2))
	r2.Release()

	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
	)

	res, err := api.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_DIFF,
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				Absolute: proto.Bool(true),
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
				Absolute: proto.Bool(true),
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
		ReportType: *pb.QueryRequest_REPORT_TYPE_TOP.Enum(),
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				// Absolute: proto.Bool(false), it's the default
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

	topList = res.Report.(*pb.QueryResponse_Top).Top.List
	require.Equal(t, 1, len(topList))
	require.Equal(t, int64(2), topList[0].Cumulative)
	require.Equal(t, int64(0), topList[0].Diff) // we expect the root to have no difference due to scaling to the same cumulative value

	res, err = api.Query(ctx, &pb.QueryRequest{
		Mode:       pb.QueryRequest_MODE_DIFF,
		ReportType: *pb.QueryRequest_REPORT_TYPE_PPROF.Enum(),
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				Absolute: proto.Bool(true),
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

	// Need to double release them because the storage will keep a reference to them.
	r1.Release()
	r2.Release()
}

func TestColumnQueryAPITypes(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	fileContent, err := os.ReadFile("testdata/alloc_space_delta.pb.gz")
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)
	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
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
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)
	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
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
	tracer := noop.NewTracerProvider().Tracer("")
	col, err := columnstore.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := profile.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		columnstore.NewTableConfig(profile.SchemaDefinition()),
	)
	require.NoError(t, err)

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

	ingester := ingester.NewIngester(
		logger,
		table,
	)
	store := profilestore.NewProfileColumnStore(
		reg,
		logger,
		tracer,
		ingester,
		schema,
		memory.DefaultAllocator,
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

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	api := NewColumnQueryAPI(
		logger,
		tracer,
		getShareServerConn(t),
		parcacol.NewQuerier(
			logger,
			tracer,
			query.NewEngine(
				mem,
				colDB.TableProvider(),
			),
			"stacktraces",
			nil,
			nil,
			mem,
		),
		mem,
		parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
		nil,
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
	tracer := noop.NewTracerProvider().Tracer("")

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	p, err := pprofprofile.ParseData(fileContent)
	require.NoError(b, err)

	sp, err := PprofToSymbolizedProfile(profile.Meta{}, p, 0, []string{})
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(b, 0)
	for i := 0; i < b.N; i++ {
		_, _ = RenderReport(
			ctx,
			tracer,
			sp,
			pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW,
			0,
			0,
			[]string{FlamegraphFieldFunctionName},
			NewTableConverterPool(),
			mem,
			parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
			nil,
			"",
			false,
		)
	}
}

func PprofToSymbolizedProfile(meta profile.Meta, prof *pprofprofile.Profile, index int, groupBy []string) (profile.Profile, error) {
	labelNameSet := make(map[string]struct{})
	for _, s := range prof.Sample {
		for k := range s.Label {
			labelNameSet[k] = struct{}{}
		}
	}
	labelNames := make([]string, 0, len(labelNameSet))
	for l := range labelNameSet {
		labelNames = append(labelNames, l)
	}

	groupBySet := make(map[string]struct{}, len(groupBy))
	for _, g := range groupBy {
		groupBySet[g] = struct{}{}
	}

	w := profile.NewWriter(memory.DefaultAllocator, labelNames)
	defer w.RecordBuilder.Release()
	for i := range prof.Sample {
		if len(prof.Sample[i].Value) <= index {
			return profile.Profile{}, status.Errorf(codes.InvalidArgument, "failed to find samples for profile type")
		}

		w.Value.Append(prof.Sample[i].Value[index])
		w.Diff.Append(0)
		w.TimeNanos.Append(prof.TimeNanos)
		w.Period.Append(prof.Period)

		for labelName, labelBuilder := range w.LabelBuildersMap {
			if prof.Sample[i].Label == nil {
				labelBuilder.AppendNull()
				continue
			}

			if labelValues, ok := prof.Sample[i].Label[labelName]; ok && len(labelValues) > 0 {
				labelBuilder.Append([]byte(labelValues[0]))
			} else {
				labelBuilder.AppendNull()
			}
		}

		w.LocationsList.Append(len(prof.Sample[i].Location) > 0)
		if len(prof.Sample[i].Location) > 0 {
			for _, loc := range prof.Sample[i].Location {
				w.Locations.Append(true)
				w.Addresses.Append(loc.Address)

				if loc.Mapping != nil {
					w.MappingStart.Append(loc.Mapping.Start)
					w.MappingLimit.Append(loc.Mapping.Limit)
					w.MappingOffset.Append(loc.Mapping.Offset)
					w.MappingFile.Append([]byte(loc.Mapping.File))
					w.MappingBuildID.Append([]byte(loc.Mapping.BuildID))
				} else {
					w.MappingStart.AppendNull()
					w.MappingLimit.AppendNull()
					w.MappingOffset.AppendNull()
					w.MappingFile.AppendNull()
					w.MappingBuildID.AppendNull()
				}

				w.Lines.Append(len(loc.Line) > 0)
				if len(loc.Line) > 0 {
					for _, line := range loc.Line {
						w.Line.Append(true)
						w.LineNumber.Append(line.Line)
						if line.Function != nil {
							w.FunctionName.Append([]byte(line.Function.Name))
							w.FunctionSystemName.Append([]byte(line.Function.SystemName))
							w.FunctionFilename.Append([]byte(line.Function.Filename))
							w.FunctionStartLine.Append(line.Function.StartLine)
						} else {
							w.FunctionName.AppendNull()
							w.FunctionSystemName.AppendNull()
							w.FunctionFilename.AppendNull()
							w.FunctionStartLine.AppendNull()
						}
					}
				}
			}
		}
	}

	return profile.Profile{
		Meta:    meta,
		Samples: []arrow.RecordBatch{w.RecordBuilder.NewRecordBatch()},
	}, nil
}

func TestFilterData(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "test",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 2, valid)
	require.Equal(t, "test", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(0)))))
	require.Equal(t, "test1", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(1)))))
}

func TestFilterUnsymbolized(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(false)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "test",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 1, valid)
}

func TestFilterDataWithPath(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libc.so.6"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("__libc_start_main"))
	w.FunctionSystemName.Append([]byte("__libc_start_main"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("/usr/lib/libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test1"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(0)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("interpreter"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test.py"))
	w.FunctionStartLine.Append(0)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "libpython3.11.so.1.0",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	validIndexes := []uint32{}
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			start, end := r.Lines.ValueOffsets(i)
			for j := int(start); j < int(end); j++ {
				if r.Line.IsValid(j) {
					validIndexes = append(validIndexes, r.LineFunctionNameIndices.Value(j))
				}
			}
		}
	}
	require.Equal(t, 1, len(validIndexes))
	require.Equal(t, "test1", string(r.LineFunctionNameDict.Value(int(validIndexes[0]))))
}

func TestFilterDataFrameFilter(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libc.so.6"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("__libc_start_main"))
	w.FunctionSystemName.Append([]byte("__libc_start_main"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("/usr/lib/libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test1"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(0)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("interpreter"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test.py"))
	w.FunctionStartLine.Append(0)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "interpreter",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 1, valid)
	require.Equal(t, "test", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(2)))))
}

func BenchmarkFilterData(t *testing.B) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	for i := 0; i < 10000; i++ {
		w.LocationsList.Append(true)
		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("test"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.FunctionName.Append([]byte("test"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)

		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("libpython3.11.so.1.0"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.FunctionName.Append([]byte("test1"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)

		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("test"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.FunctionName.Append([]byte("test1"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)
		w.Value.Append(1)
		w.Diff.Append(0)
		w.TimeNanos.Append(1)
		w.Period.Append(1)
	}

	originalRecord := w.RecordBuilder.NewRecordBatch()
	defer originalRecord.Release()
	for i := 0; i < t.N; i++ {
		originalRecord.Retain() // retain each time since FilterProfileData will release it
		recs, _, err := FilterProfileData(
			context.Background(),
			noop.NewTracerProvider().Tracer(""),
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_FrameFilter{
						FrameFilter: &pb.FrameFilter{
							Filter: &pb.FrameFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									Binary: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "test",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		for _, r := range recs {
			r.Release()
		}
	}
}

func TestFilterDataExclude(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	tracer := noop.NewTracerProvider().Tracer("")
	ctx := context.Background()

	// Create a profile with 3 samples:
	// Sample 1: function "foo" -> "bar" -> "baz"
	// Sample 2: function "main" -> "process" -> "handle"
	// Sample 3: function "foo" -> "qux"
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	// Sample 1: has "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.FunctionName.Append([]byte("foo"))
	w.FunctionSystemName.Append([]byte("foo"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(2)
	w.FunctionName.Append([]byte("bar"))
	w.FunctionSystemName.Append([]byte("bar"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(10)

	w.Locations.Append(true)
	w.Addresses.Append(0x1200)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(3)
	w.FunctionName.Append([]byte("baz"))
	w.FunctionSystemName.Append([]byte("baz"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(20)
	w.Value.Append(100)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	// Sample 2: no "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x2000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(4)
	w.FunctionName.Append([]byte("main"))
	w.FunctionSystemName.Append([]byte("main"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x2100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(5)
	w.FunctionName.Append([]byte("process"))
	w.FunctionSystemName.Append([]byte("process"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(10)

	w.Locations.Append(true)
	w.Addresses.Append(0x2200)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(6)
	w.FunctionName.Append([]byte("handle"))
	w.FunctionSystemName.Append([]byte("handle"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(20)
	w.Value.Append(200)
	w.Diff.Append(0)
	w.TimeNanos.Append(2)
	w.Period.Append(1)

	// Sample 3: has "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x3000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x4000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(7)
	w.FunctionName.Append([]byte("foo"))
	w.FunctionSystemName.Append([]byte("foo"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x3100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x4000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(8)
	w.FunctionName.Append([]byte("qux"))
	w.FunctionSystemName.Append([]byte("qux"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(30)
	w.Value.Append(300)
	w.Diff.Append(0)
	w.TimeNanos.Append(3)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	defer originalRecord.Release()

	t.Run("exclude=false filters to only samples with foo", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "foo",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should have 2 samples (sample 1 and 3 which have "foo")
		// The filtered value is the sum of values that were REMOVED, not kept
		totalRows := int64(0)
		totalValue := int64(0)
		for _, rec := range recs {
			totalRows += rec.NumRows()
			r, err := profile.NewRecordReader(rec)
			require.NoError(t, err)
			totalValue += math.Int64.Sum(r.Value)
		}
		require.Equal(t, int64(2), totalRows)
		require.Equal(t, int64(400), totalValue) // kept: 100 + 300
		require.Equal(t, int64(200), filtered)   // removed: 200 (sample 2)
	})

	t.Run("exclude=true filters out samples with foo", func(t *testing.T) {
		originalRecord.Retain()
		// Note: The new API doesn't support exclude functionality directly
		// This test now tests include behavior for non-foo functions
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_NotContains{
											NotContains: "foo",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should have 1 sample (sample 2 which doesn't have "foo")
		require.Len(t, recs, 1)
		require.Equal(t, int64(1), recs[0].NumRows())
		// The filtered value is the sum of values that were REMOVED
		require.Equal(t, int64(400), filtered) // removed: 100 + 300 (samples with foo)
	})

	t.Run("empty filter with exclude=true returns all samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{}, // no filters
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return all samples
		require.Greater(t, len(recs), 0, "Expected at least one record")
		if len(recs) > 0 {
			require.Equal(t, int64(3), recs[0].NumRows())
			// The filtered value is the sum of values that were REMOVED
			require.Equal(t, int64(0), filtered) // nothing removed with empty filter
		}
	})

	t.Run("function not found with exclude=true returns all samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{}, // no filters
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return all samples (nothing to exclude)
		totalRows := int64(0)
		for _, rec := range recs {
			totalRows += rec.NumRows()
		}
		require.Equal(t, int64(3), totalRows)
		// The filtered value is the sum of values that were REMOVED
		require.Equal(t, int64(0), filtered) // nothing removed
	})

	t.Run("function not found with exclude=false returns no samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, _, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "nonexistent",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return no samples (nothing matches)
		require.Len(t, recs, 0)
	})
}

func TestKwayMerge(t *testing.T) {
	arr1 := []string{"a", "c", "e"}
	arr2 := []string{"f", "i", "m", "o", "r"}

	merged := MergeTwoSortedSlices(arr1, arr2)

	require.Equal(t, []string{"a", "c", "e", "f", "i", "m", "o", "r"}, merged)
}

func TestSetArrayElementToNull(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	tests := []struct {
		name         string
		buildArray   func() arrow.Array
		indexToNull  int
		expectedNull int
	}{
		{
			name: "array with existing null bitmap",
			buildArray: func() arrow.Array {
				b := array.NewInt64Builder(mem)
				defer b.Release()
				b.AppendValues([]int64{1, 2, 3}, []bool{true, false, true})
				return b.NewArray()
			},
			indexToNull:  0,
			expectedNull: 2,
		},
		{
			name: "array with no null bitmap",
			buildArray: func() arrow.Array {
				b := array.NewInt64Builder(mem)
				defer b.Release()
				b.AppendValues([]int64{1, 2, 3}, nil)
				return b.NewArray()
			},
			indexToNull:  1,
			expectedNull: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arr := tc.buildArray()
			defer arr.Release()

			setArrayElementToNull(arr, tc.indexToNull, mem)

			require.True(t, arr.IsNull(tc.indexToNull))
			require.Equal(t, tc.expectedNull, arr.NullN())
		})
	}
}
