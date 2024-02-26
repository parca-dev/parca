// Copyright 2022-2024 The Parca Authors
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

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/bitutil"
	"github.com/apache/arrow/go/v15/arrow/math"
	"github.com/apache/arrow/go/v15/arrow/memory"
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
	QueryMerge(ctx context.Context, query string, start, end time.Time, aggregateByLabels bool) (profile.Profile, error)
}

var (
	ErrSourceNotFound     = errors.New("Source file not found. Either profiling metadata is wrong, or the referenced file was not included in the uploaded sources.")
	ErrNoSourceForBuildID = errors.New("No sources for this build id have been uploaded.")
)

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
		if req.SourceReference.SourceOnly {
			exists, err := q.sourceUploadExistsForBuildID(ctx, req.SourceReference)
			if err != nil {
				return nil, err
			}

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
		isDiff   bool
	)

	groupBy := req.GetGroupBy().GetFields()
	allowedGroupBy := map[string]struct{}{
		FlamegraphFieldFunctionName:     {},
		FlamegraphFieldLabels:           {},
		FlamegraphFieldLocationAddress:  {},
		FlamegraphFieldMappingFile:      {},
		FlamegraphFieldFunctionFileName: {},
	}
	groupByLabels := false
	for _, f := range groupBy {
		if f == FlamegraphFieldLabels {
			groupByLabels = true
		}
		if _, allowed := allowedGroupBy[f]; !allowed {
			return nil, status.Errorf(codes.InvalidArgument, "invalid group by field: %s", f)
		}
	}

	switch req.Mode {
	case pb.QueryRequest_MODE_SINGLE_UNSPECIFIED:
		p, err = q.selectSingle(ctx, req.GetSingle())
	case pb.QueryRequest_MODE_MERGE:
		p, err = q.selectMerge(
			ctx,
			req.GetMerge(),
			groupByLabels,
		)
	case pb.QueryRequest_MODE_DIFF:
		isDiff = true
		p, err = q.selectDiff(
			ctx,
			req.GetDiff(),
			groupByLabels,
		)
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, r := range p.Samples {
			r.Release()
		}
	}()

	p.Samples, filtered, err = FilterProfileData(
		ctx,
		q.tracer,
		q.mem,
		p.Samples,
		req.GetFilterQuery(),
		req.GetRuntimeFilter(),
	)
	if err != nil {
		return nil, fmt.Errorf("filtering profile: %w", err)
	}

	return q.renderReport(
		ctx,
		p,
		req.GetReportType(),
		req.GetNodeTrimThreshold(),
		filtered,
		groupBy,
		req.GetSourceReference(),
		source,
		isDiff,
	)
}

