// Copyright 2023-2026 The Parca Authors
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

package profile

import (
	"fmt"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
)

type LabelColumn struct {
	Col  *array.Uint32
	Dict *array.Binary
}

type Reader struct {
	Profile       Profile
	RecordReaders []*RecordReader
}

type RecordReader struct {
	Record arrow.RecordBatch

	LabelFields  []arrow.Field
	LabelColumns []LabelColumn

	Locations                     *array.List
	Location                      *array.Struct
	Address                       *array.Uint64
	Timestamp                     *array.Int64
	Period                        *array.Int64
	MappingStart                  *array.Uint64
	MappingLimit                  *array.Uint64
	MappingOffset                 *array.Uint64
	MappingFileIndices            *array.Uint32
	MappingFileDict               *array.Binary
	MappingBuildIDIndices         *array.Uint32
	MappingBuildIDDict            *array.Binary
	Lines                         *array.List
	Line                          *array.Struct
	LineNumber                    *array.Int64
	LineFunctionNameIndices       *array.Uint32
	LineFunctionNameDict          *array.Binary
	LineFunctionSystemNameIndices *array.Uint32
	LineFunctionSystemNameDict    *array.Binary
	LineFunctionFilenameIndices   *array.Uint32
	LineFunctionFilenameDict      *array.Binary
	LineFunctionStartLine         *array.Int64

	Value *array.Int64
	Diff  *array.Int64
}

func NewReader(p Profile) (Reader, error) {
	r := Reader{
		Profile: p,
	}

	for _, ar := range p.Samples {
		rr, err := NewRecordReader(ar)
		if err != nil {
			return Reader{}, err
		}
		r.RecordReaders = append(r.RecordReaders, rr)
	}
	return r, nil
}

