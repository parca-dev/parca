// Copyright 2022-2025 The Parca Authors
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

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/bitutil"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/go-kit/log"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/polarsignals/frostdb/pqarrow/arrowutils"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	sharepb "github.com/parca-dev/parca/gen/proto/go/parca/share/v1alpha1"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

type Querier interface {
	Labels(ctx context.Context, match []string, start, end time.Time, profileType string) ([]string, error)
	Values(ctx context.Context, labelName string, match []string, start, end time.Time, profileType string) ([]string, error)
	QueryRange(ctx context.Context, query string, startTime, endTime time.Time, step time.Duration, limit uint32, sumBy []string) ([]*pb.MetricsSeries, error)
	ProfileTypes(ctx context.Context, startTime, endTime time.Time) ([]*pb.ProfileType, error)
	QuerySingle(ctx context.Context, query string, time time.Time, invertCallStacks bool) (profile.Profile, error)
	QueryMerge(ctx context.Context, query string, start, end time.Time, aggregateByLabels []string, invertCallStacks bool, functionToFilterBy string) (profile.Profile, error)
	GetProfileMetadataMappings(ctx context.Context, query string, start, end time.Time) ([]string, error)
	GetProfileMetadataLabels(ctx context.Context, query string, start, end time.Time) ([]string, error)
}

var (
	ErrSourceNotFound     = errors.New("source file not found; either profiling metadata is wrong, or the referenced file was not included in the uploaded sources")
	ErrNoSourceForBuildID = errors.New("no sources for this build id have been uploaded")
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
	profileType := ""
	if req.ProfileType != nil {
		profileType = *req.ProfileType
	}
	vals, err := q.querier.Labels(ctx, req.Match, req.Start.AsTime(), req.End.AsTime(), profileType)
	if err != nil {
		return nil, err
	}

	return &pb.LabelsResponse{
		LabelNames: vals,
	}, nil
}

// Values issues a values request against the storage.
func (q *ColumnQueryAPI) Values(ctx context.Context, req *pb.ValuesRequest) (*pb.ValuesResponse, error) {
	profileType := ""
	if req.ProfileType != nil {
		profileType = *req.ProfileType
	}
	vals, err := q.querier.Values(ctx, req.LabelName, req.Match, req.Start.AsTime(), req.End.AsTime(), profileType)
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

	res, err := q.querier.QueryRange(ctx, req.Query, req.Start.AsTime(), req.End.AsTime(), req.Step.AsDuration(), req.Limit, req.SumBy)
	if err != nil {
		return nil, err
	}

	return &pb.QueryRangeResponse{
		Series: res,
	}, nil
}

// Types returns the available types of profiles.
func (q *ColumnQueryAPI) ProfileTypes(ctx context.Context, req *pb.ProfileTypesRequest) (*pb.ProfileTypesResponse, error) {
	types, err := q.querier.ProfileTypes(ctx, req.Start.AsTime(), req.End.AsTime())
	if err != nil {
		return nil, err
	}

	return &pb.ProfileTypesResponse{
		Types: types,
	}, nil
}

