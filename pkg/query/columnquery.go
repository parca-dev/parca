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
	"errors"
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

var ErrSourceNotFound = errors.New("Source file not found. Either profiling metadata is wrong, or the referenced file was not included in the uploaded sources.")
var ErrNoSourceForBuildID = errors.New("No sources for this build id have been uploaded.")

type SourceFinder interface {
	FindSource(ctx context.Context, ref *pb.SourceReference) (string, error)
	SourceExists(ctx context.Context, ref *pb.SourceReference) (bool, error)
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

	sourceFinder SourceFinder
}

func NewColumnQueryAPI(
	logger log.Logger,
	tracer trace.Tracer,
	shareClient sharepb.ShareServiceClient,
	querier Querier,
	mem memory.Allocator,
	converter *parcacol.ArrowToProfileConverter,
	sourceFinder SourceFinder,
) *ColumnQueryAPI {
	return &ColumnQueryAPI{
		logger:             logger,
		tracer:             tracer,
		shareClient:        shareClient,
		querier:            querier,
		tableConverterPool: NewTableConverterPool(),
		mem:                mem,
		converter:          converter,
		sourceFinder:       sourceFinder,
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

func (q *ColumnQueryAPI) getSource(ctx context.Context, ref *pb.SourceReference) (string, error) {
	return q.sourceFinder.FindSource(ctx, ref)
}

func (q *ColumnQueryAPI) sourceUploadExistsForBuildID(ctx context.Context, ref *pb.SourceReference) (bool, error) {
	return q.sourceFinder.SourceExists(ctx, ref)
}

// Query issues an instant query against the storage.
func (q *ColumnQueryAPI) Query(ctx context.Context, req *pb.QueryRequest) (*pb.QueryResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	var (
		source string
		err    error
	)
	if req.SourceReference != nil {
		exists, err := q.sourceUploadExistsForBuildID(ctx, req.SourceReference)
		if err != nil {
			if errors.Is(err, ErrNoSourceForBuildID) {
				return nil, status.Error(codes.NotFound, err.Error())
			}
			return nil, err
		}

		if req.SourceReference.SourceOnly {
			if !exists {
				return nil, status.Error(codes.NotFound, ErrNoSourceForBuildID.Error())
			}
			return &pb.QueryResponse{
				Report: &pb.QueryResponse_Source{
					Source: &pb.Source{},
				},
			}, nil
		}

		source, err = q.getSource(ctx, req.SourceReference)
		if err != nil {
			if errors.Is(err, ErrSourceNotFound) || errors.Is(err, ErrNoSourceForBuildID) {
				return nil, status.Error(codes.NotFound, err.Error())
			}
			return nil, err
		}
	}

	var (
		p        profile.Profile
		filtered int64
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
		p.Samples, filtered, err = FilterProfileData(ctx, q.tracer, q.mem, p.Samples, req.GetFilterQuery())
		if err != nil {
			return nil, fmt.Errorf("filtering profile: %w", err)
		}
	}
	defer func() {
		for _, r := range p.Samples {
			r.Release()
		}
	}()

	return q.renderReport(
		ctx,
		p,
		req.GetReportType(),
		req.GetNodeTrimThreshold(),
		filtered,
		req.GetGroupBy().GetFields(),
		req.GetSourceReference(),
		source,
	)
}

func FilterProfileData(
	ctx context.Context,
	tracer trace.Tracer,
	pool memory.Allocator,
	records []arrow.Record,
	filterQuery string,
) ([]arrow.Record, int64, error) {
	_, span := tracer.Start(ctx, "filterByFunction")
	defer span.End()

	// TODO: This is a bit inefficient because it completely rebuilds the
	// profile, we should only ever rebuild dictionaries once at the very end
	// before we send a result to the user. Because we are rebuilding the
	// records whe need to release the previous ones.
	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()

	// We want to filter by function name case-insensitive, so we need to lowercase the query.
	// We lower case the query here, so we don't have to do it for every sample.
	filterQuery = strings.ToLower(filterQuery)
	res := make([]arrow.Record, 0, len(records))
	allValues := int64(0)
	allFiltered := int64(0)

	for _, r := range records {
		filteredRecord, valueSum, filteredSum, err := filterRecord(ctx, tracer, pool, r, filterQuery)
		if err != nil {
			return nil, 0, fmt.Errorf("filter record: %w", err)
		}

		res = append(res, filteredRecord)
		allValues += valueSum
		allFiltered += filteredSum
	}

	return res, allValues - allFiltered, nil
}

func filterRecord(
	ctx context.Context,
	tracer trace.Tracer,
	pool memory.Allocator,
	rec arrow.Record,
	filterQuery string,
) (arrow.Record, int64, int64, error) {
	r := profile.NewRecordReader(rec)

	// Builders for the result profile.
	labelNames := make([]string, 0, len(r.LabelFields))
	for _, lf := range r.LabelFields {
		labelNames = append(labelNames, strings.TrimPrefix(lf.Name, profile.ColumnPprofLabelsPrefix))
	}

	w := profile.NewWriter(pool, labelNames)
	defer w.RecordBuilder.Release()

	for i := 0; i < int(rec.NumRows()); i++ {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
		keepRow := false
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)

			for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
				if r.LineFunction.IsValid(k) && bytes.Contains(bytes.ToLower(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(k))), []byte(filterQuery)) {
					keepRow = true
					break
				}
			}
		}

		if keepRow {
			w.Value.Append(r.Value.Value(i))
			w.Diff.Append(r.Diff.Value(i))

			for j, label := range r.LabelColumns {
				if label.Col.IsValid(i) {
					if err := w.LabelBuilders[j].Append([]byte(label.Dict.Value(label.Col.GetValueIndex(i)))); err != nil {
						return nil, 0, 0, fmt.Errorf("append label: %w", err)
					}
				} else {
					w.LabelBuilders[j].AppendNull()
				}
			}

			if lOffsetEnd-lOffsetStart > 0 {
				w.LocationsList.Append(true)
				for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
					w.Locations.Append(true)
					w.Addresses.Append(r.Address.Value(j))

					if r.Mapping.IsValid(j) {
						w.Mapping.Append(true)
						w.MappingStart.Append(r.MappingStart.Value(j))
						w.MappingLimit.Append(r.MappingLimit.Value(j))
						w.MappingOffset.Append(r.MappingOffset.Value(j))
						if r.MappingFileDict.Len() == 0 {
							w.MappingFile.AppendNull()
						} else {
							if err := w.MappingFile.Append(r.MappingFileDict.Value(r.MappingFile.GetValueIndex(j))); err != nil {
								return nil, 0, 0, fmt.Errorf("append mapping file: %w", err)
							}
						}
						if r.MappingBuildIDDict.Len() == 0 {
							w.MappingBuildID.AppendNull()
						} else {
							if err := w.MappingBuildID.Append(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(j))); err != nil {
								return nil, 0, 0, fmt.Errorf("append mapping build id: %w", err)
							}
						}
					} else {
						w.Mapping.AppendNull()
					}

					if r.Lines.IsValid(j) {
						llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)
						if llOffsetEnd-llOffsetStart > 0 {
							w.Lines.Append(true)
							for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
								w.Line.Append(true)
								w.LineNumber.Append(r.LineNumber.Value(k))
								w.Function.Append(r.LineFunction.IsValid(k))

								if r.LineFunction.IsValid(k) {
									if r.LineFunctionNameDict.Len() == 0 {
										w.FunctionName.AppendNull()
									} else {
										if err := w.FunctionName.Append(r.LineFunctionNameDict.Value(r.LineFunctionName.GetValueIndex(k))); err != nil {
											return nil, 0, 0, fmt.Errorf("append function name: %w", err)
										}
									}
									if r.LineFunctionSystemNameDict.Len() == 0 {
										w.FunctionSystemName.AppendNull()
									} else {
										if err := w.FunctionSystemName.Append(r.LineFunctionSystemNameDict.Value(r.LineFunctionSystemName.GetValueIndex(k))); err != nil {
											return nil, 0, 0, fmt.Errorf("append function system name: %w", err)
										}
									}
									if r.LineFunctionFilenameDict.Len() == 0 {
										w.FunctionFilename.AppendNull()
									} else {
										if err := w.FunctionFilename.Append(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(k))); err != nil {
											return nil, 0, 0, fmt.Errorf("append function filename: %w", err)
										}
									}
									w.FunctionStartLine.Append(r.LineFunctionStartLine.Value(k))
								}
							}
							continue
						}
					}
					w.Lines.AppendNull()
				}
			} else {
				w.LocationsList.Append(false)
			}
		}
	}

	res := w.RecordBuilder.NewRecord()
	numFields := res.Schema().NumFields()
	filteredValue := res.Column(numFields - 2).(*array.Int64)

	return res,
		math.Int64.Sum(r.Value),
		math.Int64.Sum(filteredValue),
		nil
}

