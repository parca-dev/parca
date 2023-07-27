package profile

import (
	"strings"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
)

type LabelColumn struct {
	Col  *array.Dictionary
	Dict *array.String
}

type Reader struct {
	Profile Profile

	LabelFields  []arrow.Field
	LabelColumns []LabelColumn

	Locations              *array.List
	LocationOffsets        []int32
	Location               *array.Struct
	Address                *array.Uint64
	Mapping                *array.Struct
	MappingStart           *array.Uint64
	MappingLimit           *array.Uint64
	MappingOffset          *array.Uint64
	MappingFile            *array.String
	MappingBuildID         *array.String
	Lines                  *array.List
	LineOffsets            []int32
	Line                   *array.Struct
	LineNumber             *array.Int64
	LineFunction           *array.Struct
	LineFunctionName       *array.String
	LineFunctionSystemName *array.String
	LineFunctionFilename   *array.String
	LineFunctionStartLine  *array.Int64

	Value *array.Int64
	Diff  *array.Int64
}

func NewReader(p Profile) Reader {
	ar := p.Samples
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
			Dict: col.Dictionary().(*array.String),
		}
	}
	labelNum := len(labelFields)

	// Get readers from the unfiltered profile.
	locations := ar.Column(labelNum).(*array.List)
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
	valueColumn := ar.Column(labelNum + 1).(*array.Int64)
	diffColumn := ar.Column(labelNum + 2).(*array.Int64)

	return Reader{
		Profile:                p,
		LabelFields:            labelFields,
		LabelColumns:           labelColumns,
		Locations:              locations,
		LocationOffsets:        locationOffsets,
		Location:               location,
		Address:                address,
		Mapping:                mapping,
		MappingStart:           mappingStart,
		MappingLimit:           mappingLimit,
		MappingOffset:          mappingOffset,
		MappingFile:            mappingFile,
		MappingBuildID:         mappingBuildID,
		Lines:                  lines,
		LineOffsets:            lineOffsets,
		Line:                   line,
		LineNumber:             lineNumber,
		LineFunction:           lineFunction,
		LineFunctionName:       lineFunctionName,
		LineFunctionSystemName: lineFunctionSystemName,
		LineFunctionFilename:   lineFunctionFilename,
		LineFunctionStartLine:  lineFunctionStartLine,
		Value:                  valueColumn,
		Diff:                   diffColumn,
	}
}
