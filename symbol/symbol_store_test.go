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
	"bytes"
	"context"
	"net"
	"testing"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/thanos/pkg/objstore"
	"google.golang.org/grpc"
)

func TestSymbolServer(t *testing.T) {
	bucket := objstore.NewInMemBucket()
	s := NewSymbolStore(log.NewNopLogger(), bucket, "/tmp")
	lis, err := net.Listen("tcp", ":0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	defer lis.Close()
	grpcServer := grpc.NewServer()
	storepb.RegisterSymbolStoreServer(grpcServer, s)
	go grpcServer.Serve(lis)

	conn, err := grpc.Dial(lis.Addr().String(), grpc.WithInsecure())
	require.NoError(t, err)
	defer conn.Close()
	c := NewSymbolStoreClient(storepb.NewSymbolStoreClient(conn))
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

	obj, ok := bucket.Objects()["abcd/debuginfo"]
	require.True(t, ok)
	require.Equal(t, 3072, len(obj))

	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("a")[0], obj[i])
	}
	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("b")[0], obj[i+1024])
	}
	for i := 0; i < 1024; i++ {
		require.Equal(t, []byte("c")[0], obj[i+2048])
	}

	exists, err := c.Exists(context.Background(), "abcd")
	require.NoError(t, err)
	require.True(t, exists)
}
