package api

import (
	"context"

	pb "github.com/parca-dev/parca/proto/api"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// API is the api interface for parca
// It implements the proto/api/api.proto APIServer interface
type API struct{}

// QueryRange issues a range query against the storage
func (a *API) QueryRange(ctx context.Context, req *pb.QueryRangeRequest) (*pb.QueryRangeResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Query issues a instant query against the storage
func (a *API) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Series issues a series request against the storage
func (a *API) Series(ctx context.Context, req *pb.SeriesRequest) (*pb.SeriesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Labels issues a labels request against the storage
func (a *API) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Values issues a values request against the storage
func (a *API) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Config issues a config request against the storage
func (a *API) Config(ctx context.Context, req *pb.ConfigRequest) (*pb.ConfigResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}

// Targets issues a targets request against the storage
func (a *API) Targets(ctx context.Context, req *pb.TargetsRequest) (*pb.TargetsResponse, error) {
	return nil, status.Error(codes.Unimplemented, "unimplemented")
}