func (q *ColumnQueryAPI) renderReport(
	ctx context.Context,
	p profile.Profile,
	typ pb.QueryRequest_ReportType,
	nodeTrimThreshold float32,
	filtered int64,
	groupBy []string,
	sourceReference *pb.SourceReference,
	source string,
) (*pb.QueryResponse, error) {
	return RenderReport(
		ctx,
		q.tracer,
		p,
		typ,
		nodeTrimThreshold,
		filtered,
		groupBy,
		q.tableConverterPool,
		q.mem,
		q.converter,
		sourceReference,
		source,
	)
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
	sourceReference *pb.SourceReference,
	source string,
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
	case pb.QueryRequest_REPORT_TYPE_SOURCE:
		s, total, err := GenerateSourceReport(
			ctx,
			mem,
			tracer,
			p,
			sourceReference,
			source,
		)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate arrow flamegraph: %v", err.Error())
		}

		return &pb.QueryResponse{
			Total:    total,
			Filtered: filtered,
			Report: &pb.QueryResponse_Source{
				Source: s,
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

	records := make([]arrow.Record, 0, len(compare.Samples)+len(base.Samples))

	for _, r := range compare.Samples {
		columns := r.Columns()
		// This is intentional, the diff value of the `compare` profile is the same
		// as the value of the `compare` profile, because what we're actually doing
		// is subtracting the `base` profile, but the actual calculation happens
		// when building the visualizations. We should eventually have this be done
		// directly by the query engine.
		columns[len(columns)-1] = columns[len(columns)-2]
		records = append(records, array.NewRecord(
			r.Schema(),
			columns,
			r.NumRows(),
		))
		r.Release()
	}

	for _, r := range base.Samples {
		columns := r.Columns()
		// This has to be this order as we're overriding the value column (-2)
		// in the next line.
		columns[len(columns)-1] = multiplyInt64By(q.mem, columns[len(columns)-2].(*array.Int64), -1)
		columns[len(columns)-2] = zeroArray(q.mem, int(r.NumRows()))
		records = append(records, array.NewRecord(
			r.Schema(),
			columns,
			r.NumRows(),
		))
		r.Release()
	}

	return profile.Profile{
		Meta:    compare.Meta,
		Samples: records,
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
