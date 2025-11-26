// Copyright 2023-2025 The Parca Authors
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
	LineColumn                    *array.Uint64
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

func NewReader(p Profile) Reader {
	r := Reader{
		Profile: p,
	}

	for _, ar := range p.Samples {
		r.RecordReaders = append(r.RecordReaders, NewRecordReader(ar))
	}
	return r
}

func NewRecordReader(ar arrow.RecordBatch) *RecordReader {
	schema := ar.Schema()

	labelFields := make([]arrow.Field, 0, schema.NumFields())
	for _, field := range schema.Fields() {
		if strings.HasPrefix(field.Name, ColumnLabelsPrefix) {
			labelFields = append(labelFields, field)
		}
	}

	labelColumns := make([]LabelColumn, len(labelFields))
	for i := range labelFields {
		col := ar.Column(i).(*array.Dictionary)
		labelColumns[i] = LabelColumn{
			Col:  col.Indices().(*array.Uint32),
			Dict: col.Dictionary().(*array.Binary),
		}
	}
	labelNum := len(labelFields)

	// Get readers from the unfiltered profile.
	locations := ar.Column(labelNum).(*array.List)
	location := locations.ListValues().(*array.Struct)
	address := location.Field(0).(*array.Uint64)
	mappingStart := location.Field(1).(*array.Uint64)
	mappingLimit := location.Field(2).(*array.Uint64)
	mappingOffset := location.Field(3).(*array.Uint64)
	mappingFile := location.Field(4).(*array.Dictionary)
	mappingFileIndices := mappingFile.Indices().(*array.Uint32)
	mappingFileDict := mappingFile.Dictionary().(*array.Binary)
	mappingBuildID := location.Field(5).(*array.Dictionary)
	mappingBuildIDIndices := mappingBuildID.Indices().(*array.Uint32)
	mappingBuildIDDict := mappingBuildID.Dictionary().(*array.Binary)
	lines := location.Field(6).(*array.List)
	line := lines.ListValues().(*array.Struct)
	lineNumber := line.Field(0).(*array.Int64)
	lineColumn := line.Field(1).(*array.Uint64)
	lineFunctionName := line.Field(2).(*array.Dictionary)
	lineFunctionNameIndices := lineFunctionName.Indices().(*array.Uint32)
	lineFunctionNameDict := lineFunctionName.Dictionary().(*array.Binary)
	lineFunctionSystemName := line.Field(3).(*array.Dictionary)
	lineFunctionSystemNameIndices := lineFunctionSystemName.Indices().(*array.Uint32)
	lineFunctionSystemNameDict := lineFunctionSystemName.Dictionary().(*array.Binary)
	lineFunctionFilename := line.Field(4).(*array.Dictionary)
	lineFunctionFilenameIndices := lineFunctionFilename.Indices().(*array.Uint32)
	lineFunctionFilenameDict := lineFunctionFilename.Dictionary().(*array.Binary)
	lineFunctionStartLine := line.Field(5).(*array.Int64)
	valueColumn := ar.Column(labelNum + 1).(*array.Int64)
	diffColumn := ar.Column(labelNum + 2).(*array.Int64)
	timestamp := ar.Column(labelNum + 3).(*array.Int64)
	period := ar.Column(labelNum + 4).(*array.Int64)

	return &RecordReader{
		Record:                        ar,
		LabelFields:                   labelFields,
		LabelColumns:                  labelColumns,
		Locations:                     locations,
		Location:                      location,
		Address:                       address,
		MappingStart:                  mappingStart,
		MappingLimit:                  mappingLimit,
		MappingOffset:                 mappingOffset,
		MappingFileIndices:            mappingFileIndices,
		MappingFileDict:               mappingFileDict,
		MappingBuildIDIndices:         mappingBuildIDIndices,
		MappingBuildIDDict:            mappingBuildIDDict,
		Lines:                         lines,
		Line:                          line,
		LineNumber:                    lineNumber,
		LineColumn:                    lineColumn,
		LineFunctionNameIndices:       lineFunctionNameIndices,
		LineFunctionNameDict:          lineFunctionNameDict,
		LineFunctionSystemNameIndices: lineFunctionSystemNameIndices,
		LineFunctionSystemNameDict:    lineFunctionSystemNameDict,
		LineFunctionFilenameIndices:   lineFunctionFilenameIndices,
		LineFunctionFilenameDict:      lineFunctionFilenameDict,
		LineFunctionStartLine:         lineFunctionStartLine,
		Value:                         valueColumn,
		Diff:                          diffColumn,
		Timestamp:                     timestamp,
		Period:                        period,
	}
}
