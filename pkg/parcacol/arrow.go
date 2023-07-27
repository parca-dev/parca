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

package parcacol

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/memory"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/profile"
)

type ErrMissingColumn struct {
	Column  string
	Columns int
}

func (e ErrMissingColumn) Error() string {
	return fmt.Sprintf("expected column %s, got %d columns", e.Column, e.Columns)
}

type ArrowToProfileConverter struct {
	tracer trace.Tracer
	key    *metastore.KeyMaker
}

func NewArrowToProfileConverter(
	tracer trace.Tracer,
	keyMaker *metastore.KeyMaker,
) *ArrowToProfileConverter {
	return &ArrowToProfileConverter{
		tracer: tracer,
		key:    keyMaker,
	}
}

func (c *ArrowToProfileConverter) Convert(
	ctx context.Context,
	p profile.Profile,
) (profile.OldProfile, error) {
	samples := make([]*profile.SymbolizedSample, 0, p.Samples.NumRows())

	ar := p.Samples
	schema := ar.Schema()
	indices := schema.FieldIndices("locations")
	if len(indices) != 1 {
		return profile.OldProfile{}, ErrMissingColumn{Column: "locations", Columns: len(indices)}
	}
	locations := ar.Column(indices[0]).(*array.List)
	locationOffsets := locations.Offsets()
	location := locations.ListValues().(*array.Struct)
	address := location.Field(0).(*array.Uint64)
	mapping := location.Field(1).(*array.Struct)
	mappingStart := mapping.Field(0).(*array.Uint64)
	mappingLimit := mapping.Field(1).(*array.Uint64)
	mappingOffset := mapping.Field(2).(*array.Uint64)
	mappingFile := mapping.Field(3).(*array.String)
	mappingBuildID := mapping.Field(4).(*array.String)
	lines := location.Field(2).(*array.List)
	lineOffsets := lines.Offsets()
	line := lines.ListValues().(*array.Struct)
	lineNumber := line.Field(0).(*array.Int64)
	lineFunction := line.Field(1).(*array.Struct)
	lineFunctionName := lineFunction.Field(0).(*array.String)
	lineFunctionSystemName := lineFunction.Field(1).(*array.String)
	lineFunctionFilename := lineFunction.Field(2).(*array.String)
	lineFunctionStartLine := lineFunction.Field(3).(*array.Int64)

	indices = schema.FieldIndices("value")
	if len(indices) != 1 {
		return profile.OldProfile{}, ErrMissingColumn{Column: "value", Columns: len(indices)}
	}
	valueColumn := ar.Column(indices[0]).(*array.Int64)

	indices = schema.FieldIndices("diff")
	if len(indices) != 1 {
		return profile.OldProfile{}, ErrMissingColumn{Column: "diff", Columns: len(indices)}
	}
	diffColumn := ar.Column(indices[0]).(*array.Int64)

	labelIndexes := make(map[string]int)
	for i, field := range schema.Fields() {
		if strings.HasPrefix(field.Name, profile.ColumnPprofLabelsPrefix) {
			labelIndexes[strings.TrimPrefix(field.Name, profile.ColumnPprofLabelsPrefix)] = i
		}
	}

	for i := 0; i < int(ar.NumRows()); i++ {
		labels := make(map[string]string, len(labelIndexes))
		for name, index := range labelIndexes {
			c := ar.Column(index).(*array.Dictionary)
			d := c.Dictionary().(*array.Binary)
			if !c.IsNull(i) {
				labelValue := d.Value(c.GetValueIndex(i))
				if len(labelValue) > 0 {
					labels[name] = string(labelValue)
				}
			}
		}

		lOffsetStart := locationOffsets[i]
		lOffsetEnd := locationOffsets[i+1]
		stacktrace := make([]*profile.Location, 0, lOffsetEnd-lOffsetStart)
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			llOffsetStart := lineOffsets[j]
			llOffsetEnd := lineOffsets[j+1]
			lines := make([]profile.LocationLine, 0, llOffsetEnd-llOffsetStart)

			for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
				var f *pb.Function
				if lineFunction.IsValid(k) {
					f = &pb.Function{
						Name:       lineFunctionName.Value(k),
						SystemName: lineFunctionSystemName.Value(k),
						Filename:   lineFunctionFilename.Value(k),
						StartLine:  int64(lineFunctionStartLine.Value(k)),
					}
					f.Id = c.key.MakeFunctionID(f)
				}
				lines = append(lines, profile.LocationLine{
					Line:     int64(lineNumber.Value(k)),
					Function: f,
				})
			}

			var m *pb.Mapping
			if !mapping.IsNull(j) {
				m = &pb.Mapping{
					Start:   mappingStart.Value(j),
					Limit:   mappingLimit.Value(j),
					Offset:  mappingOffset.Value(j),
					File:    mappingFile.Value(j),
					BuildId: mappingBuildID.Value(j),
				}
				m.Id = c.key.MakeMappingID(m)
			}

			loc := &profile.Location{
				Address: address.Value(j),
				Mapping: m,
				Lines:   lines,
			}
			loc.ID = c.key.MakeProfileLocationID(loc)
			stacktrace = append(stacktrace, loc)
		}

		samples = append(samples, &profile.SymbolizedSample{
			Value:     valueColumn.Value(i),
			DiffValue: diffColumn.Value(i),
			Locations: stacktrace,
			Label:     labels,
		})
	}

	return profile.OldProfile{
		Samples: samples,
		Meta:    p.Meta,
	}, nil
}

