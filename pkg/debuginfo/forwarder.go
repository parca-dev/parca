package debuginfo

import (
	"context"

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
)

type GrpcForwarder struct {
	debuginfopb.UnimplementedDebuginfoServiceServer
	client debuginfopb.DebuginfoServiceClient
}

func NewGRPCForwarder(client debuginfopb.DebuginfoServiceClient) *GrpcForwarder {
	return &GrpcForwarder{
		client: client,
	}
}

func (f *GrpcForwarder) ShouldInitiateUpload(ctx context.Context, req *debuginfopb.ShouldInitiateUploadRequest) (*debuginfopb.ShouldInitiateUploadResponse, error) {
	return f.client.ShouldInitiateUpload(ctx, req)
}

func (f *GrpcForwarder) InitiateUpload(ctx context.Context, req *debuginfopb.InitiateUploadRequest) (*debuginfopb.InitiateUploadResponse, error) {
	return f.client.InitiateUpload(ctx, req)
}

func (f *GrpcForwarder) MarkUploadFinished(ctx context.Context, req *debuginfopb.MarkUploadFinishedRequest) (*debuginfopb.MarkUploadFinishedResponse, error) {
	return f.client.MarkUploadFinished(ctx, req)
}
