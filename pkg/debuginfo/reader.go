package debuginfo

import (
	"bytes"
	"context"
	"fmt"
	"io"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UploadReader struct {
	stream debuginfopb.DebugInfoService_UploadServer
	cur    io.Reader
	size   uint64
}

func (r *UploadReader) Read(p []byte) (int, error) {
	if r.cur == nil {
		var err error
		r.cur, err = r.next()
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fmt.Errorf("get first upload chunk: %w", err)
		}
	}
	i, err := r.cur.Read(p)
	if err != nil && err != io.EOF {
		return 0, fmt.Errorf("read upload chunk (%d bytes read so far): %w", r.size, err)
	}
	if err == io.EOF {
		r.cur, err = r.next()
		if err == io.EOF {
			return 0, io.EOF
		}
		if err != nil {
			return 0, fmt.Errorf("get next upload chunk (%d bytes read so far): %w", r.size, err)
		}
		i, err = r.cur.Read(p)
		if err != nil {
			return 0, fmt.Errorf("read next upload chunk (%d bytes read so far): %w", r.size, err)
		}
	}

	r.size += uint64(i)
	return i, nil
}

func (r *UploadReader) next() (io.Reader, error) {
	err := contextError(r.stream.Context())
	if err != nil {
		return nil, err
	}

	req, err := r.stream.Recv()
	if err == io.EOF {
		return nil, io.EOF
	}
	if err != nil {
		return nil, fmt.Errorf("receive from stream: %w", err)
	}

	return bytes.NewBuffer(req.GetChunkData()), nil
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return status.Error(codes.Canceled, "request is canceled")
	case context.DeadlineExceeded:
		return status.Error(codes.DeadlineExceeded, "deadline is exceeded")
	default:
		return nil
	}
}
