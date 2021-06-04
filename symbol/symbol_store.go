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
	"encoding/hex"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"path"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gogo/status"
	"github.com/thanos-io/thanos/pkg/objstore"
	"google.golang.org/grpc/codes"

	"github.com/conprof/conprof/internal/pprof/binutils"
	"github.com/conprof/conprof/pkg/store/storepb"
)

type SymbolStore struct {
	bucket   objstore.Bucket
	logger   log.Logger
	bu       *binutils.Binutils
	cacheDir string
}

func NewSymbolStore(logger log.Logger, bucket objstore.Bucket, cacheDir string) *SymbolStore {
	return &SymbolStore{
		logger:   logger,
		bucket:   bucket,
		bu:       &binutils.Binutils{},
		cacheDir: cacheDir,
	}
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

func (s *SymbolStore) Exists(ctx context.Context, req *storepb.SymbolExistsRequest) (*storepb.SymbolExistsResponse, error) {
	err := validateId(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	path := req.Id

	found := false
	err = s.bucket.Iter(ctx, path, func(_ string) error {
		// We just need any debug files to be present.
		found = true
		return nil
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &storepb.SymbolExistsResponse{
		Exists: found,
	}, nil
}

func (s *SymbolStore) Upload(stream storepb.SymbolStore_UploadServer) error {
	req, err := stream.Recv()
	if err != nil {
		msg := "failed to receive upload info"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	id := req.GetInfo().Id
	err = validateId(id)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	path := id + "/debuginfo"

	r := &UploadReader{stream: stream}
	err = s.bucket.Upload(stream.Context(), path, r)
	if err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	return stream.SendAndClose(&storepb.SymbolUploadResponse{
		Id:    id,
		Size_: r.size,
	})
}

type UploadReader struct {
	stream storepb.SymbolStore_UploadServer
	cur    io.Reader
	size   uint64
}

func (r *UploadReader) Read(p []byte) (int, error) {
	if r.cur == nil {
		var err error
		r.cur, err = r.next()
		if err != nil {
			return 0, err
		}
	}
	i, err := r.cur.Read(p)
	if err != nil && err != io.EOF {
		return 0, err
	}
	if err == io.EOF {
		r.cur, err = r.next()
		if err == io.EOF {
			return 0, err
		}
		i, err = r.cur.Read(p)
		if err != nil {
			return 0, err
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
	if err != nil {
		return nil, err
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

func (s *SymbolStore) Symbolize(ctx context.Context, req *storepb.SymbolizeRequest) (*storepb.SymbolizeResponse, error) {
	for _, m := range req.Mappings {
		mappingPath := path.Join(s.cacheDir, m.BuildId, "debuginfo")
		if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
			r, err := s.bucket.Get(ctx, path.Join(m.BuildId, "debuginfo"))
			if s.bucket.IsObjNotFoundErr(err) {
				level.Debug(s.logger).Log("msg", "object not found", "object", m.BuildId)
				continue
			}
			if err != nil {
				level.Error(s.logger).Log("msg", "failed to get object", "object", m.BuildId, "err", err)
				return nil, err
			}
			tmpfile, err := ioutil.TempFile("", "symbol-download")
			if err != nil {
				level.Error(s.logger).Log("msg", "failed to create tmp file")
				return nil, err
			}
			_, err = io.Copy(tmpfile, r)
			if err != nil {
				os.Remove(tmpfile.Name())
				return nil, err
			}
			if err := tmpfile.Close(); err != nil {
				os.Remove(tmpfile.Name())
				return nil, err
			}

			err = os.MkdirAll(path.Join(s.cacheDir, m.BuildId), 0700)
			if err != nil {
				os.Remove(tmpfile.Name())
				return nil, err
			}
			// Need to use rename to make the "creation" atomic.
			if err := os.Rename(tmpfile.Name(), mappingPath); err != nil {
				os.Remove(tmpfile.Name())
				return nil, err
			}
		}

		objFile, err := s.bu.Open(mappingPath, m.MemoryStart, m.MemoryLimit, m.FileOffset)
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to open object file", "mappingpath", mappingPath, "start", m.MemoryStart, "limit", m.MemoryLimit, "offset", m.FileOffset, "err", err)
			return nil, err
		}

		for _, location := range m.Locations {
			frames, err := objFile.SourceLine(location.Address)
			if err != nil {
				level.Debug(s.logger).Log("msg", "failed to open object file", "mappingpath", mappingPath, "start", m.MemoryStart, "limit", m.MemoryLimit, "offset", m.FileOffset, "address", location.Address, "err", err)
				continue
			}

			for _, frame := range frames {
				location.Lines = append(location.Lines, &storepb.Line{
					Line: int64(frame.Line),
					Function: &storepb.Function{
						Name:     frame.Func,
						Filename: frame.File,
					},
				})
			}
		}
	}

	return &storepb.SymbolizeResponse{
		Mappings: req.Mappings,
	}, nil
}
