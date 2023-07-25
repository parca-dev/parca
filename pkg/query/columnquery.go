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

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/math"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"github.com/go-kit/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

type Querier interface {
	Labels(ctx context.Context, match []string, start, end time.Time) ([]string, error)
	Values(ctx context.Context, labelName string, match []string, start, end time.Time) ([]string, error)
	QueryRange(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration, limit uint32) ([]*pb.MetricsSeries, error)
	ProfileTypes(ctx context.Context) ([]*pb.ProfileType, error)
	QuerySingle(ctx context.Context, query string, time time.Time) (profile.Profile, error)
	QueryMerge(ctx context.Context, query string, start, end time.Time) (profile.Profile, error)
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
	mem                memory.Allocator
	converter          *parcacol.ArrowToProfileConverter
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	shareClient sharepb.ShareServiceClient,
	querier Querier,
	mem memory.Allocator,
	converter *parcacol.ArrowToProfileConverter,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:             logger,
		tracer:             tracer,
		shareClient:        shareClient,
		querier:            querier,
		tableConverterPool: NewTableConverterPool(),
		mem:                mem,
		converter:          converter,
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
		p        profile.Profile
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
		p, filtered, err = FilterProfileData(ctx, q.tracer, q.mem, p, req.GetFilterQuery())
		if err != nil {
			return nil, fmt.Errorf("filtering profile: %w", err)
		}
	}

	return q.renderReport(
		ctx,
		p,
		req.GetReportType(),
		req.GetNodeTrimThreshold(),
		filtered,
		req.GetGroupBy().GetFields(),
	)
}

