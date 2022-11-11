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
	"bytes"
	"context"
	"debug/elf"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/nanmu42/limitio"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var ErrDebugInfoNotFound = errors.New("debug info not found")

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
	MarkAsCorrupted(ctx context.Context, buildID string) error
	MarkAsUploading(ctx context.Context, buildID string) error
	MarkAsUploaded(ctx context.Context, buildID, hash string) error
	Fetch(ctx context.Context, buildID string) (*Metadata, error)
}

type Store struct {
	debuginfopb.UnimplementedDebugInfoServiceServer

	tracer   trace.Tracer
	logger   log.Logger
	cacheDir string

	bucket objstore.Bucket

	metadata         MetadataManager
	debuginfodClient DebugInfodClient

	debugInfoUploadAttemptsTotal prometheus.Counter
	debugInfoUploadErrorsTotal   prometheus.CounterVec
	debugInfoUploadDuration      prometheus.Histogram
	existsCheckDuration          prometheus.Histogram
}

// NewStore returns a new debug info store.
func NewStore(
	tracer trace.Tracer,
	logger log.Logger,
	reg prometheus.Registerer,
	cacheDir string,
	metadata MetadataManager,
	bucket objstore.Bucket,
	debuginfodClient DebugInfodClient,
) (*Store, error) {
	debugInfoUploadAttemptsTotal := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "debuginfo_upload_attempts_total",
			Help: "Total attempts to upload debuginfo.",
		},
	)
	debugInfoUploadErrorsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "debuginfo_upload_errors_total",
			Help: "Total number of errors in uploading debuginfo.",
		},
		[]string{"reason"},
	)
	debugInfoUploadDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "debuginfo_upload_duration_seconds",
			Help:    "How long it took in seconds to upload debuginfo.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)

	existsCheckDuration := prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "debuginfo_exists_check_duration_seconds",
			Help:    "How long it took in seconds to check existing debuginfo.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)

	return &Store{
		tracer:                       tracer,
		logger:                       log.With(logger, "component", "debuginfo"),
		bucket:                       bucket,
		cacheDir:                     cacheDir,
		metadata:                     metadata,
		debuginfodClient:             debuginfodClient,
		debugInfoUploadAttemptsTotal: debugInfoUploadAttemptsTotal,
		debugInfoUploadErrorsTotal:   *debugInfoUploadErrorsTotal,
		debugInfoUploadDuration:      debugInfoUploadDuration,
		existsCheckDuration:          existsCheckDuration,
	}, nil
}

