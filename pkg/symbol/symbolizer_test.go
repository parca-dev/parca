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
	"context"
	"net"
	"os"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"github.com/thanos-io/thanos/pkg/objstore/filesystem"
	"google.golang.org/grpc"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
)

func TestSymbolizer(t *testing.T) {
	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	dbgStr, err := debuginfo.NewStore(logger, &debuginfo.Config{
		Bucket: &client.BucketConfig{
			Type: client.FILESYSTEM,
			Config: filesystem.Config{
				Directory: "testdata/",
			},
		},
	})
	require.NoError(t, err)

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	grpcServer := grpc.NewServer()
	debuginfopb.RegisterDebugInfoServer(grpcServer, dbgStr)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	mStr, err := metastore.NewInMemoryProfileMetaStore("symbolizer")
	t.Cleanup(func() {
		mStr.Close()
	})
	require.NoError(t, err)

	sym := NewSymbolizer(log.NewNopLogger(), mStr, dbgStr)
	m := &profile.Mapping{
		ID:      uint64(1),
		Start:   4194304,
		Limit:   4603904,
		BuildID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}
	mStr.CreateMapping(m)
	locs := []*profile.Location{{
		Mapping: m,
		Address: 0x463781,
	}}
	mStr.CreateLocation(locs[0])

	err = sym.symbolize(context.Background(), locs)
	require.NoError(t, err)

	symLocs, err := mStr.GetUnsymbolizedLocations()
	require.NoError(t, err)
	require.Equal(t, 0, len(symLocs))

	allLocs, err := mStr.GetLocations()
	require.NoError(t, err)
	require.Equal(t, 1, len(allLocs))

	// TODO(kakkoyun): Ingest profile. Symbolize.
	// Generate new Pprof check if it is valid. Make them work.

	//functions, err := mStr.GetFunctions()
	//require.NoError(t, err)
	//require.Equal(t, 3, len(functions))
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[0].Filename)
	//require.Equal(t, int64(27), lines[0].Line)
	//require.Equal(t, "main.iterate", functions[0].Name)
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[1].Filename)
	//require.Equal(t, int64(23), lines[1].Line)
	//require.Equal(t, "main.iteratePerTenant", functions[1].Name)
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[2].Filename)
	//require.Equal(t, int64(10), lines[2].Line)
	//require.Equal(t, "main.main", functions[2].Name)
}

func TestRealSymbolizer(t *testing.T) {
	dbgStr, err := debuginfo.NewStore(log.NewNopLogger(), &debuginfo.Config{
		Bucket: &client.BucketConfig{
			Type: client.FILESYSTEM,
			Config: filesystem.Config{
				Directory: "testdata/",
			},
		},
	})
	require.NoError(t, err)

	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()

	grpcServer := grpc.NewServer()
	debuginfopb.RegisterDebugInfoServer(grpcServer, dbgStr)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	//mStr, err := metastore.NewInMemoryProfileMetaStore("realsymbolizer")
	//t.Cleanup(func() {
	//	mStr.Close()
	//})
	require.NoError(t, err)
	//sym := NewSymbolizer(log.NewNopLogger(), mStr, dbgStr)
	f, err := os.Open("testdata/profile.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, p.CheckValid())

	//// TODO(kakkoyun):  Generate new Pprof check if it is valid.
	//
	////storage.ProfilesFromPprof(mStr, p)
	//
	//err = sym.symbolize(context.Background(), p.Location)
	//require.NoError(t, err)
	//require.NoError(t, p.CheckValid())
	//
	//functions, err := mStr.GetFunctions()
	//require.NoError(t, err)
	//require.Equal(t, 3, len(functions))
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[0].Filename)
	////require.Equal(t, int64(27), lines[0].Line)
	//require.Equal(t, "main.iterate", functions[0].Name)
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[1].Filename)
	////require.Equal(t, int64(23), lines[1].Line)
	//require.Equal(t, "main.iteratePerTenant", functions[1].Name)
	//require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", functions[2].Filename)
	////require.Equal(t, int64(10), lines[2].Line)
	//require.Equal(t, "main.main", functions[2].Name)
}
