// Copyright 2021 The conprof Authors
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

package symbol

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"github.com/conprof/conprof/pkg/store/storepb"
)

type SymbolStoreClient struct {
	c storepb.SymbolStoreClient
}

func NewSymbolStoreClient(c storepb.SymbolStoreClient) *SymbolStoreClient {
	return &SymbolStoreClient{
		c: c,
	}
}

func (c *SymbolStoreClient) Exists(ctx context.Context, id string) (bool, error) {
	res, err := c.c.Exists(ctx, &storepb.SymbolExistsRequest{
		Id: id,
	})
	if err != nil {
		return false, err
	}

	return res.Exists, nil
}

func (c *SymbolStoreClient) Upload(ctx context.Context, id string, r io.Reader) (uint64, error) {
	stream, err := c.c.Upload(ctx)
	if err != nil {
		return 0, fmt.Errorf("initiate upload: %w", err)
	}

	err = stream.Send(&storepb.SymbolUploadRequest{
		Data: &storepb.SymbolUploadRequest_Info{
			Info: &storepb.SymbolUploadInfo{
				Id: id,
			},
		},
	})
	if err != nil {
		return 0, fmt.Errorf("send upload info: %w", err)
	}

	reader := bufio.NewReader(r)
	buffer := make([]byte, 1024)

	bytesSent := 0
	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read next chunk (%d bytes sent so far): %w", bytesSent, err)
		}

		err = stream.Send(&storepb.SymbolUploadRequest{
			Data: &storepb.SymbolUploadRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if err != nil {
			return 0, fmt.Errorf("send next chunk (%d bytes sent so far): %w", bytesSent, err)
		}
		bytesSent += n
	}

	res, err := stream.CloseAndRecv()
	if err != nil {
		return 0, fmt.Errorf("close and receive: %w:", err)
	}
	return res.Size_, nil
}
