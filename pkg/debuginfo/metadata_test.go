// Copyright 2022-2023 The Parca Authors
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
	"os"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/filesystem"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"
	"gopkg.in/yaml.v3"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

func TestMetadata(t *testing.T) {
	ctx := context.Background()
	tracer := trace.NewNoopTracerProvider().Tracer("")

	dir, err := os.MkdirTemp("", "parca-test")
	require.NoError(t, err)
	defer os.RemoveAll(dir)

	cacheDir, err := os.MkdirTemp("", "parca-test-cache")
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

	store, err := NewStore(
		tracer,
		logger,
		NewObjectStoreMetadata(logger, bucket),
		bucket,
		NopDebuginfodClient{},
		SignedUpload{
			Enabled: false,
		},
		time.Minute*15,
		1024*1024*1024,
	)
	require.NoError(t, err)

	// Test that the initial state should be empty.
	_, err = store.metadata.Fetch(ctx, "fake-build-id")
	require.ErrorIs(t, err, ErrMetadataNotFound)

	// Updating the state should be written to blob storage.
	time := time.Now()
	err = store.metadata.MarkAsUploading(ctx, "fake-build-id", "fake-upload-id", "fake-hash", timestamppb.New(time))
	require.NoError(t, err)

	dbginfo, err := store.metadata.Fetch(ctx, "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, "fake-build-id", dbginfo.BuildId)
	require.Equal(t, "fake-upload-id", dbginfo.Upload.Id)
	require.Equal(t, debuginfopb.DebuginfoUpload_STATE_UPLOADING, dbginfo.Upload.State)
}
