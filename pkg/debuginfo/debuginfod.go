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

type NopDebugInfodClient struct{}

func (NopDebugInfodClient) GetDebugInfo(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), errDebugInfoNotFound
}

type HTTPDebugInfodClient struct {
	logger          log.Logger
	UpstreamServers []*url.URL
	timeoutDuration time.Duration
}

type DebugInfodClientObjectStorageCache struct {
	logger log.Logger

	client DebugInfodClient
	bucket objstore.Bucket
}

func NewHTTPDebugInfodClient(logger log.Logger, serverURLs []string, timeoutDuration time.Duration) (*HTTPDebugInfodClient, error) {
	logger = log.With(logger, "component", "debuginfod")
	parsedURLs := make([]*url.URL, 0, len(serverURLs))
	for _, serverURL := range serverURLs {
		u, err := url.Parse(serverURL)
		if err != nil {
			return nil, err
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
		}
	}
	return &HTTPDebugInfodClient{
		logger:          logger,
		UpstreamServers: parsedURLs,
		timeoutDuration: timeoutDuration,
	}, nil
}

func NewDebugInfodClientWithObjectStorageCache(logger log.Logger, config *Config, h DebugInfodClient) (DebugInfodClient, error) {
	logger = log.With(logger, "component", "debuginfod")
	cfg, err := yaml.Marshal(config.Bucket)
	if err != nil {
		return nil, fmt.Errorf("marshal content of debuginfod object storage configuration: %w", err)
	}

	bucket, err := client.NewBucket(logger, cfg, nil, "parca/debuginfod")
	if err != nil {
		return nil, fmt.Errorf("instantiate debuginfod object storage: %w", err)
	}

	return &DebugInfodClientObjectStorageCache{
		logger: logger,
		client: h,
		bucket: bucket,
	}, nil
}

type closer func() error

func (f closer) Close() error { return f() }

type readCloser struct {
	io.Reader
	closer
}

func (c *DebugInfodClientObjectStorageCache) GetDebugInfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	logger := log.With(c.logger, "buildid", buildID)
	debugInfo, err := c.client.GetDebugInfo(ctx, buildID)
	if err != nil {
		return nil, errDebugInfoNotFound
	}

	r, w := io.Pipe()
	go func() {
		defer w.Close()
		defer debugInfo.Close()

		if err := c.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
			level.Error(logger).Log("msg", "failed to upload downloaded debuginfod file", "err", err)
		}
	}()

	return readCloser{
		Reader: io.TeeReader(debugInfo, w),
		closer: closer(func() error {
			defer debugInfo.Close()

			if err := w.Close(); err != nil {
				return err
			}
			return nil
		}),
	}, nil
}

func (c *HTTPDebugInfodClient) GetDebugInfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	logger := log.With(c.logger, "buildid", buildID)
	for _, u := range c.UpstreamServers {
		serverURL := *u
		rc, err := c.request(ctx, serverURL, buildID)
		if err == nil {
			return rc, nil
		}
		level.Warn(logger).Log(
			"msg", "failed to get debuginfo from upstream server, trying next one (if exists)",
			"server", serverURL, "err", err,
		)
	}
	return nil, errDebugInfoNotFound
}

func (c *HTTPDebugInfodClient) request(ctx context.Context, serverURL url.URL, buildID string) (io.ReadCloser, error) {
	logger := log.With(c.logger, "buildid", buildID)

	serverURL.Path = path.Join(serverURL.Path, buildID, "debuginfo")

	ctx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL.String(), nil)
	if err != nil {
		level.Debug(logger).Log("msg", "failed to create new HTTP request", "err", err)
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		level.Debug(logger).Log("msg", "object not found in public server", "object", buildID, "err", err)
		return nil, errDebugInfoNotFound
	}

	return resp.Body, nil
}
