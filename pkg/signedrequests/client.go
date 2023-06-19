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

package signedrequests

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/thanos-io/objstore/client"
	"gopkg.in/yaml.v3"
)

// DirDelim is the delimiter used to model a directory structure in an object store bucket.
const DirDelim = "/"

type ErrUnsupportedProvider struct {
	Provider client.ObjProvider
}

func (e ErrUnsupportedProvider) Error() string {
	return "provider not supported (only GCS is currently supported): " + string(e.Provider)
}

type Client interface {
	io.Closer
	SignedPUT(
		ctx context.Context,
		objectKey string,
		size int64,
		expiry time.Time,
	) (string, error)
	SignedGET(
		ctx context.Context,
		objectKey string,
		expiry time.Time,
	) (string, error)
}

func NewClient(ctx context.Context, bucketConf *client.BucketConfig) (Client, error) {
	if bucketConf.Type != client.GCS {
		return nil, ErrUnsupportedProvider{Provider: bucketConf.Type}
	}

	config, err := yaml.Marshal(bucketConf.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bucket config: %w", err)
	}

	c, err := NewGCSClient(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCS client: %w", err)
	}

	return NewPrefixedClient(c, bucketConf.Prefix), nil
}

func NewPrefixedClient(client Client, prefix string) Client {
	if validPrefix(prefix) {
		return &PrefixedClient{client: client, prefix: strings.Trim(prefix, DirDelim)}
	}

	return client
}

type PrefixedClient struct {
	client Client
	prefix string
}

func (c *PrefixedClient) SignedPUT(
	ctx context.Context,
	objectKey string,
	size int64,
	expiry time.Time,
) (string, error) {
	return c.client.SignedPUT(ctx, conditionalPrefix(c.prefix, objectKey), size, expiry)
}

func (c *PrefixedClient) SignedGET(
	ctx context.Context,
	objectKey string,
	expiry time.Time,
) (string, error) {
	return c.client.SignedGET(ctx, conditionalPrefix(c.prefix, objectKey), expiry)
}

func (c *PrefixedClient) Close() error {
	return c.client.Close()
}

func validPrefix(prefix string) bool {
	prefix = strings.Replace(prefix, "/", "", -1)
	return len(prefix) > 0
}

func conditionalPrefix(prefix, name string) string {
	if len(name) > 0 {
		return withPrefix(prefix, name)
	}

	return name
}

func withPrefix(prefix, name string) string {
	return prefix + DirDelim + name
}
