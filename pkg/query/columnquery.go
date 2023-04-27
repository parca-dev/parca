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

package query

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

type Querier interface {
	Labels(ctx context.Context, match []string, start, end time.Time) ([]string, error)
	Values(ctx context.Context, labelName string, match []string, start, end time.Time) ([]string, error)
	QueryRange(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration, limit uint32) ([]*pb.MetricsSeries, error)
	ProfileTypes(ctx context.Context) ([]*pb.ProfileType, error)
	QuerySingle(ctx context.Context, query string, time time.Time) (*profile.Profile, error)
	QueryMerge(ctx context.Context, query string, start, end time.Time) (*profile.Profile, error)
}

// ColumnQueryAPI is the read api interface for parca
// It implements the proto/query/query.proto APIServer interface.
type ColumnQueryAPI struct {
	pb.UnimplementedQueryServiceServer

	logger      log.Logger
	tracer      trace.Tracer
	shareClient sharepb.ShareServiceClient
	querier     Querier

	tableConverterPool *sync.Pool
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	shareClient sharepb.ShareServiceClient,
	querier Querier,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:             logger,
		tracer:             tracer,
		shareClient:        shareClient,
		querier:            querier,
		tableConverterPool: NewTableConverterPool(),
	}
}

func NewTableConverterPool() *sync.Pool {
	return &sync.Pool{
		New: func() any {
			return &tableConverter{
				stringsSlice:   []string{},
				stringsIndex:   map[string]uint32{},
				mappingsSlice:  []*metastorev1alpha1.Mapping{},
				mappingsIndex:  map[string]uint32{},
				locationsSlice: []*metastorev1alpha1.Location{},
				locationsIndex: map[string]uint32{},
				functionsSlice: []*metastorev1alpha1.Function{},
				functionsIndex: map[string]uint32{},
			}
		},
	}
}

// Labels issues a labels request against the storage.
func (q *ColumnQueryAPI) Labels(ctx context.Context, req *pb.LabelsRequest) (*pb.LabelsResponse, error) {
	vals, err := q.querier.Labels(ctx, req.Match, req.Start.AsTime(), req.End.AsTime())
	if err != nil {
		return nil, err
	}

	return &pb.LabelsResponse{
		LabelNames: vals,
	}, nil
}

// Values issues a values request against the storage.
func (q *ColumnQueryAPI) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	vals, err := q.querier.Values(ctx, req.LabelName, req.Match, req.Start.AsTime(), req.End.AsTime())
	if err != nil {
		return nil, err
	}

	return &pb.ValuesResponse{
		LabelValues: vals,
	}, nil
}

// QueryRange issues a range query against the storage.
func (q *ColumnQueryAPI) QueryRange(ctx context.Context, req *pb.QueryRangeRequest) (*pb.QueryRangeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	res, err := q.querier.QueryRange(ctx, req.Query, req.Start.AsTime(), req.End.AsTime(), req.Step.AsDuration(), req.Limit)
	if err != nil {
		return nil, err
	}

	return &pb.QueryRangeResponse{
		Series: res,
	}, nil
}

// Types returns the available types of profiles.
func (q *ColumnQueryAPI) ProfileTypes(ctx context.Context, req *pb.ProfileTypesRequest) (*pb.ProfileTypesResponse, error) {
	types, err := q.querier.ProfileTypes(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.ProfileTypesResponse{
		Types: types,
	}, nil
}

// Query issues an instant query against the storage.
func (q *ColumnQueryAPI) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		p        *profile.Profile
		filtered int64
		err      error
	)

	switch req.Mode {
	case pb.QueryRequest_MODE_SINGLE_UNSPECIFIED:
		p, err = q.selectSingle(ctx, req.GetSingle())
	case pb.QueryRequest_MODE_MERGE:
		p, err = q.selectMerge(ctx, req.GetMerge())
	case pb.QueryRequest_MODE_DIFF:
		p, err = q.selectDiff(ctx, req.GetDiff())
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}
	if err != nil {
		return nil, err
	}

	if req.FilterQuery != nil {
		p, filtered = FilterProfileData(ctx, q.tracer, p, req.GetFilterQuery())
	}

	return q.renderReport(
		ctx,
		p,
		req.GetReportType(),
		req.GetNodeTrimThreshold(),
		filtered,
	)
}

