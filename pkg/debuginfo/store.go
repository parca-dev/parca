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
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/google/pprof/profile"
	"github.com/thanos-io/thanos/pkg/objstore"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	"github.com/parca-dev/parca/internal/pprof/binutils"
	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

type Config struct {
	Bucket *client.BucketConfig `yaml:"bucket"`
}

type Store struct {
	bucket objstore.Bucket
	logger log.Logger

	cacheDir string
	bu       *binutils.Binutils
}

func NewStore(logger log.Logger, config *Config) (*Store, error) {
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, nil, "parca")
	if err != nil {
		return nil, fmt.Errorf("instantiate object storage: %w", err)
	}

	cacheDir := "/tmp" // TODO(kakkoyun): Parametrize through configuration.
	if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
		err := os.MkdirAll(cacheDir, 0700)
		if err != nil {
			return nil, err
		}
	}
	return &Store{
		logger:   logger,
		bucket:   bucket,
		cacheDir: cacheDir,
		bu:       &binutils.Binutils{},
	}, nil
}

func validateId(id string) error {
	_, err := hex.DecodeString(id)
	if err != nil {
		return err
	}
	if len(id) <= 2 {
		return errors.New("unexpectedly short ID")
	}

	return nil
}

func (s *Store) Exists(ctx context.Context, req *debuginfopb.ExistsRequest) (*debuginfopb.ExistsResponse, error) {
	err := validateId(req.BuildId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	path := req.BuildId

	found := false
	err = s.bucket.Iter(ctx, path, func(_ string) error {
		// We just need any debug files to be present.
		found = true
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &debuginfopb.ExistsResponse{
		Exists: found,
	}, nil
}

func (s *Store) Upload(stream debuginfopb.DebugInfoService_UploadServer) error {
	req, err := stream.Recv()
	if err != nil {
		msg := "failed to receive upload info"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	buildId := req.GetInfo().BuildId
	err = validateId(buildId)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	path := buildId + "/debuginfo"

	r := &UploadReader{stream: stream}
	err = s.bucket.Upload(stream.Context(), path, r)
	if err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	return stream.SendAndClose(&debuginfopb.UploadResponse{
		BuildId: buildId,
		Size:    r.size,
	})
}

func (s *Store) Symbolize(ctx context.Context, m *profile.Mapping, locations ...*profile.Location) (map[*profile.Location][]profile.Line, error) {
	if m.BuildID == "" {
		return nil, errors.New("empty buildID")
	}

	mappingPath, err := s.fetchObjectFile(ctx, m.BuildID)
	if err != nil {
		level.Debug(s.logger).Log("msg", "failed to fetch object", "object", m.BuildID, "err", err)
		return nil, fmt.Errorf("failed to symbolize mapping: %w", err)
	}

	// mapInfo utilize
	objFile, err := s.bu.Open(mappingPath, m.Start, m.Limit, m.Offset)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to open object file", "mappingpath", mappingPath, "start", m.Start, "limit", m.Limit, "offset", m.Offset, "err", err)
		return nil, fmt.Errorf("open object file: %w", err)
	}

	lines := map[*profile.Location][]profile.Line{}
	for _, loc := range locations {
		frames, err := objFile.SourceLine(loc.Address)
		if err != nil {
			level.Debug(s.logger).Log("msg", "failed to open object file", "mappingpath", mappingPath, "start", m.Start, "limit", m.Limit, "offset", m.Offset, "address", loc.Address, "err", err)
			continue
		}

		for _, frame := range frames {
			lines[loc] = append(lines[loc], profile.Line{
				Line: int64(frame.Line),
				Function: &profile.Function{
					Name:     frame.Func,
					Filename: frame.File,
				},
			})
		}
	}
	return lines, nil
}

func (s *Store) fetchObjectFile(ctx context.Context, buildID string) (string, error) {
	mappingPath := path.Join(s.cacheDir, buildID, "debuginfo")
	// Check if it's already cached locally; if not download.
	if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
		r, err := s.bucket.Get(ctx, path.Join(buildID, "debuginfo"))
		if s.bucket.IsObjNotFoundErr(err) {
			level.Debug(s.logger).Log("msg", "object not found", "object", buildID)
			return "", fmt.Errorf("object not found: %w", err)
		}
		if err != nil {
			return "", fmt.Errorf("get object from object storage: %w", err)
		}
		tmpfile, err := ioutil.TempFile("", "symbol-download")
		if err != nil {
			return "", fmt.Errorf("create temp file: %w", err)
		}
		defer os.Remove(tmpfile.Name())

		_, err = io.Copy(tmpfile, r)
		if err != nil {

			return "", fmt.Errorf("copy object storage file to local temp file: %w", err)
		}
		if err := tmpfile.Close(); err != nil {
			return "", fmt.Errorf("close tempfile to write object file: %w", err)
		}

		err = os.MkdirAll(path.Join(s.cacheDir, buildID), 0700)
		if err != nil {
			return "", fmt.Errorf("create object file directory: %w", err)
		}
		// Need to use rename to make the "creation" atomic.
		if err := os.Rename(tmpfile.Name(), mappingPath); err != nil {
			return "", fmt.Errorf("atomically move downloaded object file: %w", err)
		}
	}
	return mappingPath, nil
}

type UploadReader struct {
	stream debuginfopb.DebugInfoService_UploadServer
	cur    io.Reader
	size   uint64
}

func (r *UploadReader) Read(p []byte) (int, error) {
	if r.cur == nil {
		var err error
		r.cur, err = r.next()
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fmt.Errorf("get first upload chunk: %w", err)
		}
	}
	i, err := r.cur.Read(p)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("read upload chunk (%d bytes read so far): %w", r.size, err)
	}
	if err == io.EOF {
		r.cur, err = r.next()
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fmt.Errorf("get next upload chunk (%d bytes read so far): %w", r.size, err)
		}
		i, err = r.cur.Read(p)
		if err != nil {
			return 0, fmt.Errorf("read next upload chunk (%d bytes read so far): %w", r.size, err)
		}
	}

	r.size += uint64(i)
	return i, nil
}

func (r *UploadReader) next() (io.Reader, error) {
	err := contextError(r.stream.Context())
	if err != nil {
		return nil, err
	}

	req, err := r.stream.Recv()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("receive from stream: %w", err)
	}

	return bytes.NewBuffer(req.GetChunkData()), nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Error(codes.Canceled, "request is canceled")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "deadline is exceeded")
	default:
		return nil
	}
}
