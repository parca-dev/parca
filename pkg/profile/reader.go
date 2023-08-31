// Copyright 2023 The Parca Authors
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

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/array"
)

type LabelColumn struct {
	Col  *array.Dictionary
	Dict *array.Binary
}

type Reader struct {
	Profile       Profile
	RecordReaders []*RecordReader
}

type RecordReader struct {
	Record arrow.Record

	LabelFields  []arrow.Field
	LabelColumns []LabelColumn

	Locations                  *array.List
	Location                   *array.Struct
	Address                    *array.Uint64
	Mapping                    *array.Struct
	MappingStart               *array.Uint64
	MappingLimit               *array.Uint64
	MappingOffset              *array.Uint64
	MappingFile                *array.Dictionary
	MappingFileDict            *array.Binary
	MappingBuildID             *array.Dictionary
	MappingBuildIDDict         *array.Binary
	Lines                      *array.List
	Line                       *array.Struct
	LineNumber                 *array.Int64
	LineFunction               *array.Struct
	LineFunctionName           *array.Dictionary
	LineFunctionNameDict       *array.Binary
	LineFunctionSystemName     *array.Dictionary
	LineFunctionSystemNameDict *array.Binary
	LineFunctionFilename       *array.Dictionary
	LineFunctionFilenameDict   *array.Binary
	LineFunctionStartLine      *array.Int64

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

func NewRecordReader(ar arrow.Record) *RecordReader {
	schema := ar.Schema()

	labelFields := make([]arrow.Field, 0, schema.NumFields())
	for _, field := range schema.Fields() {
		if strings.HasPrefix(field.Name, ColumnPprofLabelsPrefix) {
			labelFields = append(labelFields, field)
		}
	}

	labelColumns := make([]LabelColumn, len(labelFields))
	for i := range labelFields {
		col := ar.Column(i).(*array.Dictionary)
		labelColumns[i] = LabelColumn{
			Col:  col,
			Dict: col.Dictionary().(*array.Binary),
		}
	}
	labelNum := len(labelFields)

	// Get readers from the unfiltered profile.
	locations := ar.Column(labelNum).(*array.List)
	location := locations.ListValues().(*array.Struct)
	address := location.Field(0).(*array.Uint64)
	mapping := location.Field(1).(*array.Struct)
	mappingStart := mapping.Field(0).(*array.Uint64)
	mappingLimit := mapping.Field(1).(*array.Uint64)
	mappingOffset := mapping.Field(2).(*array.Uint64)
	mappingFile := mapping.Field(3).(*array.Dictionary)
	mappingFileDict := mappingFile.Dictionary().(*array.Binary)
	mappingBuildID := mapping.Field(4).(*array.Dictionary)
	mappingBuildIDDict := mappingBuildID.Dictionary().(*array.Binary)
	lines := location.Field(2).(*array.List)
	line := lines.ListValues().(*array.Struct)
	lineNumber := line.Field(0).(*array.Int64)
	lineFunction := line.Field(1).(*array.Struct)
	lineFunctionName := lineFunction.Field(0).(*array.Dictionary)
	lineFunctionNameDict := lineFunctionName.Dictionary().(*array.Binary)
	lineFunctionSystemName := lineFunction.Field(1).(*array.Dictionary)
	lineFunctionSystemNameDict := lineFunctionSystemName.Dictionary().(*array.Binary)
	lineFunctionFilename := lineFunction.Field(2).(*array.Dictionary)
	lineFunctionFilenameDict := lineFunctionFilename.Dictionary().(*array.Binary)
	lineFunctionStartLine := lineFunction.Field(3).(*array.Int64)
	valueColumn := ar.Column(labelNum + 1).(*array.Int64)
	diffColumn := ar.Column(labelNum + 2).(*array.Int64)

	return &RecordReader{
		Record:                     ar,
		LabelFields:                labelFields,
		LabelColumns:               labelColumns,
		Locations:                  locations,
		Location:                   location,
		Address:                    address,
		Mapping:                    mapping,
		MappingStart:               mappingStart,
		MappingLimit:               mappingLimit,
		MappingOffset:              mappingOffset,
		MappingFile:                mappingFile,
		MappingFileDict:            mappingFileDict,
		MappingBuildID:             mappingBuildID,
		MappingBuildIDDict:         mappingBuildIDDict,
		Lines:                      lines,
		Line:                       line,
		LineNumber:                 lineNumber,
		LineFunction:               lineFunction,
		LineFunctionName:           lineFunctionName,
		LineFunctionNameDict:       lineFunctionNameDict,
		LineFunctionSystemName:     lineFunctionSystemName,
		LineFunctionSystemNameDict: lineFunctionSystemNameDict,
		LineFunctionFilename:       lineFunctionFilename,
		LineFunctionFilenameDict:   lineFunctionFilenameDict,
		LineFunctionStartLine:      lineFunctionStartLine,
		Value:                      valueColumn,
		Diff:                       diffColumn,
	}
}
