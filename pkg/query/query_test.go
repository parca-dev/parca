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

package query

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	profilestore "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage"
)

func Test_QueryRange_EmptyStore(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		nil,
	)

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(-5 * time.Minute)

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: 10,
	})
	require.NoError(t, err)
	require.Empty(t, resp.Series)
}

func Test_QueryRange_Valid(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		s,
	)

	app, err := db.Appender(ctx, labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
	})
	require.NoError(t, err)

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	// Overwrite the profile's timestamp to be within the last 5min.
	p.TimeNanos = time.Now().UnixNano()

	prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p, 0)
	require.NoError(t, err)
	err = app.AppendFlat(ctx, prof)
	require.NoError(t, err)

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(-5 * time.Minute)

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: 10,
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Series)
	require.Equal(t, 1, len(resp.Series))
	require.Equal(t, 1, len(resp.Series[0].Samples))
	require.Equal(t, &profilestore.LabelSet{
		Labels: []*profilestore.Label{
			{
				Name:  "__name__",
				Value: "allocs",
			},
		},
	}, resp.Series[0].Labelset)
	require.Equal(t, int64(310797348), resp.Series[0].Samples[0].Value)
}

func Test_QueryRange_Limited(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		s,
	)

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	numSeries := 10
	for i := 0; i < numSeries; i++ {
		app, err := db.Appender(ctx, labels.Labels{
			labels.Label{
				Name:  "__name__",
				Value: "allocs",
			},
			labels.Label{
				Name:  "meta",
				Value: fmt.Sprintf("series_%v", i),
			},
		})
		require.NoError(t, err)

		// Overwrite the profile's timestamp to be within the last 5min.
		p.TimeNanos = time.Now().UnixNano()

		prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p, 0)
		require.NoError(t, err)
		err = app.AppendFlat(ctx, prof)
		require.NoError(t, err)
	}

	// Query last 5 minutes
	end := time.Now()
	start := end.Add(-5 * time.Minute)

	limit := rand.Intn(numSeries)
	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
		Limit: uint32(limit),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Series)
	require.Equal(t, limit, len(resp.Series))
	for i := 0; i < limit; i++ {
		require.Equal(t, 1, len(resp.Series[i].Samples))
	}
}

func Test_QueryRange_Ranged(t *testing.T) {
	ctx := context.Background()
	reg := prometheus.NewRegistry()
	logger := log.NewNopLogger()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	db := storage.OpenDB(reg, tracer, nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)

	lset := labels.FromStrings("__name__", "allocs")

	app, err := db.Appender(ctx, lset)
	require.NoError(t, err)

	start := time.Now()

	for i := 0; i < 500; i++ {
		p.TimeNanos = start.Add(time.Duration(i) * time.Second).UnixNano()
		pprof, err := parcaprofile.FlatProfileFromPprof(ctx, logger, s, p, 0)
		require.NoError(t, err)
		err = app.AppendFlat(ctx, pprof)
		require.NoError(t, err)
	}

	q := New(logger, tracer, db, s)

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: "allocs",
		Start: timestamppb.New(start.Add(199 * time.Second)), // only 200 will be included
		End:   timestamppb.New(start.Add(1000 * time.Second)),
	})
	require.NoError(t, err)
	require.NotEmpty(t, resp.Series)
	require.NotEmpty(t, resp.Series[0].Samples)

	for i, s := range resp.Series[0].Samples {
		expectedTime := start.Add(time.Duration(200+i) * time.Second).Unix()
		require.Equal(t, expectedTime, s.Timestamp.GetSeconds(), "failed to find the correct timestamp in iteration: %d", i)
		require.Equal(t, int64(310797348), s.Value)
	}
}

