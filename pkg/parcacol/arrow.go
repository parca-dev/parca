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
	"unsafe"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/memory"
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
	sampleNum := int64(0)
	for _, r := range p.Samples {
		sampleNum += r.NumRows()
	}

	samples := make([]*profile.SymbolizedSample, 0, sampleNum)

	for _, ar := range p.Samples {
		schema := ar.Schema()
		indices := schema.FieldIndices("locations")
		if len(indices) != 1 {
			return profile.OldProfile{}, ErrMissingColumn{Column: "locations", Columns: len(indices)}
		}
		locations := ar.Column(indices[0]).(*array.List)
		locationOffsets := locations.Offsets()
		location := locations.ListValues().(*array.Struct)
		address := location.Field(0).(*array.Uint64)
		mappingStart := location.Field(1).(*array.Uint64)
		mappingLimit := location.Field(2).(*array.Uint64)
		mappingOffset := location.Field(3).(*array.Uint64)
		mappingFile := location.Field(4).(*array.Dictionary)
		mappingFileDict := mappingFile.Dictionary().(*array.Binary)
		mappingBuildID := location.Field(5).(*array.Dictionary)
		mappingBuildIDDict := mappingBuildID.Dictionary().(*array.Binary)
		lines := location.Field(6).(*array.List)
		lineOffsets := lines.Offsets()
		line := lines.ListValues().(*array.Struct)
		lineNumber := line.Field(0).(*array.Int64)
		lineFunctionName := line.Field(1).(*array.Dictionary)
		lineFunctionNameDict := lineFunctionName.Dictionary().(*array.Binary)
		lineFunctionSystemName := line.Field(2).(*array.Dictionary)
		lineFunctionSystemNameDict := lineFunctionSystemName.Dictionary().(*array.Binary)
		lineFunctionFilename := line.Field(3).(*array.Dictionary)
		lineFunctionFilenameDict := lineFunctionFilename.Dictionary().(*array.Binary)
		lineFunctionStartLine := line.Field(4).(*array.Int64)

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
					name := ""
					if lineFunctionName.IsValid(k) {
						name = string(lineFunctionNameDict.Value(lineFunctionName.GetValueIndex(k)))
					}
					systemName := ""
					if lineFunctionSystemName.IsValid(k) {
						systemName = string(lineFunctionSystemNameDict.Value(lineFunctionSystemName.GetValueIndex(k)))
					}
					filename := ""
					if lineFunctionFilename.IsValid(k) {
						filename = string(lineFunctionFilenameDict.Value(lineFunctionFilename.GetValueIndex(k)))
					}
					startLine := int64(0)
					if lineFunctionStartLine.IsValid(k) {
						startLine = int64(lineFunctionStartLine.Value(k))
					}
					var f *pb.Function
					if name != "" || systemName != "" || filename != "" || startLine != 0 {
						f = &pb.Function{
							Name:       name,
							SystemName: systemName,
							Filename:   filename,
							StartLine:  startLine,
						}
						f.Id = c.key.MakeFunctionID(f)
					}
					lines = append(lines, profile.LocationLine{
						Line:     int64(lineNumber.Value(k)),
						Function: f,
					})
				}

				start := mappingStart.Value(j)
				limit := mappingLimit.Value(j)
				offset := mappingOffset.Value(j)
				buildID := ""
				if mappingBuildID.IsValid(j) {
					buildID = string(mappingBuildIDDict.Value(mappingBuildID.GetValueIndex(j)))
				}
				file := ""
				if mappingFile.IsValid(j) {
					file = string(mappingFileDict.Value(mappingFile.GetValueIndex(j)))
				}
				var m *pb.Mapping
				if start != 0 || limit != 0 || offset != 0 || buildID != "" || file != "" {
					m = &pb.Mapping{
						Start:   start,
						Limit:   limit,
						Offset:  offset,
						File:    file,
						BuildId: buildID,
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

func BuildArrowLocations(allocator memory.Allocator, stacktraces []*pb.Stacktrace, resolvedLocations []*profile.Location, locationIndex map[string]int) (arrow.Record, error) {
	w := profile.NewLocationsWriter(allocator)
	defer w.RecordBuilder.Release()

	for _, stacktrace := range stacktraces {
		w.LocationsList.Append(true)
		for _, id := range stacktrace.LocationIds {
			w.Locations.Append(true)
			loc := resolvedLocations[locationIndex[id]]

			w.Addresses.Append(loc.Address)

			if loc.Mapping != nil {
				w.MappingStart.Append(loc.Mapping.Start)
				w.MappingLimit.Append(loc.Mapping.Limit)
				w.MappingOffset.Append(loc.Mapping.Offset)

				if len(loc.Mapping.File) > 0 {
					if err := w.MappingFile.Append(stringToBytes(loc.Mapping.File)); err != nil {
						return nil, fmt.Errorf("append mapping file: %w", err)
					}
				} else {
					if err := w.MappingFile.Append([]byte{}); err != nil {
						return nil, fmt.Errorf("append mapping file: %w", err)
					}
				}

				if len(loc.Mapping.BuildId) > 0 {
					if err := w.MappingBuildID.Append(stringToBytes(loc.Mapping.BuildId)); err != nil {
						return nil, fmt.Errorf("append mapping build id: %w", err)
					}
				} else {
					if err := w.MappingBuildID.Append([]byte{}); err != nil {
						return nil, fmt.Errorf("append mapping build id: %w", err)
					}
				}
			} else {
				w.MappingStart.AppendNull()
				w.MappingLimit.AppendNull()
				w.MappingOffset.AppendNull()
				w.MappingFile.AppendNull()
				w.MappingBuildID.AppendNull()
			}

			if loc.Lines != nil && len(loc.Lines) > 0 {
				w.Lines.Append(true)
				for _, l := range loc.Lines {
					w.Line.Append(true)
					w.LineNumber.Append(l.Line)
					if l.Function != nil {
						if len(l.Function.Name) > 0 {
							if err := w.FunctionName.Append(stringToBytes(l.Function.Name)); err != nil {
								return nil, fmt.Errorf("append function name: %w", err)
							}
						} else {
							if err := w.FunctionName.Append([]byte{}); err != nil {
								return nil, fmt.Errorf("append function name: %w", err)
							}
						}

						if len(l.Function.SystemName) > 0 {
							if err := w.FunctionSystemName.Append(stringToBytes(l.Function.SystemName)); err != nil {
								return nil, fmt.Errorf("append function system name: %w", err)
							}
						} else {
							if err := w.FunctionSystemName.Append([]byte{}); err != nil {
								return nil, fmt.Errorf("append function name: %w", err)
							}
						}

						if len(l.Function.Filename) > 0 {
							if err := w.FunctionFilename.Append(stringToBytes(l.Function.Filename)); err != nil {
								return nil, fmt.Errorf("append function filename: %w", err)
							}
						} else {
							if err := w.FunctionFilename.Append([]byte{}); err != nil {
								return nil, fmt.Errorf("append function filename: %w", err)
							}
						}
						w.FunctionStartLine.Append(l.Function.StartLine)
					} else {
						if err := w.FunctionName.Append([]byte{}); err != nil {
							return nil, fmt.Errorf("append function name: %w", err)
						}
						if err := w.FunctionSystemName.Append([]byte{}); err != nil {
							return nil, fmt.Errorf("append function system name: %w", err)
						}
						if err := w.FunctionFilename.Append([]byte{}); err != nil {
							return nil, fmt.Errorf("append function filename: %w", err)
						}
						w.FunctionStartLine.Append(0)
					}
				}
			} else {
				w.Lines.AppendNull()
			}
		}
	}

	return w.RecordBuilder.NewRecord(), nil
}

func stringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
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
