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
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/apache/arrow/go/v7/arrow/memory"
	"github.com/cenkalti/backoff/v4"
	"github.com/fatih/semgroup"
	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/parcaparquet"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	queryservice "github.com/parca-dev/parca/pkg/query"
	columnstore "github.com/polarsignals/arcticdb"
	"github.com/polarsignals/arcticdb/query"
)

func benchmarkSetup(ctx context.Context, b *testing.B) (pb.ProfileStoreServiceClient, <-chan struct{}) {
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	done := make(chan struct{})
	go func() {
		defer close(done)
		err := Run(ctx, logger, reg, &Flags{ConfigPath: "testdata/parca.yaml", Port: ":9090"}, "test-version")
		if !errors.Is(err, context.Canceled) {
			require.NoError(b, err)
		}
	}()

	var conn grpc.ClientConnInterface
	err := backoff.Retry(func() error {
		var err error
		conn, err = grpc.Dial(":9090", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return err
		}

		client := pb.NewProfileStoreServiceClient(conn)
		_, err = client.WriteRaw(ctx, &pb.WriteRawRequest{})
		return err
	}, backoff.NewConstantBackOff(time.Second))
	require.NoError(b, err)

	client := pb.NewProfileStoreServiceClient(conn)
	return client, done
}

func Benchmark_Parca_WriteRaw(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	client, done := benchmarkSetup(ctx, b)

	f, err := ioutil.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	// Benchamrk section
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.WriteRaw(ctx, &pb.WriteRawRequest{
			Series: []*pb.RawProfileSeries{
				{
					Labels: &pb.LabelSet{
						Labels: []*pb.Label{
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

func TestReplay(t *testing.T) {
	// This test is only meant to be run manually to replay a debug log to try
	// to reproduce an issue. It requires debug log output from the Parca
	// server to be available at "../../tmp/".
	t.Skip()

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
				t.Logf("failed to parse label-set %s: %v", file.Name(), err)
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

	ctx := context.Background()
	logger := log.NewNopLogger()
	reg := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col := columnstore.New(reg)
	colDB := col.DB("parca")
	table := colDB.Table("stacktraces", columnstore.NewTableConfig(parcaparquet.Schema(), 8196), logger)
	m := metastore.NewBadgerMetastore(
		logger,
		reg,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	t.Cleanup(func() {
		m.Close()
	})

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
	s := semgroup.NewGroup(context.Background(), maxWorkers)
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