func (q *ColumnQueryAPI) HasProfileData(ctx context.Context, req *pb.HasProfileDataRequest) (*pb.HasProfileDataResponse, error) {
	res, err := q.ProfileTypes(ctx, &pb.ProfileTypesRequest{})
	if err != nil {
		return nil, err
	}

	return &pb.HasProfileDataResponse{
		HasData: len(res.Types) > 0,
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
		profileMetadata *pb.ProfileMetadata
		p               profile.Profile
		filtered        int64
		isDiff          bool
		isInvert        bool
	)

	if req.InvertCallStack != nil {
		isInvert = *req.InvertCallStack
	}

	groupBy := req.GetGroupBy().GetFields()
	allowedGroupBy := map[string]struct{}{
		FlamegraphFieldFunctionFileName: {},
		FlamegraphFieldFunctionName:     {},
		FlamegraphFieldLocationAddress:  {},
		FlamegraphFieldMappingFile:      {},
		FlamegraphFieldTimestamp:        {},
	}

	if req.GetReportType() == pb.QueryRequest_REPORT_TYPE_FLAMECHART {
		groupBy = append(groupBy, FlamegraphFieldTimestamp)
	}

	groupByLabels := make([]string, 0, len(groupBy))
	for _, f := range groupBy {
		if strings.HasPrefix(f, FlamegraphFieldLabels+".") {
			// Add label to the groupByLabels passed to FrostDB
			groupByLabels = append(groupByLabels, f)
			continue
		}
		if _, allowed := allowedGroupBy[f]; allowed {
			groupByLabels = append(groupByLabels, f)
			continue
		}
		return nil, status.Errorf(codes.InvalidArgument, "invalid group by field: %s", f)
	}

	switch req.Mode {
	case pb.QueryRequest_MODE_SINGLE_UNSPECIFIED:
		p, err = q.selectSingle(ctx, req.GetSingle(), isInvert)
	case pb.QueryRequest_MODE_MERGE:
		switch req.GetReportType() {
		case pb.QueryRequest_REPORT_TYPE_PROFILE_METADATA:
			mappingFiles, labels, err := getMappingFilesAndLabels(ctx, q.querier, req.GetMerge().Query, req.GetMerge().Start.AsTime(), req.GetMerge().End.AsTime())
			if err != nil {
				return nil, err
			}

			profileMetadata = &pb.ProfileMetadata{
				MappingFiles: mappingFiles,
				Labels:       labels,
			}

		default:
			p, err = q.selectMerge(
				ctx,
				req.GetMerge(),
				groupByLabels,
				isInvert,
				req.GetSandwichByFunction(),
			)
		}
	case pb.QueryRequest_MODE_DIFF:
		isDiff = true
		switch req.GetReportType() {
		case pb.QueryRequest_REPORT_TYPE_PROFILE_METADATA:
			// When comparing, we only return the metadata for the profile we are rendering, which is the profile B.
			mappingFiles, labels, err := getMappingFilesAndLabels(ctx, q.querier, req.GetDiff().B.GetMerge().GetQuery(), req.GetDiff().B.GetMerge().Start.AsTime(), req.GetDiff().B.GetMerge().End.AsTime())
			if err != nil {
				return nil, err
			}

			profileMetadata = &pb.ProfileMetadata{
				MappingFiles: mappingFiles,
				Labels:       labels,
			}
		default:
			p, err = q.selectDiff(
				ctx,
				req.GetDiff(),
				false,
				isInvert,
			)
		}
	default:
		return nil, status.Error(codes.InvalidArgument, "unknown query mode")
	}
	if err != nil {
		return nil, err
	}
	if req.GetReportType() == pb.QueryRequest_REPORT_TYPE_PROFILE_METADATA {
		return &pb.QueryResponse{
			Total:    0,
			Filtered: 0,
			Report:   &pb.QueryResponse_ProfileMetadata{ProfileMetadata: profileMetadata},
		}, nil
	}
	defer func() {
		for _, r := range p.Samples {
			r.Release()
		}
	}()

	// Convert deprecated filters to new format for backward compatibility
	filters := ConvertDeprecatedFilters(req.GetFilter())

	p.Samples, filtered, err = FilterProfileData(
		ctx,
		q.tracer,
		q.mem,
		p.Samples,
		filters,
	)
	if err != nil {
		return nil, fmt.Errorf("filtering profile: %w", err)
	}

	// Apply sandwich view filtering if specified
	sandwichByFunction := req.GetSandwichByFunction()
	if sandwichByFunction != "" {
		// Create a stack filter for sandwich view
		sandwichFilter := &pb.Filter{
			Filter: &pb.Filter_StackFilter{
				StackFilter: &pb.StackFilter{
					Filter: &pb.StackFilter_Criteria{
						Criteria: &pb.FilterCriteria{
							FunctionName: &pb.StringCondition{
								Condition: &pb.StringCondition_Contains{
									Contains: sandwichByFunction,
								},
							},
						},
					},
				},
			},
		}
		// Combine existing filters with the sandwich filter
		sandwichFilters := make([]*pb.Filter, 0, len(filters)+1)
		sandwichFilters = append(sandwichFilters, filters...)
		sandwichFilters = append(sandwichFilters, sandwichFilter)

		var sandwichFiltered int64
		p.Samples, sandwichFiltered, err = FilterProfileData(
			ctx,
			q.tracer,
			q.mem,
			p.Samples,
			sandwichFilters,
		)
		if err != nil {
			return nil, fmt.Errorf("filtering profile for sandwich view: %w", err)
		}
		filtered += sandwichFiltered
	}

	return q.renderReport(
		ctx,
		p,
		req.GetReportType(),
		req.GetNodeTrimThreshold(),
		filtered,
		groupByLabels,
		req.GetSourceReference(),
		source,
		isDiff,
	)
}

