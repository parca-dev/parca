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

package debuginfo

import (
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/go-kit/log"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/objstore/providers/filesystem"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"
)

func TestMetadata(t *testing.T) {
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
		prometheus.NewRegistry(),
		cacheDir,
		NewObjectStoreMetadata(logger, prometheus.NewRegistry(), bucket),
		bucket,
		NopDebugInfodClient{},
	)
	require.NoError(t, err)

	// Test that the initial state should be empty.
	_, err = store.metadata.Fetch(context.Background(), "fake-build-id")
	require.ErrorIs(t, err, ErrMetadataNotFound)

	// Updating the state should be written to blob storage.
	err = store.metadata.MarkAsUploading(context.Background(), "fake-build-id")
	require.NoError(t, err)

	md, err := store.metadata.Fetch(context.Background(), "fake-build-id")
	require.NoError(t, err)
	require.Equal(t, MetadataStateUploading, md.State)
}

func TestMetadata_MarshalJSON(t *testing.T) {
	tests := []struct {
		m       Metadata
		want    string
		wantErr bool
	}{
		{
			m:    Metadata{State: MetadataStateUnknown, BuildID: "build_id", Hash: "hash"},
			want: `{"state":"METADATA_STATE_UNKNOWN","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`,
		},
		{
			m:    Metadata{State: MetadataStateUploading, BuildID: "build_id", Hash: "hash"},
			want: `{"state":"METADATA_STATE_UPLOADING","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`,
		},
		{
			m:    Metadata{State: MetadataStateUploaded, BuildID: "build_id", Hash: "hash"},
			want: `{"state":"METADATA_STATE_UPLOADED","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`,
		},
		{
			m:    Metadata{State: MetadataStateCorrupted, BuildID: "build_id", Hash: "hash"},
			want: `{"state":"METADATA_STATE_CORRUPTED","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`,
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
		want    Metadata
		wantErr bool
	}{
		{
			b:    []byte(`{"state":"METADATA_STATE_UNKNOWN","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`),
			want: Metadata{State: MetadataStateUnknown, BuildID: "build_id", Hash: "hash"},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_UPLOADING","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`),
			want: Metadata{State: MetadataStateUploading, BuildID: "build_id", Hash: "hash"},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_UPLOADED","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`),
			want: Metadata{State: MetadataStateUploaded, BuildID: "build_id", Hash: "hash"},
		},
		{
			b:    []byte(`{"state":"METADATA_STATE_CORRUPTED","build_id":"build_id","hash":"hash","upload_started_at":0,"upload_finished_at":0}`),
			want: Metadata{State: MetadataStateCorrupted, BuildID: "build_id", Hash: "hash"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Metadata{}

			if err := json.Unmarshal(tt.b, &res); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
