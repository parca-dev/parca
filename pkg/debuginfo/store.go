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
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/google/uuid"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
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
	debuginfopb.UnimplementedDebuginfoServiceServer

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

type SignedUploadClient interface {
	SignedPUT(ctx context.Context, objectKey string, size int64, expiry time.Time) (signedURL string, err error)
}

type SignedUpload struct {
	Enabled bool
	Client  SignedUploadClient
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
func (s *Store) ShouldInitiateUpload(ctx context.Context, req *debuginfopb.ShouldInitiateUploadRequest) (*debuginfopb.ShouldInitiateUploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "ShouldInitiateUpload")
	defer span.End()
	span.SetAttributes(attribute.String("build_id", req.BuildId))

	buildID := req.BuildId
	if err := validateInput(buildID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	dbginfo, err := s.metadata.Fetch(ctx, buildID)
	if err != nil && !errors.Is(err, ErrMetadataNotFound) {
		return nil, status.Error(codes.Internal, err.Error())
	} else if errors.Is(err, ErrMetadataNotFound) {
		// First time we see this Build ID.

		existsInDebuginfod, err := s.debuginfodClient.Exists(ctx, buildID)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		}

		if existsInDebuginfod {
			if err := s.metadata.MarkAsDebuginfodSource(ctx, buildID); err != nil {
				return nil, status.Error(codes.Internal, fmt.Errorf("mark Build ID to be available from debuginfod: %w", err).Error())
			}

			return &debuginfopb.ShouldInitiateUploadResponse{
				ShouldInitiateUpload: false,
				Reason:               ReasonDebuginfoInDebuginfod,
			}, nil
		}

		return &debuginfopb.ShouldInitiateUploadResponse{
			ShouldInitiateUpload: true,
			Reason:               ReasonFirstTimeSeen,
		}, nil
	} else {
		// We have seen this Build ID before and there is metadata for it.

		switch dbginfo.Source {
		case debuginfopb.Debuginfo_SOURCE_UPLOAD:
			if dbginfo.Upload == nil {
				return nil, status.Error(codes.Internal, "metadata inconsistency: upload is nil")
			}

			switch dbginfo.Upload.State {
			case debuginfopb.DebuginfoUpload_STATE_UPLOADING:
				if s.uploadIsStale(dbginfo.Upload) {
					return &debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: true,
						Reason:               ReasonUploadStale,
					}, nil
				}

				return &debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: false,
					Reason:               ReasonUploadInProgress,
				}, nil
			case debuginfopb.DebuginfoUpload_STATE_UPLOADED:
				if dbginfo.Quality == nil || !dbginfo.Quality.NotValidElf {
					return &debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: false,
						Reason:               ReasonDebuginfoAlreadyExists,
					}, nil
				}

				if req.Hash == "" {
					return &debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: true,
						Reason:               ReasonDebuginfoInvalid,
					}, nil
				}

				if dbginfo.Upload.Hash == req.Hash {
					return &debuginfopb.ShouldInitiateUploadResponse{
						ShouldInitiateUpload: false,
						Reason:               ReasonDebuginfoEqual,
					}, nil
				}

				return &debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: true,
					Reason:               ReasonDebuginfoNotEqual,
				}, nil
			default:
				return nil, status.Error(codes.Internal, "metadata inconsistency: unknown upload state")
			}
		case debuginfopb.Debuginfo_SOURCE_DEBUGINFOD:
			if dbginfo.Quality == nil || !dbginfo.Quality.NotValidElf {
				// We already have debuginfo that's also not marked to be
				// invalid, so we don't need to upload it again.
				return &debuginfopb.ShouldInitiateUploadResponse{
					ShouldInitiateUpload: false,
					Reason:               ReasonDebuginfodSource,
				}, nil
			}

			return &debuginfopb.ShouldInitiateUploadResponse{
				ShouldInitiateUpload: true,
				Reason:               ReasonDebuginfodInvalid,
			}, nil
		default:
			return nil, status.Errorf(codes.Internal, "unknown debuginfo source %q", dbginfo.Source)
		}
	}
}

