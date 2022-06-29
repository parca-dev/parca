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
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/nanmu42/limitio"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var errDebugInfoNotFound = errors.New("debug info not found")

type CacheProvider string

const (
	FILESYSTEM CacheProvider = "FILESYSTEM"
)

type Config struct {
	Bucket *client.BucketConfig `yaml:"bucket"`
	Cache  *CacheConfig         `yaml:"cache"`
}

type FilesystemCacheConfig struct {
	Directory string `yaml:"directory"`
}

type CacheConfig struct {
	Type   CacheProvider `yaml:"type"`
	Config interface{}   `yaml:"config"`
}

type Store struct {
	debuginfopb.UnimplementedDebugInfoServiceServer

	logger   log.Logger
	cacheDir string

	bucket           objstore.Bucket
	debuginfodClient DebugInfodClient

	symbolizer      *symbol.Symbolizer
	metadataManager *metadataManager

	pool sync.Pool
}

// NewStore returns a new debug info store.
func NewStore(logger log.Logger, symbolizer *symbol.Symbolizer, config *Config, debuginfodClient DebugInfodClient) (*Store, error) {
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, "parca/store")
	if err != nil {
		return nil, fmt.Errorf("instantiate object storage: %w", err)
	}

	cacheCfg, err := yaml.Marshal(config.Cache)
	if err != nil {
		return nil, fmt.Errorf("marshal content of cache configuration: %w", err)
	}

	cache, err := newCache(cacheCfg)
	if err != nil {
		return nil, fmt.Errorf("instantiate cache: %w", err)
	}

	return &Store{
		logger:           log.With(logger, "component", "debuginfo"),
		bucket:           bucket,
		cacheDir:         cache.Directory,
		symbolizer:       symbolizer,
		debuginfodClient: debuginfodClient,
		metadataManager:  newMetadataManager(logger, bucket),
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(nil)
			},
		},
	}, nil
}

func newCache(cacheCfg []byte) (*FilesystemCacheConfig, error) {
	cacheConf := &CacheConfig{}
	if err := yaml.UnmarshalStrict(cacheCfg, cacheConf); err != nil {
		return nil, fmt.Errorf("parsing config YAML file: %w", err)
	}

	config, err := yaml.Marshal(cacheConf.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal content of cache configuration: %w", err)
	}

	var c FilesystemCacheConfig
	switch strings.ToUpper(string(cacheConf.Type)) {
	case string(FILESYSTEM):
		if err := yaml.Unmarshal(config, &c); err != nil {
			return nil, err
		}
		if c.Directory == "" {
			return nil, errors.New("missing directory for filesystem bucket")
		}
	default:
		return nil, fmt.Errorf("cache with type %s is not supported", cacheConf.Type)
	}

	if _, err := os.Stat(c.Directory); os.IsNotExist(err) {
		err := os.MkdirAll(c.Directory, 0o700)
		if err != nil {
			return nil, err
		}
	}
	return &c, nil
}

