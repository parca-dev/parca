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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/thanos-io/thanos/pkg/objstore"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gopkg.in/yaml.v2"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/symbol"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var ErrDebugInfoNotFound = errors.New("debug info not found")

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

	symbolizer *symbol.Symbolizer
}

// NewStore returns a new debug info store.
func NewStore(logger log.Logger, symbolizer *symbol.Symbolizer, config *Config, debuginfodClient DebugInfodClient) (*Store, error) {
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, nil, "parca/store")
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
	if err := validateID(req.BuildId); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	found, err := s.find(ctx, req.BuildId)
	if err != nil {
		return nil, err
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

	buildID := req.GetInfo().BuildId
	if err = validateID(buildID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	level.Debug(s.logger).Log("msg", "trying to upload debug info", "buildid", buildID)
	ctx := stream.Context()
	found, err := s.find(ctx, buildID)
	if err != nil {
		return err
	}

	if found {
		level.Debug(s.logger).Log("msg", "debug info already exists", "buildid", buildID)
		return status.Error(codes.AlreadyExists, "debuginfo already exists")
	}

	r := &UploadReader{stream: stream}
	if err := s.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	level.Debug(s.logger).Log("msg", "debug info uploaded", "buildid", buildID)

	return stream.SendAndClose(&debuginfopb.UploadResponse{
		BuildId: buildID,
		Size:    r.size,
	})
}

func validateID(id string) error {
	_, err := hex.DecodeString(id)
	if err != nil {
		return fmt.Errorf("failed to validate id: %w", err)
	}
	if len(id) <= 2 {
		return errors.New("unexpectedly short ID")
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

func (s *Store) Symbolize(ctx context.Context, m *pb.Mapping, locations ...*metastore.Location) (map[*metastore.Location][]metastore.LocationLine, error) {
	logger := log.With(s.logger, "buildid", m.BuildId)

	localObjPath, err := s.fetchObjectFile(ctx, m.BuildId)
	if err != nil {
		level.Debug(logger).Log("msg", "failed to fetch object", "err", err)
		return nil, fmt.Errorf("failed to symbolize mapping: %w", err)
	}

	locationLines, err := s.symbolizer.Symbolize(ctx, m, locations, localObjPath)
	if err != nil {
		return nil, fmt.Errorf("failed to symbolize locations for mapping: %w", err)
	}
	return locationLines, nil
}

func (s *Store) fetchObjectFile(ctx context.Context, buildID string) (string, error) {
	logger := log.With(s.logger, "buildid", buildID)

	localObjectFilePath := path.Join(s.cacheDir, buildID, "debuginfo")
	// Check if it's already cached locally; if not download.
	if _, err := os.Stat(localObjectFilePath); os.IsNotExist(err) {
		// Download the debuginfo file from the bucket.
		r, err := s.bucket.Get(ctx, objectPath(buildID))
		if err != nil {
			if !s.bucket.IsObjNotFoundErr(err) {
				level.Warn(logger).Log("msg", "failed to fetch object from object storage", "err", err)
			}

			level.Debug(logger).Log("msg", "attempting to download from debuginfod servers")
			// Try downloading the debuginfo file from the debuginfod server.
			r, err = s.debuginfodClient.GetDebugInfo(ctx, buildID)
			if err != nil {
				return "", fmt.Errorf("download debug info from object store and debuginfod server: %w", err)
			}
			defer r.Close()
			level.Info(logger).Log("msg", "debug info downloaded from debuginfod server")
		}

		// Cache the file locally.
		if err := cache(localObjectFilePath, r); err != nil {
			return "", fmt.Errorf("failed to fetch debug info file: %w", err)
		}
	}

	if hasDWARF, err := elfutils.HasDWARF(localObjectFilePath); err == nil && !hasDWARF {
		level.Debug(logger).Log("msg", "trying to find a better version of the debug information file")
		r, err := s.debuginfodClient.GetDebugInfo(ctx, buildID)
		if err != nil {
			level.Debug(logger).Log("msg", "failed to find a better version of the debug information file", "err", err)
			return localObjectFilePath, nil
		}
		defer r.Close()

		if err := cache(localObjectFilePath, r); err != nil {
			level.Debug(logger).Log("msg", "failed to cache the better debug information file", "err", err)
			return localObjectFilePath, nil
		}
	}

	return localObjectFilePath, nil
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
		return fmt.Errorf("received empty debug info: %w", ErrDebugInfoNotFound)
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
