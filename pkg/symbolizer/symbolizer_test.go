// Copyright 2022 The Parca Authors
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
	"context"
	"io"
	stdlog "log"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/polarsignals/frostdb"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/filesystem"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/metastoretest"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profilestore"
	"github.com/parca-dev/parca/pkg/symbol"
)

func TestSymbolizer(t *testing.T) {
	var err error

	_, metastore, sym := setup(t)

	ctx := context.Background()

	mres, err := metastore.GetOrCreateMappings(ctx, &pb.GetOrCreateMappingsRequest{
		Mappings: []*pb.Mapping{{
			Start:   4194304,
			Limit:   4603904,
			BuildId: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
		}},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(mres.Mappings))
	m := mres.Mappings[0]

	clres, err := metastore.GetOrCreateLocations(ctx, &pb.GetOrCreateLocationsRequest{
		Locations: []*pb.Location{{
			MappingId: m.Id,
			Address:   0x463781,
		}},
	})
	require.NoError(t, err)

	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{clres.Locations[0].Id},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres.Locations))

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 1, len(ures.Locations))

	err = sym.Symbolize(ctx, ures.Locations)
	require.NoError(t, err)

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	// Get updated locations.
	lres, err = metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{lres.Locations[0].Id},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres.Locations))
	require.Equal(t, 3, len(lres.Locations[0].Lines))

	functionIds := []string{}
	for _, location := range lres.Locations {
		for _, line := range location.Lines {
			functionIds = append(functionIds, line.FunctionId)
		}
	}

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIds,
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(fres.Functions))

	require.Equal(t, fres.Functions[0].Id, lres.Locations[0].Lines[0].FunctionId)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[0].Filename)
	require.Equal(t, "main.iterate", fres.Functions[0].Name)
	require.Equal(t, int64(27), lres.Locations[0].Lines[0].Line)

	require.Equal(t, fres.Functions[1].Id, lres.Locations[0].Lines[1].FunctionId)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.iteratePerTenant", fres.Functions[1].Name)
	require.Equal(t, int64(23), lres.Locations[0].Lines[1].Line)

	require.Equal(t, fres.Functions[2].Id, lres.Locations[0].Lines[2].FunctionId)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[2].Filename)
	require.Equal(t, "main.main", fres.Functions[2].Name)
	require.Equal(t, int64(10), lres.Locations[0].Lines[2].Line)
}

func findIndexWithAddress(locs []*pb.Location, address uint64) int {
	for i, l := range locs {
		if l.Address == address {
			return i
		}
	}
	return -1
}

func TestRealSymbolizer(t *testing.T) {
	conn, metastore, sym := setup(t)

	require.NoError(t, ingest(t, conn, "testdata/profile.pb.gz"))

	ctx := context.Background()

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 11, len(ures.Locations))
	id := ures.Locations[findIndexWithAddress(ures.Locations, 0x463784)].Id

	require.NoError(t, sym.Symbolize(ctx, ures.Locations))

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	// Get updated locations.
	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{id},
	})
	require.NoError(t, err)
	require.Equal(t, 1, len(lres.Locations))
	require.Equal(t, 3, len(lres.Locations[0].Lines))

	functionIds := []string{}
	for _, location := range lres.Locations {
		for _, line := range location.Lines {
			functionIds = append(functionIds, line.FunctionId)
		}
	}

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIds,
	})
	require.NoError(t, err)
	require.Equal(t, 3, len(fres.Functions))

	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[0].Filename)
	require.Equal(t, "main.iterate", fres.Functions[0].Name)
	require.Equal(t, int64(27), lres.Locations[0].Lines[0].Line)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.iteratePerTenant", fres.Functions[1].Name)
	require.Equal(t, int64(23), lres.Locations[0].Lines[1].Line)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", fres.Functions[2].Filename)
	require.Equal(t, "main.main", fres.Functions[2].Name)
	require.Equal(t, int64(10), lres.Locations[0].Lines[2].Line)
}