func FilterProfileData(
	ctx context.Context,
	tracer trace.Tracer,
	pool memory.Allocator,
	records []arrow.RecordBatch,
	filters []*pb.Filter,
) ([]arrow.RecordBatch, int64, error) {
	_, span := tracer.Start(ctx, "filterByFunction")
	defer span.End()

	if len(filters) == 0 {
		// No filtering means all values are kept, so filtered count = 0
		return records, 0, nil
	}

	defer func() {
		for _, r := range records {
			r.Release()
		}
	}()

	res := make([]arrow.RecordBatch, 0, len(records))
	allValues := int64(0)
	allFiltered := int64(0)

	for _, r := range records {
		filteredRecords, valueSum, filteredSum, err := filterRecord(
			ctx,
			tracer,
			pool,
			r,
			filters,
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
	_ memory.Allocator,
	rec arrow.RecordBatch,
	filters []*pb.Filter,
) ([]arrow.RecordBatch, int64, int64, error) {
	_, span := tracer.Start(ctx, "filterRecord")
	defer span.End()

	r := profile.NewRecordReader(rec)

	// If no filters, return all records
	if len(filters) == 0 {
		valueSum := math.Int64.Sum(r.Value)
		return []arrow.RecordBatch{rec}, valueSum, valueSum, nil
	}

	stackFilters := make([]*pb.FilterCriteria, 0)
	frameFilters := make([]*pb.FilterCriteria, 0)

	for _, filter := range filters {
		if stackFilter := filter.GetStackFilter(); stackFilter != nil {
			if criteria := stackFilter.GetCriteria(); criteria != nil {
				stackFilters = append(stackFilters, criteria)
			}
		}
		if frameFilter := filter.GetFrameFilter(); frameFilter != nil {
			if criteria := frameFilter.GetCriteria(); criteria != nil {
				frameFilters = append(frameFilters, criteria)
			}
		}
	}

	// To keep track of which rows and frames to keep, we will build a list of row indices
	// and the corresponding frames to keep within those rows.
	type rowInfo struct {
		rowIndex     int
		framesToKeep []int // indices of frames to keep within the location list
	}

	rowsInfo := make([]rowInfo, 0)
	originalValueSum := math.Int64.Sum(r.Value)

	for i := 0; i < int(rec.NumRows()); i++ {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)

		// Check stack filters first
		keepRow := true
		if len(stackFilters) > 0 {
			stackMatches := true
			if lOffsetStart < lOffsetEnd {
				firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
				_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

				for _, filter := range stackFilters {
					if !stackMatchesFilter(r, int(firstStart), int(lastEnd-1), int(lOffsetStart), int(lOffsetEnd), filter) {
						stackMatches = false
						break
					}
				}
			}
			keepRow = stackMatches
		}

		if !keepRow {
			continue
		}

		// Apply frame filters - determine which frames to keep
		framesToKeep := make([]int, 0)
		if lOffsetEnd-lOffsetStart > 0 {
			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				lineStart, lineEnd := r.Lines.ValueOffsets(j)
				if lineStart >= lineEnd {
					// For Unsymbolized location, check at location level only
					if matchesAllFrameFilters(r, j, -1, frameFilters) {
						framesToKeep = append(framesToKeep, -1)
					}
				} else {
					// For Symbolized location, check each line/frame
					for lineIdx := int(lineStart); lineIdx < int(lineEnd); lineIdx++ {
						if matchesAllFrameFilters(r, j, lineIdx, frameFilters) {
							framesToKeep = append(framesToKeep, lineIdx)
						}
					}
				}
			}
		}

		// Only keep rows that have at least one frame after filtering
		if len(framesToKeep) > 0 {
			rowsInfo = append(rowsInfo, rowInfo{
				rowIndex:     i,
				framesToKeep: framesToKeep,
			})
		}
	}

	if len(rowsInfo) == 0 {
		// No rows match the filters
		return []arrow.RecordBatch{}, originalValueSum, 0, nil
	}

	// Now apply frame filtering by nulling out non-matching frames within rows
	for _, info := range rowsInfo {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(info.rowIndex)

		// Create a set of frames to keep for quick lookup
		keepSet := make(map[int]bool)
		for _, frameIdx := range info.framesToKeep {
			keepSet[frameIdx] = true
		}

		// Null out frames that don't match the filters
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			lineStart, lineEnd := r.Lines.ValueOffsets(j)
			allLinesFiltered := true

			// Check if any line in this location is kept (or if this is unsymbolized and kept)
			for lineIdx := int(lineStart); lineIdx < int(lineEnd); lineIdx++ {
				if keepSet[lineIdx] {
					allLinesFiltered = false
					break
				}
			}
			// Also check for unsymbolized case
			if keepSet[-1] {
				allLinesFiltered = false
			}

			// If all lines in this location are filtered out, invalidate the location
			if allLinesFiltered {
				bitutil.ClearBit(r.Locations.ListValues().NullBitmapBytes(), j)
			}

			// Null out individual line fields that don't match
			for lineIdx := int(lineStart); lineIdx < int(lineEnd); lineIdx++ {
				if !keepSet[lineIdx] {
					// Null out the frame fields for this line
					if r.LineFunctionNameIndices.Len() > 0 {
						bitutil.ClearBit(r.LineFunctionNameIndices.NullBitmapBytes(), lineIdx)
					}
					if r.LineFunctionSystemNameIndices.Len() > 0 {
						bitutil.ClearBit(r.LineFunctionSystemNameIndices.NullBitmapBytes(), lineIdx)
					}
					if r.LineFunctionFilenameIndices.Len() > 0 {
						bitutil.ClearBit(r.LineFunctionFilenameIndices.NullBitmapBytes(), lineIdx)
					}
					if r.LineFunctionStartLine.Len() > 0 {
						bitutil.ClearBit(r.LineFunctionStartLine.NullBitmapBytes(), lineIdx)
					}
					if r.LineNumber.Len() > 0 {
						bitutil.ClearBit(r.LineNumber.NullBitmapBytes(), lineIdx)
					}
				}
			}
		}
	}

	// Extract the rows we want to keep
	rowsToKeep := make([]int64, len(rowsInfo))
	for i, info := range rowsInfo {
		rowsToKeep[i] = int64(info.rowIndex)
	}

	// Split the record into slices based on the rowsToKeep
	recs := sliceRecord(rec, rowsToKeep)

	filtered := int64(0)
	for _, r := range recs {
		filtered += math.Int64.Sum(profile.NewRecordReader(r).Value)
	}

	return recs, originalValueSum, filtered, nil
}

