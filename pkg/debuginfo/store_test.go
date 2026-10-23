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

package debuginfo

import (
	"bytes"
	"context"
	"io"
	stdlog "log"
	"net"
	"testing"
	"time"

	"github.com/go-kit/log"
	"github.com/stretchr/testify/require"
	"github.com/thanos-io/objstore"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

type fakeDebuginfodClients struct {
	get       func(ctx context.Context, server, buildID string) (io.ReadCloser, error)
	getSource func(ctx context.Context, server, buildID, file string) (io.ReadCloser, error)
	exists    func(ctx context.Context, buildID string) ([]string, error)
}

func (f *fakeDebuginfodClients) Get(ctx context.Context, server, buildID string) (io.ReadCloser, error) {
	return f.get(ctx, server, buildID)
}

func (f *fakeDebuginfodClients) GetSource(ctx context.Context, server, buildID, file string) (io.ReadCloser, error) {
	return f.getSource(ctx, server, buildID, file)
}

func (f *fakeDebuginfodClients) Exists(ctx context.Context, buildID string) ([]string, error) {
	return f.exists(ctx, buildID)
}

func newFakeDebuginfodClientsWithItems(items map[string]io.ReadCloser) *fakeDebuginfodClients {
	return &fakeDebuginfodClients{
		get: func(ctx context.Context, server, buildid string) (io.ReadCloser, error) {
			item, ok := items[buildid]
			if !ok || server != "fake" {
				return nil, ErrDebuginfoNotFound
			}

			return item, nil
		},
		exists: func(ctx context.Context, buildid string) ([]string, error) {
			_, ok := items[buildid]
			if ok {
				return []string{"fake"}, nil
			}

			return nil, nil
		},
	}
}

func TestStore(t *testing.T) {
	t.Run("legacy", func(t *testing.T) { testStore(t, false) })
	t.Run("multiple uploads", func(t *testing.T) { testStore(t, true) })
}

func testStore(t *testing.T, multiple bool) {
	t.Helper()
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	logger := log.NewNopLogger()
	bucket := objstore.NewInMemBucket()

	metadata := NewObjectStoreMetadata(logger, bucket)
	s, err := NewStore(
		tracer,
		logger,
		metadata,
		bucket,
		newFakeDebuginfodClientsWithItems(map[string]io.ReadCloser{
			"deadbeef": io.NopCloser(bytes.NewBufferString("debuginfo1")),
		}),
		SignedUpload{
			Enabled: false,
		},
		time.Minute*15,
		1024*1024*1024,
	)
	require.NoError(t, err)

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	defer grpcServer.GracefulStop()
	debuginfopb.RegisterDebuginfoServiceServer(grpcServer, s)
	go func() {
		err := grpcServer.Serve(lis)
		if err != nil {
			stdlog.Fatalf("failed to serve: %v", err)
		}
	}()

	conn, err := grpc.NewClient(lis.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	defer conn.Close()

	debuginfoClient := debuginfopb.NewDebuginfoServiceClient(conn)
	grpcUploadClient := NewGrpcUploadClient(debuginfoClient)

	b := bytes.NewBuffer(nil)
	for i := 0; i < 1024; i++ {
		b.Write([]byte("a"))
	}
	for i := 0; i < 1024; i++ {
		b.Write([]byte("b"))
	}
	for i := 0; i < 1024; i++ {
		b.Write([]byte("c"))
	}

	// Totally wrong order of upload protocol sequence.
	_, err = grpcUploadClient.Upload(ctx, &debuginfopb.UploadInstructions{BuildId: "abcd"}, bytes.NewReader(b.Bytes()))
	require.EqualError(t, err, "rpc error: code = FailedPrecondition desc = metadata not found, this indicates that the upload was not previously initiated")

	// Simulate we initiated this upload 30 minutes ago.
	s.timeNow = func() time.Time { return time.Now().Add(-30 * time.Minute) }

	shouldInitiateResp, err := debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd"})
	require.NoError(t, err)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonFirstTimeSeen, shouldInitiateResp.Reason)

	_, err = debuginfoClient.InitiateUpload(ctx, &debuginfopb.InitiateUploadRequest{
		BuildId: "abcd",
		Hash:    "foo",
		Size:    2,
	})
	require.NoError(t, err)

	// An upload is already in progress. So we should not initiate another one.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd"})
	require.NoError(t, err)
	require.False(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonUploadInProgress, shouldInitiateResp.Reason)
	require.True(t, shouldInitiateResp.RetryRequired)

	// But with force=true, we should be able to restart an ongoing upload.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd", Force: true})
	require.NoError(t, err)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonUploadInProgressButForced, shouldInitiateResp.Reason)

	// Set time to current time, where the upload should be expired. So we can initiate a new one.
	s.timeNow = time.Now

	// Correct upload flow.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd"})
	require.NoError(t, err)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonUploadStale, shouldInitiateResp.Reason)

	initiateResp, err := debuginfoClient.InitiateUpload(ctx, &debuginfopb.InitiateUploadRequest{
		BuildId:                 "abcd",
		Hash:                    "foo",
		Size:                    2,
		SupportsMultipleUploads: multiple,
	})
	require.NoError(t, err)
	require.False(t, initiateResp.RetryRequired)

	instructions := initiateResp.UploadInstructions
	if multiple {
		require.Nil(t, instructions)
		require.Len(t, initiateResp.UploadTargets, 1)
		instructions = initiateResp.UploadTargets[0]
	} else {
		require.Empty(t, initiateResp.UploadTargets)
	}
	require.NotNil(t, instructions)

	size, err := grpcUploadClient.Upload(ctx, instructions, bytes.NewReader(b.Bytes()))
	require.NoError(t, err)
	require.Equal(t, 3072, int(size))

	_, err = debuginfoClient.MarkUploadFinished(ctx, &debuginfopb.MarkUploadFinishedRequest{BuildId: "abcd", UploadId: instructions.UploadId})
	require.NoError(t, err)

	obj, err := s.bucket.Get(ctx, "abcd/debuginfo")
	require.NoError(t, err)

	content, err := io.ReadAll(obj)
	require.NoError(t, err)
	require.Equal(t, 3072, len(content))
	require.Equal(t, b.Bytes(), content)

	// Uploads should not be asked to be initiated again since so far there is
	// nothing wrong with the upload. It uploaded successfully and is not
	// marked invalid.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd"})
	require.NoError(t, err)
	require.False(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonDebuginfoAlreadyExists, shouldInitiateResp.Reason)
	require.False(t, shouldInitiateResp.RetryRequired)

	// But with force=true, we should be able to re-upload existing valid debuginfo.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd", Force: true})
	require.NoError(t, err)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)
	require.Equal(t, ReasonDebuginfoAlreadyExistsButForced, shouldInitiateResp.Reason)

	// If asynchronously we figured out the debuginfo was not a valid ELF file,
	// we should allow uploading something else. Don't test the whole upload
	// flow again, just the ShouldInitiateUpload part.
	require.NoError(t, metadata.SetQuality(ctx, "abcd", debuginfopb.DebuginfoType_DEBUGINFO_TYPE_DEBUGINFO_UNSPECIFIED, &debuginfopb.DebuginfoQuality{NotValidElf: true}))
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "abcd"})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfoInvalid, shouldInitiateResp.Reason)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)

	// But we won't accept it if the hash is the same.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{
		BuildId: "abcd",
		Hash:    "foo",
	})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfoEqual, shouldInitiateResp.Reason)
	require.False(t, shouldInitiateResp.ShouldInitiateUpload)

	// An initiation request would error.
	_, err = debuginfoClient.InitiateUpload(ctx, &debuginfopb.InitiateUploadRequest{
		BuildId: "abcd",
		Hash:    "foo",
		Size:    2,
	})
	require.EqualError(t, err, "rpc error: code = AlreadyExists desc = Debuginfo already exists and is marked as invalid, but the proposed hash is the same as the one already available, therefore the upload is not accepted as it would result in the same invalid debuginfos.")

	// If the hash is different, we will accept it.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{
		BuildId: "abcd",
		Hash:    "bar",
	})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfoNotEqual, shouldInitiateResp.Reason)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)

	// The debuginfod client should be able to fetch the debuginfo, therefore don't allow uploading.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "deadbeef"})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfoInDebuginfod, shouldInitiateResp.Reason)
	require.False(t, shouldInitiateResp.ShouldInitiateUpload)

	// This stays the same, but the debuginfod client should be able to fetch the debuginfo.
	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "deadbeef"})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfodSource, shouldInitiateResp.Reason)
	require.False(t, shouldInitiateResp.ShouldInitiateUpload)

	// If we mark the debuginfo as invalid, we should allow uploading.
	require.NoError(t, metadata.SetQuality(ctx, "deadbeef", debuginfopb.DebuginfoType_DEBUGINFO_TYPE_DEBUGINFO_UNSPECIFIED, &debuginfopb.DebuginfoQuality{NotValidElf: true}))

	shouldInitiateResp, err = debuginfoClient.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{BuildId: "deadbeef"})
	require.NoError(t, err)
	require.Equal(t, ReasonDebuginfodInvalid, shouldInitiateResp.Reason)
	require.True(t, shouldInitiateResp.ShouldInitiateUpload)
}