func (s *Store) Exists(ctx context.Context, req *debuginfopb.ExistsRequest) (*debuginfopb.ExistsResponse, error) {
	defer func(begin time.Time) {
		s.existsCheckDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", req.GetBuildId()))

	buildID := req.BuildId
	if err := validateInput(buildID); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	found, err := s.find(ctx, buildID)
	if err != nil {
		return nil, err
	}

	if found {
		metadataFile, err := s.metadata.Fetch(ctx, buildID)
		if err != nil {
			if errors.Is(err, ErrMetadataNotFound) {
				return &debuginfopb.ExistsResponse{Exists: false}, nil
			}
			return nil, status.Error(codes.Internal, err.Error())
		}

		// metadata.Hash should nt be empty, but let's have the check just in case.
		if metadataFile.Hash != "" && metadataFile.Hash == req.Hash {
			return &debuginfopb.ExistsResponse{Exists: true}, nil
		}

		var exists bool
		// If it is not an exact version of the source object file what we have so, let the client try to upload it.
		if metadataFile.State == MetadataStateUploading {
			exists = !isStale(metadataFile)
		}
		return &debuginfopb.ExistsResponse{Exists: exists}, nil
	}

	return &debuginfopb.ExistsResponse{Exists: false}, nil
}

func (s *Store) Upload(stream debuginfopb.DebugInfoService_UploadServer) error {
	defer func(begin time.Time) {
		s.debugInfoUploadDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())
	s.debugInfoUploadAttemptsTotal.Inc()
	req, err := stream.Recv()
	if err != nil {
		s.debugInfoUploadErrorsTotal.WithLabelValues("stream_receive").Inc()
		msg := "failed to receive upload info"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	var (
		buildID = req.GetInfo().BuildId
		hash    = req.GetInfo().Hash
		r       = &UploadReader{stream: stream}
	)

	ctx := stream.Context()
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(attribute.String("build_id", buildID))
	span.SetAttributes(attribute.String("hash", hash))

	if err := s.upload(ctx, buildID, hash, r); err != nil {
		s.debugInfoUploadErrorsTotal.WithLabelValues("store_upload").Inc()
		return err
	}

	level.Debug(s.logger).Log("msg", "debug info uploaded", "buildid", buildID)
	return stream.SendAndClose(&debuginfopb.UploadResponse{
		BuildId: buildID,
		Size:    r.size,
	})
}

func (s *Store) upload(ctx context.Context, buildID, hash string, r io.Reader) error {
	if err := validateInput(buildID); err != nil {
		err = fmt.Errorf("invalid build ID: %w", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if err := validateInput(hash); err != nil {
		err = fmt.Errorf("invalid hash: %w", err)
		return status.Error(codes.InvalidArgument, err.Error())
	}

	level.Debug(s.logger).Log("msg", "trying to upload debug info", "buildid", buildID)

	metadataFile, err := s.metadata.Fetch(ctx, buildID)
	if err == nil {
		level.Debug(s.logger).Log("msg", "fetching metadata state", "result", metadataFile)

		switch metadataFile.State {
		case MetadataStateCorrupted:
			// Corrupted. Re-upload.
		case MetadataStateUploaded:
			// The debug info was fully uploaded.
			return status.Error(codes.AlreadyExists, "debuginfo already exists")
		case MetadataStateUploading:
			if !isStale(metadataFile) {
				return status.Error(codes.AlreadyExists, "debuginfo already exists, being uploaded right now")
			}
			// The debug info upload operation most likely failed.
		default:
			return status.Error(codes.Internal, "unknown metadata state")
		}
	} else {
		if !errors.Is(err, ErrMetadataNotFound) {
			level.Error(s.logger).Log("msg", "failed to fetch metadata state", "err", err)
		}
	}

	found, err := s.find(ctx, buildID)
	if err != nil {
		return err
	}

	if found {
		if hash != "" && metadataFile != nil {
			if metadataFile.Hash == hash {
				level.Debug(s.logger).Log("msg", "debug info already exists", "buildid", buildID)
				return status.Error(codes.AlreadyExists, "debuginfo already exists")
			}
		}

		objFile, _, err := s.FetchDebugInfo(ctx, buildID)
		if err != nil {
			return status.Error(codes.Internal, err.Error())
		}
		if err := elfutils.ValidateFile(objFile); err != nil {
			s.debugInfoUploadErrorsTotal.WithLabelValues("validation").Inc()
			// Failed to validate. Mark the file as corrupted, and let the client try to upload it again.
			if err := s.metadata.MarkAsCorrupted(ctx, buildID); err != nil {
				level.Warn(s.logger).Log("msg", "failed to update metadata as corrupted", "err", err)
			}
			level.Error(s.logger).Log("msg", "failed to validate object file", "buildid", buildID)
			// Client will retry.
			return status.Error(codes.Internal, err.Error())
		}

		// Valid.
		f, err := elf.Open(objFile)
		if err != nil {
			level.Debug(s.logger).Log("msg", "failed to open object file", "err", err)
		} else {
			hasDWARF, err := elfutils.HasDWARF(f)
			if err != nil {
				level.Debug(s.logger).Log("msg", "failed to check for DWARF", "err", err)
			}
			f.Close()
			if hasDWARF {
				return status.Error(codes.AlreadyExists, "debuginfo already exists")
			}
		}
	}

	// At this point we know that we received a better version of the debug information file,
	// so let the client upload it.
	if err := s.metadata.MarkAsUploading(ctx, buildID); err != nil {
		err = fmt.Errorf("failed to update metadata before uploading: %w", err)
		return status.Error(codes.Internal, err.Error())
	}

	// limitio.Writer is used to avoid buffer overflow.
	// We only need to read the first 64 bytes (at most).
	// The ELF header is 52 or 64 bytes long for 32-bit and 64-bit binaries respectively.
	// If we receive a longer data, we will ignore the rest without an error.
	b := bytes.NewBuffer(nil)
	w := limitio.NewWriter(b, 64, true)

	// Here we're optimistically uploading the received stream directly to the bucket,
	// and if something goes wrong we mark it as corrupted, so it could be overwritten in subsequent calls.
	// We only want to make sure we don't read a corrupted file while symbolizing.
	// Ww also wanted to prevent any form of buffering for this data on the server-side,
	// thus the optimistic writes directly to the object-store while also writing the header of the file into a buffer,
	// so we can validate the ELF header.
	if err := s.bucket.Upload(ctx, objectPath(buildID), io.TeeReader(r, w)); err != nil {
		msg := "failed to upload"
		level.Error(s.logger).Log("msg", msg, "err", err)
		return status.Errorf(codes.Unknown, msg)
	}

	if err := elfutils.ValidateHeader(b); err != nil {
		// Failed to validate. Mark the incoming stream as corrupted, and let the client try to upload it again.
		if err := s.metadata.MarkAsCorrupted(ctx, buildID); err != nil {
			err = fmt.Errorf("failed to update metadata after uploaded, as corrupted: %w", err)
			return status.Error(codes.Internal, err.Error())
		}
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if err := s.metadata.MarkAsUploaded(ctx, buildID, hash); err != nil {
		err = fmt.Errorf("failed to update metadata after uploaded: %w", err)
		return status.Error(codes.Internal, err.Error())
	}

	return nil
}

func isStale(metadataFile *Metadata) bool {
	return time.Now().Add(-15 * time.Minute).After(time.Unix(metadataFile.UploadStartedAt, 0))
}

func (s *Store) Download(req *debuginfopb.DownloadRequest, stream debuginfopb.DebugInfoService_DownloadServer) error {
	if err := validateInput(req.BuildId); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	ctx := stream.Context()
	found, err := s.find(ctx, req.BuildId)
	if err != nil {
		return err
	}

	if !found {
		return status.Error(codes.NotFound, "debuginfo not found")
	}

	metadata, err := s.metadata.Fetch(ctx, req.BuildId)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	switch metadata.State {
	case MetadataStateCorrupted:
		return status.Error(codes.FailedPrecondition, "debuginfo is corrupted")
	case MetadataStateUploading:
		return status.Error(codes.Unavailable, "debuginfo is being uploaded")
	}

	objFile, source, err := s.FetchDebugInfo(ctx, req.BuildId)
	if err != nil {
		if errors.Is(err, ErrDebugInfoNotFound) {
			return status.Error(codes.NotFound, err.Error())
		}
		return status.Error(codes.Internal, err.Error())
	}

	if err := stream.Send(&debuginfopb.DownloadResponse{
		Data: &debuginfopb.DownloadResponse_Info{
			Info: &debuginfopb.DownloadInfo{
				Source: source,
			},
		},
	}); err != nil {
		return err
	}

	f, err := os.Open(objFile)
	if err != nil {
		return status.Error(codes.Internal, err.Error())
	}

	buf := make([]byte, ChunkSize)
	bytesSent := 0
	for {
		n, err := f.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read next chunk (%d bytes sent so far): %w", bytesSent, err)
		}
		bytesSent += n

		if err := stream.Send(&debuginfopb.DownloadResponse{
			Data: &debuginfopb.DownloadResponse_ChunkData{
				ChunkData: buf[:n],
			},
		}); err != nil {
			return err
		}
	}

	return nil
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

func (s *Store) find(ctx context.Context, key string) (bool, error) {
	found := false
	err := s.bucket.Iter(ctx, key, func(_ string) error {
		// We just need any debug files to be present, so if a file under the directory for the build ID exists,
		// it's found: <buildid>/debuginfo, or <buildid>/metadata.
		found = true
		return nil
	})
	if err != nil {
		return false, status.Error(codes.Internal, err.Error())
	}
	return found, nil
}

func (s *Store) FetchDebugInfo(ctx context.Context, buildID string) (string, debuginfopb.DownloadInfo_Source, error) {
	logger := log.With(s.logger, "buildid", buildID)

	source := debuginfopb.DownloadInfo_SOURCE_UNKNOWN_UNSPECIFIED
	objFile, err := s.fetchFromObjectStore(ctx, buildID)
	if err != nil {
		// It's ok if we don't have the symbols for given BuildID, it happens too often.
		level.Warn(logger).Log("msg", "failed to fetch object", "err", err)

		// Let's try to find a debug file from debuginfod servers.
		objFile, err = s.fetchDebuginfodFile(ctx, buildID)
		if err != nil {
			return "", source, fmt.Errorf("failed to fetch: %w", err)
		}
		source = debuginfopb.DownloadInfo_SOURCE_DEBUGINFOD
	} else {
		source = debuginfopb.DownloadInfo_SOURCE_UPLOAD
	}

	// Let's make sure we have the best version of the debug file.
	if err := elfutils.ValidateFile(objFile); err != nil {
		level.Warn(logger).Log("msg", "failed to validate debug information", "err", err)
		// Mark the file as corrupted, and let the client try to upload it again.
		err := s.metadata.MarkAsCorrupted(ctx, buildID)
		if err != nil {
			level.Warn(logger).Log(
				"msg", "failed to mark debug information",
				"err", fmt.Errorf("failed to update metadata for corrupted: %w", err),
			)
		}
		if source != debuginfopb.DownloadInfo_SOURCE_DEBUGINFOD {
			dbgFile, err := s.fetchDebuginfodFile(ctx, buildID)
			if err != nil {
				level.Warn(logger).Log("msg", "failed to fetch debuginfod file", "err", err)
			} else {
				objFile = dbgFile
				source = debuginfopb.DownloadInfo_SOURCE_DEBUGINFOD
			}
		}
	}

	if source != debuginfopb.DownloadInfo_SOURCE_DEBUGINFOD {
		f, err := elf.Open(objFile)
		if err != nil {
			level.Debug(logger).Log("msg", "failed to open object file", "err", err)
		} else {
			hasDWARF, err := elfutils.HasDWARF(f)
			if err != nil {
				level.Debug(logger).Log("msg", "failed to check for DWARF", "err", err)
			}
			f.Close()

			if !hasDWARF {
				// Try to download a better version from debuginfod servers.
				dbgFile, err := s.fetchDebuginfodFile(ctx, buildID)
				if err != nil {
					level.Warn(logger).Log("msg", "failed to fetch debuginfod file", "err", err)
				} else {
					objFile = dbgFile
					source = debuginfopb.DownloadInfo_SOURCE_DEBUGINFOD
				}
			}
		}
	}

	return objFile, source, nil
}

func (s *Store) fetchFromObjectStore(ctx context.Context, buildID string) (string, error) {
	logger := log.With(s.logger, "buildid", buildID)

	objFile := s.localCachePath(buildID)
	// Check if it's already cached locally; if not download.
	if _, err := os.Stat(objFile); os.IsNotExist(err) {
		// Download the debuginfo file from the bucket.
		r, err := s.bucket.Get(ctx, objectPath(buildID))
		if err != nil {
			if s.bucket.IsObjNotFoundErr(err) {
				level.Debug(logger).Log("msg", "failed to fetch object from object storage", "err", err)
				return "", ErrDebugInfoNotFound
			}
			return "", fmt.Errorf("failed to fetch object: %w", err)
		}

		// Cache the file locally.
		if err := s.cache(objFile, r); err != nil {
			return "", fmt.Errorf("failed to fetch debug info file: %w", err)
		}
	}

	return objFile, nil
}

func (s *Store) fetchDebuginfodFile(ctx context.Context, buildID string) (string, error) {
	logger := log.With(s.logger, "buildid", buildID)
	level.Debug(logger).Log("msg", "attempting to download from debuginfod servers")

	objFile := s.localCachePath(buildID)
	// Try downloading the debuginfo file from the debuginfod server.
	r, err := s.debuginfodClient.GetDebugInfo(ctx, buildID)
	if err != nil {
		level.Debug(logger).Log("msg", "failed to download debuginfo from debuginfod", "err", err)
		return "", fmt.Errorf("failed to fetch from debuginfod: %w", err)
	}
	defer r.Close()
	level.Info(logger).Log("msg", "debug info downloaded from debuginfod server")

	// Cache the file locally.
	if err := s.cache(objFile, r); err != nil {
		level.Debug(logger).Log("msg", "failed to cache debuginfo", "err", err)
		return "", fmt.Errorf("failed to fetch from debuginfod: %w", err)
	}

	return objFile, nil
}

func (s *Store) localCachePath(buildID string) string {
	return path.Join(s.cacheDir, buildID, "debuginfo")
}

func (s *Store) cache(localPath string, r io.ReadCloser) error {
	tmpfile, err := os.CreateTemp(s.cacheDir, "symbol-download-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	written, err := io.Copy(tmpfile, r)
	if err != nil {
		return fmt.Errorf("copy debug info file to local temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return fmt.Errorf("close tempfile to write debug info file: %w", err)
	}
	if written == 0 {
		return fmt.Errorf("received empty debug info: %w", ErrDebugInfoNotFound)
	}

	err = os.MkdirAll(path.Dir(localPath), 0o700)
	if err != nil {
		return fmt.Errorf("create debug info file directory: %w", err)
	}
	// Need to use rename to make the "creation" atomic.
	if err := os.Rename(tmpfile.Name(), localPath); err != nil {
		return fmt.Errorf("atomically move downloaded debug info file: %w", err)
	}
	return nil
}

func objectPath(buildID string) string {
	return path.Join(buildID, "debuginfo")
}
