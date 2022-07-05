// Copyright 2021 The Parca Authors
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
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

var ErrDebugInfoAlreadyExists = errors.New("debug info already exists")

// ChunkSize 8MB is the size of the chunks in which debuginfo files are
// uploaded and downloaded. AWS S3 has a minimum of 5MB for multi part uploads
// and a maximum of 15MB, and a default of 8MB.
var ChunkSize = 1024 * 1024 * 8

type Client struct {
	c debuginfopb.DebugInfoServiceClient
}

func NewDebugInfoClient(conn *grpc.ClientConn) *Client {
	return &Client{
		c: debuginfopb.NewDebugInfoServiceClient(conn),
	}
}

func (c *Client) Exists(ctx context.Context, buildID, hash string) (bool, error) {
	res, err := c.c.Exists(ctx, &debuginfopb.ExistsRequest{
		BuildId: buildID,
		Hash:    hash,
	})
	if err != nil {
		return false, err
	}

	return res.Exists, nil
}

func (c *Client) Upload(ctx context.Context, buildID, hash string, r io.Reader) (uint64, error) {
	stream, err := c.c.Upload(ctx)
	if err != nil {
		return 0, fmt.Errorf("initiate upload: %w", err)
	}

	err = stream.Send(&debuginfopb.UploadRequest{
		Data: &debuginfopb.UploadRequest_Info{
			Info: &debuginfopb.UploadInfo{
				BuildId: buildID,
				Hash:    hash,
			},
		},
	})
	if err != nil {
		if err := sentinelError(err); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("send upload info: %w", err)
	}

	reader := bufio.NewReader(r)

	buffer := make([]byte, ChunkSize)

	bytesSent := 0
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read next chunk (%d bytes sent so far): %w", bytesSent, err)
		}

		err = stream.Send(&debuginfopb.UploadRequest{
			Data: &debuginfopb.UploadRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if err == io.EOF {
			// When the stream is closed, the server will send an EOF.
			// To get the correct error code, we need the status.
			// So receive the message and check the status.
			err = stream.RecvMsg(nil)
			if err := sentinelError(err); err != nil {
				return 0, err
			}
			return 0, fmt.Errorf("send chunk: %w", err)
		}
		if err != nil {
			return 0, fmt.Errorf("send next chunk (%d bytes sent so far): %w", bytesSent, err)
		}
		bytesSent += n
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		if err := sentinelError(err); err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("close and receive: %w", err)
	}
	return res.Size, nil
}

type Downloader struct {
	stream debuginfopb.DebugInfoService_DownloadClient
	info   *debuginfopb.DownloadInfo
}

func (c *Client) Downloader(ctx context.Context, buildID string) (*Downloader, error) {
	stream, err := c.c.Download(ctx, &debuginfopb.DownloadRequest{
		BuildId: buildID,
	})
	if err != nil {
		return nil, fmt.Errorf("initiate download: %w", err)
	}

	res, err := stream.Recv()
	if err != nil {
		return nil, fmt.Errorf("receive download info: %w", err)
	}

	info := res.GetInfo()
	if info == nil {
		return nil, fmt.Errorf("download info is nil")
	}

	return &Downloader{
		stream: stream,
		info:   info,
	}, nil
}

func (d *Downloader) Info() *debuginfopb.DownloadInfo {
	return d.info
}

func (d *Downloader) Close() error {
	return d.stream.CloseSend()
}

func (d *Downloader) Download(ctx context.Context, w io.Writer) (int, error) {
	bytesWritten := 0
	for {
		res, err := d.stream.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			return bytesWritten, fmt.Errorf("receive next chunk: %w", err)
		}

		chunkData := res.GetChunkData()
		if chunkData == nil {
			return bytesWritten, fmt.Errorf("chunk does not contain data")
		}

		n, err := w.Write(chunkData)
		if err != nil {
			return bytesWritten, fmt.Errorf("write next chunk: %w", err)
		}
		bytesWritten += n
	}

	return bytesWritten, nil
}

func sentinelError(err error) error {
	if sts, ok := status.FromError(err); ok {
		if sts.Code() == codes.AlreadyExists {
			return ErrDebugInfoAlreadyExists
		}
	}
	return nil
}
