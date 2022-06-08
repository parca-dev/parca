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
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/go-kit/log"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/filesystem"
)

func TestMetadata(t *testing.T) {
	dir, err := ioutil.TempDir("", "parca-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cacheDir, err := ioutil.TempDir("", "parca-test-cache")
	require.NoError(t, err)
	defer os.RemoveAll(cacheDir)

	logger := log.NewNopLogger()
	sym, err := symbol.NewSymbolizer(logger)
	require.NoError(t, err)

	cfg := &Config{
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
	}

	store, err := NewStore(
		logger,
		sym,
		cfg,
		NopDebugInfodClient{},
	)
	require.NoError(t, err)

	// Test that the initial state should be empty.
	setMetadataLogger(logger)
	state, err := fetchMetadataState(context.Background(), store.bucket, "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, metadataStateEmpty, state)

	// Updating the state should be written to blob storage.
	err = metadataUpdate(context.Background(), store.bucket, "fake-build-id", metadataStateUploading)
	require.NoError(t, err)

	state, err = fetchMetadataState(context.Background(), store.bucket, "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, metadataStateUploading, state)
}