func FilterProfileData(
	ctx context.Context,
	tracer trace.Tracer,
	pool memory.Allocator,
	p profile.Profile,
	filterQuery string,
) (profile.Profile, int64, error) {
	_, span := tracer.Start(ctx, "filterByFunction")
	defer span.End()

	// We want to filter by function name case-insensitive, so we need to lowercase the query.
	// We lower case the query here, so we don't have to do it for every sample.
	filterQuery = strings.ToLower(filterQuery)

	r := profile.NewReader(p)

	// Builders for the result profile.
	// TODO: This is a bit inefficient because it completely rebuilds the
	// profile, we should only ever rebuild dictionaries once at the very end
	// before we send a result to the user.
	resBuilder := array.NewRecordBuilder(pool, profile.ArrowSchema(r.LabelFields))
	defer resBuilder.Release()

	labelBuilders := make([]*array.BinaryDictionaryBuilder, len(r.LabelFields))
	labelNum := len(r.LabelFields)
	for i := 0; i < labelNum; i++ {
		labelBuilders[i] = resBuilder.Field(i).(*array.BinaryDictionaryBuilder)
	}
	resLocationsList := resBuilder.Field(labelNum).(*array.ListBuilder)
	resLocations := resLocationsList.ValueBuilder().(*array.StructBuilder)

	resAddresses := resLocations.FieldBuilder(0).(*array.Uint64Builder)

	resMapping := resLocations.FieldBuilder(1).(*array.StructBuilder)
	resMappingStart := resMapping.FieldBuilder(0).(*array.Uint64Builder)
	resMappingLimit := resMapping.FieldBuilder(1).(*array.Uint64Builder)
	resMappingOffset := resMapping.FieldBuilder(2).(*array.Uint64Builder)
	resMappingFile := resMapping.FieldBuilder(3).(*array.StringBuilder)
	resMappingBuildID := resMapping.FieldBuilder(4).(*array.StringBuilder)

	resLines := resLocations.FieldBuilder(2).(*array.ListBuilder)
	resLine := resLines.ValueBuilder().(*array.StructBuilder)
	resLineNumber := resLine.FieldBuilder(0).(*array.Int64Builder)
	resFunction := resLine.FieldBuilder(1).(*array.StructBuilder)
	resFunctionName := resFunction.FieldBuilder(0).(*array.StringBuilder)
	resFunctionSystemName := resFunction.FieldBuilder(1).(*array.StringBuilder)
	resFunctionFilename := resFunction.FieldBuilder(2).(*array.StringBuilder)
	resFunctionStartLine := resFunction.FieldBuilder(3).(*array.Int64Builder)

	resValues := resBuilder.Field(labelNum + 1).(*array.Int64Builder)
	resDiff := resBuilder.Field(labelNum + 2).(*array.Int64Builder)

	for i := 0; i < int(r.Profile.Samples.NumRows()); i++ {
		lOffsetStart := r.LocationOffsets[i]
		lOffsetEnd := r.LocationOffsets[i+1]
		keepRow := false
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			llOffsetStart := r.LineOffsets[j]
			llOffsetEnd := r.LineOffsets[j+1]

			for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
				if r.LineFunction.IsValid(k) && strings.Contains(strings.ToLower(r.LineFunctionName.Value(k)), filterQuery) {
					keepRow = true
					break
				}
			}
		}

		if keepRow {
			resValues.Append(r.Value.Value(i))
			resDiff.Append(r.Diff.Value(i))

			for j, label := range r.LabelColumns {
				if label.Col.IsValid(i) {
					labelBuilders[j].Append(label.Dict.Value(label.Col.GetValueIndex(i)))
				} else {
					labelBuilders[j].AppendNull()
				}
			}

			resLocationsList.Append(true)
			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				resLocations.Append(true)
				resAddresses.Append(r.Address.Value(j))

				resMapping.Append(r.Mapping.IsValid(j))
				if r.Mapping.IsValid(j) {
					resMappingStart.Append(r.MappingStart.Value(j))
					resMappingLimit.Append(r.MappingLimit.Value(j))
					resMappingOffset.Append(r.MappingOffset.Value(j))
					resMappingFile.Append(r.MappingFile.Value(j))
					resMappingBuildID.Append(r.MappingBuildID.Value(j))
				}

				resLines.Append(r.Lines.IsValid(j))

				if r.Lines.IsValid(j) {
					llOffsetStart := r.LineOffsets[j]
					llOffsetEnd := r.LineOffsets[j+1]
					for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
						resLine.Append(true)
						resLineNumber.Append(r.LineNumber.Value(k))
						resFunction.Append(r.LineFunction.IsValid(k))

						if r.LineFunction.IsValid(k) {
							resFunctionName.Append(r.LineFunctionName.Value(k))
							resFunctionSystemName.Append(r.LineFunctionSystemName.Value(k))
							resFunctionFilename.Append(r.LineFunctionFilename.Value(k))
							resFunctionStartLine.Append(r.LineFunctionStartLine.Value(k))
						}
					}
				}
			}
		}
	}

	res := profile.Profile{
		Meta:    p.Meta,
		Samples: resBuilder.NewRecord(),
	}
	numFields := res.Samples.Schema().NumFields()
	filteredValue := res.Samples.Column(numFields - 1).(*array.Int64)
	return res, math.Int64.Sum(r.Value) - math.Int64.Sum(filteredValue), nil
}

func (q *ColumnQueryAPI) renderReport(
	ctx context.Context,
	p profile.Profile,
	typ pb.QueryRequest_ReportType,
	nodeTrimThreshold float32,
	filtered int64,
	groupBy []string,
) (*pb.QueryResponse, error) {
	return RenderReport(ctx, q.tracer, p, typ, nodeTrimThreshold, filtered, groupBy, q.tableConverterPool, q.mem, q.converter)
}

