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
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math"
	"path"
	"time"

	"github.com/bufbuild/connect-go"
	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/types/known/timestamppb"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1/debuginfov1alpha1connect"
	"github.com/parca-dev/parca/pkg/signedupload"
)

var ErrDebuginfoNotFound = errors.New("debuginfo not found")

type CacheProvider string

const (
	FILESYSTEM CacheProvider = "FILESYSTEM"
)

type Config struct {
	Bucket *client.BucketConfig `yaml:"bucket"`
	Cache  *CacheConfig         `yaml:"cache"`
}

type FilesystemCacheConfig struct {
	Directory string `yaml:"directory"`
}

type CacheConfig struct {
	Type   CacheProvider `yaml:"type"`
	Config interface{}   `yaml:"config"`
}

type MetadataManager interface {
	MarkAsDebuginfodSource(ctx context.Context, buildID string) error
	MarkAsUploading(ctx context.Context, buildID, uploadID, hash string, startedAt *timestamppb.Timestamp) error
	MarkAsUploaded(ctx context.Context, buildID, uploadID string, finishedAt *timestamppb.Timestamp) error
	Fetch(ctx context.Context, buildID string) (*debuginfopb.Debuginfo, error)
}

type Store struct {
	debuginfov1alpha1connect.UnimplementedDebuginfoServiceHandler

	tracer trace.Tracer
	logger log.Logger

	bucket objstore.Bucket

	metadata         MetadataManager
	debuginfodClient DebuginfodClient

	signedUpload SignedUpload

	maxUploadDuration time.Duration
	maxUploadSize     int64

	timeNow func() time.Time
}

type SignedUpload struct {
	Enabled bool
	Client  signedupload.Client
}

// NewStore returns a new debug info store.
func NewStore(
	tracer trace.Tracer,
	logger log.Logger,
	metadata MetadataManager,
	bucket objstore.Bucket,
	debuginfodClient DebuginfodClient,
	signedUpload SignedUpload,
	maxUploadDuration time.Duration,
	maxUploadSize int64,
) (*Store, error) {
	return &Store{
		tracer:            tracer,
		logger:            log.With(logger, "component", "debuginfo"),
		bucket:            bucket,
		metadata:          metadata,
		debuginfodClient:  debuginfodClient,
		signedUpload:      signedUpload,
		maxUploadDuration: maxUploadDuration,
		maxUploadSize:     maxUploadSize,
		timeNow:           time.Now,
	}, nil
}

const (
	ReasonDebuginfoInDebuginfod  = "Debuginfo exists in debuginfod, therefore no upload is necessary."
	ReasonFirstTimeSeen          = "First time we see this Build ID, and it does not exist in debuginfod, therefore please upload!"
	ReasonUploadStale            = "A previous upload was started but not finished and is now stale, so it can be retried."
	ReasonUploadInProgress       = "A previous upload is still in-progress and not stale yet (only stale uploads can be retried)."
	ReasonDebuginfoAlreadyExists = "Debuginfo already exists and is not marked as invalid, therefore no new upload is needed."
	ReasonDebuginfoInvalid       = "Debuginfo already exists but is marked as invalid, therefore a new upload is needed. Hash the debuginfo and initiate the upload."
	ReasonDebuginfoEqual         = "Debuginfo already exists and is marked as invalid, but the proposed hash is the same as the one already available, therefore the upload is not accepted as it would result in the same invalid debuginfos."
	ReasonDebuginfoNotEqual      = "Debuginfo already exists but is marked as invalid, therefore a new upload will be accepted."
	ReasonDebuginfodSource       = "Debuginfo is available from debuginfod already and not marked as invalid, therefore no new upload is needed."
	ReasonDebuginfodInvalid      = "Debuginfo is available from debuginfod already but is marked as invalid, therefore a new upload is needed."
)