func keepSample(s *profile.SymbolizedSample, filterQuery string) bool {
	for _, loc := range s.Locations {
		for _, l := range loc.Lines {
			if l.Function != nil && strings.Contains(strings.ToLower(l.Function.Name), filterQuery) {
				return true
			}
		}
	}
	return false
}

type FilteredProfile struct {
	TotalUnfiltered int64
	*profile.Profile
}

func FilterProfileData(
	ctx context.Context,
	tracer trace.Tracer,
	p *profile.Profile,
	filterQuery string,
) (*profile.Profile, int64) {
	_, span := tracer.Start(ctx, "filterByFunction")
	defer span.End()

	// We want to filter by function name case-insensitive, so we need to lowercase the query.
	// We lower case the query here, so we don't have to do it for every sample.
	filterQuery = strings.ToLower(filterQuery)

	var (
		totalUnfiltered int64
		totalFiltered   int64
	)
	filteredSamples := []*profile.SymbolizedSample{}
	for _, s := range p.Samples {
		// We sum up the total number of values here, regardless whether it's filtered or not,
		// to get the unfiltered total.
		totalUnfiltered += s.Value

		if keepSample(s, filterQuery) {
			filteredSamples = append(filteredSamples, s)
			totalFiltered += s.Value
		}
	}

	return &profile.Profile{
		Samples: filteredSamples,
		Meta:    p.Meta,
	}, totalUnfiltered - totalFiltered
}

func (q *ColumnQueryAPI) renderReport(
	ctx context.Context,
	p *profile.Profile,
	typ pb.QueryRequest_ReportType,
	nodeTrimThreshold float32,
	filtered int64,
) (*pb.QueryResponse, error) {
	return RenderReport(ctx, q.tracer, p, typ, nodeTrimThreshold, filtered, q.tableConverterPool)
}

