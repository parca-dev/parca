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
	"context"
	"errors"
	"io"

	"github.com/thanos-io/objstore"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

var (
	ErrUnknownDebuginfoSource = errors.New("unknown debuginfo source")
	ErrNotUploadedYet         = errors.New("debuginfo not uploaded yet")
)

type Fetcher struct {
	debuginfodClient DebuginfodClient
	bucket           objstore.Bucket
}

func NewFetcher(
	debuginfodClient DebuginfodClient,
	bucket objstore.Bucket,
) *Fetcher {
	return &Fetcher{
		debuginfodClient: debuginfodClient,
		bucket:           bucket,
	}
}

func (f *Fetcher) FetchDebuginfo(ctx context.Context, dbginfo *debuginfopb.Debuginfo) (io.ReadCloser, error) {
	switch dbginfo.Source {
	case debuginfopb.Debuginfo_SOURCE_UPLOAD:
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
