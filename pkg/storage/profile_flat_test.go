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

package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestProfileFlatNormalizer(t *testing.T) {
	f, err := os.Open("testdata/profile1.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"copyinstantprofiletree",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)

	p, err := FlatProfileFromPprof(context.Background(), log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)

	require.Equal(t, InstantProfileMeta{
		PeriodType: ValueType{Type: "cpu", Unit: "nanoseconds"},
		SampleType: ValueType{Type: "samples", Unit: "count"},
		Timestamp:  1626013307085,
		Duration:   30000181568,
		Period:     10 * time.Millisecond.Nanoseconds(),
	}, p.ProfileMeta())

	require.Len(t, p.Samples(), 32)
}