// ShouldInitiateUpload returns whether an upload should be initiated for the
// given build ID. Checking if an upload should even be initiated allows the
// parca-agent to avoid extracting debuginfos unnecessarily from a binary.
func (s *Store) ShouldInitiateUpload(ctx context.Context, req *connect.Request[debuginfopb.ShouldInitiateUploadRequest]) (*connect.Response[debuginfopb.ShouldInitiateUploadResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", req.Msg.BuildId))

	buildID := req.Msg.BuildId
	if err := validateInput(buildID); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	dbginfo, err := s.metadata.Fetch(ctx, buildID)
	if err != nil && !errors.Is(err, ErrMetadataNotFound) {
		return nil, connect.NewError(connect.CodeInternal, err)
	} else if errors.Is(err, ErrMetadataNotFound) {
		// First time we see this Build ID.

		existsInDebuginfod, err := s.debuginfodClient.Exists(ctx, buildID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		if existsInDebuginfod {
			if err := s.metadata.MarkAsDebuginfodSource(ctx, buildID); err != nil {
				return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("mark Build ID to be available from debuginfod: %w", err))
			}

			return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
				ShouldInitiateUpload: false,
				Reason:               ReasonDebuginfoInDebuginfod,
			}), nil
		}

		return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
			ShouldInitiateUpload: true,
			Reason:               ReasonFirstTimeSeen,
		}), nil
	} else {
		// We have seen this Build ID before and there is metadata for it.

		switch dbginfo.Source {
		case debuginfopb.Debuginfo_SOURCE_UPLOAD:
			if dbginfo.Upload == nil {
				return nil, connect.NewError(connect.CodeInternal, errors.New("metadata inconsistency: upload is nil"))
			}

			switch dbginfo.Upload.State {
			case debuginfopb.DebuginfoUpload_STATE_UPLOADING:
				if s.uploadIsStale(dbginfo.Upload) {
					return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: true,
						Reason:               ReasonUploadStale,
					}), nil
				}

				return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: false,
					Reason:               ReasonUploadInProgress,
				}), nil
			case debuginfopb.DebuginfoUpload_STATE_UPLOADED:
				if dbginfo.Quality == nil || !dbginfo.Quality.NotValidElf {
					return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: false,
						Reason:               ReasonDebuginfoAlreadyExists,
					}), nil
				}

				if req.Msg.Hash == "" {
					return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: true,
						Reason:               ReasonDebuginfoInvalid,
					}), nil
				}

				if dbginfo.Upload.Hash == req.Msg.Hash {
					return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: false,
						Reason:               ReasonDebuginfoEqual,
					}), nil
				}

				return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: true,
					Reason:               ReasonDebuginfoNotEqual,
				}), nil
			default:
				return nil, connect.NewError(connect.CodeInternal, errors.New("metadata inconsistency: unknown upload state"))
			}
		case debuginfopb.Debuginfo_SOURCE_DEBUGINFOD:
			if dbginfo.Quality == nil || !dbginfo.Quality.NotValidElf {
				// We already have debuginfo that's also not marked to be
				// invalid, so we don't need to upload it again.
				return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: false,
					Reason:               ReasonDebuginfodSource,
				}), nil
			}

			return connect.NewResponse(&debuginfopb.ShouldInitiateUploadResponse{
				ShouldInitiateUpload: true,
				Reason:               ReasonDebuginfodInvalid,
			}), nil
		default:
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unknown debuginfo source %q", dbginfo.Source))
		}
	}
}

func (s *Store) InitiateUpload(ctx context.Context, req *connect.Request[debuginfopb.InitiateUploadRequest]) (*connect.Response[debuginfopb.InitiateUploadResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", req.Msg.BuildId))

	if req.Msg.Hash == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("hash must be set"))
	}
	if req.Msg.Size == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("size must be set"))
	}

	// We don't want to blindly accept upload initiation requests that
	// shouldn't have happened.
	shouldInitiateResp, err := s.ShouldInitiateUpload(ctx, connect.NewRequest(&debuginfopb.ShouldInitiateUploadRequest{
		BuildId: req.Msg.BuildId,
		Hash:    req.Msg.Hash,
	}))
	if err != nil {
		return nil, err
	}
	if !shouldInitiateResp.Msg.ShouldInitiateUpload {
		if shouldInitiateResp.Msg.Reason == ReasonDebuginfoEqual {
			return nil, connect.NewError(connect.CodeAlreadyExists, errors.New(ReasonDebuginfoEqual))
		}
		return nil, connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("upload should not have been attempted to be initiated, a previous check should have failed with: %s", shouldInitiateResp.Msg.Reason))
	}

	if req.Msg.Size > s.maxUploadSize {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("upload size %d exceeds maximum allowed size %d", req.Msg.Size, s.maxUploadSize))
	}

	uploadID := uuid.New().String()
	uploadStarted := s.timeNow()
	uploadExpiry := uploadStarted.Add(s.maxUploadDuration)

	if !s.signedUpload.Enabled {
		if err := s.metadata.MarkAsUploading(ctx, req.Msg.BuildId, uploadID, req.Msg.Hash, timestamppb.New(uploadStarted)); err != nil {
			return nil, fmt.Errorf("mark debuginfo upload as uploading via gRPC: %w", err)
		}

		return connect.NewResponse(&debuginfopb.InitiateUploadResponse{
			UploadInstructions: &debuginfopb.UploadInstructions{
				BuildId:        req.Msg.BuildId,
				UploadId:       uploadID,
				UploadStrategy: debuginfopb.UploadInstructions_UPLOAD_STRATEGY_GRPC,
			},
		}), nil
	}

	signedURL, err := s.signedUpload.Client.SignedPUT(ctx, objectPath(req.Msg.BuildId), req.Msg.Size, uploadExpiry)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if err := s.metadata.MarkAsUploading(ctx, req.Msg.BuildId, uploadID, req.Msg.Hash, timestamppb.New(uploadStarted)); err != nil {
		return nil, fmt.Errorf("mark debuginfo upload as uploading via signed URL: %w", err)
	}

	return connect.NewResponse(&debuginfopb.InitiateUploadResponse{
		UploadInstructions: &debuginfopb.UploadInstructions{
			BuildId:        req.Msg.BuildId,
			UploadId:       uploadID,
			UploadStrategy: debuginfopb.UploadInstructions_UPLOAD_STRATEGY_SIGNED_URL,
			SignedUrl:      signedURL,
		},
	}), nil
}