func Test_QueryRange_InputValidation(t *testing.T) {
	ctx := context.Background()
	end := time.Now()
	start := end.Add(-5 * time.Minute)

	tests := map[string]struct {
		req *pb.QueryRangeRequest
	}{
		"Empty query": {
			req: &pb.QueryRangeRequest{
				Query: "",
				Start: timestamppb.New(start),
				End:   timestamppb.New(end),
			},
		},
		"Empty start": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: nil,
				End:   timestamppb.New(end),
			},
		},
		"Empty End": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: timestamppb.New(start),
				End:   nil,
			},
		},
		"End before start": {
			req: &pb.QueryRangeRequest{
				Query: "allocs",
				Start: timestamppb.New(end),
				End:   timestamppb.New(start),
			},
		},
	}

	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		nil,
		nil,
	)

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := q.QueryRange(ctx, test.req)
			require.Error(t, err)
			require.Empty(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func Test_Query_InputValidation(t *testing.T) {
	ctx := context.Background()

	invalidMode := pb.QueryRequest_Mode(1000)
	invalidReportType := pb.QueryRequest_ReportType(1000)

	tests := map[string]struct {
		req *pb.QueryRequest
	}{
		"Invalid mode": {
			req: &pb.QueryRequest{
				Mode:       invalidMode,
				Options:    &pb.QueryRequest_Single{Single: &pb.SingleProfile{}},
				ReportType: *pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED.Enum(),
			},
		},
		"Invalid report type": {
			req: &pb.QueryRequest{
				Mode:       *pb.QueryRequest_MODE_SINGLE_UNSPECIFIED.Enum(),
				Options:    &pb.QueryRequest_Single{Single: &pb.SingleProfile{}},
				ReportType: invalidReportType,
			},
		},
		"option doesn't match mode": {
			req: &pb.QueryRequest{
				Mode:       *pb.QueryRequest_MODE_SINGLE_UNSPECIFIED.Enum(),
				Options:    &pb.QueryRequest_Merge{Merge: &pb.MergeProfile{}},
				ReportType: *pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED.Enum(),
			},
		},
		"option not provided": {
			req: &pb.QueryRequest{
				Mode:       *pb.QueryRequest_MODE_SINGLE_UNSPECIFIED.Enum(),
				Options:    nil,
				ReportType: *pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED.Enum(),
			},
		},
	}

	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		nil,
		nil,
	)

	t.Parallel()
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := q.Query(ctx, test.req)
			require.Error(t, err)
			require.Empty(t, resp)
			require.Equal(t, codes.InvalidArgument, status.Code(err))
		})
	}
}

func Test_Query_Simple(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		s,
	)

	app, err := db.Appender(ctx, labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
	})
	require.NoError(t, err)

	f, err := os.Open("../storage/testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	t1 := (time.Now().UnixNano() / 1000000) * 1000000
	p1.TimeNanos = t1

	prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p1, 0)
	require.NoError(t, err)
	err = app.AppendFlat(ctx, prof)
	require.NoError(t, err)

	_, err = q.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_SINGLE_UNSPECIFIED,
		Options: &pb.QueryRequest_Single{
			Single: &pb.SingleProfile{
				Query: "allocs",
				Time:  timestamppb.New(time.Unix(0, t1)),
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED,
	})
	require.NoError(t, err)

	//out, err := proto.Marshal(resp)
	//require.NoError(t, err)
	//err = ioutil.WriteFile("../../ui/packages/shared/profile/src/testdata/fg-simple.pb", out, 0644)
	//require.NoError(t, err)
}

func Test_Query_Diff(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		s,
	)

	app, err := db.Appender(ctx, labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
	})
	require.NoError(t, err)

	f, err := os.Open("../storage/testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	f, err = os.Open("../storage/testdata/profile2.pb.gz")
	require.NoError(t, err)
	p2, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	t1 := (time.Now().UnixNano() / 1000000) * 1000000
	p1.TimeNanos = t1

	prof1, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p1, 0)
	require.NoError(t, err)
	err = app.AppendFlat(ctx, prof1)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 10)

	t2 := (time.Now().UnixNano() / 1000000) * 1000000
	p2.TimeNanos = t2

	prof2, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p2, 0)
	require.NoError(t, err)
	err = app.AppendFlat(ctx, prof2)
	require.NoError(t, err)

	_, err = q.Query(ctx, &pb.QueryRequest{
		Mode: pb.QueryRequest_MODE_DIFF,
		Options: &pb.QueryRequest_Diff{
			Diff: &pb.DiffProfile{
				A: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: "allocs",
							Time:  timestamppb.New(time.Unix(0, t1)),
						},
					},
				},
				B: &pb.ProfileDiffSelection{
					Mode: pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED,
					Options: &pb.ProfileDiffSelection_Single{
						Single: &pb.SingleProfile{
							Query: "allocs",
							Time:  timestamppb.New(time.Unix(0, t2)),
						},
					},
				},
			},
		},
		ReportType: pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED,
	})
	require.NoError(t, err)

	//	out, err := proto.Marshal(resp)
	//	require.NoError(t, err)
	//	err = ioutil.WriteFile("../../ui/packages/shared/profile/src/testdata/fg-diff.pb", out, 0644)
	//	require.NoError(t, err)
}