func TestRealSymbolizerDwarfAndSymbols(t *testing.T) {
	conn, metastore, sym := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/normal-cpu.stripped.pprof"))

	ctx := context.Background()

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 174, len(ures.Locations))
	id1 := ures.Locations[findIndexWithAddress(ures.Locations, 0x6491de)].Id
	id2 := ures.Locations[findIndexWithAddress(ures.Locations, 0x649e46)].Id

	require.NoError(t, sym.Symbolize(ctx, ures.Locations))

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{id1, id2},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(lres.Locations))
	require.Equal(t, 1, len(lres.Locations[0].Lines))
	require.Equal(t, 1, len(lres.Locations[1].Lines))

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: []string{lres.Locations[0].Lines[0].FunctionId, lres.Locations[1].Lines[0].FunctionId},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fres.Functions))

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", fres.Functions[0].Filename)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", fres.Functions[0].Name)
	require.Equal(t, int64(13), lres.Locations[0].Lines[0].Line)

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.busyCPU", fres.Functions[1].Name)
	require.Equal(t, int64(89), lres.Locations[1].Lines[0].Line)
}

func TestRealSymbolizerInliningDisabled(t *testing.T) {
	conn, metastore, sym := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/inlining-disabled-cpu.stripped.pprof"))

	ctx := context.Background()

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 223, len(ures.Locations))
	id1 := ures.Locations[findIndexWithAddress(ures.Locations, 0x77157c)].Id
	id2 := ures.Locations[findIndexWithAddress(ures.Locations, 0x77265c)].Id

	require.NoError(t, sym.Symbolize(ctx, ures.Locations))

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{id1, id2},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(lres.Locations))
	require.Equal(t, 1, len(lres.Locations[0].Lines))
	require.Equal(t, 1, len(lres.Locations[1].Lines))

	functionIds := []string{}
	for _, location := range lres.Locations {
		for _, line := range location.Lines {
			functionIds = append(functionIds, line.FunctionId)
		}
	}

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIds,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fres.Functions))

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", fres.Functions[0].Filename)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", fres.Functions[0].Name)
	require.Equal(t, int64(13), lres.Locations[0].Lines[0].Line)

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.busyCPU", fres.Functions[1].Name)
	require.Equal(t, int64(89), lres.Locations[1].Lines[0].Line)
}

func TestRealSymbolizerWithoutDWARF(t *testing.T) {
	// NOTICE: Uses custom Go symbolizer!

	conn, metastore, sym := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/without-dwarf-cpu.stripped.pprof"))

	ctx := context.Background()

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 159, len(ures.Locations))
	id1 := ures.Locations[findIndexWithAddress(ures.Locations, 0x6491de)].Id
	id2 := ures.Locations[findIndexWithAddress(ures.Locations, 0x649e46)].Id

	require.NoError(t, sym.Symbolize(ctx, ures.Locations))

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	// Get updated locations.
	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{id1, id2},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(lres.Locations))
	require.Equal(t, 1, len(lres.Locations[0].Lines))
	require.Equal(t, 1, len(lres.Locations[1].Lines))

	functionIds := []string{}
	for _, location := range lres.Locations {
		for _, line := range location.Lines {
			functionIds = append(functionIds, line.FunctionId)
		}
	}

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIds,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fres.Functions))

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/fib/fib.go", fres.Functions[0].Filename)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", fres.Functions[0].Name)
	require.Equal(t, int64(13), lres.Locations[0].Lines[0].Line) // with DWARF 5

	require.Equal(t, "/home/kakkoyun/Workspace/PolarSignals/pprof-example-app-go/main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.busyCPU", fres.Functions[1].Name)
	require.Equal(t, int64(89), lres.Locations[1].Lines[0].Line) // with DWARF 86
}