func FilterProfileData(
	ctx context.Context,
	tracer trace.Tracer,
	pool memory.Allocator,
	records []arrow.Record,
	filterQuery string,
	runtimeFilter *pb.RuntimeFilter,
) ([]arrow.Record, int64, error) {
	_, span := tracer.Start(ctx, "filterByFunction")
	defer span.End()

	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()

	// We want to filter by function name case-insensitive, so we need to lowercase the query.
	// We lower case the query here, so we don't have to do it for every sample.
	filterQueryBytes := []byte(strings.ToLower(filterQuery))
	res := make([]arrow.Record, 0, len(records))
	allValues := int64(0)
	allFiltered := int64(0)

	if runtimeFilter == nil {
		runtimeFilter = &pb.RuntimeFilter{}
	}

	binariesToExclude := [][]byte{}

	interpretedOnly := false
	if runtimeFilter != nil {
		if !runtimeFilter.ShowPython {
			binariesToExclude = append(binariesToExclude, []byte("libpython"))
		}
		if !runtimeFilter.ShowRuby {
			binariesToExclude = append(binariesToExclude, []byte("libruby"))
		}
		interpretedOnly = runtimeFilter.ShowInterpretedOnly
	}

	for _, r := range records {
		filteredRecords, valueSum, filteredSum, err := filterRecord(
			ctx,
			tracer,
			pool,
			r,
			filterQueryBytes,
			binariesToExclude,
			interpretedOnly,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("filter record: %w", err)
		}

		if len(filteredRecords) != 0 {
			res = append(res, filteredRecords...)
		}
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
	filterQueryBytes []byte,
	binariesToExclude [][]byte,
	showInterpretedOnly bool,
) ([]arrow.Record, int64, int64, error) {
	r := profile.NewRecordReader(rec)

	indexMatches := map[uint32]struct{}{}
	for i := 0; i < r.LineFunctionNameDict.Len(); i++ {
		if bytes.Contains(bytes.ToLower(r.LineFunctionNameDict.Value(i)), filterQueryBytes) {
			indexMatches[uint32(i)] = struct{}{}
		}
	}

	if len(indexMatches) == 0 {
		return nil, math.Int64.Sum(r.Value), 0, nil
	}

	rowsToKeep := make([]int64, 0, int(rec.NumRows()))
	for i := 0; i < int(rec.NumRows()); i++ {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
		keepRow := false
		if len(filterQueryBytes) > 0 {
			if lOffsetStart < lOffsetEnd {
				firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
				_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))
				for k := int(firstStart); k < int(lastEnd); k++ {
					if r.LineFunctionNameIndices.IsValid(k) {
						if _, ok := indexMatches[r.LineFunctionNameIndices.Value(k)]; ok {
							keepRow = true
							break
						}
					}
				}
			}
		} else {
			keepRow = true
		}

		if !keepRow {
			continue
		}

		rowsToKeep = append(rowsToKeep, int64(i))
		if lOffsetEnd-lOffsetStart > 0 {
			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				validMappingStart := r.MappingStart.IsValid(j)
				var mappingFile []byte
				skipLocation := false
				if validMappingStart {
					mappingFile = r.MappingFileDict.Value(int(r.MappingFileIndices.Value(j)))
					lastSlash := bytes.LastIndex(mappingFile, []byte("/"))
					mappingFileBase := mappingFile
					if lastSlash >= 0 {
						mappingFileBase = mappingFile[lastSlash+1:]
					}
					if len(mappingFileBase) > 0 {
						for _, binaryToExclude := range binariesToExclude {
							if bytes.HasPrefix(mappingFileBase, binaryToExclude) {
								skipLocation = true
								break
							}
						}
					}
				}
				if skipLocation {
					bitutil.ClearBit(r.Locations.ListValues().NullBitmapBytes(), j)
					continue
				}
				if showInterpretedOnly && !bytes.Equal(mappingFile, []byte("interpreter")) {
					bitutil.ClearBit(r.Locations.ListValues().NullBitmapBytes(), j)
					continue
				}
			}
		}
	}

	// Split the record into slices based on the rowsToKeep.
	recs := sliceRecord(rec, rowsToKeep)

	filtered := int64(0)
	for _, r := range recs {
		filtered += math.Int64.Sum(profile.NewRecordReader(r).Value)
	}

	return recs,
		math.Int64.Sum(r.Value),
		filtered,
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
	isDiff bool,
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
		isDiff,
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
	isDiff bool,
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
		pp, err := GenerateFlatPprof(ctx, isDiff, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		buf, err := SerializePprof(pp)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate pprof: %v", err.Error())
		}

		return &pb.QueryResponse{
			Total:    0, // TODO: Figure out how to get total for pprof
			Filtered: filtered,
			Report:   &pb.QueryResponse_Pprof{Pprof: buf},
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
	case pb.QueryRequest_REPORT_TYPE_TABLE_ARROW:
		table, cumulative, err := GenerateTable(ctx, mem, tracer, p)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to generate table: %v", err.Error())
		}

		return &pb.QueryResponse{
			Total:    cumulative,
			Filtered: filtered,
			Report:   &pb.QueryResponse_TableArrow{TableArrow: table},
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

func (q *ColumnQueryAPI) selectMerge(
	ctx context.Context,
	m *pb.MergeProfile,
	aggregateByLabels bool,
) (profile.Profile, error) {
	p, err := q.querier.QueryMerge(
		ctx,
		m.Query,
		m.Start.AsTime(),
		m.End.AsTime(),
		aggregateByLabels,
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectDiff(ctx context.Context, d *pb.DiffProfile, aggregateByLabels bool) (profile.Profile, error) {
	ctx, span := q.tracer.Start(ctx, "diffRequest")
	defer span.End()

	if d == nil {
		return profile.Profile{}, status.Error(codes.InvalidArgument, "requested diff mode, but did not provide parameters for diff")
	}

	g, ctx := errgroup.WithContext(ctx)
	var base profile.Profile
	defer func() {
		for _, r := range base.Samples {
			r.Release()
		}
	}()
	g.Go(func() error {
		var err error
		base, err = q.selectProfileForDiff(ctx, d.A, aggregateByLabels)
		if err != nil {
			return fmt.Errorf("reading base profile: %w", err)
		}
		return nil
	})

	var compare profile.Profile
	defer func() {
		for _, r := range compare.Samples {
			r.Release()
		}
	}()
	g.Go(func() error {
		var err error
		compare, err = q.selectProfileForDiff(ctx, d.B, aggregateByLabels)
		if err != nil {
			return fmt.Errorf("reading compared profile: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return profile.Profile{}, err
	}

	return ComputeDiff(ctx, q.tracer, base, compare, q.mem)
}

func ComputeDiff(ctx context.Context, tracer trace.Tracer, base, compare profile.Profile, mem memory.Allocator) (profile.Profile, error) {
	_, span := tracer.Start(ctx, "ComputeDiff")
	defer span.End()

	records := make([]arrow.Record, 0, len(compare.Samples)+len(base.Samples))

	for _, r := range compare.Samples {
		columns := r.Columns()
		cols := make([]arrow.Array, len(columns))
		copy(cols, columns)
		// This is intentional, the diff value of the `compare` profile is the same
		// as the value of the `compare` profile, because what we're actually doing
		// is subtracting the `base` profile, but the actual calculation happens
		// when building the visualizations. We should eventually have this be done
		// directly by the query engine.
		cols[len(cols)-1] = cols[len(cols)-2]
		records = append(records, array.NewRecord(
			r.Schema(),
			cols,
			r.NumRows(),
		))
	}

	for _, r := range base.Samples {
		func() {
			columns := r.Columns()
			cols := make([]arrow.Array, len(columns))
			copy(cols, columns)
			mult := multiplyInt64By(mem, columns[len(columns)-2].(*array.Int64), -1)
			defer mult.Release()
			zero := zeroArray(mem, int(r.NumRows()))
			defer zero.Release()
			records = append(records, array.NewRecord(
				r.Schema(),
				append(cols[:len(cols)-2], zero, mult),
				r.NumRows(),
			))
		}()
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

func (q *ColumnQueryAPI) selectProfileForDiff(ctx context.Context, s *pb.ProfileDiffSelection, aggregateByLabels bool) (profile.Profile, error) {
	switch s.Mode {
	case pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		return q.selectSingle(ctx, s.GetSingle())
	case pb.ProfileDiffSelection_MODE_MERGE:
		return q.selectMerge(ctx, s.GetMerge(), aggregateByLabels)
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

type IndexRange struct {
	Start int64
	End   int64
}

// sliceRecord returns a set of continguous index ranges from the given indicies
// ex: [1,2,7,8,9] would return two records of [{Start:1, End:3},{Start:7,End:10}]
func sliceRecord(r arrow.Record, indices []int64) []arrow.Record {
	if len(indices) == 0 {
		return []arrow.Record{}
	}

	slices := []arrow.Record{}
	cur := IndexRange{
		Start: indices[0],
		End:   indices[0] + 1,
	}

	for _, i := range indices[1:] {
		if i == cur.End {
			cur.End++
		} else {
			slices = append(slices, r.NewSlice(cur.Start, cur.End))
			cur = IndexRange{
				Start: i,
				End:   i + 1,
			}
		}
	}

	slices = append(slices, r.NewSlice(cur.Start, cur.End))
	return slices
}
