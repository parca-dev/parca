// Copyright 2023-2026 The Parca Authors
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

package query

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"

	"github.com/klauspost/compress/zstd"
	"github.com/thanos-io/objstore"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
)

type DebuginfodClients interface {
	GetSource(ctx context.Context, server, buildid, file string) (io.ReadCloser, error)
	Exists(ctx context.Context, buildid string) ([]string, error)
}

type BucketSourceFinder struct {
	bucket     objstore.BucketReader
	debuginfod DebuginfodClients
}

func NewBucketSourceFinder(
	bucket objstore.BucketReader,
	debuginfod DebuginfodClients,
) *BucketSourceFinder {
	return &BucketSourceFinder{
		bucket:     bucket,
		debuginfod: debuginfod,
	}
}

func (f *BucketSourceFinder) findDebuginfodSource(ctx context.Context, ref *pb.SourceReference) (string, error) {
	servers, err := f.debuginfod.Exists(ctx, ref.BuildId)
	if err != nil {
		return "", err
	}

	for _, server := range servers {
		r, err := f.debuginfod.GetSource(ctx, server, ref.BuildId, ref.Filename)
		if err != nil {
			continue
		}
		defer r.Close()

		b, err := io.ReadAll(r)
		if err != nil {
			return "", err
		}

		return string(b), nil
	}

	return "", ErrNoSourceForBuildID
}

func (f *BucketSourceFinder) FindSource(ctx context.Context, ref *pb.SourceReference) (string, error) {
	r, err := f.bucket.Get(ctx, path.Join(ref.BuildId, "sources"))
	if err != nil {
		if f.bucket.IsObjNotFoundErr(err) {
			return f.findDebuginfodSource(ctx, ref)
		}
		return "", err
	}
	defer r.Close()

	b, err := io.ReadAll(r)
	if err != nil {
		return "", err
	}

	sr, err := NewSourcesReader(bytes.NewReader(b))
	if err != nil {
		return "", err
	}

	source, err := sr.Find(ref.Filename)
	if err != nil {
		return "", err
	}

	return string(source), nil
}

func (f *BucketSourceFinder) SourceExists(ctx context.Context, ref *pb.SourceReference) (bool, error) {
	exists, err := f.bucket.Exists(ctx, path.Join(ref.BuildId, "sources"))
	if err != nil {
		return false, err
	}

	if !exists {
		servers, err := f.debuginfod.Exists(ctx, ref.BuildId)
		if err != nil {
			return false, err
		}

		if len(servers) == 0 {
			return false, nil
		}
	}

	return true, nil
}

type SourcesReader struct {
	r           *tar.Reader
	maxFileSize int64
}

func NewSourcesReader(r io.ReaderAt) (*SourcesReader, error) {
	magic := make([]byte, 4)
	_, err := r.ReadAt(magic, 0)
	if err != nil {
		return nil, fmt.Errorf("read magic: %v", err)
	}

	const maxint64 = 1<<63 - 1
	// 1MB buffer means we read the underlying reader in chunks of 1MB.
	sr := io.NewSectionReader(r, 0, maxint64)

	var compressedReader io.Reader
	if magic[0] == 0x1f && magic[1] == 0x8b && magic[2] == 0x08 {
		compressedReader, err = gzip.NewReader(sr)
		if err != nil {
			return nil, fmt.Errorf("new gzip reader: %v", err)
		}
	}

	if magic[0] == 0x28 && magic[1] == 0xb5 && magic[2] == 0x2f && magic[3] == 0xfd {
		compressedReader, err = zstd.NewReader(sr)
		if err != nil {
			return nil, fmt.Errorf("new zstd reader: %v", err)
		}
	}

	if compressedReader == nil {
		return nil, errors.New("unknown compression format")
	}

	return &SourcesReader{
		r:           tar.NewReader(compressedReader),
		maxFileSize: 1024 * 1024 * 100, // 100MB
	}, nil
}

func (s *SourcesReader) Find(filename string) ([]byte, error) {
	path := trimLeadingSlash(filename)
	for {
		header, err := s.r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("next tar entry: %v", err)
		}

		if header.Typeflag == tar.TypeReg {
			if header.Size > s.maxFileSize {
				return nil, fmt.Errorf("file %s is too large (%d bytes)", header.Name, header.Size)
			}

			sourceFile := trimLeadingSlash(header.Name)
			if sourceFile == path {
				buf := make([]byte, header.Size)
				_, err := io.ReadFull(s.r, buf)
				if err != nil {
					return nil, fmt.Errorf("read file %s: %v", header.Name, err)
				}
				return buf, nil
			}

			// Skip the file.
			_, err := io.CopyN(io.Discard, s.r, header.Size)
			if err != nil {
				return nil, fmt.Errorf("skip file %s: %v", header.Name, err)
			}
		}
	}

	return nil, ErrSourceNotFound
}

func trimLeadingSlash(s string) string {
	if len(s) > 0 && s[0] == '/' {
		return s[1:]
	}
	if len(s) > 1 && s[0] == '.' && s[1] == '/' {
		return s[2:]
	}
	return s
}
