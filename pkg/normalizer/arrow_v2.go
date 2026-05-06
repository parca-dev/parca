// Copyright 2024-2026 The Parca Authors
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

package normalizer

import (
	"context"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"

	"github.com/parca-dev/parca/pkg/profile"
)

// AddSampleRecordV2 decodes a v2 sample record into the internal record builder.
//
// The v2 schema differs from v1 in two key ways:
//   - Stacktraces are inlined as a ListView of dictionary-encoded location structs
//     rather than referenced by id with a separate locations record.
//   - Several fields use Arrow native types (Timestamp, StringView, UUID extension,
//     run-end encoded String) instead of Binary/Int64 with explicit dictionaries.
//
// The output mirrors the v1 internal record so the rest of the ingestion pipeline
// can remain unchanged.
func (c *arrowToInternalConverter) AddSampleRecordV2(
	ctx context.Context,
	rec arrow.RecordBatch,
) error {
	var (
		labelsStruct  *array.Struct
		stacktraceArr *array.ListView
		err           error
	)

	s := rec.Schema()
	for i, field := range s.Fields() {
		switch field.Name {
		case "labels":
			ls, ok := rec.Column(i).(*array.Struct)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Struct, got %T", field.Name, rec.Column(i))
			}
			labelsStruct = ls
		case profile.ColumnStacktrace:
			lv, ok := rec.Column(i).(*array.ListView)
			if !ok {
				return fmt.Errorf("expected column %q to be of type ListView, got %T", field.Name, rec.Column(i))
			}
			stacktraceArr = lv
		case "stacktrace_id":
			c.b.StacktraceIDs, err = uuidExtensionToBinaryDictV2(c.mem, rec.Column(i))
			if err != nil {
				return fmt.Errorf("convert stacktrace_id: %w", err)
			}
		case profile.ColumnValue:
			v, ok := rec.Column(i).(*array.Int64)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Int64, got %T", field.Name, rec.Column(i))
			}
			v.Retain()
			c.b.Value = v
		case "producer":
			c.b.Producer, err = expandREEStringToBinaryDictV2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnSampleType:
			c.b.SampleType, err = expandREEStringToBinaryDictV2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnSampleUnit:
			c.b.SampleUnit, err = expandREEStringToBinaryDictV2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnPeriodType:
			c.b.PeriodType, err = expandREEStringToBinaryDictV2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnPeriodUnit:
			c.b.PeriodUnit, err = expandREEStringToBinaryDictV2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case "temporality":
			// No internal storage column.
		case profile.ColumnPeriod:
			c.b.Period, err = expandREEInt64(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnDuration:
			c.b.Duration, err = expandREEUint64ToInt64V2(c.mem, rec.Column(i), field.Name)
			if err != nil {
				return err
			}
		case profile.ColumnTimestamp:
			ts, ok := rec.Column(i).(*array.Timestamp)
			if !ok {
				return fmt.Errorf("expected column %q to be of type Timestamp, got %T", field.Name, rec.Column(i))
			}
			c.b.TimeNanos, c.b.Timestamp = expandTimestampV2(c.mem, ts)
		}
	}

	if labelsStruct != nil {
		structType := labelsStruct.DataType().(*arrow.StructType)
		for j := 0; j < labelsStruct.NumField(); j++ {
			field := structType.Field(j)
			d, err := expandREEStringDictToBinaryDictV2(c.mem, labelsStruct.Field(j), field.Name)
			if err != nil {
				return fmt.Errorf("decode label %q: %w", field.Name, err)
			}
			c.b.Labels = append(c.b.Labels, d)
			c.b.LabelFields = append(c.b.LabelFields, arrow.Field{
				Name: profile.ColumnLabelsPrefix + field.Name,
				Type: d.DataType(),
			})
		}
	}

	if stacktraceArr != nil {
		c.b.Locations, err = encodeStacktraceListViewV2(c.mem, stacktraceArr)
		if err != nil {
			return fmt.Errorf("encode stacktrace: %w", err)
		}
	}

	return nil
}

