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

	debuginfodClientCache DebugInfodClient

	bucket objstore.Bucket
	logger log.Logger

	cacheDir   string
	symbolizer *symbol.Symbolizer
}

// NewStore returns a new debug info store
func NewStore(logger log.Logger, symbolizer *symbol.Symbolizer, config *Config, debuginfodClientCache DebugInfodClient) (*Store, error) {
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, nil, "parca")
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
		debuginfodClientCache: debuginfodClientCache,
		logger:                log.With(logger, "component", "debuginfo"),
		bucket:                bucket,
		cacheDir:              cache.Directory,
		symbolizer:            symbolizer,
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
	err := validateID(req.BuildId)
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

	buildID := req.GetInfo().BuildId
	err = validateID(buildID)
	if err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}
	path := buildID + "/debuginfo"

	r := &UploadReader{stream: stream}
	err = s.bucket.Upload(stream.Context(), path, r)
	if err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

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

func (s *Store) Symbolize(ctx context.Context, m *pb.Mapping, locations ...*metastore.Location) (map[*metastore.Location][]metastore.LocationLine, error) {
	localObjPath, err := s.fetchObjectFile(ctx, m.BuildId)
	if err != nil {
		level.Debug(s.logger).Log("msg", "failed to fetch object", "object", m.BuildId, "err", err)
		return nil, fmt.Errorf("failed to symbolize mapping: %w", err)
	}

	liner, err := s.symbolizer.NewLiner(m, localObjPath)
	if err != nil {
		const msg = "failed to create liner"
		level.Debug(s.logger).Log("msg", msg, "object", m.BuildId, "err", err)
		return nil, fmt.Errorf(msg+": %w", err)
	}

	locationLines := map[*metastore.Location][]metastore.LocationLine{}
	for _, loc := range locations {
		lines, err := liner.PCToLines(loc.Address)
		if err != nil {
			level.Debug(s.logger).Log("msg", "failed to extract source lines", "object", m.BuildId, "err", err)
			continue
		}
		locationLines[loc] = append(locationLines[loc], lines...)
	}
	return locationLines, nil
}

func (s *Store) fetchObjectFile(ctx context.Context, buildID string) (string, error) {
	mappingPath := path.Join(s.cacheDir, buildID, "debuginfo")
	// Check if it's already cached locally; if not download.
	if _, err := os.Stat(mappingPath); os.IsNotExist(err) {
		r, err := s.bucket.Get(ctx, path.Join(buildID, "debuginfo"))

		if s.bucket.IsObjNotFoundErr(err) {
			level.Debug(s.logger).Log("msg", "object not found in parca object storage", "object", buildID, "err", err)

			r, err = s.debuginfodClientCache.GetDebugInfo(ctx, buildID)
			if err != nil {
				return "", fmt.Errorf("get object files from debuginfod storage: %w", err)
			}

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

		err = os.MkdirAll(path.Join(s.cacheDir, buildID), 0o700)
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
