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

package parca

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/cenkalti/backoff/v4"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/polarsignals/frostdb"
	"github.com/polarsignals/frostdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profilestore"
	queryservice "github.com/parca-dev/parca/pkg/query"
)

func getShareServerConn(t Testing) sharepb.ShareServiceClient {
	conn, err := grpc.Dial("api.pprof.me:443", grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	require.NoError(t, err)
	return sharepb.NewShareServiceClient(conn)
}

func benchmarkSetup(ctx context.Context, b *testing.B) (pb.ProfileStoreServiceClient, <-chan struct{}) {
	addr := "127.0.0.1:7077"

	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Run(ctx, logger, reg, &Flags{
			ConfigPath: "testdata/parca.yaml",
			Port:       addr,
			Metastore:  metaStoreBadger,
			Storage: FlagsStorage{
				GranuleSize:  8 * 1024,
				ActiveMemory: 512 * 1024 * 1024,
			},
			ProfileShareServer: "api.pprof.dummy:443",
			Hidden: FlagsHidden{
				DebugNormalizeAddresses: true,
			},
		}, "test-version")
		if !errors.Is(err, context.Canceled) {
			require.NoError(b, err)
		}
	}()

	var conn grpc.ClientConnInterface
	err := backoff.Retry(func() error {
		var err error
		conn, err = grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			// b.Logf("failed to connect to parca: %v", err)
			return err
		}

		client := pb.NewProfileStoreServiceClient(conn)
		_, err = client.WriteRaw(ctx, &pb.WriteRawRequest{})
		if err != nil {
			// b.Logf("failed to connect to write raw profile: %v", err)
			return err
		}
		return nil
	}, backoff.NewConstantBackOff(time.Second))
	require.NoError(b, err)

	client := pb.NewProfileStoreServiceClient(conn)
	return client, done
}

// go test -bench=Benchmark_WriteRaw --count=3 --benchtime=100x -benchmem -memprofile ./pkg/parca/writeraw-memory.pb.gz -cpuprofile ./pkg/parca/writeraw-cpu.pb.gz ./pkg/parca | tee ./pkg/parca/writeraw.txt

func Benchmark_WriteRaw(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, done := benchmarkSetup(ctx, b)

	f, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	// Benchmark section
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.WriteRaw(ctx, &pb.WriteRawRequest{
			Series: []*pb.RawProfileSeries{
				{
					Labels: &pb.LabelSet{
						Labels: []*pb.Label{
							{
								Name:  labels.MetricName,
								Value: "allocs",
							},
							{
								Name:  "test",
								Value: b.Name(),
							},
						},
					},
					Samples: []*pb.RawSample{
						{
							RawProfile: f,
						},
					},
				},
			},
		})
		require.NoError(b, err)
	}
	b.StopTimer()

	cancel()
	<-done
}

type Testing interface {
	require.TestingT
	Helper()
	Name() string
}

func MustReadAllGzip(t require.TestingT, filename string) []byte {
	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

func TestConsistency(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := frostdb.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		frostdb.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	metastore := metastore.NewInProcessClient(m)

	f, err := os.Open("../query/testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	pprofProf, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	compactedOriginalProfile := pprofProf.Compact()

	fileContent, err := os.ReadFile("../query/testdata/alloc_objects.pb.gz")
	require.NoError(t, err)

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
	api := queryservice.NewColumnQueryAPI(
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

	ts := timestamppb.New(timestamp.Time(1608199718549)) // time_nanos of the profile divided by 1e6
	res, err := api.Query(ctx, &querypb.QueryRequest{
		ReportType: querypb.QueryRequest_REPORT_TYPE_PPROF,
		Options: &querypb.QueryRequest_Single{
			Single: &querypb.SingleProfile{
				Query: `memory:alloc_objects:count:space:bytes`,
				Time:  ts,
			},
		},
	})
	require.NoError(t, err)

	resProf, err := profile.ParseData(res.Report.(*querypb.QueryResponse_Pprof).Pprof)
	require.NoError(t, err)

	require.Equal(t, len(compactedOriginalProfile.Sample), len(resProf.Sample))
}

func runCmd(t *testing.T, name string, arg ...string) {
	t.Helper()

	cmd := exec.Command(name, arg...)
	var outb bytes.Buffer
	cmd.Stdout = &outb
	cmd.Stderr = &outb
	require.NoError(t, cmd.Run(), outb.String())
}

func TestPGOE2e(t *testing.T) {
	runCmd(t, "go", "build", "-o", "testdata/pgotest", "./testdata/pgotest.go")
	runCmd(t, "./testdata/pgotest")

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := frostdb.New()
	require.NoError(t, err)
	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		frostdb.NewTableConfig(parcacol.SchemaDefinition()),
	)
	require.NoError(t, err)
	m := metastoretest.NewTestMetastore(
		t,
		logger,
		reg,
		tracer,
	)

	metastore := metastore.NewInProcessClient(m)

	fileContent, err := os.ReadFile("./testdata/pgotest.prof")
	require.NoError(t, err)

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
						Value: "process_cpu",
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
	api := queryservice.NewColumnQueryAPI(
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

	res, err := api.Query(ctx, &querypb.QueryRequest{
		Mode:       querypb.QueryRequest_MODE_MERGE,
		ReportType: querypb.QueryRequest_REPORT_TYPE_PPROF,
		Options: &querypb.QueryRequest_Merge{
			Merge: &querypb.MergeProfile{
				Query: `process_cpu:samples:count:cpu:nanoseconds:delta`,
				Start: timestamppb.New(timestamp.Time(math.MinInt64)),
				End:   timestamppb.New(timestamp.Time(math.MaxInt64)),
			},
		},
	})
	require.NoError(t, err)

	rawPprof := res.Report.(*querypb.QueryResponse_Pprof).Pprof

	require.NoError(t, os.WriteFile("./testdata/pgotest.res.prof", rawPprof, 0o644))
	runCmd(t, "go", "build", "-pgo", "./testdata/pgotest.res.prof", "-o", "./testdata/pgotest", "./testdata/pgotest.go")
}