func TestRealSymbolizerEverythingStrippedInliningEnabled(t *testing.T) {
	// NOTICE: Uses custom Go symbolizer!

	conn, metastore, sym := setup(t)

	// Generated from https://github.com/polarsignals/pprof-example-app-go
	require.NoError(t, ingest(t, conn, "testdata/stripped-cpu.stripped.pprof"))

	ctx := context.Background()

	ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 136, len(ures.Locations))
	id1 := ures.Locations[findIndexWithAddress(ures.Locations, 0x6491de)].Id
	id2 := ures.Locations[findIndexWithAddress(ures.Locations, 0x649e46)].Id

	require.NoError(t, sym.Symbolize(ctx, ures.Locations))

	ures, err = metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
	require.NoError(t, err)
	require.Equal(t, 0, len(ures.Locations))

	// Get updated locations.
	lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
		LocationIds: []string{id1, id2},
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(lres.Locations))
	require.Equal(t, 1, len(lres.Locations[0].Lines))
	require.Equal(t, 1, len(lres.Locations[1].Lines))

	functionIds := []string{}
	for _, location := range lres.Locations {
		for _, line := range location.Lines {
			functionIds = append(functionIds, line.FunctionId)
		}
	}

	fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIds,
	})
	require.NoError(t, err)
	require.Equal(t, 2, len(fres.Functions))

	// go -trimpath
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib/fib.go", fres.Functions[0].Filename)
	require.Equal(t, "github.com/polarsignals/pprof-example-app-go/fib.Fibonacci", fres.Functions[0].Name)
	require.Equal(t, int64(13), lres.Locations[0].Lines[0].Line) // with DWARF 5

	// go -trimpath
	require.Equal(t, "./main.go", fres.Functions[1].Filename)
	require.Equal(t, "main.busyCPU", fres.Functions[1].Name)
	require.Equal(t, int64(89), lres.Locations[1].Lines[0].Line) // with DWARF 86
}

func TestDwarfEqualAddr2line(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("do not run this test on non-linux system")
		return
	}

	dirname := "./testdata"
	var pprofFiles []string

	// without dwarf, addr2line could not work
	files := []string{"profile.pb.gz"}
	for _, file := range files {
		pprofFiles = append(pprofFiles, filepath.Join(dirname, file))
	}

	for _, pprofFile := range pprofFiles {
		// use metastore to tran unsymbolized location
		conn, metastore, sym := setup(t)
		require.NoError(t, ingest(t, conn, pprofFile))
		ctx := context.Background()
		ures, err := metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
		require.NoError(t, err)
		require.NoError(t, sym.Symbolize(ctx, ures.Locations))

		ids := make([]string, 0, len(ures.Locations))
		for _, l := range ures.Locations {
			ids = append(ids, l.Id)
		}
		lres, err := metastore.Locations(ctx, &pb.LocationsRequest{
			LocationIds: ids,
		})
		require.NoError(t, err)

		// use addr2liner to tran all pprof location to frames
		data, err := os.ReadFile(pprofFile)
		require.NoError(t, err)
		p, err := profile.ParseData(data)
		require.NoError(t, err)

		// build addr2liner
		// do not use addr2liner to tran system address
		goMapping := p.Mapping[0]
		executableFile := filepath.Join(dirname, goMapping.BuildID, "debuginfo")
		addr2liner, err := newAddr2Liner("addr2line", executableFile, 0)
		require.NoError(t, err)

		addr2lineFrames := make(map[uint64][]Frame, len(p.Location))
		for _, l := range p.Location {
			if l.Mapping != goMapping {
				continue
			}
			frames, err := addr2liner.addrInfo(l.Address)
			require.NoErrorf(t, err, "error using addr2line parse address:%x for file:%s", l.Address, pprofFile)
			addr2lineFrames[l.Address] = frames
		}

		// make sure addr2line output function order, name, line number
		// equal with metastore
		for _, l := range lres.Locations {
			var functionIds []string
			for _, line := range l.Lines {
				functionIds = append(functionIds, line.FunctionId)
			}
			fres, err := metastore.Functions(ctx, &pb.FunctionsRequest{
				FunctionIds: functionIds,
			})
			require.NoError(t, err)

			if expectedFrames, ok := addr2lineFrames[l.Address]; ok {
				require.Equalf(t, len(expectedFrames), len(fres.Functions),
					"frames not equal length, address:%x for file:%s", l.Address, pprofFile)

				for i, expectedFrame := range expectedFrames {
					gotFrame := Frame{
						Func: fres.Functions[i].Name,
						File: fres.Functions[i].Filename,
						Line: int(l.Lines[i].Line),
					}
					require.Equal(t, expectedFrame, gotFrame,
						"metastore result not equal with addr2line, address:%x for file:%s", l.Address, pprofFile)
				}
			} else {
				t.Errorf("not found address in addr2line:%d", l.Address)
			}
		}
	}
}

