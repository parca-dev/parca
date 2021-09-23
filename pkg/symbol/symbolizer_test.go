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

package symbol

import (
	"bytes"
	"context"
	"io/ioutil"
	stdlog "log"
	"net"
	"os"
	"sort"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/storage"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"github.com/thanos-io/thanos/pkg/objstore/filesystem"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
)

type TestProfileMetaStore interface {
	TestLocationStore
	TestFunctionStore
	metastore.MappingStore
	Close() error
	Ping() error
}

type TestLocationStore interface {
	metastore.LocationStore
	GetLocations(ctx context.Context) ([]*profile.Location, error)
}

type TestFunctionStore interface {
	metastore.FunctionStore
	GetFunctions(ctx context.Context) ([]*profile.Function, error)
}

func TestSymbolizer(t *testing.T) {
	ctx := context.Background()

	cacheDir, err := ioutil.TempDir("", "parca-test-cache")
	require.NoError(t, err)
	defer os.RemoveAll(cacheDir)

	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	s, err := debuginfo.NewStore(logger, &debuginfo.Config{
		Bucket: &client.BucketConfig{
			Type: client.FILESYSTEM,
			Config: filesystem.Config{
				Directory: "testdata/",
			},
		},
		Cache: &debuginfo.CacheConfig{
			Type: debuginfo.FILESYSTEM,
			Config: &debuginfo.FilesystemCacheConfig{
				Directory: cacheDir,
			},
		},
	})
	require.NoError(t, err)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()
	debuginfopb.RegisterDebugInfoServiceServer(grpcServer, s)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			stdlog.Fatalf("failed to serve: %v", err)
		}
	}()

	var mStr TestProfileMetaStore
	mStr, err = metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
		"symbolizer",
	)
	t.Cleanup(func() {
		mStr.Close()
	})
	require.NoError(t, err)

	sym := NewSymbolizer(log.NewNopLogger(), mStr, s)
	m := &profile.Mapping{
		ID:      uint64(1),
		Start:   4194304,
		Limit:   4603904,
		BuildID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}
	_, err = mStr.CreateMapping(ctx, m)
	require.NoError(t, err)

	locs := []*profile.Location{{
		Mapping: m,
		Address: 0x463781,
	}}
	_, err = mStr.CreateLocation(ctx, locs[0])
	require.NoError(t, err)

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(symLocs))

	err = sym.symbolize(context.Background(), symLocs)
	require.NoError(t, err)

	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, len(allLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(functions))

	lines := allLocs[0].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(10), lines[0].Line)
	require.Equal(t, "main.main", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(27), lines[2].Line)
	require.Equal(t, "main.iterate", lines[2].Function.Name)
}

func TestRealSymbolizer(t *testing.T) {
	ctx := context.Background()

	cacheDir, err := ioutil.TempDir("", "parca-test-cache")
	require.NoError(t, err)
	defer os.RemoveAll(cacheDir)

	dbgStr, err := debuginfo.NewStore(log.NewNopLogger(), &debuginfo.Config{
		Bucket: &client.BucketConfig{
			Type: client.FILESYSTEM,
			Config: filesystem.Config{
				Directory: "testdata/",
			},
		},
		Cache: &debuginfo.CacheConfig{
			Type: debuginfo.FILESYSTEM,
			Config: &debuginfo.FilesystemCacheConfig{
				Directory: cacheDir,
			},
		},
	})
	require.NoError(t, err)

	var mStr TestProfileMetaStore
	mStr, err = metastore.NewInMemorySQLiteProfileMetaStore(
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		mStr.Close()
	})

	db := storage.OpenDB(prometheus.NewRegistry(), trace.NewNoopTracerProvider().Tracer(""), nil)
	pStr := profilestore.NewProfileStore(
		log.NewNopLogger(),
		trace.NewNoopTracerProvider().Tracer(""),
		db,
		mStr,
	)

	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()

	debuginfopb.RegisterDebugInfoServiceServer(grpcServer, dbgStr)
	profilestorepb.RegisterProfileStoreServiceServer(grpcServer, pStr)

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			stdlog.Fatalf("failed to serve: %v", err)
		}
	}()

	f, err := os.Open("testdata/profile.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, p.CheckValid())

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	buf := bytes.NewBuffer(nil)
	require.NoError(t, p.Write(buf))

	wc := profilestorepb.NewProfileStoreServiceClient(conn)
	_, err = wc.WriteRaw(context.Background(), &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{Labels: []*profilestorepb.Label{}},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: buf.Bytes(),
			}},
		}},
	})
	require.NoError(t, err)

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 32, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 11, len(symLocs))

	sym := NewSymbolizer(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs)) // p.Location to symbolize the profile.

	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 32, len(allLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 31, len(functions))

	lines := allLocs[4].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(10), lines[0].Line)
	require.Equal(t, "main.main", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(27), lines[2].Line)
	require.Equal(t, "main.iterate", lines[2].Function.Name)
}
