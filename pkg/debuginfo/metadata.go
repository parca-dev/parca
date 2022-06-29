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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/thanos-io/objstore"
)

var (
	ErrMetadataShouldExist     = errors.New("debug info metadata should exist")
	ErrMetadataUnexpectedState = errors.New("debug info metadata state is unexpected")
	// There's no debug info metadata. This could mean that an older version
	// uploaded the debug info files, but there's no record of the metadata, yet.
	ErrMetadataNotFound = errors.New("debug info metadata not found")
)

type metadataState int64

const (
	metadataStateUnknown metadataState = iota
	// The debug info file is being uploaded.
	metadataStateUploading
	// The debug info file is fully uploaded.
	metadataStateUploaded
	// The debug info file is corrupted.
	metadataStateCorrupted
)

var mdStateStr = map[metadataState]string{
	metadataStateUnknown:   "METADATA_STATE_UNKNOWN",
	metadataStateUploading: "METADATA_STATE_UPLOADING",
	metadataStateUploaded:  "METADATA_STATE_UPLOADED",
	metadataStateCorrupted: "METADATA_STATE_CORRUPTED",
}

var strMdState = map[string]metadataState{
	"METADATA_STATE_UNKNOWN":   metadataStateUnknown,
	"METADATA_STATE_UPLOADING": metadataStateUploading,
	"METADATA_STATE_UPLOADED":  metadataStateUploaded,
	"METADATA_STATE_CORRUPTED": metadataStateCorrupted,
}

func (m metadataState) String() string {
	val, ok := mdStateStr[m]
	if !ok {
		return "<not found>"
	}
	return val
}

func (m metadataState) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(mdStateStr[m])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (m *metadataState) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*m = strMdState[s]
	return nil
}

type metadataManager struct {
	logger log.Logger

	bucket objstore.Bucket
}

func newMetadataManager(logger log.Logger, bucket objstore.Bucket) *metadataManager {
	return &metadataManager{logger: log.With(logger, "component", "debuginfo-metadata"), bucket: bucket}
}

type metadata struct {
	State            metadataState `json:"state"`
	Hash             string        `json:"hash"`
	UploadStartedAt  int64         `json:"upload_started_at"`
	UploadFinishedAt int64         `json:"upload_finished_at"`
}

func (m *metadataManager) markAsCorrupted(ctx context.Context, buildID string) error {
	if err := m.write(ctx, buildID, &metadata{
		State: metadataStateCorrupted,
	}); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	return nil
}

func (m *metadataManager) markAsUploading(ctx context.Context, buildID string) error {
	_, err := m.bucket.Get(ctx, metadataObjectPath(buildID))
	// The metadata file should not exist yet. Not erroring here because there's
	// room for a race condition.
	if err == nil {
		level.Info(m.logger).Log("msg", "there should not be a metadata file")
		return nil
	}

	if !m.bucket.IsObjNotFoundErr(err) {
		level.Error(m.logger).Log("msg", "unexpected error", "err", err)
		return err
	}

	if err := m.write(ctx, buildID, &metadata{
		State:           metadataStateUploading,
		UploadStartedAt: time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	return nil
}

func (m *metadataManager) markAsUploaded(ctx context.Context, buildID, hash string) error {
	r, err := m.bucket.Get(ctx, metadataObjectPath(buildID))
	if err != nil {
		level.Error(m.logger).Log("msg", "expected metadata file", "err", err)
		return ErrMetadataShouldExist
	}
	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return err
	}

	metaData := &metadata{}
	if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
		return err
	}

	// There's a small window where a race could happen.
	if metaData.State == metadataStateUploaded {
		return nil
	}

	if metaData.State != metadataStateUploading {
		return ErrMetadataUnexpectedState
	}

	metaData.State = metadataStateUploaded
	metaData.Hash = hash
	metaData.UploadFinishedAt = time.Now().Unix()

	metadataBytes, _ := json.MarshalIndent(&metaData, "", "\t")
	newData := bytes.NewReader(metadataBytes)

	if err := m.bucket.Upload(ctx, metadataObjectPath(buildID), newData); err != nil {
		return err
	}
	return nil
}

func (m *metadataManager) fetch(ctx context.Context, buildID string) (*metadata, error) {
	r, err := m.bucket.Get(ctx, metadataObjectPath(buildID))
	if err != nil {
		if m.bucket.IsObjNotFoundErr(err) {
			return nil, ErrMetadataNotFound
		}
		return nil, err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	metaData := &metadata{}
	if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
		return nil, err
	}
	return metaData, nil
}

func (m *metadataManager) write(ctx context.Context, buildID string, md *metadata) error {
	metadataBytes, _ := json.MarshalIndent(md, "", "\t")
	r := bytes.NewReader(metadataBytes)
	if err := m.bucket.Upload(ctx, metadataObjectPath(buildID), r); err != nil {
		level.Error(m.logger).Log("msg", "failed to create metadata file", "err", err)
		return err
	}
	return nil
}

func metadataObjectPath(buildID string) string {
	return path.Join(buildID, "metadata")
}