type fakeSignedUploadClient struct {
	url string
}

func (f fakeSignedUploadClient) SignedPUT(context.Context, string, int64, time.Time) (string, error) {
	return f.url, nil
}

func TestStoreInitiateSignedUpload(t *testing.T) {
	for _, multiple := range []bool{false, true} {
		name := "legacy"
		if multiple {
			name = "multiple uploads"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			logger := log.NewNopLogger()
			bucket := objstore.NewInMemBucket()
			metadata := NewObjectStoreMetadata(logger, bucket)
			const url = "https://storage.example.com/signed-upload"
			s, err := NewStore(
				noop.NewTracerProvider().Tracer(""), logger, metadata, bucket,
				newFakeDebuginfodClientsWithItems(nil),
				SignedUpload{Enabled: true, Client: fakeSignedUploadClient{url: url}},
				15*time.Minute, 1024,
			)
			require.NoError(t, err)

			resp, err := s.InitiateUpload(ctx, &debuginfopb.InitiateUploadRequest{
				BuildId:                 "abcd",
				Hash:                    "hash",
				Size:                    42,
				Type:                    debuginfopb.DebuginfoType_DEBUGINFO_TYPE_EXECUTABLE,
				SupportsMultipleUploads: multiple,
			})
			require.NoError(t, err)
			require.False(t, resp.RetryRequired)
			instructions := resp.UploadInstructions
			if multiple {
				require.Nil(t, instructions)
				require.Len(t, resp.UploadTargets, 1)
				instructions = resp.UploadTargets[0]
			} else {
				require.Empty(t, resp.UploadTargets)
			}
			require.NotNil(t, instructions)
			require.Equal(t, "abcd", instructions.BuildId)
			require.NotEmpty(t, instructions.UploadId)
			require.Equal(t, debuginfopb.UploadInstructions_UPLOAD_STRATEGY_SIGNED_URL, instructions.UploadStrategy)
			require.Equal(t, url, instructions.SignedUrl)
			require.Equal(t, debuginfopb.DebuginfoType_DEBUGINFO_TYPE_EXECUTABLE, instructions.Type)
		})
	}
}
