// Copyright 2022-2026 The Parca Authors
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
	"net/url"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/thanos-io/objstore/providers/s3"
	"gopkg.in/yaml.v3"
)

type S3Client struct {
	minioClient *minio.Client
	bucketName  string
	closer      io.Closer
}

func NewS3Client(ctx context.Context, conf []byte) (*S3Client, error) {
	var s3c s3.Config
	if err := yaml.Unmarshal(conf, &s3c); err != nil {
		return nil, err
	}

	return NewS3BucketWithConfig(ctx, s3c)
}

func NewS3BucketWithConfig(ctx context.Context, s3c s3.Config) (*S3Client, error) {
	if s3c.Bucket == "" {
		return nil, errors.New("missing S3 bucket name for stored blocks")
	}

	// Setup credentials
	var creds *credentials.Credentials
	if s3c.AccessKey != "" && s3c.SecretKey != "" {
		creds = credentials.NewStaticV4(s3c.AccessKey, s3c.SecretKey, s3c.SessionToken)
	} else {
		// Use default credential chain (IAM roles, env vars, etc.)
		creds = credentials.NewIAM("")
	}

	// Setup endpoint
	endpoint := s3c.Endpoint
	if endpoint == "" {
		// Default to AWS S3 endpoint for the region
		if s3c.Region != "" {
			endpoint = fmt.Sprintf("s3.%s.amazonaws.com", s3c.Region)
		} else {
			endpoint = "s3.amazonaws.com"
		}
	}

	// Create minio client
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:        creds,
		Secure:       !s3c.Insecure,
		Region:       s3c.Region,
		BucketLookup: s3c.BucketLookupType.MinioType(),
	})
	if err != nil {
		return nil, fmt.Errorf("create minio client: %w", err)
	}

	return &S3Client{
		minioClient: minioClient,
		bucketName:  s3c.Bucket,
		closer:      nil, // minio client doesn't need explicit closing
	}, nil
}

func (c *S3Client) Close() error {
	// AWS session doesn't have a Close method, but we implement it for consistency
	// with the Client interface
	return nil
}

func (c *S3Client) SignedPUT(
	ctx context.Context,
	objectKey string,
	size int64,
	expiry time.Time,
) (string, error) {
	duration := time.Until(expiry)
	if duration <= 0 {
		return "", errors.New("expiry time must be in the future")
	}

	// Create presigned URL for PUT operation
	reqParams := make(url.Values)
	reqParams.Set("Content-Length", fmt.Sprintf("%d", size))

	presignedURL, err := c.minioClient.PresignedPutObject(ctx, c.bucketName, objectKey, duration)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned PUT URL: %w", err)
	}

	return presignedURL.String(), nil
}

func (c *S3Client) SignedGET(
	ctx context.Context,
	objectKey string,
	expiry time.Time,
) (string, error) {
	duration := time.Until(expiry)
	if duration <= 0 {
		return "", errors.New("expiry time must be in the future")
	}

	// Create presigned URL for GET operation
	reqParams := make(url.Values)

	presignedURL, err := c.minioClient.PresignedGetObject(ctx, c.bucketName, objectKey, duration, reqParams)
	if err != nil {
		return "", fmt.Errorf("failed to generate presigned GET URL: %w", err)
	}

	return presignedURL.String(), nil
}