// expandREEStringToBinaryDictV2 expands a RunEndEncoded[String] array into a
// Dictionary[Uint32, Binary] array, which is the type the internal columnar
// store expects.
func expandREEStringToBinaryDictV2(mem memory.Allocator, arr arrow.Array, fieldName string) (*array.Dictionary, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}
	str, ok := ree.Values().(*array.String)
	if !ok {
		return nil, fmt.Errorf("expected column %q RunEndEncoded values to be String, got %T", fieldName, ree.Values())
	}

	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary}
	b := array.NewBuilder(mem, typ).(*array.BinaryDictionaryBuilder)
	defer b.Release()

	runEnds := ree.RunEndsArr().(*array.Int32)
	prevEnd := int32(0)
	for i := 0; i < runEnds.Len(); i++ {
		end := runEnds.Value(i)
		isNull := str.IsNull(i)
		var v string
		if !isNull {
			v = str.Value(i)
		}
		for j := prevEnd; j < end; j++ {
			if isNull || len(v) == 0 {
				b.AppendNull()
				continue
			}
			if err := b.AppendString(v); err != nil {
				return nil, fmt.Errorf("append value: %w", err)
			}
		}
		prevEnd = end
	}

	return b.NewArray().(*array.Dictionary), nil
}

// expandREEStringDictToBinaryDictV2 expands a RunEndEncoded[Dictionary[Uint32, String]]
// array (used for v2 labels) into a Dictionary[Uint32, Binary] array.
func expandREEStringDictToBinaryDictV2(mem memory.Allocator, arr arrow.Array, fieldName string) (*array.Dictionary, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}
	dict, ok := ree.Values().(*array.Dictionary)
	if !ok {
		return nil, fmt.Errorf("expected column %q RunEndEncoded values to be Dictionary, got %T", fieldName, ree.Values())
	}
	strDict, ok := dict.Dictionary().(*array.String)
	if !ok {
		return nil, fmt.Errorf("expected column %q dictionary values to be String, got %T", fieldName, dict.Dictionary())
	}

	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary}
	b := array.NewBuilder(mem, typ).(*array.BinaryDictionaryBuilder)
	defer b.Release()

	runEnds := ree.RunEndsArr().(*array.Int32)
	prevEnd := int32(0)
	for i := 0; i < runEnds.Len(); i++ {
		end := runEnds.Value(i)
		isNull := dict.IsNull(i)
		var v string
		if !isNull {
			v = strDict.Value(dict.GetValueIndex(i))
		}
		for j := prevEnd; j < end; j++ {
			if isNull || len(v) == 0 {
				b.AppendNull()
				continue
			}
			if err := b.AppendString(v); err != nil {
				return nil, fmt.Errorf("append value: %w", err)
			}
		}
		prevEnd = end
	}

	return b.NewArray().(*array.Dictionary), nil
}

// expandREEUint64ToInt64V2 expands a RunEndEncoded[Uint64] array into a flat Int64
// array. The internal storage uses Int64 for duration but the v2 wire type is Uint64.
func expandREEUint64ToInt64V2(mem memory.Allocator, arr arrow.Array, fieldName string) (*array.Int64, error) {
	ree, ok := arr.(*array.RunEndEncoded)
	if !ok {
		return nil, fmt.Errorf("expected column %q to be of type RunEndEncoded, got %T", fieldName, arr)
	}
	uint64Arr, ok := ree.Values().(*array.Uint64)
	if !ok {
		return nil, fmt.Errorf("expected column %q RunEndEncoded values to be Uint64, got %T", fieldName, ree.Values())
	}

	b := array.NewInt64Builder(mem)
	defer b.Release()

	runEnds := ree.RunEndsArr().(*array.Int32)
	prevEnd := int32(0)
	for i := 0; i < runEnds.Len(); i++ {
		end := runEnds.Value(i)
		v := int64(uint64Arr.Value(i))
		for j := prevEnd; j < end; j++ {
			b.Append(v)
		}
		prevEnd = end
	}

	return b.NewInt64Array(), nil
}

// expandTimestampV2 splits a nanosecond-resolution Timestamp array into the two
// internal columns the database expects: time_nanos (raw nanoseconds) and
// timestamp (milliseconds).
func expandTimestampV2(mem memory.Allocator, ts *array.Timestamp) (timeNanos, timestampMs *array.Int64) {
	nanosB := array.NewInt64Builder(mem)
	defer nanosB.Release()
	msB := array.NewInt64Builder(mem)
	defer msB.Release()

	nsPerMs := time.Millisecond.Nanoseconds()
	for i := 0; i < ts.Len(); i++ {
		n := int64(ts.Value(i))
		nanosB.Append(n)
		msB.Append(n / nsPerMs)
	}

	return nanosB.NewInt64Array(), msB.NewInt64Array()
}

