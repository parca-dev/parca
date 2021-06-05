// Copyright 2021 The conprof Authors
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

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/go-kit/kit/log"
	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore/filesystem"
	"google.golang.org/grpc"
)

func TestSymbolizer(t *testing.T) {
	bucket, err := filesystem.NewBucket("testdata/")
	require.NoError(t, err)

	dir, err := os.MkdirTemp("", "symbolizer-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir) // clean up

	w := log.NewSyncWriter(os.Stderr)
	logger := log.NewLogfmtLogger(w)
	st := NewSymbolStore(logger, bucket, dir)
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()
	grpcServer := grpc.NewServer()
	storepb.RegisterSymbolStoreServer(grpcServer, st)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	c := storepb.NewSymbolizeClient(conn)
	s := NewSymbolizer(log.NewNopLogger(), c)
	m := &profile.Mapping{
		ID:      uint64(1),
		Start:   4194304,
		Limit:   4603904,
		BuildID: "2d6912fd3dd64542f6f6294f4bf9cb6c265b3085",
	}
	p := &profile.Profile{
		Location: []*profile.Location{{
			Mapping: m,
			Address: 0x463781,
		}},
		Mapping: []*profile.Mapping{m},
	}

	err = s.Symbolize(context.Background(), p)
	require.NoError(t, err)

	require.Equal(t, 1, len(p.Location))
	require.Equal(t, 1, len(p.Mapping))

	lines := p.Location[0].Line
	require.Equal(t, 3, len(lines))
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[0].Function.Filename)
	require.Equal(t, int64(27), lines[0].Line)
	require.Equal(t, "main.iterate", lines[0].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[1].Function.Filename)
	require.Equal(t, int64(23), lines[1].Line)
	require.Equal(t, "main.iteratePerTenant", lines[1].Function.Name)
	require.Equal(t, "/home/brancz/src/github.com/polarsignals/pprof-labels-example/main.go", lines[2].Function.Filename)
	require.Equal(t, int64(10), lines[2].Line)
	require.Equal(t, "main.main", lines[2].Function.Name)
}

func TestRealSymbolizer(t *testing.T) {
	bucket, err := filesystem.NewBucket("testdata/")
	require.NoError(t, err)

	dir, err := os.MkdirTemp("", "symbolizer-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir) // clean up

	st := NewSymbolStore(log.NewNopLogger(), bucket, dir)
	lis, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer lis.Close()
	grpcServer := grpc.NewServer()
	storepb.RegisterSymbolStoreServer(grpcServer, st)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()

	c := storepb.NewSymbolizeClient(conn)
	s := NewSymbolizer(log.NewNopLogger(), c)
	f, err := os.Open("testdata/profile.pb.gz")
	require.NoError(t, err)
	p, err := profile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, p.CheckValid())

	err = s.Symbolize(context.Background(), p)
	require.NoError(t, err)
	require.NoError(t, p.CheckValid())
}