func (s *Store) InitiateUpload(ctx context.Context, req *debuginfopb.InitiateUploadRequest) (*debuginfopb.InitiateUploadResponse, error) {
	ctx, span := s.tracer.Start(ctx, "InitiateUpload")
	defer span.End()
	span.SetAttributes(attribute.String("build_id", req.BuildId))

	if req.Hash == "" {
		return nil, status.Error(codes.InvalidArgument, "hash must be set")
	}
	if req.Size == 0 {
		return nil, status.Error(codes.InvalidArgument, "size must be set")
	}

	// We don't want to blindly accept upload initiation requests that
	// shouldn't have happened.
	shouldInitiateResp, err := s.ShouldInitiateUpload(ctx, &debuginfopb.ShouldInitiateUploadRequest{
		BuildId: req.BuildId,
		Hash:    req.Hash,
	})
	if err != nil {
		return nil, err
	}
	if !shouldInitiateResp.ShouldInitiateUpload {
		if shouldInitiateResp.Reason == ReasonDebuginfoEqual {
			return nil, status.Error(codes.AlreadyExists, ReasonDebuginfoEqual)
		}
		return nil, status.Errorf(codes.FailedPrecondition, "upload should not have been attempted to be initiated, a previous check should have failed with: %s", shouldInitiateResp.Reason)
	}

	if req.Size > s.maxUploadSize {
		return nil, status.Errorf(codes.InvalidArgument, "upload size %d exceeds maximum allowed size %d", req.Size, s.maxUploadSize)
	}

	uploadID := uuid.New().String()
	uploadStarted := s.timeNow()
	uploadExpiry := uploadStarted.Add(s.maxUploadDuration)

	if !s.signedUpload.Enabled {
		if err := s.metadata.MarkAsUploading(ctx, req.BuildId, uploadID, req.Hash, timestamppb.New(uploadStarted)); err != nil {
			return nil, fmt.Errorf("mark debuginfo upload as uploading via gRPC: %w", err)
		}

		return &debuginfopb.InitiateUploadResponse{
			UploadInstructions: &debuginfopb.UploadInstructions{
				BuildId:        req.BuildId,
				UploadId:       uploadID,
				UploadStrategy: debuginfopb.UploadInstructions_UPLOAD_STRATEGY_GRPC,
			},
		}, nil
	}

	signedURL, err := s.signedUpload.Client.SignedPUT(ctx, objectPath(req.BuildId), req.Size, uploadExpiry)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err := s.metadata.MarkAsUploading(ctx, req.BuildId, uploadID, req.Hash, timestamppb.New(uploadStarted)); err != nil {
		return nil, fmt.Errorf("mark debuginfo upload as uploading via signed URL: %w", err)
	}

	return &debuginfopb.InitiateUploadResponse{
		UploadInstructions: &debuginfopb.UploadInstructions{
			BuildId:        req.BuildId,
			UploadId:       uploadID,
			UploadStrategy: debuginfopb.UploadInstructions_UPLOAD_STRATEGY_SIGNED_URL,
			SignedUrl:      signedURL,
		},
	}, nil
}

func (s *Store) MarkUploadFinished(ctx context.Context, req *debuginfopb.MarkUploadFinishedRequest) (*debuginfopb.MarkUploadFinishedResponse, error) {
	ctx, span := s.tracer.Start(ctx, "MarkUploadFinished")
	defer span.End()
	span.SetAttributes(attribute.String("build_id", req.BuildId))
	span.SetAttributes(attribute.String("upload_id", req.UploadId))

	buildID := req.BuildId
	if err := validateInput(buildID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	err := s.metadata.MarkAsUploaded(ctx, buildID, req.UploadId, timestamppb.New(s.timeNow()))
	if errors.Is(err, ErrDebuginfoNotFound) {
		return nil, status.Error(codes.NotFound, "no debuginfo metadata found for build id")
	}
	if errors.Is(err, ErrUploadMetadataNotFound) {
		return nil, status.Error(codes.NotFound, "no debuginfo upload metadata found for build id")
	}
	if errors.Is(err, ErrUploadIDMismatch) {
		return nil, status.Error(codes.InvalidArgument, "upload id mismatch")
	}
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &debuginfopb.MarkUploadFinishedResponse{}, nil
}

func (s *Store) Upload(stream debuginfopb.DebuginfoService_UploadServer) error {
	if s.signedUpload.Enabled {
		return status.Error(codes.Unimplemented, "signed URL uploads are the only supported upload strategy for this service")
	}

	req, err := stream.Recv()
	if err != nil {
		return status.Errorf(codes.Unknown, "failed to receive upload info: %q", err)
	}

	var (
		buildID  = req.GetInfo().BuildId
		uploadID = req.GetInfo().UploadId
		r        = &UploadReader{stream: stream}
	)

	ctx, span := s.tracer.Start(stream.Context(), "Upload")
	defer span.End()
	span.SetAttributes(attribute.String("build_id", buildID))
	span.SetAttributes(attribute.String("upload_id", uploadID))

	if err := s.upload(ctx, buildID, uploadID, r); err != nil {
		return err
	}

	return stream.SendAndClose(&debuginfopb.UploadResponse{
		BuildId: buildID,
		Size:    r.size,
	})
}

func (s *Store) upload(ctx context.Context, buildID, uploadID string, r io.Reader) error {
	if err := validateInput(buildID); err != nil {
		return status.Errorf(codes.InvalidArgument, "invalid build ID: %q", err)
	}

	dbginfo, err := s.metadata.Fetch(ctx, buildID)
	if err != nil {
		if errors.Is(err, ErrMetadataNotFound) {
			return status.Error(codes.FailedPrecondition, "metadata not found, this indicates that the upload was not previously initiated")
		}
		return status.Error(codes.Internal, err.Error())
	}

	if dbginfo.Upload == nil {
		return status.Error(codes.FailedPrecondition, "metadata not found, this indicates that the upload was not previously initiated")
	}

	if dbginfo.Upload.Id != uploadID {
		return status.Error(codes.InvalidArgument, "the upload ID does not match the one returned by the InitiateUpload call")
	}

	if err := s.bucket.Upload(ctx, objectPath(buildID), r); err != nil {
		return status.Error(codes.Internal, fmt.Errorf("upload debuginfo: %w", err).Error())
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