// uuidExtensionToBinaryDictV2 turns the UUID extension column (16-byte
// FixedSizeBinary storage) into the Dictionary[Uint32, Binary] shape used as
// the stacktrace id column in the internal builder.
func uuidExtensionToBinaryDictV2(mem memory.Allocator, arr arrow.Array) (*array.Dictionary, error) {
	ext, ok := arr.(array.ExtensionArray)
	if !ok {
		return nil, fmt.Errorf("expected ExtensionArray for stacktrace_id, got %T", arr)
	}
	storage, ok := ext.Storage().(*array.FixedSizeBinary)
	if !ok {
		return nil, fmt.Errorf("expected FixedSizeBinary storage for stacktrace_id, got %T", ext.Storage())
	}

	typ := &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary}
	b := array.NewBuilder(mem, typ).(*array.BinaryDictionaryBuilder)
	defer b.Release()

	for i := 0; i < storage.Len(); i++ {
		if storage.IsNull(i) {
			b.AppendNull()
			continue
		}
		if err := b.Append(storage.Value(i)); err != nil {
			return nil, fmt.Errorf("append: %w", err)
		}
	}

	return b.NewArray().(*array.Dictionary), nil
}

// v2LocationReader holds typed views into the v2 location dictionary so each
// location can be encoded by index.
type v2LocationReader struct {
	address         *array.Uint64
	mappingFile     *array.Dictionary
	mappingFileVals *array.String
	buildID         *array.Dictionary
	buildIDVals     *array.String
	lines           *array.ListView

	lineNumber  *array.Uint64
	lineColumn  *array.Uint64
	function    *array.Dictionary
	systemName  *array.StringView
	filename    *array.Dictionary
	filenameVal *array.String
	startLine   *array.Uint64
}

func newV2LocationReader(locStruct *array.Struct) (*v2LocationReader, error) {
	r := &v2LocationReader{}

	locType := locStruct.DataType().(*arrow.StructType)
	for i := 0; i < locType.NumFields(); i++ {
		f := locType.Field(i)
		switch f.Name {
		case "address":
			arr, ok := locStruct.Field(i).(*array.Uint64)
			if !ok {
				return nil, fmt.Errorf("expected location.address to be Uint64, got %T", locStruct.Field(i))
			}
			r.address = arr
		case "frame_type":
			// Not encoded into the binary location format.
		case "mapping_file":
			d, ok := locStruct.Field(i).(*array.Dictionary)
			if !ok {
				return nil, fmt.Errorf("expected location.mapping_file to be Dictionary, got %T", locStruct.Field(i))
			}
			vals, ok := d.Dictionary().(*array.String)
			if !ok {
				return nil, fmt.Errorf("expected location.mapping_file dictionary values to be String, got %T", d.Dictionary())
			}
			r.mappingFile = d
			r.mappingFileVals = vals
		case "mapping_build_id":
			d, ok := locStruct.Field(i).(*array.Dictionary)
			if !ok {
				return nil, fmt.Errorf("expected location.mapping_build_id to be Dictionary, got %T", locStruct.Field(i))
			}
			vals, ok := d.Dictionary().(*array.String)
			if !ok {
				return nil, fmt.Errorf("expected location.mapping_build_id dictionary values to be String, got %T", d.Dictionary())
			}
			r.buildID = d
			r.buildIDVals = vals
		case "lines":
			lv, ok := locStruct.Field(i).(*array.ListView)
			if !ok {
				return nil, fmt.Errorf("expected location.lines to be ListView, got %T", locStruct.Field(i))
			}
			r.lines = lv
			if err := r.indexLineFields(lv); err != nil {
				return nil, err
			}
		}
	}

	if r.address == nil {
		return nil, fmt.Errorf("v2 location struct missing address field")
	}

	return r, nil
}

