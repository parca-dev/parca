package profile

import (
	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/memory"
)

type Writer struct {
	RecordBuilder      *array.RecordBuilder
	LabelBuildersMap   map[string]*array.BinaryDictionaryBuilder
	LabelBuilders      []*array.BinaryDictionaryBuilder
	LocationsList      *array.ListBuilder
	Locations          *array.StructBuilder
	Addresses          *array.Uint64Builder
	Mapping            *array.StructBuilder
	MappingStart       *array.Uint64Builder
	MappingLimit       *array.Uint64Builder
	MappingOffset      *array.Uint64Builder
	MappingFile        *array.StringBuilder
	MappingBuildID     *array.StringBuilder
	Lines              *array.ListBuilder
	Line               *array.StructBuilder
	LineNumber         *array.Int64Builder
	Function           *array.StructBuilder
	FunctionName       *array.StringBuilder
	FunctionSystemName *array.StringBuilder
	FunctionFilename   *array.StringBuilder
	FunctionStartLine  *array.Int64Builder
	Value              *array.Int64Builder
	Diff               *array.Int64Builder
}

func NewWriter(pool memory.Allocator, labelNames []string) Writer {
	labelFields := make([]arrow.Field, len(labelNames))
	for i, name := range labelNames {
		labelFields[i] = arrow.Field{
			Name:     ColumnPprofLabelsPrefix + name,
			Type:     &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint16, ValueType: arrow.BinaryTypes.Binary},
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

	value := b.Field(labelNum + 1).(*array.Int64Builder)
	diff := b.Field(labelNum + 2).(*array.Int64Builder)

	return Writer{
		RecordBuilder:      b,
		LabelBuildersMap:   labelBuildersMap,
		LabelBuilders:      labelBuilders,
		LocationsList:      locationsList,
		Locations:          locations,
		Addresses:          addresses,
		Mapping:            mapping,
		MappingStart:       mappingStart,
		MappingLimit:       mappingLimit,
		MappingOffset:      mappingOffset,
		MappingFile:        mappingFile,
		MappingBuildID:     mappingBuildID,
		Lines:              lines,
		Line:               line,
		LineNumber:         lineNumber,
		Function:           function,
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
	Mapping            *array.StructBuilder
	MappingStart       *array.Uint64Builder
	MappingLimit       *array.Uint64Builder
	MappingOffset      *array.Uint64Builder
	MappingFile        *array.StringBuilder
	MappingBuildID     *array.StringBuilder
	Lines              *array.ListBuilder
	Line               *array.StructBuilder
	LineNumber         *array.Int64Builder
	Function           *array.StructBuilder
	FunctionName       *array.StringBuilder
	FunctionSystemName *array.StringBuilder
	FunctionFilename   *array.StringBuilder
	FunctionStartLine  *array.Int64Builder
	Value              *array.Int64Builder
	Diff               *array.Int64Builder
}

func NewLocationsWriter(pool memory.Allocator) LocationsWriter {
	b := array.NewRecordBuilder(pool, LocationsArrowSchema())

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

	return LocationsWriter{
		RecordBuilder:      b,
		LocationsList:      locationsList,
		Locations:          locations,
		Addresses:          addresses,
		Mapping:            mapping,
		MappingStart:       mappingStart,
		MappingLimit:       mappingLimit,
		MappingOffset:      mappingOffset,
		MappingFile:        mappingFile,
		MappingBuildID:     mappingBuildID,
		Lines:              lines,
		Line:               line,
		LineNumber:         lineNumber,
		Function:           function,
		FunctionName:       functionName,
		FunctionSystemName: functionSystemName,
		FunctionFilename:   functionFilename,
		FunctionStartLine:  functionStartLine,
	}
}