func (s *Store) MarkUploadFinished(ctx context.Context, req *connect.Request[debuginfopb.MarkUploadFinishedRequest]) (*connect.Response[debuginfopb.MarkUploadFinishedResponse], error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", req.Msg.BuildId))
	span.SetAttributes(attribute.String("upload_id", req.Msg.UploadId))

	buildID := req.Msg.BuildId
	if err := validateInput(buildID); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	err := s.metadata.MarkAsUploaded(ctx, buildID, req.Msg.UploadId, timestamppb.New(s.timeNow()))
	if errors.Is(err, ErrDebuginfoNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no debuginfo metadata found for build id"))
	}
	if errors.Is(err, ErrUploadMetadataNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("no debuginfo upload metadata found for build id"))
	}
	if errors.Is(err, ErrUploadIDMismatch) {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("upload id mismatch"))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&debuginfopb.MarkUploadFinishedResponse{}), nil
}

func (s *Store) Upload(ctx context.Context, stream *connect.ClientStream[debuginfopb.UploadRequest]) (*connect.Response[debuginfopb.UploadResponse], error) {
	if s.signedUpload.Enabled {
		return nil, connect.NewError(connect.CodeUnimplemented, errors.New("signed URL uploads are the only supported upload strategy for this service"))
	}

	stream.Receive()
	if err := stream.Err(); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, err)
	}

	var (
		buildID  = stream.Msg().GetInfo().GetBuildId()
		uploadID = stream.Msg().GetInfo().UploadId
		r        = &UploadReader{stream: stream}
	)
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", buildID))
	span.SetAttributes(attribute.String("upload_id", uploadID))

	if err := s.upload(ctx, buildID, uploadID, r); err != nil {
		return nil, connect.NewError(connect.CodeUnknown, err)
	}

	return connect.NewResponse(&debuginfopb.UploadResponse{
		BuildId: buildID,
		Size:    math.MaxUint64, // TODO: return the size of the uploaded file
	}), nil
}

func (s *Store) upload(ctx context.Context, buildID, uploadID string, r io.Reader) error {
	if err := validateInput(buildID); err != nil {
		return connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid build ID: %q", err))
	}

	dbginfo, err := s.metadata.Fetch(ctx, buildID)
	if err != nil {
		if errors.Is(err, ErrMetadataNotFound) {
			return connect.NewError(connect.CodeFailedPrecondition, errors.New("metadata not found, this indicates that the upload was not previously initiated"))
		}
		return connect.NewError(connect.CodeInternal, err)
	}

	if dbginfo.Upload == nil {
		return connect.NewError(connect.CodeFailedPrecondition, errors.New("metadata not found, this indicates that the upload was not previously initiated"))
	}

	if dbginfo.Upload.Id != uploadID {
		return connect.NewError(connect.CodeInvalidArgument, errors.New("the upload ID does not match the one returned by the InitiateUpload call"))
	}

	if err := s.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("upload debuginfo: %w", err))
	}

	return nil
}

func (s *Store) uploadIsStale(upload *debuginfopb.DebuginfoUpload) bool {
	return upload.StartedAt.AsTime().Add(s.maxUploadDuration + 2*time.Minute).Before(s.timeNow())
}

func validateInput(id string) error {
	_, err := hex.DecodeString(id)
	if err != nil {
		return fmt.Errorf("failed to validate input: %w", err)
	}
	if len(id) <= 2 {
		return errors.New("unexpectedly short input")
	}

	return nil
}

func objectPath(buildID string) string {
	return path.Join(buildID, "debuginfo")
}