func Benchmark_Query_Merge(b *testing.B) {
	ctx := context.Background()
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	b.Cleanup(func() {
		s.Close()
	})
	f, err := os.Open("../storage/testdata/profile1.pb.gz")
	require.NoError(b, err)
	p1, err := profile.Parse(f)
	require.NoError(b, err)
	require.NoError(b, f.Close())

	p, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p1, 0)
	require.NoError(b, err)

	for k := 0.; k <= 10; k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				ctx := context.Background()
				db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
				q := New(
					log.NewNopLogger(),
					trace.NewNoopTracerProvider().Tracer(""),
					db,
					s,
				)

				app, err := db.Appender(ctx, labels.Labels{
					labels.Label{
						Name:  "__name__",
						Value: "allocs",
					},
				})

				require.NoError(b, err)
				for j := 0; j < n; j++ {
					p.Meta.Timestamp = int64(j + 1)
					err = app.AppendFlat(ctx, p)
					require.NoError(b, err)
				}
				b.StartTimer()

				_, err = q.Query(ctx, &pb.QueryRequest{
					Mode: pb.QueryRequest_MODE_MERGE,
					Options: &pb.QueryRequest_Merge{
						Merge: &pb.MergeProfile{
							Query: "allocs",
							Start: timestamppb.New(time.Unix(0, 0)),
							End:   timestamppb.New(time.Unix(0, int64(time.Millisecond)*int64(n+1))),
						},
					},
					ReportType: pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED,
				})
				require.NoError(b, err)
			}
		})
	}
}

func Test_Query_Merge(t *testing.T) {
	ctx := context.Background()

	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})

	f, err := os.Open("../storage/testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	p, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p1, 0)
	require.NoError(t, err)

	for k := 0.; k <= 10; k++ {
		db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
		q := New(
			log.NewNopLogger(),
			trace.NewNoopTracerProvider().Tracer(""),
			db,
			s,
		)

		app, err := db.Appender(ctx, labels.Labels{
			labels.Label{
				Name:  "__name__",
				Value: "allocs",
			},
		})

		require.NoError(t, err)
		n := int(math.Pow(2, k))
		t.Run(fmt.Sprintf("%d", n), func(t *testing.T) {
			for j := 0; j < n; j++ {
				p.Meta.Timestamp = int64(j + 1)
				err = app.AppendFlat(ctx, p)
				require.NoError(t, err)
			}

			_, err = q.Query(ctx, &pb.QueryRequest{
				Mode: pb.QueryRequest_MODE_MERGE,
				Options: &pb.QueryRequest_Merge{
					Merge: &pb.MergeProfile{
						Query: "allocs",
						Start: timestamppb.New(time.Unix(0, 0)),
						End:   timestamppb.New(time.Unix(0, int64(time.Millisecond)*int64(n+1))),
					},
				},
				ReportType: pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED,
			})
			require.NoError(t, err)
		})
	}
}

func Test_QueryRange_MultipleLabels_NoMatch(t *testing.T) {
	ctx := context.Background()
	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	s := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		s.Close()
	})
	q := New(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		s,
	)

	ls1 := labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
		labels.Label{
			Name:  "namespace",
			Value: "gitops-platform-metrics",
		},
		labels.Label{
			Name:  "container",
			Value: "c1",
		},
	}

	ls2 := labels.Labels{
		labels.Label{
			Name:  "__name__",
			Value: "allocs",
		},
		labels.Label{
			Name:  "namespace",
			Value: "gitops-platform-storage",
		},
		labels.Label{
			Name:  "container",
			Value: "c2",
		},
	}

	end := time.Now()
	start := end.Add(-5 * time.Minute)
	t1 := (end.UnixNano() / 1000000) * 1000000
	for _, ls := range []labels.Labels{ls1, ls2} {
		app, err := db.Appender(ctx, ls)
		require.NoError(t, err)

		f, err := os.Open("../storage/testdata/profile1.pb.gz")
		require.NoError(t, err)
		p1, err := profile.Parse(f)
		require.NoError(t, err)
		require.NoError(t, f.Close())

		p1.TimeNanos = t1

		prof, err := parcaprofile.FlatProfileFromPprof(ctx, log.NewNopLogger(), s, p1, 0)
		require.NoError(t, err)
		err = app.AppendFlat(ctx, prof)
		require.NoError(t, err)
	}

	resp, err := q.QueryRange(ctx, &pb.QueryRangeRequest{
		Query: `allocs{namespace="gitops-platform-metrics", container="c3"}`,
		Start: timestamppb.New(start),
		End:   timestamppb.New(end),
	})
	require.NoError(t, err)
	require.Equal(t, 0, len(resp.GetSeries()))
}
