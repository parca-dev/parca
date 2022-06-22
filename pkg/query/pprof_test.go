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
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/parcacol"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlatPprof(t *testing.T) {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	p := &pprofpb.Profile{}
	require.NoError(t, p.UnmarshalVT(fileContent))

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		tracer,
	)
	metastore := metastore.NewInProcessClient(l)
	normalizer := parcacol.NewNormalizer(metastore)
	profiles, err := normalizer.NormalizePprof(ctx, "memory", p, false)
	require.NoError(t, err)

	symbolizedProfile, err := parcacol.SymbolizeNormalizedProfile(ctx, metastore, profiles[0])
	require.NoError(t, err)

	res, err := GenerateFlatPprof(ctx, symbolizedProfile)
	require.NoError(t, err)

	require.Equal(t, &profile.ValueType{Type: "space", Unit: "bytes"}, res.PeriodType)
	require.Equal(t, []*profile.ValueType{{Type: "alloc_objects", Unit: "count"}}, res.SampleType)
	require.Equal(t, time.Date(2020, 12, 17, 10, 8, 38, 549000000, time.UTC).UnixNano(), res.TimeNanos)
	require.Equal(t, int64(0), res.DurationNanos)
	require.Equal(t, int64(524288), res.Period)

	require.Equal(t, []*profile.Mapping{{
		ID:              1,
		Start:           4194304,
		Limit:           23252992,
		Offset:          0,
		File:            "/bin/operator",
		BuildID:         "",
		HasFunctions:    true,
		HasFilenames:    false,
		HasLineNumbers:  false,
		HasInlineFrames: false,
	}}, res.Mapping)

	require.Equal(t, 974, len(res.Function))
	require.Equal(t, 1886, len(res.Location))
	require.Equal(t, 4661, len(res.Sample))

	tmpfile, err := ioutil.TempFile("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	require.NoError(t, res.Write(tmpfile))
	require.NoError(t, tmpfile.Close())

	f, err := os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := profile.Parse(f)

	for _, s := range resProf.Sample {
		if s.Location == nil {
			fmt.Println("locations nil")
		}
	}

	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}

func TestGeneratePprofNilMapping(t *testing.T) {
	ctx := context.Background()
	var err error

	l := metastore.NewBadgerMetastore(
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)
	metastore := metastore.NewInProcessClient(l)

	fres, err := metastore.GetOrCreateFunctions(ctx, &pb.GetOrCreateFunctionsRequest{
		Functions: []*pb.Function{{
			Name: "1",
		}, {
			Name: "2",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fres.Functions))
	f1 := fres.Functions[0]
	f2 := fres.Functions[1]

	lres, err := metastore.GetOrCreateLocations(ctx, &pb.GetOrCreateLocationsRequest{
		Locations: []*pb.Location{{
			Lines: &pb.LocationLines{
				Entries: []*pb.Line{{
					FunctionId: f1.Id,
				}},
			},
		}, {
			Lines: &pb.LocationLines{
				Entries: []*pb.Line{{
					FunctionId: f2.Id,
				}},
			},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(lres.Locations))
	l1 := lres.Locations[0]
	l2 := lres.Locations[1]

	sres, err := metastore.GetOrCreateStacktraces(ctx, &pb.GetOrCreateStacktracesRequest{
		Stacktraces: []*pb.Stacktrace{{
			LocationIds: []string{l2.Id, l1.Id},
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(sres.Stacktraces))
	s := sres.Stacktraces[0]

	symbolizedProfile, err := parcacol.SymbolizeNormalizedProfile(ctx, metastore, &parcaprofile.NormalizedProfile{
		Samples: []*parcaprofile.NormalizedSample{{
			StacktraceID: s.Id,
			Value:        1,
		}},
	})
	require.NoError(t, err)

	res, err := GenerateFlatPprof(ctx, symbolizedProfile)
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	require.NoError(t, res.Write(tmpfile))
	require.NoError(t, tmpfile.Close())

	f, err := os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}
