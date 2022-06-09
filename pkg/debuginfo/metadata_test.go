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
	"encoding/json"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/filesystem"

	"github.com/parca-dev/parca/pkg/symbol"
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
	state, err := store.metadataManager.fetch(context.Background(), "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, metadataStateEmpty, state)

	// Updating the state should be written to blob storage.
	err = store.metadataManager.update(context.Background(), "fake-build-id", metadataStateUploading)
	require.NoError(t, err)

	state, err = store.metadataManager.fetch(context.Background(), "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, metadataStateUploading, state)
}

func TestMetadata_MarshalJSON(t *testing.T) {
	tests := []struct {
		m       metadata
		want    string
		wantErr bool
	}{
		{
			m:    metadata{State: metadataStateEmpty},
			want: `{"state":"METADATA_STATE_EMPTY","started_upload_at":0,"finished_upload_at":0}`,
		},
		{
			m:    metadata{State: metadataStateError},
			want: `{"state":"METADATA_STATE_ERROR","started_upload_at":0,"finished_upload_at":0}`,
		},
		{
			m:    metadata{State: metadataStateUploading},
			want: `{"state":"METADATA_STATE_UPLOADING","started_upload_at":0,"finished_upload_at":0}`,
		},
		{
			m:    metadata{State: metadataStateUploaded},
			want: `{"state":"METADATA_STATE_UPLOADED","started_upload_at":0,"finished_upload_at":0}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.m.State.String(), func(t *testing.T) {
			got, err := json.Marshal(tt.m)
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			gotStr := string(got)
			if !reflect.DeepEqual(gotStr, tt.want) {
				t.Errorf("MarshalJSON() got = %v, want %v", gotStr, tt.want)
			}
		})
	}
}

func TestMetadata_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name    string
		b       []byte
		want    metadata
		wantErr bool
	}{
		{
			b:    []byte(`{"state":"METADATA_STATE_EMPTY","started_upload_at":0,"finished_upload_at":0}`),
			want: metadata{State: metadataStateEmpty},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_ERROR","started_upload_at":0,"finished_upload_at":0}`),
			want: metadata{State: metadataStateError},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_UPLOADING","started_upload_at":0,"finished_upload_at":0}`),
			want: metadata{State: metadataStateUploading},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_UPLOADED","started_upload_at":0,"finished_upload_at":0}`),
			want: metadata{State: metadataStateUploaded},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := metadata{}

			if err := json.Unmarshal(tt.b, &res); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
