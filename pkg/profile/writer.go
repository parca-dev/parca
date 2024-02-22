// Copyright 2023-2024 The Parca Authors
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
	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/memory"
)

type Writer struct {
	RecordBuilder      *array.RecordBuilder
	LabelBuildersMap   map[string]*array.BinaryDictionaryBuilder
	LabelBuilders      []*array.BinaryDictionaryBuilder
	LocationsList      *array.ListBuilder
	Locations          *array.StructBuilder
	Addresses          *array.Uint64Builder
	MappingStart       *array.Uint64Builder
	MappingLimit       *array.Uint64Builder
	MappingOffset      *array.Uint64Builder
	MappingFile        *array.BinaryDictionaryBuilder
	MappingBuildID     *array.BinaryDictionaryBuilder
	Lines              *array.ListBuilder
	Line               *array.StructBuilder
	LineNumber         *array.Int64Builder
	FunctionName       *array.BinaryDictionaryBuilder
	FunctionSystemName *array.BinaryDictionaryBuilder
	FunctionFilename   *array.BinaryDictionaryBuilder
	FunctionStartLine  *array.Int64Builder
	Value              *array.Int64Builder
	Diff               *array.Int64Builder
}

func (w *Writer) Release() {
	w.RecordBuilder.Release()
}

func NewWriter(pool memory.Allocator, labelNames []string) Writer {
	labelFields := make([]arrow.Field, len(labelNames))
	for i, name := range labelNames {
		labelFields[i] = arrow.Field{
			Name:     ColumnPprofLabelsPrefix + name,
			Type:     &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
			Nullable: true,
		}
	}

	b := array.NewRecordBuilder(pool, ArrowSchema(labelFields))

	labelNum := len(labelFields)
	labelBuilders := make([]*array.BinaryDictionaryBuilder, labelNum)
	labelBuildersMap := make(map[string]*array.BinaryDictionaryBuilder, labelNum)
	for i := 0; i < labelNum; i++ {
		labelBuilders[i] = b.Field(i).(*array.BinaryDictionaryBuilder)
		labelBuildersMap[labelNames[i]] = labelBuilders[i]
	}

	locationsList := b.Field(labelNum).(*array.ListBuilder)
	locations := locationsList.ValueBuilder().(*array.StructBuilder)

	addresses := locations.FieldBuilder(0).(*array.Uint64Builder)

	mappingStart := locations.FieldBuilder(1).(*array.Uint64Builder)
	mappingLimit := locations.FieldBuilder(2).(*array.Uint64Builder)
	mappingOffset := locations.FieldBuilder(3).(*array.Uint64Builder)
	mappingFile := locations.FieldBuilder(4).(*array.BinaryDictionaryBuilder)
	mappingBuildID := locations.FieldBuilder(5).(*array.BinaryDictionaryBuilder)

	lines := locations.FieldBuilder(6).(*array.ListBuilder)
	line := lines.ValueBuilder().(*array.StructBuilder)
	lineNumber := line.FieldBuilder(0).(*array.Int64Builder)
	functionName := line.FieldBuilder(1).(*array.BinaryDictionaryBuilder)
	functionSystemName := line.FieldBuilder(2).(*array.BinaryDictionaryBuilder)
	functionFilename := line.FieldBuilder(3).(*array.BinaryDictionaryBuilder)
	functionStartLine := line.FieldBuilder(4).(*array.Int64Builder)

	value := b.Field(labelNum + 1).(*array.Int64Builder)
	diff := b.Field(labelNum + 2).(*array.Int64Builder)

	return Writer{
		RecordBuilder:      b,
		LabelBuildersMap:   labelBuildersMap,
		LabelBuilders:      labelBuilders,
		LocationsList:      locationsList,
		Locations:          locations,
		Addresses:          addresses,
		MappingStart:       mappingStart,
		MappingLimit:       mappingLimit,
		MappingOffset:      mappingOffset,
		MappingFile:        mappingFile,
		MappingBuildID:     mappingBuildID,
		Lines:              lines,
		Line:               line,
		LineNumber:         lineNumber,
		FunctionName:       functionName,
		FunctionSystemName: functionSystemName,
		FunctionFilename:   functionFilename,
		FunctionStartLine:  functionStartLine,
		Value:              value,
		Diff:               diff,
	}
}

type LocationsWriter struct {
	RecordBuilder      *array.RecordBuilder
	LabelBuildersMap   map[string]*array.BinaryDictionaryBuilder
	LabelBuilders      []*array.BinaryDictionaryBuilder
	LocationsList      *array.ListBuilder
	Locations          *array.StructBuilder
	Addresses          *array.Uint64Builder
	MappingStart       *array.Uint64Builder
	MappingLimit       *array.Uint64Builder
	MappingOffset      *array.Uint64Builder
	MappingFile        *array.BinaryDictionaryBuilder
	MappingBuildID     *array.BinaryDictionaryBuilder
	Lines              *array.ListBuilder
	Line               *array.StructBuilder
	LineNumber         *array.Int64Builder
	FunctionName       *array.BinaryDictionaryBuilder
	FunctionSystemName *array.BinaryDictionaryBuilder
	FunctionFilename   *array.BinaryDictionaryBuilder
	FunctionStartLine  *array.Int64Builder
	Value              *array.Int64Builder
	Diff               *array.Int64Builder
}

func NewLocationsWriter(pool memory.Allocator) LocationsWriter {
	b := array.NewRecordBuilder(pool, LocationsArrowSchema())

	locationsList := b.Field(0).(*array.ListBuilder)
	locations := locationsList.ValueBuilder().(*array.StructBuilder)

	addresses := locations.FieldBuilder(0).(*array.Uint64Builder)

	mappingStart := locations.FieldBuilder(1).(*array.Uint64Builder)
	mappingLimit := locations.FieldBuilder(2).(*array.Uint64Builder)
	mappingOffset := locations.FieldBuilder(3).(*array.Uint64Builder)
	mappingFile := locations.FieldBuilder(4).(*array.BinaryDictionaryBuilder)
	mappingBuildID := locations.FieldBuilder(5).(*array.BinaryDictionaryBuilder)

	lines := locations.FieldBuilder(6).(*array.ListBuilder)
	line := lines.ValueBuilder().(*array.StructBuilder)
	lineNumber := line.FieldBuilder(0).(*array.Int64Builder)
	functionName := line.FieldBuilder(1).(*array.BinaryDictionaryBuilder)
	functionSystemName := line.FieldBuilder(2).(*array.BinaryDictionaryBuilder)
	functionFilename := line.FieldBuilder(3).(*array.BinaryDictionaryBuilder)
	functionStartLine := line.FieldBuilder(4).(*array.Int64Builder)

	return LocationsWriter{
		RecordBuilder:      b,
		LocationsList:      locationsList,
		Locations:          locations,
		Addresses:          addresses,
		MappingStart:       mappingStart,
		MappingLimit:       mappingLimit,
		MappingOffset:      mappingOffset,
		MappingFile:        mappingFile,
		MappingBuildID:     mappingBuildID,
		Lines:              lines,
		Line:               line,
		LineNumber:         lineNumber,
		FunctionName:       functionName,
		FunctionSystemName: functionSystemName,
		FunctionFilename:   functionFilename,
		FunctionStartLine:  functionStartLine,
	}
}
