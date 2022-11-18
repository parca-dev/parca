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
	"github.com/thanos-io/objstore"
	"golang.org/x/net/context"
)

type DebuginfodClient interface {
	GetDebuginfo(ctx context.Context, buildid string) (io.ReadCloser, error)
}

type NopDebuginfodClient struct{}

func (NopDebuginfodClient) GetDebuginfo(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), ErrDebuginfoNotFound
}

type HTTPDebuginfodClient struct {
	logger log.Logger
	client *http.Client

	UpstreamServers []*url.URL
	timeoutDuration time.Duration
}

type DebuginfodClientObjectStorageCache struct {
	logger log.Logger

	client DebuginfodClient
	bucket objstore.Bucket
}

// NewHTTPDebuginfodClient returns a new HTTP debuginfo client.
func NewHTTPDebuginfodClient(logger log.Logger, serverURLs []string, timeoutDuration time.Duration) (*HTTPDebuginfodClient, error) {
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
	return &HTTPDebuginfodClient{
		logger:          logger,
		UpstreamServers: parsedURLs,
		timeoutDuration: timeoutDuration,
		client:          http.DefaultClient,
	}, nil
}

// NewDebuginfodClientWithObjectStorageCache creates a new DebuginfodClient that caches the debug information in the object storage.
func NewDebuginfodClientWithObjectStorageCache(logger log.Logger, bucket objstore.Bucket, h DebuginfodClient) (DebuginfodClient, error) {
	return &DebuginfodClientObjectStorageCache{
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

// GetDebuginfo returns debuginfo for given buildid while caching it in object storage.
func (c *DebuginfodClientObjectStorageCache) GetDebuginfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	logger := log.With(c.logger, "buildid", buildID)
	debuginfo, err := c.client.GetDebuginfo(ctx, buildID)
	if err != nil {
		return nil, err
	}

	r, w := io.Pipe()
	go func() {
		defer w.Close()
		defer debuginfo.Close()

		// TODO(kakkoyun): Use store.upload() to upload the debuginfo to object storage.
		if err := c.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
			level.Error(logger).Log("msg", "failed to upload downloaded debuginfod file", "err", err)
		}
	}()

	return readCloser{
		Reader: io.TeeReader(debuginfo, w),
		closer: closer(func() error {
			defer debuginfo.Close()

			if err := w.Close(); err != nil {
				return err
			}
			return nil
		}),
	}, nil
}

// GetDebuginfo returns debug information file for given buildID by downloading it from upstream servers.
func (c *HTTPDebuginfodClient) GetDebuginfo(ctx context.Context, buildID string) (io.ReadCloser, error) {
	logger := log.With(c.logger, "buildid", buildID)

	// e.g:
	// "https://debuginfod.elfutils.org/"
	// "https://debuginfod.systemtap.org/"
	// "https://debuginfod.opensuse.org/"
	// "https://debuginfod.s.voidlinux.org/"
	// "https://debuginfod.debian.net/"
	// "https://debuginfod.fedoraproject.org/"
	// "https://debuginfod.altlinux.org/"
	// "https://debuginfod.archlinux.org/"
	// "https://debuginfod.centos.org/"
	for _, u := range c.UpstreamServers {
		serverURL := *u
		rc, err := func(serverURL url.URL) (io.ReadCloser, error) {
			ctx, cancel := context.WithTimeout(ctx, c.timeoutDuration)
			defer cancel()

			rc, err := c.request(ctx, serverURL, buildID)
			if err != nil {
				return nil, err
			}
			return rc, nil
		}(serverURL)
		if err != nil {
			level.Warn(logger).Log(
				"msg", "failed to download debuginfo file from upstream debuginfod server, trying next one (if exists)",
				"server", serverURL, "err", err,
			)
			continue
		}
		if rc != nil {
			return rc, nil
		}
	}
	return nil, ErrDebuginfoNotFound
}

func (c *HTTPDebuginfodClient) request(ctx context.Context, u url.URL, buildID string) (io.ReadCloser, error) {
	// https://www.mankier.com/8/debuginfod#Webapi
	// Endpoint: /buildid/BUILDID/debuginfo
	// If the given buildid is known to the server,
	// this request will result in a binary object that contains the customary .*debug_* sections.
	u.Path = path.Join(u.Path, "buildid", buildID, "debuginfo")

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	switch resp.StatusCode / 100 {
	case 2:
		return resp.Body, nil
	case 4:
		if resp.StatusCode == http.StatusNotFound {
			return nil, ErrDebuginfoNotFound
		}
		return nil, fmt.Errorf("client error: %s", resp.Status)
	case 5:
		return nil, fmt.Errorf("server error: %s", resp.Status)
	default:
		return nil, fmt.Errorf("unexpected status code: %s", resp.Status)
	}
}
