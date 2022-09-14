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

type MetadataState int64

const (
	MetadataStateUnknown MetadataState = iota
	// The debug info file is being uploaded.
	MetadataStateUploading
	// The debug info file is fully uploaded.
	MetadataStateUploaded
	// The debug info file is corrupted.
	MetadataStateCorrupted
)

var mdStateStr = map[MetadataState]string{
	MetadataStateUnknown:   "METADATA_STATE_UNKNOWN",
	MetadataStateUploading: "METADATA_STATE_UPLOADING",
	MetadataStateUploaded:  "METADATA_STATE_UPLOADED",
	MetadataStateCorrupted: "METADATA_STATE_CORRUPTED",
}

var strMdState = map[string]MetadataState{
	"METADATA_STATE_UNKNOWN":   MetadataStateUnknown,
	"METADATA_STATE_UPLOADING": MetadataStateUploading,
	"METADATA_STATE_UPLOADED":  MetadataStateUploaded,
	"METADATA_STATE_CORRUPTED": MetadataStateCorrupted,
}

func (m MetadataState) String() string {
	val, ok := mdStateStr[m]
	if !ok {
		return "<not found>"
	}
	return val
}

func (m MetadataState) MarshalJSON() ([]byte, error) {
	buffer := bytes.NewBufferString(`"`)
	buffer.WriteString(mdStateStr[m])
	buffer.WriteString(`"`)
	return buffer.Bytes(), nil
}

func (m *MetadataState) UnmarshalJSON(b []byte) error {
	var s string
	err := json.Unmarshal(b, &s)
	if err != nil {
		return err
	}
	*m = strMdState[s]
	return nil
}

type ObjectStoreMetadata struct {
	logger log.Logger

	bucket objstore.Bucket
}

func NewObjectStoreMetadata(logger log.Logger, bucket objstore.Bucket) *ObjectStoreMetadata {
	return &ObjectStoreMetadata{logger: log.With(logger, "component", "debuginfo-metadata"), bucket: bucket}
}

type Metadata struct {
	State            MetadataState `json:"state"`
	BuildID          string        `json:"build_id"`
	Hash             string        `json:"hash"`
	UploadStartedAt  int64         `json:"upload_started_at"`
	UploadFinishedAt int64         `json:"upload_finished_at"`
}

func (m *ObjectStoreMetadata) MarkAsCorrupted(ctx context.Context, buildID string) error {
	if err := m.write(ctx, buildID, &Metadata{
		State: MetadataStateCorrupted,
	}); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}
	level.Debug(m.logger).Log("msg", "marked as corrupted", "buildid", buildID)
	return nil
}

func (m *ObjectStoreMetadata) MarkAsUploading(ctx context.Context, buildID string) error {
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

	if err := m.write(ctx, buildID, &Metadata{
		State:           MetadataStateUploading,
		BuildID:         buildID,
		UploadStartedAt: time.Now().Unix(),
	}); err != nil {
		return fmt.Errorf("failed to write metadata: %w", err)
	}

	level.Debug(m.logger).Log("msg", "marked as uploading", "buildid", buildID)
	return nil
}

func (m *ObjectStoreMetadata) MarkAsUploaded(ctx context.Context, buildID, hash string) error {
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

	metaData := &Metadata{}
	if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
		return err
	}

	// There's a small window where a race could happen.
	if metaData.State == MetadataStateUploaded {
		return nil
	}

	if metaData.State == MetadataStateUploading && metaData.BuildID != buildID {
		return errors.New("build ids do not match")
	}

	metaData.State = MetadataStateUploaded
	metaData.BuildID = buildID
	metaData.Hash = hash
	metaData.UploadFinishedAt = time.Now().Unix()

	metadataBytes, _ := json.MarshalIndent(&metaData, "", "\t")
	newData := bytes.NewReader(metadataBytes)

	if err := m.bucket.Upload(ctx, metadataObjectPath(buildID), newData); err != nil {
		return err
	}

	level.Debug(m.logger).Log("msg", "marked as uploaded", "buildid", buildID)
	return nil
}

func (m *ObjectStoreMetadata) Fetch(ctx context.Context, buildID string) (*Metadata, error) {
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

	metaData := &Metadata{}
	if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
		return nil, err
	}
	return metaData, nil
}

func (m *ObjectStoreMetadata) write(ctx context.Context, buildID string, md *Metadata) error {
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
