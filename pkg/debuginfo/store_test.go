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

package debuginfo

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	stdlog "log"
	"net"
	"os"
	"testing"

	"github.com/go-kit/log"
	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"github.com/thanos-io/thanos/pkg/objstore/filesystem"
	"google.golang.org/grpc"
)

func TestStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "parca-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cacheDir, err := ioutil.TempDir("", "parca-test-cache")
	require.NoError(t, err)
	defer os.RemoveAll(cacheDir)

	logger := log.NewNopLogger()
	s, err := NewStore(
		logger,
		symbol.NewSymbolizer(logger),
		&Config{
			Bucket: &client.BucketConfig{
				Type: client.FILESYSTEM,
				Config: filesystem.Config{
					Directory: dir,
				},
			},
			Cache: &CacheConfig{
				Type: FILESYSTEM,
				Config: &FilesystemCacheConfig{
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

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()
	c := NewDebugInfoClient(conn)
	b := bytes.NewBuffer(nil)
	for i := 0; i < 1024; i++ {
		b.Write([]byte("a"))
	}
	for i := 0; i < 1024; i++ {
		b.Write([]byte("b"))
	}
	for i := 0; i < 1024; i++ {
		b.Write([]byte("c"))
	}
	size, err := c.Upload(context.Background(), "abcd", b)
	require.NoError(t, err)
	require.Equal(t, uint64(3072), size)

	obj, err := s.bucket.Get(context.Background(), "abcd/debuginfo")
	require.NoError(t, err)

	content, err := io.ReadAll(obj)
	require.NoError(t, err)
	require.Equal(t, 3072, len(content))

	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("a")[0], content[i])
	}
	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("b")[0], content[i+1024])
	}
	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("c")[0], content[i+2048])
	}

	exists, err := c.Exists(context.Background(), "abcd")
	require.NoError(t, err)
	require.True(t, exists)
}