func mustReadAll(t require.TestingT, filename string) []byte {
	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	content, err := io.ReadAll(f)
	require.NoError(t, err)
	return content
}

func ingest(t *testing.T, conn *grpc.ClientConn, path string) error {
	fileContent := mustReadAll(t, path)
	wc := profilestorepb.NewProfileStoreServiceClient(conn)
	_, err := wc.WriteRaw(context.Background(), &profilestorepb.WriteRawRequest{
		Series: []*profilestorepb.RawProfileSeries{{
			Labels: &profilestorepb.LabelSet{Labels: []*profilestorepb.Label{{Name: "__name__", Value: "process_cpu"}}},
			Samples: []*profilestorepb.RawSample{{
				RawProfile: fileContent,
			}},
		}},
	})
	return err
}

func setup(t *testing.T) (*grpc.ClientConn, pb.MetastoreServiceClient, *Symbolizer) {
	t.Helper()

	logger := log.NewNopLogger()
	tracer := trace.NewNoopTracerProvider().Tracer("")
	col, err := frostdb.New()
	require.NoError(t, err)

	colDB, err := col.DB(context.Background(), "parca")
	require.NoError(t, err)

	schema, err := parcacol.Schema()
	require.NoError(t, err)

	table, err := colDB.Table(
		"stacktraces",
		frostdb.NewTableConfig(schema),
	)
	require.NoError(t, err)

	debugInfoCacheDir, err := os.MkdirTemp("", "parca-debuginfo-test-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(debugInfoCacheDir)
	})

	symbolizerCacheDir, err := os.MkdirTemp("", "parca-symbolizer-test-cache-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(symbolizerCacheDir)
	})

	sym, err := symbol.NewSymbolizer(logger)
	require.NoError(t, err)

	cfg, err := yaml.Marshal(&client.BucketConfig{
		Type: client.FILESYSTEM,
		Config: filesystem.Config{
			Directory: "testdata/",
		},
	})
	require.NoError(t, err)

	bucket, err := client.NewBucket(logger, cfg, prometheus.NewRegistry(), "parca/store")
	require.NoError(t, err)

	metadata := debuginfo.NewObjectStoreMetadata(logger, bucket)
	dbgStr, err := debuginfo.NewStore(
		tracer,
		logger,
		debugInfoCacheDir,
		metadata,
		bucket,
		debuginfo.NopDebugInfodClient{},
	)
	require.NoError(t, err)

	mStr := metastoretest.NewTestMetastore(
		t,
		log.NewNopLogger(),
		prometheus.NewRegistry(),
		trace.NewNoopTracerProvider().Tracer(""),
	)

	metastore := metastore.NewInProcessClient(mStr)

	pStr := profilestore.NewProfileColumnStore(
		logger,
		tracer,
		metastore,
		table,
		schema,
		false,
	)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
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

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	t.Cleanup(func() {
		conn.Close()
	})

	return conn, metastore, New(
		logger,
		prometheus.NewRegistry(),
		metastore,
		dbgStr,
		sym,
		symbolizerCacheDir,
		symbolizerCacheDir,
		0,
	)
}
