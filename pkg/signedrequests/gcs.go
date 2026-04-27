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
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/storage"
	"github.com/thanos-io/objstore/providers/gcs"
	"golang.org/x/oauth2/google"
	iamcredentials "google.golang.org/api/iamcredentials/v1"
	"google.golang.org/api/option"
	"gopkg.in/yaml.v3"
)

type GCSClient struct {
	bucket *storage.BucketHandle
	closer io.Closer

	// googleAccessID is the service account email used to sign URLs. Detected
	// once at construction time; passing it explicitly into SignedURLOptions
	// lets us also override SignBytes, which is what enables trace-context
	// propagation into the IAM SignBlob call.
	googleAccessID string

	// privateKey is set when the service account JSON contains one, enabling
	// local signing without a round-trip to IAM.
	privateKey []byte

	// iamService is used for remote signing via IAM SignBlob when no private
	// key is available. It is context-aware, so spans emitted by its HTTP
	// transport are parented to the caller's trace.
	iamService *iamcredentials.Service
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

	var (
		creds *google.Credentials
		err   error
	)
	if gc.ServiceAccount != "" {
		creds, err = google.CredentialsFromJSONWithType(ctx, []byte(gc.ServiceAccount), google.ServiceAccount, storage.ScopeFullControl)
	} else {
		creds, err = google.FindDefaultCredentials(ctx, storage.ScopeFullControl)
	}
	if err != nil {
		return nil, fmt.Errorf("load credentials: %w", err)
	}

	opts := []option.ClientOption{
		option.WithCredentials(creds),
		option.WithUserAgent("parca"),
	}

	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, err
	}

	googleAccessID, privateKey, err := signingIdentity(ctx, creds)
	if err != nil {
		return nil, fmt.Errorf("resolve signing identity: %w", err)
	}

	c := &GCSClient{
		bucket:         gcsClient.Bucket(gc.Bucket),
		closer:         gcsClient,
		googleAccessID: googleAccessID,
		privateKey:     privateKey,
	}

	if len(privateKey) == 0 {
		iamService, err := iamcredentials.NewService(ctx, opts...)
		if err != nil {
			return nil, fmt.Errorf("create iamcredentials service: %w", err)
		}
		c.iamService = iamService
	}

	return c, nil
}

// signingIdentity returns the service account email and (optionally) a private
// key for URL signing. Mirrors storage.BucketHandle.detectDefaultGoogleAccessID,
// which is unexported — we need this surfaced because overriding SignBytes (to
// thread context through to IAM) means the library no longer fills in
// GoogleAccessID for our closure.
func signingIdentity(ctx context.Context, creds *google.Credentials) (string, []byte, error) {
	if len(creds.JSON) > 0 {
		var sa struct {
			ClientEmail string `json:"client_email"`
			PrivateKey  string `json:"private_key"`
		}
		if err := json.Unmarshal(creds.JSON, &sa); err == nil && sa.ClientEmail != "" {
			return sa.ClientEmail, []byte(sa.PrivateKey), nil
		}
	}

	if !metadata.OnGCE() {
		return "", nil, errors.New("could not resolve service account email from credentials JSON and not running on GCE")
	}
	email, err := metadata.EmailWithContext(ctx, "default")
	if err != nil {
		return "", nil, fmt.Errorf("read service account email from GCE metadata: %w", err)
	}
	if email == "" {
		return "", nil, errors.New("empty service account email from GCE metadata")
	}
	return email, nil, nil
}

// signBytesWithContext returns a SignBytes function that calls IAM SignBlob
// with the given context. Using our context here means the HTTP span emitted
// by the IAM client inherits the caller's trace.
func (c *GCSClient) signBytesWithContext(ctx context.Context) func([]byte) ([]byte, error) {
	return func(in []byte) ([]byte, error) {
		resp, err := c.iamService.Projects.ServiceAccounts.SignBlob(
			fmt.Sprintf("projects/-/serviceAccounts/%s", c.googleAccessID),
			&iamcredentials.SignBlobRequest{
				Payload: base64.StdEncoding.EncodeToString(in),
			},
		).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("iam sign blob: %w", err)
		}
		return base64.StdEncoding.DecodeString(resp.SignedBlob)
	}
}

func (c *GCSClient) signedURLOptions(ctx context.Context, base *storage.SignedURLOptions) *storage.SignedURLOptions {
	base.GoogleAccessID = c.googleAccessID
	if len(c.privateKey) > 0 {
		base.PrivateKey = c.privateKey
	} else {
		base.SignBytes = c.signBytesWithContext(ctx)
	}
	return base
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
	return c.bucket.SignedURL(objectKey, c.signedURLOptions(ctx, &storage.SignedURLOptions{
		Method:  "PUT",
		Expires: expiry,
		Headers: []string{
			"X-Upload-Content-Length:" + strconv.FormatInt(size, 10),
		},
	}))
}

func (c *GCSClient) SignedGET(
	ctx context.Context,
	objectKey string,
	expiry time.Time,
) (string, error) {
	return c.bucket.SignedURL(objectKey, c.signedURLOptions(ctx, &storage.SignedURLOptions{
		Method:  "GET",
		Expires: expiry,
	}))
}
