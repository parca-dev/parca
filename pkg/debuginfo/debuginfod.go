// Copyright 2022-2023 The Parca Authors
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
	"errors"
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
	Get(ctx context.Context, buildid string) (io.ReadCloser, error)
	Exists(ctx context.Context, buildid string) (bool, error)
}

type NopDebuginfodClient struct{}

func (NopDebuginfodClient) Get(context.Context, string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(nil)), ErrDebuginfoNotFound
}

func (NopDebuginfodClient) Exists(context.Context, string) (bool, error) {
	return false, nil
}

type HTTPDebuginfodClient struct {
	logger log.Logger
	client *http.Client

	upstreamServers []*url.URL
	timeoutDuration time.Duration
}

type DebuginfodClientObjectStorageCache struct {
	logger log.Logger

	client DebuginfodClient
	bucket objstore.Bucket
}

// NewHTTPDebuginfodClient returns a new HTTP debug info client.
func NewHTTPDebuginfodClient(logger log.Logger, serverURLs []string, client *http.Client) (*HTTPDebuginfodClient, error) {
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

		parsedURLs = append(parsedURLs, u)
	}

	return &HTTPDebuginfodClient{
		logger:          logger,
		upstreamServers: parsedURLs,
		client:          client,
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

// Get returns debuginfo for given buildid while caching it in object storage.
func (c *DebuginfodClientObjectStorageCache) Get(ctx context.Context, buildID string) (io.ReadCloser, error) {
	rc, err := c.bucket.Get(ctx, objectPath(buildID))
	if err != nil {
		if c.bucket.IsObjNotFoundErr(err) {
			return c.getAndCache(ctx, buildID)
		}

		return nil, err
	}

	return rc, nil
}

func (c *DebuginfodClientObjectStorageCache) getAndCache(ctx context.Context, buildID string) (io.ReadCloser, error) {
	r, err := c.client.Get(ctx, buildID)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	if err := c.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
		level.Error(c.logger).Log("msg", "failed to upload downloaded debuginfod file", "err", err, "build_id", buildID)
	}

	r, err = c.bucket.Get(ctx, objectPath(buildID))
	if err != nil {
		return nil, err
	}

	return r, nil
}

// Exists returns true if debuginfo for given buildid exists.
func (c *DebuginfodClientObjectStorageCache) Exists(ctx context.Context, buildID string) (bool, error) {
	exists, err := c.bucket.Exists(ctx, objectPath(buildID))
	if err != nil {
		return false, err
	}

	if exists {
		return true, nil
	}

	return c.client.Exists(ctx, buildID)
}

// Get returns debug information file for given buildID by downloading it from upstream servers.
func (c *HTTPDebuginfodClient) Get(ctx context.Context, buildID string) (io.ReadCloser, error) {
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
	for _, u := range c.upstreamServers {
		rc, err := c.request(ctx, *u, buildID)
		if err != nil {
			continue
		}
		if rc != nil {
			return rc, nil
		}
	}
	return nil, ErrDebuginfoNotFound
}

func (c *HTTPDebuginfodClient) Exists(ctx context.Context, buildID string) (bool, error) {
	r, err := c.Get(ctx, buildID)
	if err != nil {
		if err == ErrDebuginfoNotFound {
			return false, nil
		}
		return false, err
	}

	return true, r.Close()
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

	return c.handleResponse(ctx, resp)
}

func (c *HTTPDebuginfodClient) handleResponse(ctx context.Context, resp *http.Response) (io.ReadCloser, error) {
	// Follow at most 2 redirects.
	for i := 0; i < 2; i++ {
		switch resp.StatusCode / 100 {
		case 2:
			return resp.Body, nil
		case 3:
			req, err := http.NewRequestWithContext(ctx, "GET", resp.Header.Get("Location"), nil)
			if err != nil {
				return nil, fmt.Errorf("create request: %w", err)
			}

			resp, err = c.client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("request failed: %w", err)
			}

			continue
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

	return nil, errors.New("too many redirects")
}
