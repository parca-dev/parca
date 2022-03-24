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

// This file contains some more specific benchmarks that one needs a benchmark dataset for and more.
// Therefore, it is usually excluded unless it is enabled through the specific build tag.

package storage_test

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage"
)

func loadProfiles(b *testing.B, amount int) ([]*profile.Profile, error) {
	b.Helper()

	filepathTimestamp, err := regexp.Compile(`(\d{13}).pb.gz$`)
	if err != nil {
		return nil, err
	}

	// Adjust this path to your local file system to where benchmark dataset is located.
	files, err := filepath.Glob("./benchmark/data/heap/*.pb.gz")
	if err != nil {
		return nil, err
	}

	if amount < 0 {
		amount = len(files)
	}

	profiles := make([]*profile.Profile, 0, amount)

	for _, file := range files[:amount] {
		submatch := filepathTimestamp.FindStringSubmatch(file)
		if len(submatch) != 2 {
			return nil, fmt.Errorf("expected 2 matches in timestamp regexp")
		}
		f, err := os.Open(file)
		if err != nil {
			return nil, err
		}
		p, err := profile.Parse(f)
		if err != nil {
			return nil, err
		}

		p.SampleType = []*profile.ValueType{p.SampleType[0]}

		profiles = append(profiles, p)
	}

	return profiles, nil
}

// go test -bench=BenchmarkAppends --count=5 --benchtime=2500x -benchmem -memprofile ./pkg/storage/benchmark/db-appends-memory.pb.gz -cpuprofile ./pkg/storage/benchmark/db-appends-cpu.pb.gz ./pkg/storage | tee ./pkg/storage/benchmark/db-appends.txt

func BenchmarkAppends(b *testing.B) {
	ctx := context.Background()

	profiles, err := loadProfiles(b, b.N)
	require.NoError(b, err)

	logger := log.NewNopLogger()

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		otel.Tracer("foo"),
		metastore.NewRandomUUIDGenerator(),
	)
	require.NoError(b, err)
	b.Cleanup(func() {
		l.Close()
	})

	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)

	lset := labels.FromStrings("job", "parca", "n", strconv.Itoa(b.N))
	app, err := db.Appender(context.Background(), lset)
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	//for i := 0; i < b.N; i++ {
	//	p := profiles[i%len(profiles)]
	//	pprof, err := storage.FromPprof(ctx, logger, l, p, 0)
	//	require.NoError(b, err)
	//	err = app.Append(ctx, pprof)
	//	require.NoError(b, err)
	//}

	for i := 0; i < b.N; i++ {
		p := profiles[i%len(profiles)]
		pprof, err := parcaprofile.FromPprof(ctx, logger, l, p, 0, false)
		require.NoError(b, err)
		err = app.AppendFlat(ctx, pprof)
		require.NoError(b, err)
	}

	b.StopTimer()
}

// go test -bench=BenchmarkIterator --count=5 --benchtime=2500x -benchmem -memprofile ./pkg/storage/benchmark/db-iterator-memory.pb.gz -cpuprofile ./pkg/storage/benchmark/db-iterator-cpu.pb.gz ./pkg/storage | tee ./pkg/storage/benchmark/db-iterator.txt

func BenchmarkIterator(b *testing.B) {
	ctx := context.Background()

	profiles, err := loadProfiles(b, b.N)
	require.NoError(b, err)

	registry := prometheus.NewRegistry()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	logger := log.NewNopLogger()

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		registry,
		tracer,
		metastore.NewRandomUUIDGenerator(),
	)
	require.NoError(b, err)
	b.Cleanup(func() {
		l.Close()
	})

	db := storage.OpenDB(registry, tracer, nil)

	lset := labels.FromStrings("job", "parca", "n", strconv.Itoa(b.N))
	app, err := db.Appender(context.Background(), lset)
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		pprof := profiles[i%len(profiles)]
		p, err := parcaprofile.FromPprof(ctx, logger, l, pprof, 0, false)
		require.NoError(b, err)
		err = app.AppendFlat(ctx, p)
		require.NoError(b, err)
	}

	// 1614253659535 - 130th sample
	// 1614255868920 - 400th sample
	// 1614262838920 - 1250th sample

	var q storage.Querier
	if b.N == 1 {
		q = db.Querier(context.Background(), math.MinInt64, math.MaxInt64)
	} else {
		q = db.Querier(context.Background(), 1614253659535, 1614262838920)
	}

	b.ReportAllocs()
	b.ResetTimer()

	set := q.Select(nil, labels.MustNewMatcher(labels.MatchEqual, "job", "parca"))
	seen := 0
	for set.Next() {
		s := set.At()
		it := s.Iterator()
		for it.Next() {
			seen++
		}
	}

	if b.N == 1 {
		require.Equal(b, 1, seen)
	} else {
		require.Equal(b, 1121, seen) // 1250 - 130 and then +1 for the next value we'd see.
	}
}