func RenderReport(
	ctx context.Context,
	tracer trace.Tracer,
	p *profile.Profile,
	typ pb.QueryRequest_ReportType,
	nodeTrimThreshold float32,
	filtered int64,
	pool *sync.Pool,
) (*pb.QueryResponse, error) {
	ctx, span := tracer.Start(ctx, "renderReport")
	span.SetAttributes(attribute.String("reportType", typ.String()))
	defer span.End()

	nodeTrimFraction := float32(0)
	if nodeTrimThreshold != 0 {
		nodeTrimFraction = nodeTrimThreshold / 100
	}

	switch typ {
	//nolint:staticcheck // SA1019: Fow now we want to support these APIs
	case pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_UNSPECIFIED:
		fg, err := GenerateFlamegraphFlat(ctx, tracer, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate flamegraph: %v", err.Error())
		}
		return &pb.QueryResponse{
			Total:    fg.Total,
			Filtered: filtered,
			Report: &pb.QueryResponse_Flamegraph{
				Flamegraph: fg,
			},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_TABLE:
		fg, err := GenerateFlamegraphTable(ctx, tracer, p, nodeTrimFraction, pool)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate flamegraph: %v", err.Error())
		}
		return &pb.QueryResponse{
			//nolint:staticcheck // SA1019: TODO: The cumulative should be passed differently in the future.
			Total:    fg.Total,
			Filtered: filtered,
			Report: &pb.QueryResponse_Flamegraph{
				Flamegraph: fg,
			},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW:
		_, err := GenerateFlamegraphArrow(ctx, tracer, p, nodeTrimFraction)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate arrow flamegraph: %v", err.Error())
		}

		return &pb.QueryResponse{}, nil
	case pb.QueryRequest_REPORT_TYPE_PPROF:
		pp, err := GenerateFlatPprof(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		var buf bytes.Buffer
		if err := pp.Write(&buf); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		return &pb.QueryResponse{
			Total:    0, // TODO: Figure out how to get total for pprof
			Filtered: filtered,
			Report:   &pb.QueryResponse_Pprof{Pprof: buf.Bytes()},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_TOP:
		top, cumulative, err := GenerateTopTable(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		return &pb.QueryResponse{
			//nolint:staticcheck // SA1019: TODO: The cumulative should be passed differently in the future.
			Total:    cumulative,
			Filtered: filtered,
			Report:   &pb.QueryResponse_Top{Top: top},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_CALLGRAPH:
		callgraph, err := GenerateCallgraph(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate callgraph: %v", err.Error())
		}
		return &pb.QueryResponse{
			//nolint:staticcheck // SA1019: TODO: The cumulative should be passed differently in the future.
			Total:    callgraph.Cumulative,
			Filtered: filtered,
			Report:   &pb.QueryResponse_Callgraph{Callgraph: callgraph},
		}, nil
	default:
		return nil, status.Error(codes.InvalidArgument, "requested report type does not exist")
	}
}

func (q *ColumnQueryAPI) selectSingle(ctx context.Context, s *pb.SingleProfile) (*profile.Profile, error) {
	p, err := q.querier.QuerySingle(
		ctx,
		s.Query,
		s.Time.AsTime(),
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectMerge(ctx context.Context, m *pb.MergeProfile) (*profile.Profile, error) {
	p, err := q.querier.QueryMerge(
		ctx,
		m.Query,
		m.Start.AsTime(),
		m.End.AsTime(),
	)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectDiff(ctx context.Context, d *pb.DiffProfile) (*profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "diffRequest")
	defer span.End()

	if d == nil {
		return nil, status.Error(codes.InvalidArgument, "requested diff mode, but did not provide parameters for diff")
	}

	g, ctx := errgroup.WithContext(ctx)
	var base *profile.Profile
	g.Go(func() error {
		var err error
		base, err = q.selectProfileForDiff(ctx, d.A)
		if err != nil {
			return fmt.Errorf("reading base profile: %w", err)
		}
		return nil
	})

	var compare *profile.Profile
	g.Go(func() error {
		var err error
		compare, err = q.selectProfileForDiff(ctx, d.B)
		if err != nil {
			return fmt.Errorf("reading compared profile: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	// TODO: This is cheating a bit. This should be done with a sub-query in the columnstore.
	diff := &profile.Profile{}

	// TODO: Use parcacol.Sample for comparing these
	for i := range compare.Samples {
		diff.Samples = append(diff.Samples, &profile.SymbolizedSample{
			Locations: compare.Samples[i].Locations,
			Value:     compare.Samples[i].Value,
			DiffValue: compare.Samples[i].Value,
			Label:     compare.Samples[i].Label,
			NumLabel:  compare.Samples[i].NumLabel,
		})
	}

	for i := range base.Samples {
		diff.Samples = append(diff.Samples, &profile.SymbolizedSample{
			Locations: base.Samples[i].Locations,
			DiffValue: -base.Samples[i].Value,
			Label:     base.Samples[i].Label,
			NumLabel:  base.Samples[i].NumLabel,
		})
	}

	return diff, nil
}

func (q *ColumnQueryAPI) selectProfileForDiff(ctx context.Context, s *pb.ProfileDiffSelection) (*profile.Profile, error) {
	switch s.Mode {
	case pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		return q.selectSingle(ctx, s.GetSingle())
	case pb.ProfileDiffSelection_MODE_MERGE:
		return q.selectMerge(ctx, s.GetMerge())
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown mode for diff profile selection")
	}
}

func (q *ColumnQueryAPI) ShareProfile(ctx context.Context, req *pb.ShareProfileRequest) (*pb.ShareProfileResponse, error) {
	req.QueryRequest.ReportType = pb.QueryRequest_REPORT_TYPE_PPROF
	resp, err := q.Query(ctx, req.QueryRequest)
	if err != nil {
		return nil, err
	}
	uploadResp, err := q.shareClient.Upload(ctx, &sharepb.UploadRequest{
		Profile:     resp.GetPprof(),
		Description: *req.Description,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to upload profile: %s", err.Error())
	}
	return &pb.ShareProfileResponse{
		Link: uploadResp.Link,
	}, nil
}
