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

package parca

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow/go/v8/arrow/memory"
	"github.com/cenkalti/backoff/v4"
	"github.com/fatih/semgroup"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/polarsignals/arcticdb"
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	queryservice "github.com/parca-dev/parca/pkg/query"
)

func benchmarkSetup(ctx context.Context, b *testing.B) (pb.ProfileStoreServiceClient, <-chan struct{}) {
	addr := ":7077"

	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Run(ctx, logger, reg, &Flags{
			ConfigPath:          "testdata/parca.yaml",
			Port:                addr,
			Metastore:           metaStoreBadgerInMemory,
			StorageGranuleSize:  8 * 1024,
			StorageActiveMemory: 512 * 1024 * 1024,
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

	f, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
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

func replayDebugLog(ctx context.Context, t require.TestingT) (querypb.QueryServiceServer, *arcticdb.Table, *semgroup.Group, func()) {
	dir := "../../tmp/"
	files, err := ioutil.ReadDir(dir)
	require.NoError(t, err)

	type Sample struct {
		Timestamp int64
		Labels    labels.Labels
		FilePath  string
	}

	var samples []Sample
	for _, file := range files {
		if file.IsDir() {
			matchers, err := parser.ParseMetricSelector(file.Name())
			if err != nil {
				t.Errorf("failed to parse label-set %s: %v", file.Name(), err)
				continue
			}

			ls := make(labels.Labels, 0, len(matchers))
			for _, matcher := range matchers {
				ls = append(ls, labels.Label{
					Name:  matcher.Name,
					Value: matcher.Value,
				})
			}
			sort.Sort(ls)

			sampleFiles, err := ioutil.ReadDir(filepath.Join(dir, file.Name()))
			require.NoError(t, err)

			for _, sampleFile := range sampleFiles {
				if sampleFile.IsDir() {
					continue
				}
				if strings.HasSuffix(sampleFile.Name(), ".pb.gz") {
					samples = append(samples, Sample{
						Timestamp: sampleFile.ModTime().Unix(),
						Labels:    ls,
						FilePath:  filepath.Join(dir, file.Name(), sampleFile.Name()),
					})
				}
			}
		}
	}

	sort.Slice(samples, func(i, j int) bool {
		return samples[i].Timestamp < samples[j].Timestamp
	})

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

	api := queryservice.NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)

	const maxWorkers = 8
	s := semgroup.NewGroup(ctx, maxWorkers)
	for _, sample := range samples {
		s.Go(func() error {
			fileContent, err := ioutil.ReadFile(sample.FilePath)
			if err != nil {
				return err
			}
			p, err := profile.Parse(bytes.NewBuffer(fileContent))
			if err != nil {
				return err
			}
			profiles, err := parcaprofile.ProfilesFromPprof(ctx, logger, m, p, false)
			if err != nil {
				return err
			}

			for _, profile := range profiles {
				_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, sample.Labels, profile)
				if err != nil {
					return err
				}
			}

			return nil
		})
	}

	return api, table, s, func() {
		m.Close()
	}
}

func TestReplay(t *testing.T) {
	// This test is only meant to be run manually to replay a debug log to try
	// to reproduce an issue. It requires debug log output from the Parca
	// server to be available at "../../tmp/".
	t.Skip()

	ctx := context.Background()
	api, table, s, cleanup := replayDebugLog(ctx, t)
	t.Cleanup(cleanup)

	go func() {
		for {
			time.Sleep(time.Second)

			select {
			case <-ctx.Done():
				return
			default:
				_, err := api.Values(ctx, &querypb.ValuesRequest{
					LabelName: "__name__",
				})
				require.NoError(t, err)
			}
		}
	}()

	require.NoError(t, s.Wait())
	table.Sync()
}

func BenchmarkValuesAPI(b *testing.B) {
	ctx := context.Background()
	api, table, s, cleanup := replayDebugLog(ctx, b)
	b.Cleanup(cleanup)
	require.NoError(b, s.Wait())
	table.Sync()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := api.Values(ctx, &querypb.ValuesRequest{
			LabelName: "__name__",
		})
		require.NoError(b, err)
	}
}

func TestConsistency(t *testing.T) {
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

	f, err := os.Open("../query/testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	p1 = p1.Compact()

	p, err := parcaprofile.FromPprof(ctx, logger, m, p1, 0, false)
	require.NoError(t, err)

	_, err = parcacol.InsertProfileIntoTable(ctx, logger, table, labels.Labels{{
		Name:  "__name__",
		Value: "memory",
	}}, p)
	require.NoError(t, err)

	table.Sync()
	api := queryservice.NewColumnQueryAPI(
		logger,
		tracer,
		m,
		query.NewEngine(
			memory.DefaultAllocator,
			colDB.TableProvider(),
		),
		"stacktraces",
	)

	ts := timestamppb.New(timestamp.Time(p1.TimeNanos / time.Millisecond.Nanoseconds()))
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

	require.Equal(t, len(p1.Sample), len(resProf.Sample))
}
