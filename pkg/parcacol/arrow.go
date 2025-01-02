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

package parcacol

import (
	"context"
	"fmt"
	"strings"
	"unsafe"

	"github.com/apache/arrow/go/v17/arrow"
	"github.com/apache/arrow/go/v17/arrow/array"
	"github.com/apache/arrow/go/v17/arrow/memory"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/kv"
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
	key    *kv.KeyMaker
}

func NewArrowToProfileConverter(
	tracer trace.Tracer,
	keyMaker *kv.KeyMaker,
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
		locationOffsets := locations.Offsets()[locations.Offset() : locations.Offset()+1+locations.Len()] // Adjust offsets by the data offset. This happens if this list is a slice of a larger list.
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
		lineOffsets := lines.Offsets()[lines.Offset() : lines.Offset()+1+lines.Len()] // Adjust offsets by the data offset. This happens if this list is a slice of a larger list.
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
			if strings.HasPrefix(field.Name, profile.ColumnLabelsPrefix) {
				labelIndexes[strings.TrimPrefix(field.Name, profile.ColumnLabelsPrefix)] = i
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
				if locations.ListValues().IsNull(j) { // Ignore null locations; they have been filtered out.
					continue
				}

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

			if len(loc.Lines) > 0 {
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
