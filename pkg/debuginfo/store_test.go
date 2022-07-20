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
	"encoding/hex"
	"io"
	"io/ioutil"
	stdlog "log"
	"net"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/filesystem"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

func TestStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "parca-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cacheDir, err := ioutil.TempDir("", "parca-test-cache")
	require.NoError(t, err)
	defer os.RemoveAll(cacheDir)

	logger := log.NewNopLogger()
	cfg, err := yaml.Marshal(&client.BucketConfig{
		Type: client.FILESYSTEM,
		Config: filesystem.Config{
			Directory: dir,
		},
	})
	require.NoError(t, err)

	bucket, err := client.NewBucket(logger, cfg, prometheus.NewRegistry(), "parca/store")
	require.NoError(t, err)

	cache, err := NewCache(
		&CacheConfig{
			Type: FILESYSTEM,
			Config: &FilesystemCacheConfig{
				Directory: cacheDir,
			},
		},
	)
	require.NoError(t, err)

	s, err := NewStore(
		logger,
		cache.Directory,
		NewObjectStoreMetadata(logger, bucket),
		bucket,
		NopDebugInfodClient{},
	)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
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

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
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
	_, err = c.Upload(context.Background(), "abcd", "abcd", b)
	require.Error(t, err)

	nf, err := os.Open("testdata/validelf_nosections")
	require.NoError(t, err)

	_, err = c.Upload(context.Background(), hex.EncodeToString([]byte("nosection")), "abcd", nf)
	require.Error(t, err)

	wf, err := os.Open("testdata/validelf_withsections")
	require.NoError(t, err)

	size, err := c.Upload(context.Background(), hex.EncodeToString([]byte("section")), "abcd", wf)
	require.NoError(t, err)
	require.Equal(t, 7079, int(size))

	obj, err := s.bucket.Get(context.Background(), hex.EncodeToString([]byte("section"))+"/debuginfo")
	require.NoError(t, err)

	content, err := io.ReadAll(obj)
	require.NoError(t, err)
	require.Equal(t, 7079, len(content))
	require.Equal(t, []byte{0x7f, 'E', 'L', 'F'}, content[:4])

	ctx := context.Background()
	exists, err := c.Exists(context.Background(), hex.EncodeToString([]byte("section")), "abcd")
	require.NoError(t, err)
	require.True(t, exists)

	buf := bytes.NewBuffer(nil)
	downloader, err := c.Downloader(ctx, hex.EncodeToString([]byte("section")))
	require.NoError(t, err)
	require.Equal(t, debuginfopb.DownloadInfo_SOURCE_UPLOAD, downloader.Info().Source)

	written, err := downloader.Download(ctx, buf)
	require.NoError(t, err)
	require.Equal(t, 7079, written)
	require.Equal(t, 7079, buf.Len())
	require.NoError(t, downloader.Close())

	// Test only reading the download info.
	downloader, err = c.Downloader(ctx, hex.EncodeToString([]byte("section")))
	require.NoError(t, err)
	require.Equal(t, debuginfopb.DownloadInfo_SOURCE_UPLOAD, downloader.Info().Source)
	require.NoError(t, downloader.Close())
}