func (s *Store) Exists(ctx context.Context, req *debuginfopb.ExistsRequest) (*debuginfopb.ExistsResponse, error) {
	buildID := req.BuildId

	var exists bool
	var err error
	defer func() {
		level.Debug(s.logger).Log("msg", "debug info exists", "exists", exists, "buildid", buildID, "err", err)
	}()

	if err := validateInput(buildID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	found, err := s.find(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if found {
		metadataFile, err := s.metadataManager.fetch(ctx, buildID)
		if err != nil {
			if errors.Is(err, ErrMetadataNotFound) {
				return &debuginfopb.ExistsResponse{Exists: exists}, nil
			}
			return nil, status.Error(codes.Internal, err.Error())
		}

		if metadataFile.Hash != "" && metadataFile.Hash == req.Hash {
			exists = true
			return &debuginfopb.ExistsResponse{Exists: exists}, nil
		}

		// If it is not an exact version of the source object file what we have so, let the client try to upload it.
		if metadataFile.State == metadataStateUploading {
			exists = !isStale(metadataFile)
		}
		return &debuginfopb.ExistsResponse{Exists: exists}, nil
	}

	return &debuginfopb.ExistsResponse{Exists: exists}, nil
}

func (s *Store) Upload(stream debuginfopb.DebugInfoService_UploadServer) error {
	req, err := stream.Recv()
	if err != nil {
		msg := "failed to receive upload info"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	var (
		buildID = req.GetInfo().BuildId
		hash    = req.GetInfo().Hash
		r       = &UploadReader{stream: stream}
	)
	if err := s.upload(stream.Context(), buildID, hash, r); err != nil {
		return fmt.Errorf("failed to upload debug information file: %w", err)
	}

	level.Debug(s.logger).Log("msg", "debug info uploaded", "buildid", buildID)
	return stream.SendAndClose(&debuginfopb.UploadResponse{
		BuildId: buildID,
		Size:    r.size,
	})
}

func (s *Store) upload(ctx context.Context, buildID, hash string, r io.Reader) error {
	if err := validateInput(buildID); err != nil {
		err = fmt.Errorf("invalid build ID: %w", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validateInput(hash); err != nil {
		err = fmt.Errorf("invalid hash: %w", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	level.Debug(s.logger).Log("msg", "trying to upload debug info", "buildid", buildID)

	metadataFile, err := s.metadataManager.fetch(ctx, buildID)
	if err == nil {
		level.Debug(s.logger).Log("msg", "fetching metadata state", "result", metadataFile)

		switch metadataFile.State {
		case metadataStateCorrupted:
			// Corrupted. Re-upload.
		case metadataStateUploaded:
			// The debug info was fully uploaded.
			return status.Error(codes.AlreadyExists, "debuginfo already exists")
		case metadataStateUploading:
			if !isStale(metadataFile) {
				return status.Error(codes.AlreadyExists, "debuginfo already exists, being uploaded right now")
			}
			// The debug info upload operation most likely failed.
		default:
			return status.Error(codes.Internal, "unknown metadata state")
		}
	} else {
		if !errors.Is(err, ErrMetadataNotFound) {
			level.Error(s.logger).Log("msg", "failed to fetch metadata state", "err", err)
		}
	}

	found, err := s.find(ctx, buildID)
	if err != nil {
		return err
	}

	if found {
		if hash != "" && metadataFile != nil {
			if metadataFile.Hash == hash {
				level.Debug(s.logger).Log("msg", "debug info already exists", "buildid", buildID)
				return status.Error(codes.AlreadyExists, "debuginfo already exists")
			}
		}

		objFile, err := s.fetchObjectFile(ctx, buildID)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}

		if err := elfutils.ValidateFile(objFile); err != nil {
			// Failed to validate. Mark the file as corrupted, and let the client try to upload it again.
			if err := s.metadataManager.markAsCorrupted(ctx, buildID); err != nil {
				level.Warn(s.logger).Log("msg", "failed to update metadata as corrupted", "err", err)
			}
			level.Error(s.logger).Log("msg", "failed to validate object file", "buildid", buildID)
			// Client will retry.
			return status.Error(codes.Internal, err.Error())
		}

		// Valid.
		hasDWARF, err := elfutils.HasDWARF(objFile)
		if err != nil {
			level.Debug(s.logger).Log("msg", "failed to check for DWARF", "err", err)
		}
		if hasDWARF {
			return status.Error(codes.AlreadyExists, "debuginfo already exists")
		}
	}

	if err := s.metadataManager.markAsUploading(ctx, buildID); err != nil {
		err = fmt.Errorf("failed to update metadata before uploading: %w", err)
		return status.Error(codes.Internal, err.Error())
	}

	//nolint:forcetypeassert
	b := s.pool.Get().(*bytes.Buffer)
	// NOTICE: The ELF header is 52 or 64 bytes long for 32-bit and 64-bit binaries respectively.
	buf := limitio.NewWriter(b, 64, true)
	defer func() {
		b.Reset()
		s.pool.Put(b)
	}()

	// At this point we know that we received a better version of the debug information file,
	// so let the client upload it.

	if err := s.bucket.Upload(ctx, objectPath(buildID), io.TeeReader(r, buf)); err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	if err := elfutils.ValidateHeader(b); err != nil {
		// Failed to validate. Mark the incoming stream as corrupted, and let the client try to upload it again.
		if err := s.metadataManager.markAsCorrupted(ctx, buildID); err != nil {
			err = fmt.Errorf("failed to update metadata after uploaded, as corrupted: %w", err)
			return status.Error(codes.Internal, err.Error())
		}
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.metadataManager.markAsUploaded(ctx, buildID, hash); err != nil {
		err = fmt.Errorf("failed to update metadata after uploaded: %w", err)
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func isStale(metadataFile *metadata) bool {
	return time.Now().Add(-15 * time.Minute).After(time.Unix(metadataFile.UploadStartedAt, 0))
}

func validateInput(id string) error {
	_, err := hex.DecodeString(id)
	if err != nil {
		return fmt.Errorf("failed to validate input: %w", err)
	}
	if len(id) <= 2 {
		return errors.New("unexpectedly short input")
	}

	return nil
}

func (s *Store) find(ctx context.Context, key string) (bool, error) {
	found := false
	err := s.bucket.Iter(ctx, key, func(_ string) error {
		// We just need any debug files to be present, so if a file under the directory for the build ID exists,
		// it's found: <buildid>/debuginfo
		found = true
		return nil
	})
	if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}
	return found, nil
}

// Symbolize fetches the debug info for a given build ID and symbolizes it the
// given location.
func (s *Store) Symbolize(ctx context.Context, m *pb.Mapping, locations []*pb.Location) ([][]profile.LocationLine, error) {
	buildID := m.BuildId
	logger := log.With(s.logger, "buildid", buildID)

	var downloadedFromDebugInfod bool
	objFile, err := s.fetchObjectFile(ctx, buildID)
	if err != nil {
		// It's ok if we don't have the symbols for given BuildID, it happens too often.
		level.Warn(logger).Log("msg", "failed to fetch object", "err", err)

		// Let's try to find a debug file from debuginfod servers.
		objFile, err = s.fetchDebuginfodFile(ctx, buildID)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch: %w", err)
		}
		downloadedFromDebugInfod = true
	}

	// Let's make sure we have the best version of the debug file.
	if err := elfutils.ValidateFile(objFile); err != nil {
		level.Warn(logger).Log("msg", "failed to validate debug information", "err", err)
		// Mark the file as corrupted, and let the client try to upload it again.
		err := s.metadataManager.markAsCorrupted(ctx, buildID)
		if err != nil {
			level.Warn(logger).Log(
				"msg", "failed to mar debug information",
				"err", fmt.Errorf("failed to update metadata for corrupted: %w", err),
			)
		}
		if !downloadedFromDebugInfod {
			dbgFile, err := s.fetchDebuginfodFile(ctx, buildID)
			if err != nil {
				level.Warn(logger).Log("msg", "failed to fetch debuginfod file", "err", err)
			} else {
				objFile = dbgFile
				downloadedFromDebugInfod = true
			}
		}
	}
	if !downloadedFromDebugInfod {
		hasDWARF, err := elfutils.HasDWARF(objFile)
		if err != nil {
			level.Debug(logger).Log("msg", "failed to check for DWARF", "err", err)
		}
		if !hasDWARF {
			// Try to download a better version from debuginfod servers.
			dbgFile, err := s.fetchDebuginfodFile(ctx, buildID)
			if err != nil {
				level.Warn(logger).Log("msg", "failed to fetch debuginfod file", "err", err)
			} else {
				objFile = dbgFile
			}
		}
	}

	// At this point we have the best version of the debug information file that we could find.
	// Let's symbolize it.
	lines, err := s.symbolizer.Symbolize(ctx, m, locations, objFile)
	if err != nil {
		if errors.Is(err, symbol.ErrLinerCreationFailedBefore) {
			level.Debug(logger).Log("msg", "failed to symbolize before", "err", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to symbolize locations for mapping: %w", err)
	}
	return lines, nil
}

func (s *Store) fetchObjectFile(ctx context.Context, buildID string) (string, error) {
	logger := log.With(s.logger, "buildid", buildID)

	objFile := s.localCachePath(buildID)
	// Check if it's already cached locally; if not download.
	if _, err := os.Stat(objFile); os.IsNotExist(err) {
		// Download the debuginfo file from the bucket.
		r, err := s.bucket.Get(ctx, objectPath(buildID))
		if err != nil {
			if s.bucket.IsObjNotFoundErr(err) {
				level.Debug(logger).Log("msg", "failed to fetch object from object storage", "err", err)
				return "", errDebugInfoNotFound
			}
			return "", fmt.Errorf("failed to fetch object: %w", err)
		}

		// Cache the file locally.
		if err := cache(objFile, r); err != nil {
			return "", fmt.Errorf("failed to fetch debug info file: %w", err)
		}
	}

	return objFile, nil
}

func (s *Store) fetchDebuginfodFile(ctx context.Context, buildID string) (string, error) {
	logger := log.With(s.logger, "buildid", buildID)
	level.Debug(logger).Log("msg", "attempting to download from debuginfod servers")

	objFile := s.localCachePath(buildID)
	// Try downloading the debuginfo file from the debuginfod server.
	r, err := s.debuginfodClient.GetDebugInfo(ctx, buildID)
	if err != nil {
		level.Debug(logger).Log("msg", "failed to download debuginfo from debuginfod", "err", err)
		return "", fmt.Errorf("failed to fetch from debuginfod: %w", err)
	}
	defer r.Close()
	level.Info(logger).Log("msg", "debug info downloaded from debuginfod server")

	// Cache the file locally.
	if err := cache(objFile, r); err != nil {
		level.Debug(logger).Log("msg", "failed to cache debuginfo", "err", err)
		return "", fmt.Errorf("failed to fetch from debuginfod: %w", err)
	}

	return objFile, nil
}

func (s *Store) localCachePath(buildID string) string {
	return path.Join(s.cacheDir, buildID, "debuginfo")
}

func cache(localPath string, r io.ReadCloser) error {
	tmpfile, err := ioutil.TempFile("", "symbol-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	written, err := io.Copy(tmpfile, r)
	if err != nil {
		return fmt.Errorf("copy debug info file to local temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("close tempfile to write debug info file: %w", err)
	}
	if written == 0 {
		return fmt.Errorf("received empty debug info: %w", errDebugInfoNotFound)
	}

	err = os.MkdirAll(path.Dir(localPath), 0o700)
	if err != nil {
		return fmt.Errorf("create debug info file directory: %w", err)
	}
	// Need to use rename to make the "creation" atomic.
	if err := os.Rename(tmpfile.Name(), localPath); err != nil {
		return fmt.Errorf("atomically move downloaded debug info file: %w", err)
	}
	return nil
}

func objectPath(buildID string) string {
	return path.Join(buildID, "debuginfo")
}