func NewRecordReader(ar arrow.RecordBatch) (*RecordReader, error) {
	schema := ar.Schema()

	rr := &RecordReader{
		Record: ar,
	}

	labelFields := make([]arrow.Field, 0, schema.NumFields())
	labelColumns := make([]LabelColumn, 0, schema.NumFields())

	// Iterate over schema fields once and populate the RecordReader
	for i, field := range schema.Fields() {
		if strings.HasPrefix(field.Name, ColumnLabelsPrefix) {
			labelFields = append(labelFields, field)
			col := ar.Column(i).(*array.Dictionary)
			labelColumns = append(labelColumns, LabelColumn{
				Col:  col.Indices().(*array.Uint32),
				Dict: col.Dictionary().(*array.Binary),
			})
			continue
		}

		switch field.Name {
		case "locations":
			rr.Locations = ar.Column(i).(*array.List)
			rr.Location = rr.Locations.ListValues().(*array.Struct)

			// Process location struct fields by name
			locationType := rr.Location.DataType().(*arrow.StructType)
			for j := 0; j < locationType.NumFields(); j++ {
				locField := locationType.Field(j)
				switch locField.Name {
				case "address":
					rr.Address = rr.Location.Field(j).(*array.Uint64)
				case "mapping_start":
					rr.MappingStart = rr.Location.Field(j).(*array.Uint64)
				case "mapping_limit":
					rr.MappingLimit = rr.Location.Field(j).(*array.Uint64)
				case "mapping_offset":
					rr.MappingOffset = rr.Location.Field(j).(*array.Uint64)
				case "mapping_file":
					mappingFile := rr.Location.Field(j).(*array.Dictionary)
					rr.MappingFileIndices = mappingFile.Indices().(*array.Uint32)
					rr.MappingFileDict = mappingFile.Dictionary().(*array.Binary)
				case "mapping_build_id":
					mappingBuildID := rr.Location.Field(j).(*array.Dictionary)
					rr.MappingBuildIDIndices = mappingBuildID.Indices().(*array.Uint32)
					rr.MappingBuildIDDict = mappingBuildID.Dictionary().(*array.Binary)
				case "lines":
					rr.Lines = rr.Location.Field(j).(*array.List)
					rr.Line = rr.Lines.ListValues().(*array.Struct)

					// Process line struct fields by name
					lineType := rr.Line.DataType().(*arrow.StructType)
					for k := 0; k < lineType.NumFields(); k++ {
						lineField := lineType.Field(k)
						switch lineField.Name {
						case "line":
							rr.LineNumber = rr.Line.Field(k).(*array.Int64)
						case "function_name":
							lineFunctionName := rr.Line.Field(k).(*array.Dictionary)
							rr.LineFunctionNameIndices = lineFunctionName.Indices().(*array.Uint32)
							rr.LineFunctionNameDict = lineFunctionName.Dictionary().(*array.Binary)
						case "function_system_name":
							lineFunctionSystemName := rr.Line.Field(k).(*array.Dictionary)
							rr.LineFunctionSystemNameIndices = lineFunctionSystemName.Indices().(*array.Uint32)
							rr.LineFunctionSystemNameDict = lineFunctionSystemName.Dictionary().(*array.Binary)
						case "function_filename":
							lineFunctionFilename := rr.Line.Field(k).(*array.Dictionary)
							rr.LineFunctionFilenameIndices = lineFunctionFilename.Indices().(*array.Uint32)
							rr.LineFunctionFilenameDict = lineFunctionFilename.Dictionary().(*array.Binary)
						case "function_start_line":
							rr.LineFunctionStartLine = rr.Line.Field(k).(*array.Int64)
						}
					}
				}
			}
		case "value":
			rr.Value = ar.Column(i).(*array.Int64)
		case "diff":
			rr.Diff = ar.Column(i).(*array.Int64)
		case ColumnTimestamp:
			rr.Timestamp = ar.Column(i).(*array.Int64)
		case ColumnPeriod:
			rr.Period = ar.Column(i).(*array.Int64)
		}
	}

	rr.LabelFields = labelFields
	rr.LabelColumns = labelColumns

	// Validate that all required fields were found
	if rr.Locations == nil {
		return nil, fmt.Errorf("missing required field: locations")
	}
	if rr.Location == nil {
		return nil, fmt.Errorf("missing required field: location")
	}
	if rr.Address == nil {
		return nil, fmt.Errorf("missing required field: address")
	}
	if rr.MappingStart == nil {
		return nil, fmt.Errorf("missing required field: mapping_start")
	}
	if rr.MappingLimit == nil {
		return nil, fmt.Errorf("missing required field: mapping_limit")
	}
	if rr.MappingOffset == nil {
		return nil, fmt.Errorf("missing required field: mapping_offset")
	}
	if rr.MappingFileIndices == nil || rr.MappingFileDict == nil {
		return nil, fmt.Errorf("missing required field: mapping_file")
	}
	if rr.MappingBuildIDIndices == nil || rr.MappingBuildIDDict == nil {
		return nil, fmt.Errorf("missing required field: mapping_build_id")
	}
	if rr.Lines == nil {
		return nil, fmt.Errorf("missing required field: lines")
	}
	if rr.Line == nil {
		return nil, fmt.Errorf("missing required field: line")
	}
	if rr.LineNumber == nil {
		return nil, fmt.Errorf("missing required field: line")
	}
	if rr.LineFunctionNameIndices == nil || rr.LineFunctionNameDict == nil {
		return nil, fmt.Errorf("missing required field: function_name")
	}
	if rr.LineFunctionSystemNameIndices == nil || rr.LineFunctionSystemNameDict == nil {
		return nil, fmt.Errorf("missing required field: function_system_name")
	}
	if rr.LineFunctionFilenameIndices == nil || rr.LineFunctionFilenameDict == nil {
		return nil, fmt.Errorf("missing required field: function_filename")
	}
	if rr.LineFunctionStartLine == nil {
		return nil, fmt.Errorf("missing required field: function_start_line")
	}
	if rr.Value == nil {
		return nil, fmt.Errorf("missing required field: value")
	}
	if rr.Diff == nil {
		return nil, fmt.Errorf("missing required field: diff")
	}
	if rr.Timestamp == nil {
		return nil, fmt.Errorf("missing required field: %s", ColumnTimestamp)
	}
	if rr.Period == nil {
		return nil, fmt.Errorf("missing required field: %s", ColumnPeriod)
	}

	return rr, nil
}
