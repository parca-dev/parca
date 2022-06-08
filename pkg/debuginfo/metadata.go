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
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/thanos-io/objstore"
)

var (
	logger                                     log.Logger
	ErrDebugInfoMetadataShouldExist            = errors.New("debug info metadata should exist")
	ErrDebugInfoMetadataExpectedStateUploading = errors.New("debug info metadata state should be uploading")
)

type metadataState int64

const (
	// There was an unexpected error. The error will be filled in in the return
	// value
	metadataStateError metadataState = iota
	// There's no debug info metadata. This could mean that an older Parca version
	// uploaded the debug info files, but there's not record of the metadata, yet.
	metadataStateEmpty
	// The debug info file is being uploaded.
	metadataStateUploading
	// The debug info file is fully uploaded.
	metadataStateUploaded
)

func setMetadataLogger(l log.Logger) {
	logger = log.With(l, "component", "debuginfo-metadata")
}

func (m metadataState) String() string {
	d := map[metadataState]string{
		metadataStateError:     "METADATA_STATE_ERROR",
		metadataStateEmpty:     "METADATA_STATE_EMTPY",
		metadataStateUploading: "METADATA_STATE_UPLOADING",
		metadataStateUploaded:  "METADATA_STATE_UPLOADED",
	}

	val, ok := d[m]
	if !ok {
		return "<not found>"
	}
	return val
}

type DebugInfoMetadata struct {
	State            metadataState `json:"state"`
	StartedUploadAt  int64         `json:"started_upload_at"`
	FinishedUploadAt int64         `json:"finished_upload_at"`
}

func metadataUpdate(ctx context.Context, bucket objstore.Bucket, buildID string, state metadataState) error {
	level.Debug(logger).Log("msg", "Attempting state update to", "state", state)

	switch state {
	case metadataStateUploading:
		_, err := bucket.Get(ctx, metadataPath(buildID))
		// The metadata file should not exist yet. Not erroring here because there's
		// room for a race condition.
		if err == nil {
			level.Info(logger).Log("msg", "There should not be a metadata file")
			return nil
		}

		if !bucket.IsObjNotFoundErr(err) {
			level.Error(logger).Log("msg", "Expected IsObjNotFoundErr but got", "err", err)
			return err
		}

		// Let's write the metadata.
		metadataBytes, _ := json.MarshalIndent(&DebugInfoMetadata{
			State:           metadataStateUploading,
			StartedUploadAt: time.Now().Unix(),
		}, "", "\t")
		r := bytes.NewReader(metadataBytes)
		if err := bucket.Upload(ctx, metadataPath(buildID), r); err != nil {
			level.Error(logger).Log("msg", "Creating the metadata file failed", "err", err)
			return err
		}

	case metadataStateUploaded:
		r, err := bucket.Get(ctx, metadataPath(buildID))
		if err != nil {
			level.Error(logger).Log("msg", "Expected metadata file", "err", err)
			return ErrDebugInfoMetadataShouldExist
		}
		buf := new(bytes.Buffer)
		_, err = buf.ReadFrom(r)
		if err != nil {
			level.Error(logger).Log("msg", "ReadFrom failed", "err", err)
			return err
		}

		metaData := &DebugInfoMetadata{}

		if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
			level.Error(logger).Log("msg", "Parsing JSON metadata failed", "err", err)
			return err
		}

		// There's a small window where a race could happen.
		if metaData.State == metadataStateUploaded {
			return nil
		}

		if metaData.State != metadataStateUploading {
			return ErrDebugInfoMetadataExpectedStateUploading
		}

		metaData.State = metadataStateUploaded
		metaData.FinishedUploadAt = time.Now().Unix()

		metadataBytes, _ := json.MarshalIndent(&metaData, "", "\t")
		newData := bytes.NewReader(metadataBytes)

		if err := bucket.Upload(ctx, metadataPath(buildID), newData); err != nil {
			return err
		}

	}
	return nil
}

func fetchMetadataState(ctx context.Context, bucket objstore.Bucket, buildID string) (metadataState, error) {
	r, err := bucket.Get(ctx, metadataPath(buildID))
	if err != nil {
		return metadataStateEmpty, nil
	}

	buf := new(bytes.Buffer)
	buf.ReadFrom(r)

	metaData := &DebugInfoMetadata{}
	if err := json.Unmarshal(buf.Bytes(), metaData); err != nil {
		return metadataStateError, err
	}
	return metaData.State, nil
}

func metadataPath(buildID string) string {
	return path.Join(buildID, "metadata")
}
