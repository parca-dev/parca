package metastore

import (
	"context"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"google.golang.org/grpc"
)

type InProcessClient struct {
	m pb.MetastoreServiceServer
}

func NewInProcessClient(m pb.MetastoreServiceServer) *InProcessClient {
	return &InProcessClient{m: m}
}

func (c *InProcessClient) GetOrCreateMappings(ctx context.Context, in *pb.GetOrCreateMappingsRequest, opts ...grpc.CallOption) (*pb.GetOrCreateMappingsResponse, error) {
	return c.m.GetOrCreateMappings(ctx, in)
}

func (c *InProcessClient) GetOrCreateFunctions(ctx context.Context, in *pb.GetOrCreateFunctionsRequest, opts ...grpc.CallOption) (*pb.GetOrCreateFunctionsResponse, error) {
	return c.m.GetOrCreateFunctions(ctx, in)
}

func (c *InProcessClient) GetOrCreateLocations(ctx context.Context, in *pb.GetOrCreateLocationsRequest, opts ...grpc.CallOption) (*pb.GetOrCreateLocationsResponse, error) {
	return c.m.GetOrCreateLocations(ctx, in)
}

func (c *InProcessClient) GetOrCreateStacktraces(ctx context.Context, in *pb.GetOrCreateStacktracesRequest, opts ...grpc.CallOption) (*pb.GetOrCreateStacktracesResponse, error) {
	return c.m.GetOrCreateStacktraces(ctx, in)
}

func (c *InProcessClient) UnsymbolizedLocations(ctx context.Context, in *pb.UnsymbolizedLocationsRequest, opts ...grpc.CallOption) (*pb.UnsymbolizedLocationsResponse, error) {
	return c.m.UnsymbolizedLocations(ctx, in)
}

func (c *InProcessClient) CreateLocationLines(ctx context.Context, in *pb.CreateLocationLinesRequest, opts ...grpc.CallOption) (*pb.CreateLocationLinesResponse, error) {
	return c.m.CreateLocationLines(ctx, in)
}

func (c *InProcessClient) Locations(ctx context.Context, in *pb.LocationsRequest, opts ...grpc.CallOption) (*pb.LocationsResponse, error) {
	return c.m.Locations(ctx, in)
}

func (c *InProcessClient) LocationLines(ctx context.Context, in *pb.LocationLinesRequest, opts ...grpc.CallOption) (*pb.LocationLinesResponse, error) {
	return c.m.LocationLines(ctx, in)
}

func (c *InProcessClient) Functions(ctx context.Context, in *pb.FunctionsRequest, opts ...grpc.CallOption) (*pb.FunctionsResponse, error) {
	return c.m.Functions(ctx, in)
}

func (c *InProcessClient) Mappings(ctx context.Context, in *pb.MappingsRequest, opts ...grpc.CallOption) (*pb.MappingsResponse, error) {
	return c.m.Mappings(ctx, in)
}

func (c *InProcessClient) Stacktraces(ctx context.Context, in *pb.StacktracesRequest, opts ...grpc.CallOption) (*pb.StacktracesResponse, error) {
	return c.m.Stacktraces(ctx, in)
}
