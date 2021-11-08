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
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestGeneratePprof(t *testing.T) {
	ctx := context.Background()

	f, err := os.Open("testdata/alloc_objects.pb.gz")
	require.NoError(t, err)
	p1, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	l, err := metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"generatepprof",
	)
	t.Cleanup(func() {
		l.Close()
	})
	require.NoError(t, err)
	p, err := ProfileFromPprof(ctx, log.NewNopLogger(), l, p1, 0)
	require.NoError(t, err)
	res, err := GeneratePprof(ctx, l, p)
	require.NoError(t, err)

	tmpfile, err := ioutil.TempFile("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	require.NoError(t, res.Write(tmpfile))
	require.NoError(t, tmpfile.Close())

	f, err = os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}

func TestGeneratePprofNilMapping(t *testing.T) {
	ctx := context.Background()
	var err error

	l := metastore.NewBadgerMetastore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		metastore.NewRandomUUIDGenerator(),
	)
	f1 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "1",
		},
	}
	f1.ID, err = l.CreateFunction(ctx, f1)
	require.NoError(t, err)

	f2 := &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name: "2",
		},
	}
	f2.ID, err = l.CreateFunction(ctx, f2)
	require.NoError(t, err)

	l1 := &metastore.Location{
		Lines: []metastore.LocationLine{
			{
				Function: f1,
			},
		},
	}
	l1.ID, err = l.CreateLocation(ctx, l1)
	require.NoError(t, err)

	l2 := &metastore.Location{
		Lines: []metastore.LocationLine{
			{
				Function: f2,
			},
		},
	}
	l2.ID, err = l.CreateLocation(ctx, l2)
	require.NoError(t, err)

	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uuid.UUID{
		l2.ID,
		l1.ID,
	}))

	res, err := GeneratePprof(ctx, l, &Profile{Tree: pt})
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