func (r *v2LocationReader) indexLineFields(lines *array.ListView) error {
	lineStruct, ok := lines.ListValues().(*array.Struct)
	if !ok {
		return fmt.Errorf("expected lines list values to be Struct, got %T", lines.ListValues())
	}

	lineType := lineStruct.DataType().(*arrow.StructType)
	for i := 0; i < lineType.NumFields(); i++ {
		f := lineType.Field(i)
		switch f.Name {
		case "line":
			arr, ok := lineStruct.Field(i).(*array.Uint64)
			if !ok {
				return fmt.Errorf("expected lines.line to be Uint64, got %T", lineStruct.Field(i))
			}
			r.lineNumber = arr
		case "column":
			arr, ok := lineStruct.Field(i).(*array.Uint64)
			if !ok {
				return fmt.Errorf("expected lines.column to be Uint64, got %T", lineStruct.Field(i))
			}
			r.lineColumn = arr
		case "function":
			d, ok := lineStruct.Field(i).(*array.Dictionary)
			if !ok {
				return fmt.Errorf("expected lines.function to be Dictionary, got %T", lineStruct.Field(i))
			}
			r.function = d

			funcStruct, ok := d.Dictionary().(*array.Struct)
			if !ok {
				return fmt.Errorf("expected function dictionary values to be Struct, got %T", d.Dictionary())
			}
			funcType := funcStruct.DataType().(*arrow.StructType)
			for j := 0; j < funcType.NumFields(); j++ {
				ff := funcType.Field(j)
				switch ff.Name {
				case "system_name":
					sv, ok := funcStruct.Field(j).(*array.StringView)
					if !ok {
						return fmt.Errorf("expected function.system_name to be StringView, got %T", funcStruct.Field(j))
					}
					r.systemName = sv
				case "filename":
					fd, ok := funcStruct.Field(j).(*array.Dictionary)
					if !ok {
						return fmt.Errorf("expected function.filename to be Dictionary, got %T", funcStruct.Field(j))
					}
					fvals, ok := fd.Dictionary().(*array.String)
					if !ok {
						return fmt.Errorf("expected function.filename dictionary values to be String, got %T", fd.Dictionary())
					}
					r.filename = fd
					r.filenameVal = fvals
				case "start_line":
					arr, ok := funcStruct.Field(j).(*array.Uint64)
					if !ok {
						return fmt.Errorf("expected function.start_line to be Uint64, got %T", funcStruct.Field(j))
					}
					r.startLine = arr
				}
			}
		}
	}

	return nil
}

// encodeStacktraceListViewV2 walks the v2 stacktrace ListView and emits a
// List[Dictionary[Uint32, Binary]] where each binary entry is a single location
// encoded with the same byte format profile.EncodeArrowLocation produces, so the
// rest of the storage path can treat it uniformly with v1.
func encodeStacktraceListViewV2(mem memory.Allocator, stacktrace *array.ListView) (*array.List, error) {
	locDict, ok := stacktrace.ListValues().(*array.Dictionary)
	if !ok {
		return nil, fmt.Errorf("expected stacktrace values to be Dictionary, got %T", stacktrace.ListValues())
	}
	locStruct, ok := locDict.Dictionary().(*array.Struct)
	if !ok {
		return nil, fmt.Errorf("expected stacktrace dictionary values to be Struct, got %T", locDict.Dictionary())
	}

	r, err := newV2LocationReader(locStruct)
	if err != nil {
		return nil, err
	}

	typ := arrow.ListOf(&arrow.DictionaryType{
		IndexType: arrow.PrimitiveTypes.Uint32,
		ValueType: arrow.BinaryTypes.Binary,
	})
	b := array.NewListBuilderWithField(mem, typ.ElemField())
	defer b.Release()
	lv := b.ValueBuilder().(*array.BinaryDictionaryBuilder)

	for row := 0; row < stacktrace.Len(); row++ {
		if stacktrace.IsNull(row) {
			b.Append(false)
			continue
		}
		startOff, endOff := stacktrace.ValueOffsets(row)
		if endOff == startOff {
			b.Append(false)
			continue
		}
		b.Append(true)

		for j := startOff; j < endOff; j++ {
			locIdx := locDict.GetValueIndex(int(j))
			encoded := encodeV2Location(locIdx, r)
			if err := lv.Append(encoded); err != nil {
				return nil, fmt.Errorf("append location: %w", err)
			}
		}
	}

	return b.NewListArray(), nil
}

