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
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/thanos-io/thanos/pkg/objstore"
	"golang.org/x/net/context"
)

type DebugInfodClient interface {
	GetDebugInfo(buildid string) (io.ReadCloser, error)
}

type HttpDebuginfodClient struct {
	UpstreamServer *url.URL
}
type ObjectStorageDebugInfodClientCache struct {
	ctx    context.Context
	logger log.Logger
	client DebugInfodClient
	bucket objstore.Bucket
}

func NewHttpDebugInfoClient(serverUrl string) (*HttpDebuginfodClient, error) {
	parsedUrl, err := url.Parse(serverUrl)
	if err != nil {
		return nil, err
	}
	return &HttpDebuginfodClient{UpstreamServer: parsedUrl}, nil
}

func NewObjectStorageDebugInfodClientCache(ctx context.Context, logger log.Logger, h *HttpDebuginfodClient) *ObjectStorageDebugInfodClientCache {
	return &ObjectStorageDebugInfodClientCache{
		ctx:    ctx,
		logger: logger,
		client: h,
	}
}

func (c *ObjectStorageDebugInfodClientCache) GetDebugInfo(buildId string) (io.ReadCloser, error) {
	debugInfo, err := c.client.GetDebugInfo(buildId)
	if err != nil {
		return nil, ErrDebugInfoNotFound
	}

	path := buildId + "/debuginfo"

	err = c.bucket.Upload(c.ctx, path, debugInfo)
	if err != nil {
		level.Error(c.logger).Log("msg", "failed to upload to debuginfod cache", "err", err)
		return nil, err
	}
	return debugInfo, nil
}

func (c *HttpDebuginfodClient) GetDebugInfo(buildID string) (io.ReadCloser, error) {
	buildIdUrl := *c.UpstreamServer
	buildIdUrl.Path = path.Join(buildIdUrl.Path, buildID, "debuginfo")

	resp, err := http.Get(buildIdUrl.String())
	if err != nil {
		//level.Debug(logger).Log("msg", "object not found in public server", "object", buildID, "err", err)
		return nil, ErrDebugInfoNotFound
	}

	return resp.Body, nil
}