func RenderReport(
	ctx context.Context,
	tracer trace.Tracer,
	p profile.Profile,
	typ pb.QueryRequest_ReportType,
	nodeTrimThreshold float32,
	filtered int64,
	groupBy []string,
	pool *sync.Pool,
	mem memory.Allocator,
	converter *parcacol.ArrowToProfileConverter,
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
		op, err := converter.Convert(ctx, p)
		if err != nil {
			return nil, err
		}

		fg, err := GenerateFlamegraphFlat(ctx, tracer, op)
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
		op, err := converter.Convert(ctx, p)
		if err != nil {
			return nil, err
		}

		fg, err := GenerateFlamegraphTable(ctx, tracer, op, nodeTrimFraction, pool)
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
		allowedGroupBy := map[string]struct{}{
			FlamegraphFieldFunctionName: {},
			FlamegraphFieldLabels:       {},
		}
		for _, f := range groupBy {
			if _, allowed := allowedGroupBy[f]; !allowed {
				return nil, status.Errorf(codes.InvalidArgument, "invalid group by field: %s", f)
			}
		}

		fa, total, err := GenerateFlamegraphArrow(ctx, mem, tracer, p, groupBy, nodeTrimFraction)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate arrow flamegraph: %v", err.Error())
		}

		return &pb.QueryResponse{
			Total:    total,
			Filtered: filtered,
			Report: &pb.QueryResponse_FlamegraphArrow{
				FlamegraphArrow: fa,
			},
		}, nil
	case pb.QueryRequest_REPORT_TYPE_PPROF:
		op, err := converter.Convert(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert profile: %v", err.Error())
		}

		pp, err := GenerateFlatPprof(ctx, op)
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
		op, err := converter.Convert(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert profile: %v", err.Error())
		}

		top, cumulative, err := GenerateTopTable(ctx, op)
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
		op, err := converter.Convert(ctx, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to convert profile: %v", err.Error())
		}

		callgraph, err := GenerateCallgraph(ctx, op)
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