// stackMatchesFilter checks if a stack matches the given filter criteria.
func stackMatchesFilter(r *profile.RecordReader, firstStart, lastEnd, locStart, locEnd int, filter *pb.FilterCriteria) bool {
	if fnCond := filter.GetFunctionName(); fnCond != nil {
		if r.LineFunctionNameIndices.Len() == 0 {
			return handleUnsymbolizedFunctionCondition(fnCond)
		}
		return matchesFunctionNameInRange(r, firstStart, lastEnd, fnCond)
	}

	if binCond := filter.GetBinary(); binCond != nil {
		found := false
		for locIdx := locStart; locIdx < locEnd; locIdx++ {
			if r.MappingStart.IsValid(locIdx) {
				mappingFile := r.MappingFileDict.Value(int(r.MappingFileIndices.Value(locIdx)))
				lastSlash := bytes.LastIndex(mappingFile, []byte("/"))
				mappingFileBase := mappingFile
				if lastSlash >= 0 {
					mappingFileBase = mappingFile[lastSlash+1:]
				}
				if matchesStringCondition(mappingFileBase, binCond) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	if sysCond := filter.GetSystemName(); sysCond != nil {
		if r.LineFunctionSystemNameIndices.Len() == 0 {
			return handleUnsymbolizedFunctionCondition(sysCond)
		}
		if !matchesSystemNameInRange(r, firstStart, lastEnd, sysCond) {
			return false
		}
	}

	if fileCond := filter.GetFilename(); fileCond != nil {
		if r.LineFunctionFilenameIndices.Len() == 0 {
			return handleUnsymbolizedFunctionCondition(fileCond)
		}
		if !matchesFilenameInRange(r, firstStart, lastEnd, fileCond) {
			return false
		}
	}

	if addrCond := filter.GetAddress(); addrCond != nil {
		found := false
		for locIdx := locStart; locIdx < locEnd; locIdx++ {
			if r.Address.IsValid(locIdx) {
				address := r.Address.Value(locIdx)
				if matchesNumberCondition(address, addrCond) {
					found = true
					break
				}
			}
		}
		if !found {
			return false
		}
	}

	if lineCond := filter.GetLineNumber(); lineCond != nil {
		if !matchesLineNumberInRange(r, firstStart, lastEnd, lineCond) {
			return false
		}
	}

	return true
}

func handleUnsymbolizedFunctionCondition(fnCond *pb.StringCondition) bool {
	switch fnCond.GetCondition().(type) {
	case *pb.StringCondition_Contains, *pb.StringCondition_Equal:
		// For Contains/Equal: no function names means no matches
		return false
	case *pb.StringCondition_NotContains, *pb.StringCondition_NotEqual:
		// For NotContains/NotEqual: no function names means condition is satisfied
		return true
	}
	return false
}

func matchesFunctionNameInRange(r *profile.RecordReader, firstStart, lastEnd int, fnCond *pb.StringCondition) bool {
	// For NotContains/NotEqual, we need ALL functions to not contain/equal the target (AND logic)
	// For Contains/Equal, we need ANY function to contain/equal the target (OR logic)
	isNegativeCondition := false
	switch fnCond.GetCondition().(type) {
	case *pb.StringCondition_NotContains, *pb.StringCondition_NotEqual:
		isNegativeCondition = true
	}

	// Iterate through all line indices in the range
	for lineIndex := firstStart; lineIndex <= lastEnd; lineIndex++ {
		if lineIndex >= r.LineFunctionNameIndices.Len() {
			break
		}

		// Check if this line has a valid function name
		if r.LineFunctionNameIndices.IsValid(lineIndex) {
			fnIndex := r.LineFunctionNameIndices.Value(lineIndex)
			functionName := r.LineFunctionNameDict.Value(int(fnIndex))

			if isNegativeCondition {
				// For negative conditions (NotContains/NotEqual), if ANY function matches the negative condition, return false
				if !matchesStringCondition(functionName, fnCond) {
					return false
				}
			} else {
				// For positive conditions (Contains/Equal), if ANY function matches, return true
				if matchesStringCondition(functionName, fnCond) {
					return true
				}
			}
		}
	}

	if isNegativeCondition {
		// For negative conditions, if we got here, ALL functions passed the negative condition
		return true
	} else {
		// For positive conditions, if we got here, NO function matched
		return false
	}
}

func matchesSystemNameInRange(r *profile.RecordReader, firstStart, lastEnd int, sysCond *pb.StringCondition) bool {
	isNegativeCondition := false
	switch sysCond.GetCondition().(type) {
	case *pb.StringCondition_NotContains, *pb.StringCondition_NotEqual:
		isNegativeCondition = true
	}

	for lineIndex := firstStart; lineIndex <= lastEnd; lineIndex++ {
		if lineIndex >= r.LineFunctionSystemNameIndices.Len() {
			break
		}

		if r.LineFunctionSystemNameIndices.IsValid(lineIndex) {
			sysIndex := r.LineFunctionSystemNameIndices.Value(lineIndex)
			systemName := r.LineFunctionSystemNameDict.Value(int(sysIndex))

			if isNegativeCondition {
				if !matchesStringCondition(systemName, sysCond) {
					return false
				}
			} else {
				if matchesStringCondition(systemName, sysCond) {
					return true
				}
			}
		}
	}

	return isNegativeCondition
}

func matchesFilenameInRange(r *profile.RecordReader, firstStart, lastEnd int, fileCond *pb.StringCondition) bool {
	isNegativeCondition := false
	switch fileCond.GetCondition().(type) {
	case *pb.StringCondition_NotContains, *pb.StringCondition_NotEqual:
		isNegativeCondition = true
	}

	for lineIndex := firstStart; lineIndex <= lastEnd; lineIndex++ {
		if lineIndex >= r.LineFunctionFilenameIndices.Len() {
			break
		}

		if r.LineFunctionFilenameIndices.IsValid(lineIndex) {
			fileIndex := r.LineFunctionFilenameIndices.Value(lineIndex)
			filename := r.LineFunctionFilenameDict.Value(int(fileIndex))

			if isNegativeCondition {
				if !matchesStringCondition(filename, fileCond) {
					return false
				}
			} else {
				if matchesStringCondition(filename, fileCond) {
					return true
				}
			}
		}
	}

	return isNegativeCondition
}

func matchesLineNumberInRange(r *profile.RecordReader, firstStart, lastEnd int, lineCond *pb.NumberCondition) bool {
	isNegativeCondition := false
	switch lineCond.GetCondition().(type) {
	case *pb.NumberCondition_NotEqual:
		isNegativeCondition = true
	}

	for lineIndex := firstStart; lineIndex <= lastEnd; lineIndex++ {
		if lineIndex >= r.LineNumber.Len() {
			break
		}

		if r.LineNumber.IsValid(lineIndex) {
			lineNumber := uint64(r.LineNumber.Value(lineIndex))

			if isNegativeCondition {
				if !matchesNumberCondition(lineNumber, lineCond) {
					return false
				}
			} else {
				if matchesNumberCondition(lineNumber, lineCond) {
					return true
				}
			}
		}
	}

	return isNegativeCondition
}

func matchesAllFrameFilters(r *profile.RecordReader, locationIndex, lineIndex int, frameFilters []*pb.FilterCriteria) bool {
	// If no frame filters are provided, keep all frames
	if len(frameFilters) == 0 {
		return true
	}

	for _, filter := range frameFilters {
		if !matchesFrameFilter(r, locationIndex, lineIndex, filter) {
			return false
		}
	}
	return true
}

// matchesFrameFilter checks if a single frame matches the filter criteria.
func matchesFrameFilter(r *profile.RecordReader, locationIndex, lineIndex int, filter *pb.FilterCriteria) bool {
	if fnCond := filter.GetFunctionName(); fnCond != nil {
		// If lineIndex is -1, skip function name check
		if lineIndex >= 0 {
			// If unsymbolized, always return false
			if r.LineFunctionNameIndices.Len() == 0 {
				return false
			}
			if r.LineFunctionNameIndices.IsValid(lineIndex) {
				fnIndex := r.LineFunctionNameIndices.Value(lineIndex)
				functionName := r.LineFunctionNameDict.Value(int(fnIndex))
				if !matchesStringCondition(functionName, fnCond) {
					return false
				}
			} else {
				// Frame has no function name, so function name filter doesn't match
				return false
			}
		}
	}

	if binCond := filter.GetBinary(); binCond != nil {
		if r.MappingStart.IsValid(locationIndex) {
			mappingFile := r.MappingFileDict.Value(int(r.MappingFileIndices.Value(locationIndex)))
			lastSlash := bytes.LastIndex(mappingFile, []byte("/"))
			mappingFileBase := mappingFile
			if lastSlash >= 0 {
				mappingFileBase = mappingFile[lastSlash+1:]
			}
			if !matchesStringCondition(mappingFileBase, binCond) {
				return false
			}
		}
	}

	if sysCond := filter.GetSystemName(); sysCond != nil {
		// If lineIndex is -1, skip system name check
		if lineIndex >= 0 {
			if r.LineFunctionSystemNameIndices.Len() == 0 {
				return false
			}
			if r.LineFunctionSystemNameIndices.IsValid(lineIndex) {
				sysIndex := r.LineFunctionSystemNameIndices.Value(lineIndex)
				systemName := r.LineFunctionSystemNameDict.Value(int(sysIndex))
				if !matchesStringCondition(systemName, sysCond) {
					return false
				}
			} else {
				// Frame has no system name, so system name filter doesn't match
				return false
			}
		}
	}

	if fileCond := filter.GetFilename(); fileCond != nil {
		// If lineIndex is -1, skip filename check
		if lineIndex >= 0 {
			if r.LineFunctionFilenameIndices.Len() == 0 {
				return false
			}
			if r.LineFunctionFilenameIndices.IsValid(lineIndex) {
				fileIndex := r.LineFunctionFilenameIndices.Value(lineIndex)
				filename := r.LineFunctionFilenameDict.Value(int(fileIndex))
				if !matchesStringCondition(filename, fileCond) {
					return false
				}
			} else {
				// Frame has no filename, so filename filter doesn't match
				return false
			}
		}
	}

	if addrCond := filter.GetAddress(); addrCond != nil {
		if r.Address.IsValid(locationIndex) {
			address := r.Address.Value(locationIndex)
			if !matchesNumberCondition(address, addrCond) {
				return false
			}
		} else {
			// Frame has no address, so address filter doesn't match
			return false
		}
	}

	if lineCond := filter.GetLineNumber(); lineCond != nil {
		// If lineIndex is -1, skip line number check
		if lineIndex >= 0 {
			if r.LineNumber.IsValid(lineIndex) {
				lineNumber := uint64(r.LineNumber.Value(lineIndex))
				if !matchesNumberCondition(lineNumber, lineCond) {
					return false
				}
			} else {
				// Frame has no line number, so line number filter doesn't match
				return false
			}
		}
	}

	return true
}

// matchesStringCondition checks if a value matches a string condition.
func matchesStringCondition(value []byte, condition *pb.StringCondition) bool {
	if condition == nil {
		return true
	}

	valueLower := bytes.ToLower(value)

	switch condition.GetCondition().(type) {
	case *pb.StringCondition_Equal:
		target := bytes.ToLower([]byte(condition.GetEqual()))
		return bytes.Equal(valueLower, target)
	case *pb.StringCondition_NotEqual:
		target := bytes.ToLower([]byte(condition.GetNotEqual()))
		return !bytes.Equal(valueLower, target)
	case *pb.StringCondition_Contains:
		target := bytes.ToLower([]byte(condition.GetContains()))
		return bytes.Contains(valueLower, target)
	case *pb.StringCondition_NotContains:
		target := bytes.ToLower([]byte(condition.GetNotContains()))
		return !bytes.Contains(valueLower, target)
	default:
		return true
	}
}

// matchesNumberCondition checks if a numeric value matches a number condition.
func matchesNumberCondition(value uint64, condition *pb.NumberCondition) bool {
	if condition == nil {
		return true
	}

	switch condition.GetCondition().(type) {
	case *pb.NumberCondition_Equal:
		return value == condition.GetEqual()
	case *pb.NumberCondition_NotEqual:
		return value != condition.GetNotEqual()
	default:
		return true
	}
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
	case pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW, pb.QueryRequest_REPORT_TYPE_FLAMECHART:
		if typ == pb.QueryRequest_REPORT_TYPE_FLAMECHART {
			// Generating the flame chart assumes a single record that is sorted by timestamp.
			for i, sample := range p.Samples {
				indices := sample.Schema().FieldIndices(FlamegraphFieldTimestamp)
				if len(indices) != 1 {
					return nil, status.Errorf(codes.Internal, "invalid flame chart timestamp indices: %v", indices)
				}
				sortedIndices, err := arrowutils.SortRecord(sample, []arrowutils.SortingColumn{
					{Index: indices[0], Direction: arrowutils.Ascending},
				})
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to sort flame chart record: %v", err.Error())
				}

				isSorted := true
				for j := 0; j < sortedIndices.Len(); j++ {
					if sortedIndices.Value(j) != int32(j) {
						isSorted = false
						break
					}
				}
				if isSorted {
					// Don't sort if the indices are already sorted.
					continue
				}

				sorted, err := arrowutils.Take(ctx, sample, sortedIndices)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to sort flame chart record: %v", err.Error())
				}

				p.Samples[i] = sorted
			}

			if len(p.Samples) > 1 {
				indices := p.Samples[0].Schema().FieldIndices(FlamegraphFieldTimestamp)
				if len(indices) != 1 {
					return nil, status.Errorf(codes.Internal, "invalid flame chart timestamp indices: %v", indices)
				}
				sorted, err := arrowutils.MergeRecords(mem, p.Samples, []arrowutils.SortingColumn{
					{Index: indices[0], Direction: arrowutils.Ascending},
				}, 0)
				if err != nil {
					return nil, status.Errorf(codes.Internal, "failed to merge flame chart records: %v", err.Error())
				}
				p.Samples = []arrow.RecordBatch{sorted}
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

func (q *ColumnQueryAPI) selectSingle(ctx context.Context, s *pb.SingleProfile, isInverted bool) (profile.Profile, error) {
	p, err := q.querier.QuerySingle(
		ctx,
		s.Query,
		s.Time.AsTime(),
		isInverted,
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectMerge(ctx context.Context, m *pb.MergeProfile, groupByLabels []string, isInverted bool, functionToFilterBy string) (profile.Profile, error) {
	p, err := q.querier.QueryMerge(
		ctx,
		m.Query,
		m.Start.AsTime(),
		m.End.AsTime(),
		groupByLabels,
		isInverted,
		functionToFilterBy,
	)
	if err != nil {
		return profile.Profile{}, err
	}

	return p, nil
}

func (q *ColumnQueryAPI) selectDiff(ctx context.Context, d *pb.DiffProfile, aggregateByLabels, isInverted bool) (profile.Profile, error) {
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
		base, err = q.selectProfileForDiff(ctx, d.A, aggregateByLabels, isInverted)
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
		compare, err = q.selectProfileForDiff(ctx, d.B, aggregateByLabels, isInverted)
		if err != nil {
			return fmt.Errorf("reading compared profile: %w", err)
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return profile.Profile{}, err
	}

	return ComputeDiff(ctx, q.tracer, q.mem, base, compare, d.GetAbsolute())
}

type Releasable interface {
	Release()
}

func ComputeDiff(ctx context.Context, tracer trace.Tracer, mem memory.Allocator, base, compare profile.Profile, absolute bool) (profile.Profile, error) {
	_, span := tracer.Start(ctx, "ComputeDiff")
	defer span.End()
	cleanupArrs := make([]Releasable, 0, len(base.Samples))
	defer func() {
		for _, r := range cleanupArrs {
			r.Release()
		}
	}()

	records := make([]arrow.RecordBatch, 0, len(compare.Samples)+len(base.Samples))

	var (
		compareCumulativeRatio = 1.0
		baseCumulativeRatio    = 1.0
	)

	if !absolute {
		compareCumulativeTotal := int64(0)
		for _, r := range compare.Samples {
			cols := r.Columns()
			compareCumulativeTotal += math.Int64.Sum(cols[len(cols)-4].(*array.Int64))
		}

		baseCumulativeTotal := int64(0)
		for _, r := range base.Samples {
			cols := r.Columns()
			baseCumulativeTotal += math.Int64.Sum(cols[len(cols)-4].(*array.Int64))
		}

		// Scale up base if compare is bigger
		if compareCumulativeTotal > baseCumulativeTotal {
			baseCumulativeRatio = float64(compareCumulativeTotal) / float64(baseCumulativeTotal)
		}
		// Scale up compare if base is bigger
		if baseCumulativeTotal > compareCumulativeTotal {
			compareCumulativeRatio = float64(baseCumulativeTotal) / float64(compareCumulativeTotal)
		}
	}

	for _, r := range compare.Samples {
		columns := r.Columns()
		cols := make([]arrow.Array, len(columns))
		copy(cols, columns)
		// This is intentional, the diff value of the `compare` profile is the same
		// as the value of the `compare` profile, because what we're actually doing
		// is subtracting the `base` profile, but the actual calculation happens
		// when building the visualizations. We should eventually have this be done
		// directly by the query engine.

		if compareCumulativeRatio > 1.0 {
			// If compareCumulativeRatio is bigger than 1.0 we have to scale all values
			multi := multiplyInt64By(mem, cols[len(cols)-4].(*array.Int64), compareCumulativeRatio)
			cols[len(cols)-3] = multi
			cleanupArrs = append(cleanupArrs, multi)
		} else {
			// otherwise we simply use the original values.
			cols[len(cols)-3] = cols[len(cols)-4] // value as diff
		}

		records = append(records, array.NewRecordBatch(
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
			diff := multiplyInt64By(mem, columns[len(columns)-4].(*array.Int64), -1*baseCumulativeRatio)
			defer diff.Release()
			value := zeroInt64Array(mem, int(r.NumRows()))
			defer value.Release()
			timestamp := zeroInt64Array(mem, int(r.NumRows()))
			defer timestamp.Release()
			duration := zeroInt64Array(mem, int(r.NumRows()))
			defer duration.Release()
			records = append(records, array.NewRecordBatch(
				r.Schema(),
				append(
					cols[:len(cols)-4], // all other columns like locations
					value,
					diff,
					timestamp,
					duration,
				),
				r.NumRows(),
			))
		}()
	}

	return profile.Profile{
		Meta:    compare.Meta,
		Samples: records,
	}, nil
}

func multiplyInt64By(pool memory.Allocator, arr *array.Int64, factor float64) arrow.Array {
	b := array.NewInt64Builder(pool)
	defer b.Release()

	values := arr.Int64Values()
	valid := make([]bool, len(values))
	for i := range values {
		nv := float64(values[i]) * factor
		values[i] = int64(nv)
		valid[i] = true
	}

	b.AppendValues(values, valid)
	return b.NewArray()
}

func zeroInt64Array(pool memory.Allocator, rows int) arrow.Array {
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

func (q *ColumnQueryAPI) selectProfileForDiff(ctx context.Context, s *pb.ProfileDiffSelection, _, isInverted bool) (profile.Profile, error) {
	switch s.Mode {
	case pb.ProfileDiffSelection_MODE_SINGLE_UNSPECIFIED:
		return q.selectSingle(ctx, s.GetSingle(), isInverted)
	case pb.ProfileDiffSelection_MODE_MERGE:
		return q.selectMerge(ctx, s.GetMerge(), []string{}, isInverted, "")
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
func sliceRecord(r arrow.RecordBatch, indices []int64) []arrow.RecordBatch {
	if len(indices) == 0 {
		return []arrow.RecordBatch{}
	}

	slices := []arrow.RecordBatch{}
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

func getMappingFilesAndLabels(
	ctx context.Context,
	q Querier,
	query string,
	startTime, endTime time.Time,
) ([]string, []string, error) {
	mappingFiles, err := q.GetProfileMetadataMappings(ctx, query, startTime, endTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get mappings: %w", err)
	}

	labels, err := q.GetProfileMetadataLabels(ctx, query, startTime, endTime)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get labels: %w", err)
	}

	return mappingFiles, labels, nil
}

// This is a deduplicating k-way merge.
// The two slices that are passed in are assumed to be sorted.
func MergeTwoSortedSlices(arr1, arr2 []string) []string {
	merged := make([]string, 0, len(arr1)+len(arr2))
	i, j := 0, 0

	for i < len(arr1) && j < len(arr2) {
		if arr1[i] < arr2[j] {
			if len(merged) == 0 || merged[len(merged)-1] != arr1[i] {
				merged = append(merged, arr1[i])
			}
			i++
		} else {
			if len(merged) == 0 || merged[len(merged)-1] != arr2[j] {
				merged = append(merged, arr2[j])
			}
			j++
		}
	}

	for i < len(arr1) {
		if len(merged) == 0 || merged[len(merged)-1] != arr1[i] {
			merged = append(merged, arr1[i])
		}
		i++
	}

	for j < len(arr2) {
		if len(merged) == 0 || merged[len(merged)-1] != arr2[j] {
			merged = append(merged, arr2[j])
		}
		j++
	}

	return merged
}

// ConvertDeprecatedFilters converts deprecated filter fields to the new schema for backward compatibility.
func ConvertDeprecatedFilters(filters []*pb.Filter) []*pb.Filter {
	convertedFilters := make([]*pb.Filter, 0, len(filters))

	for _, filter := range filters {
		newFilter := &pb.Filter{}

		if stackFilter := filter.GetStackFilter(); stackFilter != nil {
			// Handle new oneof structure - prefer new criteria over deprecated function_name_stack_filter
			if criteria := stackFilter.GetCriteria(); criteria != nil {
				newFilter.Filter = &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: criteria,
						},
					},
				}
			} else if funcFilter := stackFilter.GetFunctionNameStackFilter(); funcFilter != nil { //nolint:staticcheck // deprecated but needed for backward compatibility
				// Handle deprecated function_name_stack_filter for backward compatibility
				criteria := &pb.FilterCriteria{
					FunctionName: &pb.StringCondition{},
				}
				if funcFilter.GetExclude() {
					criteria.FunctionName.Condition = &pb.StringCondition_NotContains{
						NotContains: funcFilter.GetFunctionToFilter(),
					}
				} else {
					criteria.FunctionName.Condition = &pb.StringCondition_Contains{
						Contains: funcFilter.GetFunctionToFilter(),
					}
				}
				newFilter.Filter = &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: criteria,
						},
					},
				}
			}
		}

		if frameFilter := filter.GetFrameFilter(); frameFilter != nil {
			// Handle new oneof structure - prefer new criteria over deprecated binary_frame_filter
			if criteria := frameFilter.GetCriteria(); criteria != nil {
				newFilter.Filter = &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: criteria,
						},
					},
				}
			} else if binaryFilter := frameFilter.GetBinaryFrameFilter(); binaryFilter != nil { //nolint:staticcheck // deprecated but needed for backward compatibility
				// Handle deprecated binary_frame_filter for backward compatibility
				for _, binary := range binaryFilter.GetIncludeBinaries() {
					criteria := &pb.FilterCriteria{
						Binary: &pb.StringCondition{
							Condition: &pb.StringCondition_Contains{
								Contains: binary,
							},
						},
					}
					binaryFilter := &pb.Filter{
						Filter: &pb.Filter_FrameFilter{
							FrameFilter: &pb.FrameFilter{
								Filter: &pb.FrameFilter_Criteria{
									Criteria: criteria,
								},
							},
						},
					}
					convertedFilters = append(convertedFilters, binaryFilter)
				}
				continue // Skip adding the original filter since we added converted ones
			}
		}

		// Add the converted filter (or original if no conversion needed)
		if newFilter.Filter != nil {
			convertedFilters = append(convertedFilters, newFilter)
		} else {
			convertedFilters = append(convertedFilters, filter)
		}
	}

	return convertedFilters
}
