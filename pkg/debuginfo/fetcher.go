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
	"context"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/thanos-io/objstore"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

var ErrUnknownDebuginfoSource = errors.New("unknown debuginfo source")

type Fetcher struct {
	metadata         MetadataManager
	debuginfodClient DebuginfodClient
	bucket           objstore.Bucket
}

func NewFetcher(
	metadata MetadataManager,
	debuginfodClient DebuginfodClient,
	bucket objstore.Bucket,
) *Fetcher {
	return &Fetcher{
		metadata:         metadata,
		debuginfodClient: debuginfodClient,
		bucket:           bucket,
	}
}

func (f *Fetcher) FetchDebuginfo(ctx context.Context, buildid string) (io.ReadCloser, error) {
	dbginfo, err := f.metadata.Fetch(ctx, buildid)
	if err != nil {
		return nil, fmt.Errorf("fetching metadata: %w", err)
	}

	switch dbginfo.Source {
	case debuginfopb.Debuginfo_SOURCE_UPLOAD:
		if dbginfo.Upload.State != debuginfopb.DebuginfoUpload_STATE_UPLOADED {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()

			ctx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()
			for {
				select {
				case <-ctx.Done():
					return nil, errors.New("timed out waiting for upload to finish")
				default:
				}

				dbginfo, err = f.metadata.Fetch(ctx, buildid)
				if err != nil {
					return nil, fmt.Errorf("fetching metadata: %w", err)
				}

				if dbginfo.Upload.State == debuginfopb.DebuginfoUpload_STATE_UPLOADED {
					break
				}
			}
		}

		return f.fetchFromBucket(ctx, dbginfo)
	case debuginfopb.Debuginfo_SOURCE_DEBUGINFOD:
		return f.fetchFromDebuginfod(ctx, dbginfo)
	default:
		return nil, ErrUnknownDebuginfoSource
	}
}

func (f *Fetcher) fetchFromBucket(ctx context.Context, dbginfo *debuginfopb.Debuginfo) (io.ReadCloser, error) {
	return f.bucket.Get(ctx, objectPath(dbginfo.BuildId))
}

func (f *Fetcher) fetchFromDebuginfod(ctx context.Context, dbginfo *debuginfopb.Debuginfo) (io.ReadCloser, error) {
	return f.debuginfodClient.Get(ctx, dbginfo.BuildId)
}