func (q *ColumnQueryAPI) selectSingle(ctx context.Context, s *pb.SingleProfile) (profile.Profile, error) {
	p, err := q.querier.QuerySingle(
		ctx,
		s.Query,
		s.Time.AsTime(),
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectMerge(ctx context.Context, m *pb.MergeProfile) (profile.Profile, error) {
	p, err := q.querier.QueryMerge(
		ctx,
		m.Query,
		m.Start.AsTime(),
		m.End.AsTime(),
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectDiff(ctx context.Context, d *pb.DiffProfile) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "diffRequest")
	defer span.End()

	if d == nil {
		return profile.Profile{}, status.Error(codes.InvalidArgument, "requested diff mode, but did not provide parameters for diff")
	}

	g, ctx := errgroup.WithContext(ctx)
	var base profile.Profile
	g.Go(func() error {
		var err error
		base, err = q.selectProfileForDiff(ctx, d.A)
		if err != nil {
			return fmt.Errorf("reading base profile: %w", err)
		}
		return nil
	})

	var compare profile.Profile
	g.Go(func() error {
		var err error
		compare, err = q.selectProfileForDiff(ctx, d.B)
		if err != nil {
			return fmt.Errorf("reading compared profile: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return profile.Profile{}, err
	}

	labelFields, labelArrays, err := mergeDiffLabelColumns(q.mem, compare, base)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("merging label columns: %w", err)
	}

	compareColumns := compare.Samples.Columns()
	baseColumns := base.Samples.Columns()
	compareValueColumn := compareColumns[len(compareColumns)-2].(*array.Int64)
	baseValueColumn := baseColumns[len(baseColumns)-2].(*array.Int64)

	locationsColumn, err := array.Concatenate([]arrow.Array{compareColumns[len(compareColumns)-3], baseColumns[len(baseColumns)-3]}, q.mem)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("concatenate locations column: %w", err)
	}

	valueColumn, err := array.Concatenate([]arrow.Array{compareValueColumn, zeroArray(q.mem, int(base.Samples.NumRows()))}, q.mem)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("concatenate value column: %w", err)
	}

	// This is intentional, the diff value of the `compare` profile is the same
	// as the value of the `compare` profile, because what we're actually doing
	// is subtracting the `base` profile, but the actual calculation happens
	// when building the visualizations. We should eventually have this be done
	// directly by the query engine.
	diffColumn, err := array.Concatenate([]arrow.Array{compareValueColumn, multiplyInt64By(q.mem, baseValueColumn, -1)}, q.mem)
	if err != nil {
		return profile.Profile{}, fmt.Errorf("concatenate diff column: %w", err)
	}

	return profile.Profile{
		Meta: compare.Meta,
		Samples: array.NewRecord(
			profile.ArrowSchema(labelFields),
			append(
				labelArrays,
				locationsColumn,
				valueColumn,
				diffColumn,
			),
			compare.Samples.NumRows()+base.Samples.NumRows(),
		),
	}, nil
}

func multiplyInt64By(pool memory.Allocator, arr *array.Int64, factor int64) arrow.Array {
	b := array.NewInt64Builder(pool)
	defer b.Release()

	values := arr.Int64Values()
	valid := make([]bool, len(values))
	for i := range values {
		values[i] *= factor
		valid[i] = true
	}

	b.AppendValues(values, valid)
	return b.NewArray()
}

func zeroArray(pool memory.Allocator, rows int) arrow.Array {
	b := array.NewInt64Builder(pool)
	defer b.Release()

	values := make([]int64, rows)
	valid := make([]bool, len(values))
	for i := range values {
		valid[i] = true
	}

	b.AppendValues(values, valid)
	return b.NewArray()
}

func mergeDiffLabelColumns(pool memory.Allocator, compare, base profile.Profile) ([]arrow.Field, []arrow.Array, error) {
	labelFields := []arrow.Field{}
	columns := [][2]arrow.Array{}
	labelIndex := map[string]int{}
	for i, f := range compare.Samples.Schema().Fields() {
		if strings.HasPrefix(f.Name, profile.ColumnPprofLabelsPrefix) {
			labelFields = append(labelFields, f)
			columns = append(columns, [2]arrow.Array{
				compare.Samples.Column(i),
				nil,
			})
			labelIndex[f.Name] = len(labelFields) - 1
		}
	}

	for i, f := range base.Samples.Schema().Fields() {
		if strings.HasPrefix(f.Name, profile.ColumnPprofLabelsPrefix) {
			j, ok := labelIndex[f.Name]
			if !ok {
				labelFields = append(labelFields, f)
				columns = append(columns, [2]arrow.Array{
					nil,
					base.Samples.Column(i),
				})
				labelIndex[f.Name] = len(labelFields) - 1
			}
			columns[j][1] = compare.Samples.Column(i)
		}
	}

	firstLength := compare.Samples.NumRows()
	secondLength := base.Samples.NumRows()

	for _, cols := range columns {
		if cols[0] == nil {
			builder := array.NewBuilder(pool, cols[1].DataType())
			builder.AppendNulls(int(firstLength))
			cols[0] = builder.NewArray()
		}
		if cols[1] == nil {
			builder := array.NewBuilder(pool, cols[0].DataType())
			builder.AppendNulls(int(secondLength))
			cols[0] = builder.NewArray()
		}
	}

	labelArrays := make([]arrow.Array, len(labelFields))
	var err error
	for i, cols := range columns {
		labelArrays[i], err = array.Concatenate(cols[:], pool)
		if err != nil {
			return nil, nil, fmt.Errorf("concatenate label column %q: %w", labelFields[i].Name, err)
		}
	}

	return labelFields, labelArrays, nil
}

func (q *ColumnQueryAPI) selectProfileForDiff(ctx context.Context, s *pb.ProfileDiffSelection) (profile.Profile, error) {
	switch s.Mode {
	case pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		return q.selectSingle(ctx, s.GetSingle())
	case pb.ProfileDiffSelection_MODE_MERGE:
		return q.selectMerge(ctx, s.GetMerge())
	default:
		return profile.Profile{}, status.Error(codes.InvalidArgument, "unknown mode for diff profile selection")
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