// encodeV2Location encodes a single location at index locIdx into the same
// binary format produced by profile.EncodeArrowLocation. The format is decoded by
// profile.DecodeInto, so any change here must stay in lockstep with that decoder.
//
// Format:
//
//	uvarint(address)
//	uvarint(numLines)
//	byte hasMapping
//	if hasMapping:
//	  string(buildID), string(mappingFile)
//	  uvarint(mappingStart), uvarint(mappingLimit-mappingStart), uvarint(mappingOffset)
//	for each line:
//	  uvarint(lineNumber), uvarint(column)
//	  byte hasFunction
//	  if hasFunction: uvarint(startLine), string(name), string(systemName), string(filename)
//
// v2 has no mapping_start/limit/offset, so they are written as zero. v2 has no
// separate function name vs. system_name, so name is written as the empty string.
func encodeV2Location(locIdx int, r *v2LocationReader) []byte {
	address := r.address.Value(locIdx)

	var (
		buildID, mappingFile string
		hasMapping           bool
	)
	if r.mappingFile != nil && r.mappingFile.IsValid(locIdx) {
		mappingFile = r.mappingFileVals.Value(r.mappingFile.GetValueIndex(locIdx))
		hasMapping = true
	}
	if r.buildID != nil && r.buildID.IsValid(locIdx) {
		buildID = r.buildIDVals.Value(r.buildID.GetValueIndex(locIdx))
		hasMapping = true
	}

	var (
		lineStart, lineEnd int64
		hasLines           bool
	)
	if r.lines != nil && r.lines.IsValid(locIdx) {
		lineStart, lineEnd = r.lines.ValueOffsets(locIdx)
		hasLines = lineEnd > lineStart
	}

	size := uvarintSize(address)
	size++ // hasMapping byte
	if hasLines {
		size += uvarintSize(uint64(lineEnd - lineStart))
	} else {
		size += uvarintSize(0)
	}

	if hasMapping {
		size += sizeOfString(buildID)
		size += sizeOfString(mappingFile)
		size += uvarintSize(0) // mappingStart
		size += uvarintSize(0) // mappingLimit-mappingStart
		size += uvarintSize(0) // mappingOffset
	}

	if hasLines {
		for i := lineStart; i < lineEnd; i++ {
			size += uvarintSize(r.lineNumber.Value(int(i)))
			size += uvarintSize(r.lineColumn.Value(int(i)))
			size++ // hasFunction byte

			fIdx, ok := v2FunctionIndex(r, int(i))
			if !ok {
				continue
			}
			size += uvarintSize(r.startLine.Value(fIdx))
			size += sizeOfString("") // function name (not present in v2)
			size += sizeOfString(v2FunctionSystemName(r, fIdx))
			size += sizeOfString(v2FunctionFilename(r, fIdx))
		}
	}

	buf := make([]byte, size)
	offset := binary.PutUvarint(buf, address)
	if hasLines {
		offset += binary.PutUvarint(buf[offset:], uint64(lineEnd-lineStart))
	} else {
		offset += binary.PutUvarint(buf[offset:], 0)
	}

	if hasMapping {
		buf[offset] = 0x1
		offset++
		offset = writeStringV2(buf, offset, buildID)
		offset = writeStringV2(buf, offset, mappingFile)
		offset += binary.PutUvarint(buf[offset:], 0)
		offset += binary.PutUvarint(buf[offset:], 0)
		offset += binary.PutUvarint(buf[offset:], 0)
	} else {
		buf[offset] = 0x0
		offset++
	}

	if hasLines {
		for i := lineStart; i < lineEnd; i++ {
			offset += binary.PutUvarint(buf[offset:], r.lineNumber.Value(int(i)))
			offset += binary.PutUvarint(buf[offset:], r.lineColumn.Value(int(i)))

			fIdx, ok := v2FunctionIndex(r, int(i))
			if !ok {
				buf[offset] = 0x0
				offset++
				continue
			}
			buf[offset] = 0x1
			offset++
			offset += binary.PutUvarint(buf[offset:], r.startLine.Value(fIdx))
			offset = writeStringV2(buf, offset, "")
			offset = writeStringV2(buf, offset, v2FunctionSystemName(r, fIdx))
			offset = writeStringV2(buf, offset, v2FunctionFilename(r, fIdx))
		}
	}

	return buf[:offset]
}

func v2FunctionIndex(r *v2LocationReader, lineIdx int) (int, bool) {
	if r.function == nil || !r.function.IsValid(lineIdx) {
		return 0, false
	}
	return r.function.GetValueIndex(lineIdx), true
}

func v2FunctionSystemName(r *v2LocationReader, fIdx int) string {
	if r.systemName == nil || r.systemName.IsNull(fIdx) {
		return ""
	}
	return r.systemName.Value(fIdx)
}

func v2FunctionFilename(r *v2LocationReader, fIdx int) string {
	if r.filename == nil || !r.filename.IsValid(fIdx) {
		return ""
	}
	return r.filenameVal.Value(r.filename.GetValueIndex(fIdx))
}

func writeStringV2(buf []byte, offset int, s string) int {
	offset += binary.PutUvarint(buf[offset:], uint64(len(s)))
	copy(buf[offset:], s)
	return offset + len(s)
}

func sizeOfString(s string) int {
	return uvarintSize(uint64(len(s))) + len(s)
}

func uvarintSize(v uint64) int {
	return profile.UvarintSize(v)
}
