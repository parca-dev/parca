package symbol

import (
	"bufio"
	"context"
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
		return 0, err
	}

	err = stream.Send(&storepb.SymbolUploadRequest{
		Data: &storepb.SymbolUploadRequest_Info{
			Info: &storepb.SymbolUploadInfo{
				Id: id,
			},
		},
	})
	if err != nil {
		return 0, err
	}

	reader := bufio.NewReader(r)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, err
		}

		err = stream.Send(&storepb.SymbolUploadRequest{
			Data: &storepb.SymbolUploadRequest_ChunkData{
				ChunkData: buffer[:n],
			},
		})
		if err != nil {
			return 0, err
		}
	}

	res, err := stream.CloseAndRecv()
	return res.Size_, err
}
