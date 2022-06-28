// Copyright 2022 The Parca Authors
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

package parcacol

import (
	"context"
	"fmt"

	"github.com/apache/arrow/go/v8/arrow"
	"github.com/apache/arrow/go/v8/arrow/array"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func ArrowRecordToStacktraceSamples(
	ctx context.Context,
	m pb.MetastoreServiceClient,
	ar arrow.Record,
	valueColumnName string,
	meta profile.Meta,
) (*profile.Profile, error) {
	schema := ar.Schema()
	indices := schema.FieldIndices("stacktrace")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one stacktrace column, got %d", len(indices))
	}
	stacktraceColumn := ar.Column(indices[0]).(*array.Binary)

	indices = schema.FieldIndices("sum(value)")
	if len(indices) != 1 {
		return nil, fmt.Errorf("expected exactly one value column, got %d", len(indices))
	}
	valueColumn := ar.Column(indices[0]).(*array.Int64)

	rows := int(ar.NumRows())
	stacktraceIDs := make([]string, rows)
	for i := 0; i < rows; i++ {
		stacktraceIDs[i] = string(stacktraceColumn.Value(i))
	}

	stacktraceLocations, err := resolveStacktraces(ctx, m, stacktraceIDs)
	if err != nil {
		return nil, fmt.Errorf("read stacktrace metadata: %w", err)
	}

	samples := make([]*profile.SymbolizedSample, 0, rows)
	for i := 0; i < rows; i++ {
		samples = append(samples, &profile.SymbolizedSample{
			Value:     valueColumn.Value(i),
			Locations: stacktraceLocations[i],
		})
	}

	return &profile.Profile{
		Samples: samples,
		Meta:    meta,
	}, nil
}

func SymbolizeNormalizedProfile(ctx context.Context, m pb.MetastoreServiceClient, p *profile.NormalizedProfile) (*profile.Profile, error) {
	stacktraceIDs := make([]string, len(p.Samples))
	for i, sample := range p.Samples {
		stacktraceIDs[i] = sample.StacktraceID
	}

	stacktraceLocations, err := resolveStacktraces(ctx, m, stacktraceIDs)
	if err != nil {
		return nil, fmt.Errorf("read stacktrace metadata: %w", err)
	}

	samples := make([]*profile.SymbolizedSample, len(p.Samples))
	for i, sample := range p.Samples {
		samples[i] = &profile.SymbolizedSample{
			Value:     sample.Value,
			DiffValue: sample.DiffValue,
			Locations: stacktraceLocations[i],
		}
	}

	return &profile.Profile{
		Samples: samples,
		Meta:    p.Meta,
	}, nil
}

func resolveStacktraces(ctx context.Context, m pb.MetastoreServiceClient, stacktraceIDs []string) (
	[][]*profile.Location,
	error,
) {
	sres, err := m.Stacktraces(ctx, &pb.StacktracesRequest{
		StacktraceIds: stacktraceIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("read stacktraces: %w", err)
	}

	locationNum := 0
	for _, stacktrace := range sres.Stacktraces {
		locationNum += len(stacktrace.LocationIds)
	}

	locationIndex := make(map[string]int, locationNum)
	locationIDs := make([]string, 0, locationNum)
	for _, s := range sres.Stacktraces {
		for _, id := range s.LocationIds {
			if _, seen := locationIndex[id]; !seen {
				locationIDs = append(locationIDs, id)
				locationIndex[id] = len(locationIDs) - 1
			}
		}
	}

	lres, err := m.Locations(ctx, &pb.LocationsRequest{LocationIds: locationIDs})
	if err != nil {
		return nil, err
	}

	locations, err := getLocationsFromSerializedLocations(ctx, m, locationIDs, lres.Locations)
	if err != nil {
		return nil, err
	}

	stacktraceLocations := make([][]*profile.Location, len(sres.Stacktraces))
	for i, stacktrace := range sres.Stacktraces {
		stacktraceLocations[i] = make([]*profile.Location, len(stacktrace.LocationIds))
		for j, id := range stacktrace.LocationIds {
			stacktraceLocations[i][j] = locations[locationIndex[id]]
		}
	}

	return stacktraceLocations, nil
}

func getLocationsFromSerializedLocations(
	ctx context.Context,
	s pb.MetastoreServiceClient,
	locationIds []string,
	locations []*pb.Location,
) (
	[]*profile.Location,
	error,
) {
	mappingIndex := map[string]int{}
	mappingIDs := []string{}
	for _, location := range locations {
		if location.MappingId == "" {
			continue
		}

		if _, found := mappingIndex[location.MappingId]; !found {
			mappingIDs = append(mappingIDs, location.MappingId)
			mappingIndex[location.MappingId] = len(mappingIDs) - 1
		}
	}

	var mappings []*pb.Mapping
	if len(mappingIDs) > 0 {
		mres, err := s.Mappings(ctx, &pb.MappingsRequest{
			MappingIds: mappingIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("get mappings by IDs: %w", err)
		}
		mappings = mres.Mappings
	}

	lres, err := s.LocationLines(ctx, &pb.LocationLinesRequest{
		LocationIds: locationIds,
	})
	if err != nil {
		return nil, fmt.Errorf("get lines by location IDs: %w", err)
	}

	functionIndex := map[string]int{}
	functionIDs := []string{}
	for _, lines := range lres.LocationLines {
		if lines == nil {
			continue
		}
		for _, line := range lines.Entries {
			if _, found := functionIndex[line.FunctionId]; !found {
				functionIDs = append(functionIDs, line.FunctionId)
				functionIndex[line.FunctionId] = len(functionIDs) - 1
			}
		}
	}

	fres, err := s.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get functions by ids: %w", err)
	}

	res := make([]*profile.Location, 0, len(locations))
	for i, location := range locations {
		var mapping *pb.Mapping
		if location.MappingId != "" {
			mapping = mappings[mappingIndex[location.MappingId]]
		}

		symbolizedLines := []profile.LocationLine{}
		if lres.LocationLines[i] != nil {
			lines := lres.LocationLines[i].Entries
			symbolizedLines = make([]profile.LocationLine, 0, len(lines))
			for _, line := range lines {
				symbolizedLines = append(symbolizedLines, profile.LocationLine{
					Function: fres.Functions[functionIndex[line.FunctionId]],
					Line:     line.Line,
				})
			}
		}

		res = append(res, &profile.Location{
			ID:       location.Id,
			Address:  location.Address,
			IsFolded: location.IsFolded,
			Mapping:  mapping,
			Lines:    symbolizedLines,
		})
	}

	return res, nil
}