type ProfileSymbolizer struct {
	tracer trace.Tracer
	m      pb.MetastoreServiceClient
}

func NewProfileSymbolizer(
	tracer trace.Tracer,
	m pb.MetastoreServiceClient,
) *ProfileSymbolizer {
	return &ProfileSymbolizer{
		tracer: tracer,
		m:      m,
	}
}

func (s *ProfileSymbolizer) SymbolizeNormalizedProfile(ctx context.Context, p *profile.NormalizedProfile) (profile.OldProfile, error) {
	stacktraceIDs := make([]string, len(p.Samples))
	for i, sample := range p.Samples {
		stacktraceIDs[i] = sample.StacktraceID
	}

	stacktraceLocations, err := s.resolveStacktraces(ctx, stacktraceIDs)
	if err != nil {
		return profile.OldProfile{}, fmt.Errorf("read stacktrace metadata: %w", err)
	}

	samples := make([]*profile.SymbolizedSample, len(p.Samples))
	for i, sample := range p.Samples {
		samples[i] = &profile.SymbolizedSample{
			Value:     sample.Value,
			DiffValue: sample.DiffValue,
			Locations: stacktraceLocations[i],
			Label:     sample.Label,
			NumLabel:  sample.NumLabel,
		}
	}

	return profile.OldProfile{
		Samples: samples,
		Meta:    p.Meta,
	}, nil
}

func (s *ProfileSymbolizer) resolveStacktraces(ctx context.Context, stacktraceIDs []string) (
	[][]*profile.Location,
	error,
) {
	ctx, span := s.tracer.Start(ctx, "resolve-stacktraces")
	defer span.End()

	stacktraces, locations, locationIndex, err := s.resolveStacktraceLocations(ctx, stacktraceIDs)
	if err != nil {
		return nil, fmt.Errorf("resolve stacktrace locations: %w", err)
	}

	stacktraceLocations := make([][]*profile.Location, len(stacktraces))
	for i, stacktrace := range stacktraces {
		stacktraceLocations[i] = make([]*profile.Location, len(stacktrace.LocationIds))
		for j, id := range stacktrace.LocationIds {
			stacktraceLocations[i][j] = locations[locationIndex[id]]
		}
	}

	return stacktraceLocations, nil
}

