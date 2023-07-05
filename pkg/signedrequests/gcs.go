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
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"cloud.google.com/go/storage"
	"github.com/thanos-io/objstore/providers/gcs"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type GCSClient struct {
	bucket *storage.BucketHandle
	closer io.Closer
}

func NewGCSClient(ctx context.Context, conf []byte) (*GCSClient, error) {
	var gc gcs.Config
	if err := yaml.Unmarshal(conf, &gc); err != nil {
		return nil, err
	}

	return NewGCSBucketWithConfig(ctx, gc)
}

func NewGCSBucketWithConfig(ctx context.Context, gc gcs.Config) (*GCSClient, error) {
	if gc.Bucket == "" {
		return nil, errors.New("missing Google Cloud Storage bucket name for stored blocks")
	}

	var opts []option.ClientOption

	// If ServiceAccount is provided, use them in GCS client, otherwise fallback to Google default logic.
	if gc.ServiceAccount != "" {
		credentials, err := google.CredentialsFromJSON(ctx, []byte(gc.ServiceAccount), storage.ScopeFullControl)
		if err != nil {
			return nil, fmt.Errorf("create credentials from JSON: %w", err)
		}
		opts = append(opts, option.WithCredentials(credentials))
	}

	opts = append(opts, option.WithUserAgent("parca"))

	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &GCSClient{
		bucket: gcsClient.Bucket(gc.Bucket),
		closer: gcsClient,
	}, nil
}

func (c *GCSClient) Close() error {
	return c.closer.Close()
}

func (c *GCSClient) SignedPUT(
	ctx context.Context,
	objectKey string,
	size int64,
	expiry time.Time,
) (string, error) {
	return c.bucket.SignedURL(objectKey, &storage.SignedURLOptions{
		Method:  "PUT",
		Expires: expiry,
		Headers: []string{
			"X-Upload-Content-Length:" + strconv.FormatInt(size, 10),
		},
	})
}

func (c *GCSClient) SignedGET(
	ctx context.Context,
	objectKey string,
	expiry time.Time,
) (string, error) {
	return c.bucket.SignedURL(objectKey, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: expiry,
	})
}
