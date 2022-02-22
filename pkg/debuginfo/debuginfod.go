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
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/thanos-io/thanos/pkg/objstore"
	"github.com/thanos-io/thanos/pkg/objstore/client"
	"golang.org/x/net/context"
	"gopkg.in/yaml.v2"
)

type DebugInfodClient interface {
	GetDebugInfo(ctx context.Context, buildid string) (io.ReadCloser, error)
}

type HTTPDebugInfodClient struct {
	logger          log.Logger
	UpstreamServer  *url.URL
	timeoutDuration time.Duration
}

type DebugInfodClientObjectStorageCache struct {
	logger log.Logger
	client DebugInfodClient
	bucket objstore.Bucket
}

func NewHTTPDebugInfodClient(logger log.Logger, serverURL string, timeoutDuration time.Duration) (*HTTPDebugInfodClient, error) {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}
	return &HTTPDebugInfodClient{
		logger:          logger,
		UpstreamServer:  parsedURL,
		timeoutDuration: timeoutDuration,
	}, nil
}

func NewDebugInfodClientWithObjectStorageCache(logger log.Logger, config *Config, h DebugInfodClient) (*DebugInfodClientObjectStorageCache, error) {
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of debuginfod object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, nil, "parca")
	if err != nil {
		return nil, fmt.Errorf("instantiate debuginfod object storage: %w", err)
	}
	return &DebugInfodClientObjectStorageCache{
		logger: logger,
		client: h,
		bucket: bucket,
	}, nil
}

func (c *DebugInfodClientObjectStorageCache) GetDebugInfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	path := "/debuginfod-cache/" + buildID + "/debuginfo"

	if exists, _ := c.bucket.Exists(ctx, path); exists {
		debuginfoFile, err := c.bucket.Get(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("failed to download object file from debuginfod cache: build_id: %v: %w", buildID, err)
		}
		return debuginfoFile, nil
	}

	debugInfo, err := c.client.GetDebugInfo(ctx, buildID)
	if err != nil {
		return nil, ErrDebugInfoNotFound
	}

	err = c.bucket.Upload(ctx, path, debugInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to upload to debuginfod cache: %w", err)
	}

	debugInfoReader, err := c.bucket.Get(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("failed to download object file from debuginfod cache: build_id: %v: %w", buildID, err)
	}

	return debugInfoReader, nil
}

func (c *HTTPDebugInfodClient) GetDebugInfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	buildIDURL := *c.UpstreamServer
	buildIDURL.Path = path.Join(buildIDURL.Path, buildID, "debuginfo")

	ctx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", buildIDURL.String(), nil)
	if err != nil {
		level.Debug(c.logger).Log("msg", "failed to create new HTTP request", "err", err)
		return nil, err
	}

	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		level.Debug(c.logger).Log("msg", "object not found in public server", "object", buildID, "err", err)
		return nil, ErrDebugInfoNotFound
	}

	return resp.Body, nil
}