func (s *ProfileSymbolizer) resolveStacktraceLocations(ctx context.Context, stacktraceIDs []string) (
	[]*pb.Stacktrace,
	[]*profile.Location,
	map[string]int,
	error,
) {
	ctx, span := s.tracer.Start(ctx, "resolve-stacktraces")
	defer span.End()

	sres, err := s.m.Stacktraces(ctx, &pb.StacktracesRequest{
		StacktraceIds: stacktraceIDs,
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read stacktraces: %w", err)
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

	lres, err := s.m.Locations(ctx, &pb.LocationsRequest{LocationIds: locationIDs})
	if err != nil {
		return nil, nil, nil, err
	}

	locations, err := s.getLocationsFromSerializedLocations(ctx, locationIDs, lres.Locations)
	if err != nil {
		return nil, nil, nil, err
	}

	return sres.Stacktraces, locations, locationIndex, nil
}

func BuildArrowLocations(allocator memory.Allocator, stacktraces []*pb.Stacktrace, resolvedLocations []*profile.Location, locationIndex map[string]int) arrow.Record {
	b := array.NewRecordBuilder(allocator, profile.LocationsArrowSchema())
	defer b.Release()

	locationsList := b.Field(0).(*array.ListBuilder)
	locations := locationsList.ValueBuilder().(*array.StructBuilder)

	addresses := locations.FieldBuilder(0).(*array.Uint64Builder)

	mapping := locations.FieldBuilder(1).(*array.StructBuilder)
	mappingStart := mapping.FieldBuilder(0).(*array.Uint64Builder)
	mappingLimit := mapping.FieldBuilder(1).(*array.Uint64Builder)
	mappingOffset := mapping.FieldBuilder(2).(*array.Uint64Builder)
	mappingFile := mapping.FieldBuilder(3).(*array.StringBuilder)
	mappingBuildID := mapping.FieldBuilder(4).(*array.StringBuilder)

	lines := locations.FieldBuilder(2).(*array.ListBuilder)
	line := lines.ValueBuilder().(*array.StructBuilder)
	lineNumber := line.FieldBuilder(0).(*array.Int64Builder)
	function := line.FieldBuilder(1).(*array.StructBuilder)
	functionName := function.FieldBuilder(0).(*array.StringBuilder)
	functionSystemName := function.FieldBuilder(1).(*array.StringBuilder)
	functionFilename := function.FieldBuilder(2).(*array.StringBuilder)
	functionStartLine := function.FieldBuilder(3).(*array.Int64Builder)

	for _, stacktrace := range stacktraces {
		locationsList.Append(true)
		for _, id := range stacktrace.LocationIds {
			locations.Append(true)
			loc := resolvedLocations[locationIndex[id]]

			addresses.Append(loc.Address)

			mapping.Append(loc.Mapping != nil)
			if loc.Mapping != nil {
				mappingStart.Append(loc.Mapping.Start)
				mappingLimit.Append(loc.Mapping.Limit)
				mappingOffset.Append(loc.Mapping.Offset)
				mappingFile.Append(loc.Mapping.File)
				mappingBuildID.Append(loc.Mapping.BuildId)
			}

			lines.Append(len(loc.Lines) > 0)
			if loc.Lines != nil {
				for _, l := range loc.Lines {
					line.Append(true)
					lineNumber.Append(l.Line)
					function.Append(l.Function != nil)
					if l.Function != nil {
						functionName.Append(l.Function.Name)
						functionSystemName.Append(l.Function.SystemName)
						functionFilename.Append(l.Function.Filename)
						functionStartLine.Append(l.Function.StartLine)
					}
				}
			}
		}
	}

	return b.NewRecord()
}

func (s *ProfileSymbolizer) getLocationsFromSerializedLocations(
	ctx context.Context,
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
		mres, err := s.m.Mappings(ctx, &pb.MappingsRequest{
			MappingIds: mappingIDs,
		})
		if err != nil {
			return nil, fmt.Errorf("get mappings by IDs: %w", err)
		}
		mappings = mres.Mappings
	}

	functionIndex := map[string]int{}
	functionIDs := []string{}
	for _, location := range locations {
		if location.Lines == nil {
			continue
		}
		for _, line := range location.Lines {
			if _, found := functionIndex[line.FunctionId]; !found {
				functionIDs = append(functionIDs, line.FunctionId)
				functionIndex[line.FunctionId] = len(functionIDs) - 1
			}
		}
	}

	fres, err := s.m.Functions(ctx, &pb.FunctionsRequest{
		FunctionIds: functionIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("get functions by ids: %w", err)
	}

	res := make([]*profile.Location, 0, len(locations))
	for _, location := range locations {
		var mapping *pb.Mapping
		if location.MappingId != "" {
			mapping = mappings[mappingIndex[location.MappingId]]
		}

		symbolizedLines := []profile.LocationLine{}
		if location.Lines != nil {
			lines := location.Lines
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
