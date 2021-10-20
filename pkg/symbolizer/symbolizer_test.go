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

package symbolizer

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
	"github.com/parca-dev/parca/pkg/symbol"
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
	_, dbgStr, mStr := setup(t)

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	m := &profile.Mapping{
		ID:      uint64(1),
		Start:   4194304,
		Limit:   4603904,
		BuildID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}

	ctx := context.Background()

	_, err := mStr.CreateMapping(ctx, m)
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

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[0].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(7), lines[0].Line) // llvm-addr2line gives 10
	require.Equal(t, "main.main", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(27), lines[2].Line)
	require.Equal(t, "main.iterate", lines[2].Function.Name)
}

func TestRealSymbolizer(t *testing.T) {
	conn, dbgStr, mStr := setup(t)

	require.NoError(t, ingest(t, conn, "testdata/profile.pb.gz"))

	ctx := context.Background()

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 32, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 11, len(symLocs))

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 31, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[4].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(7), lines[0].Line) // llvm-addr2line gives 10
	require.Equal(t, "main.main", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(27), lines[2].Line)
	require.Equal(t, "main.iterate", lines[2].Function.Name)
}

func TestRealSymbolizerDwarfAndSymbols(t *testing.T) {
	conn, dbgStr, mStr := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/normal-cpu.stripped.pprof"))

	ctx := context.Background()

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 174, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 174, len(symLocs))

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 127, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[2].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", lines[0].Function.Filename)
	require.Equal(t, int64(5), lines[0].Line)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", lines[0].Function.Name)

	lines = allLocs[3].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(86), lines[0].Line)
	require.Equal(t, "main.busyCPU", lines[0].Function.Name)
}

func TestRealSymbolizerInliningDisabled(t *testing.T) {
	conn, dbgStr, mStr := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/inlining-disabled-cpu.stripped.pprof"))

	ctx := context.Background()

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 223, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 223, len(symLocs))

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 136, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[1].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", lines[0].Function.Filename)
	require.Equal(t, int64(5), lines[0].Line)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", lines[0].Function.Name)

	lines = allLocs[2].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(86), lines[0].Line)
	require.Equal(t, "main.busyCPU", lines[0].Function.Name)
}

func TestRealSymbolizerWithoutDWARF(t *testing.T) {
	// NOTICE: Uses custom Go symbolizer!

	conn, dbgStr, mStr := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/without-dwarf-cpu.stripped.pprof"))

	ctx := context.Background()

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 159, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 159, len(symLocs))

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 99, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[13].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", lines[0].Function.Filename)
	require.Equal(t, int64(13), lines[0].Line) // with DWARF 5
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", lines[0].Function.Name)

	lines = allLocs[14].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(89), lines[0].Line) // with DWARF 86
	require.Equal(t, "main.busyCPU", lines[0].Function.Name)
}

func TestRealSymbolizerEverythingStrippedInliningEnabled(t *testing.T) {
	// NOTICE: Uses custom Go symbolizer!

	conn, dbgStr, mStr := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/stripped-cpu.stripped.pprof"))

	ctx := context.Background()

	allLocs, err := mStr.GetLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 136, len(allLocs))

	symLocs, err := mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 136, len(symLocs))

	sym := New(log.NewNopLogger(), mStr, dbgStr)
	require.NoError(t, sym.symbolize(ctx, symLocs))

	symLocs, err = mStr.GetSymbolizableLocations(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	functions, err := mStr.GetFunctions(ctx)
	require.NoError(t, err)
	require.Equal(t, 80, len(functions))

	// Get updated locations.
	allLocs, err = mStr.GetLocations(ctx)
	require.NoError(t, err)

	lines := allLocs[1].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	// go -trimpath
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib/fib.go", lines[0].Function.Filename)
	require.Equal(t, int64(13), lines[0].Line) // with DWARF 5
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", lines[0].Function.Name)

	lines = allLocs[2].Line
	sort.SliceStable(lines, func(i, j int) bool {
		return lines[i].Line < lines[j].Line
	})
	require.Equal(t, 1, len(lines))
	// go -trimpath
	require.Equal(t, "./main.go", lines[0].Function.Filename)
	require.Equal(t, int64(89), lines[0].Line) // with DWARF 86
	require.Equal(t, "main.busyCPU", lines[0].Function.Name)
}

func ingest(t *testing.T, conn *grpc.ClientConn, path string) error {
	f, err := os.Open(path)
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, p.CheckValid())

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
	return err
}

func setup(t *testing.T) (*grpc.ClientConn, *debuginfo.Store, TestProfileMetaStore) {
	t.Helper()

	cacheDir, err := ioutil.TempDir("", "parca-test-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(cacheDir)
	})

	logger := log.NewNopLogger()
	sym, err := symbol.NewSymbolizer(logger)
	require.NoError(t, err)

	dbgStr, err := debuginfo.NewStore(
		logger,
		sym,
		&debuginfo.Config{
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
		t.Name(),
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
	t.Cleanup(func() {
		grpcServer.GracefulStop()
	})

	debuginfopb.RegisterDebugInfoServiceServer(grpcServer, dbgStr)
	profilestorepb.RegisterProfileStoreServiceServer(grpcServer, pStr)

	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			stdlog.Fatalf("failed to serve: %v", err)
		}
	}()

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return conn, dbgStr, mStr
}
