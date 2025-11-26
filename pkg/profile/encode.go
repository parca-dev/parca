// Copyright 2024-2025 The Parca Authors
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
	"encoding/binary"

	"github.com/apache/arrow-go/v18/arrow/array"
	semconv "go.opentelemetry.io/otel/semconv/v1.28.0"
	pprofextended "go.opentelemetry.io/proto/otlp/profiles/v1development"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func EncodeOtelLocation(
	attributeTable []*pprofextended.KeyValueAndUnit,
	l *pprofextended.Location,
	m *pprofextended.Mapping,
	funcs []*pprofextended.Function,
	stringTable []string,
) []byte {
	buf := make([]byte, serializedOtelLocationSize(attributeTable, l, m, funcs, stringTable))
	offset := binary.PutUvarint(buf, l.Address)
	offset = writeIntAsUvarint(buf, offset, len(l.Line))
	if m == nil {
		buf[offset] = 0x0
		offset++
	} else {
		buf[offset] = 0x1
		offset++

		buildID := ""
		for _, idx := range m.AttributeIndices {
			if stringTable[attributeTable[idx].KeyStrindex] == string(semconv.ProcessExecutableBuildIDGnuKey) {
				buildID = attributeTable[idx].Value.GetStringValue()
				break
			}
		}
		offset = writeString(buf, offset, buildID)

		filename := ""
		if m.FilenameStrindex != 0 {
			filename = stringTable[m.FilenameStrindex]
		}
		offset = writeString(buf, offset, filename)
		offset = writeUint64(buf, offset, m.MemoryStart)
		offset = writeUint64(buf, offset, m.MemoryLimit-m.MemoryStart)
		offset = writeUint64(buf, offset, m.FileOffset)
	}

	for _, line := range l.Line {
		offset = writeInt64AsUvarint(buf, offset, line.Line)
		offset = writeUint64(buf, offset, uint64(line.Column))

		if line.FunctionIndex != 0 {
			buf[offset] = 0x1
			offset++

			f := funcs[line.FunctionIndex-1]
			offset = writeInt64AsUvarint(buf, offset, f.StartLine)

			name := ""
			if f.NameStrindex != 0 {
				name = stringTable[f.NameStrindex]
			}
			offset = writeString(buf, offset, name)

			systemName := ""
			if f.SystemNameStrindex != 0 {
				systemName = stringTable[f.SystemNameStrindex]
			}
			offset = writeString(buf, offset, systemName)

			filename := ""
			if f.FilenameStrindex != 0 {
				filename = stringTable[f.FilenameStrindex]
			}
			offset = writeString(buf, offset, filename)
		} else {
			buf[offset] = 0x0
			offset++
		}
	}

	return buf
}

func serializedOtelLocationSize(
	attributeTable []*pprofextended.KeyValueAndUnit,
	l *pprofextended.Location,
	m *pprofextended.Mapping,
	funcs []*pprofextended.Function,
	stringTable []string,
) int {
	size := UvarintSize(l.Address)
	size++ // 1 byte for whether there is a mapping

	size = addSerializedIntAsUvarintSize(size, len(l.Line))

	if m != nil {
		buildID := ""
		for _, idx := range m.AttributeIndices {
			if stringTable[attributeTable[idx].KeyStrindex] == string(semconv.ProcessExecutableBuildIDGnuKey) {
				buildID = attributeTable[idx].Value.GetStringValue()
				break
			}
		}
		size = addSerializedStringSize(size, buildID)

		filename := ""
		if m.FilenameStrindex != 0 {
			filename = stringTable[m.FilenameStrindex]
		}
		size = addSerializedStringSize(size, filename)
		size = addSerializedUint64Size(size, m.MemoryStart)
		size = addSerializedUint64Size(size, m.MemoryLimit-m.MemoryStart)
		size = addSerializedUint64Size(size, m.FileOffset)
	}

	for _, line := range l.Line {
		size = addSerializedInt64AsUvarintSize(size, line.Line)
		size = addSerializedUint64Size(size, uint64(line.Column))

		size++ // 1 byte for whether there is a function
		if line.FunctionIndex != 0 {
			f := funcs[line.FunctionIndex-1]
			size = addSerializedInt64AsUvarintSize(size, f.StartLine)

			name := ""
			if f.NameStrindex != 0 {
				name = stringTable[f.NameStrindex]
			}
			size = addSerializedStringSize(size, name)

			systemName := ""
			if f.SystemNameStrindex != 0 {
				systemName = stringTable[f.SystemNameStrindex]
			}
			size = addSerializedStringSize(size, systemName)

			filename := ""
			if f.FilenameStrindex != 0 {
				filename = stringTable[f.FilenameStrindex]
			}
			size = addSerializedStringSize(size, filename)
		}
	}

	return size
}

func EncodePprofLocation(
	l *pprofpb.Location,
	m *pprofpb.Mapping,
	funcs []*pprofpb.Function,
	stringTable []string,
) []byte {
	buf := make([]byte, serializedPprofLocationSize(l, m, funcs, stringTable))
	offset := binary.PutUvarint(buf, l.Address)
	offset = writeIntAsUvarint(buf, offset, len(l.Line))
	if m == nil {
		buf[offset] = 0x0
		offset++
	} else {
		buf[offset] = 0x1
		offset++

		buildID := ""
		if m.BuildId != 0 {
			buildID = stringTable[m.BuildId]
		}
		offset = writeString(buf, offset, buildID)

		filename := ""
		if m.Filename != 0 {
			filename = stringTable[m.Filename]
		}
		offset = writeString(buf, offset, filename)
		offset = writeUint64(buf, offset, m.MemoryStart)
		offset = writeUint64(buf, offset, m.MemoryLimit-m.MemoryStart)
		offset = writeUint64(buf, offset, m.FileOffset)
	}

	for _, line := range l.Line {
		offset = writeInt64AsUvarint(buf, offset, line.Line)
		offset = writeUint64(buf, offset, 0) // pprof doesn't have column info

		if line.FunctionId != 0 {
			buf[offset] = 0x1
			offset++

			f := funcs[line.FunctionId-1]
			offset = writeInt64AsUvarint(buf, offset, f.StartLine)

			name := ""
			if f.Name != 0 {
				name = stringTable[f.Name]
			}
			offset = writeString(buf, offset, name)

			systemName := ""
			if f.SystemName != 0 {
				systemName = stringTable[f.SystemName]
			}
			offset = writeString(buf, offset, systemName)

			filename := ""
			if f.Filename != 0 {
				filename = stringTable[f.Filename]
			}
			offset = writeString(buf, offset, filename)
		} else {
			buf[offset] = 0x0
			offset++
		}
	}

	return buf
}

func serializedPprofLocationSize(
	l *pprofpb.Location,
	m *pprofpb.Mapping,
	funcs []*pprofpb.Function,
	stringTable []string,
) int {
	size := UvarintSize(l.Address)
	size++ // 1 byte for whether there is a mapping

	size = addSerializedIntAsUvarintSize(size, len(l.Line))

	if m != nil {
		buildID := ""
		if m.BuildId != 0 {
			buildID = stringTable[m.BuildId]
		}
		size = addSerializedStringSize(size, buildID)

		filename := ""
		if m.Filename != 0 {
			filename = stringTable[m.Filename]
		}
		size = addSerializedStringSize(size, filename)
		size = addSerializedUint64Size(size, m.MemoryStart)
		size = addSerializedUint64Size(size, m.MemoryLimit-m.MemoryStart)
		size = addSerializedUint64Size(size, m.FileOffset)
	}

	for _, line := range l.Line {
		size = addSerializedInt64AsUvarintSize(size, line.Line)
		size = addSerializedUint64Size(size, 0) // pprof doesn't have column info

		size++ // 1 byte for whether there is a function
		if line.FunctionId != 0 {
			f := funcs[line.FunctionId-1]
			size = addSerializedInt64AsUvarintSize(size, f.StartLine)

			name := ""
			if f.Name != 0 {
				name = stringTable[f.Name]
			}
			size = addSerializedStringSize(size, name)

			systemName := ""
			if f.SystemName != 0 {
				systemName = stringTable[f.SystemName]
			}
			size = addSerializedStringSize(size, systemName)

			filename := ""
			if f.Filename != 0 {
				filename = stringTable[f.Filename]
			}
			size = addSerializedStringSize(size, filename)
		}
	}

	return size
}

func EncodeArrowLocation(
	address uint64,
	hasMapping bool,
	mappingStart uint64,
	mappingLimit uint64,
	mappingOffset uint64,
	mappingFile []byte,
	buildID []byte,
	linesStartOffset int,
	linesEndOffset int,
	_ *array.List,
	_ *array.Struct,
	lineNumber *array.Int64,
	columnNumber *array.Uint64,
	lineFunctionName *array.Dictionary,
	lineFunctionNameDict *array.Binary,
	lineFunctionSystemName *array.Dictionary,
	lineFunctionSystemNameDict *array.Binary,
	lineFunctionFilename *array.RunEndEncoded,
	lineFunctionFilenameDict *array.Dictionary,
	lineFunctionFilenameDictValues *array.Binary,
	lineFunctionStartLine *array.Int64,
) []byte {
	buf := make([]byte, serializedArrowLocationSize(
		address,
		hasMapping,
		mappingStart,
		mappingLimit,
		mappingOffset,
		mappingFile,
		buildID,
		linesStartOffset,
		linesEndOffset,
		nil,
		nil,
		lineNumber,
		columnNumber,
		lineFunctionName,
		lineFunctionNameDict,
		lineFunctionSystemName,
		lineFunctionSystemNameDict,
		lineFunctionFilename,
		lineFunctionFilenameDict,
		lineFunctionFilenameDictValues,
		lineFunctionStartLine,
	))
	offset := binary.PutUvarint(buf, address)
	offset = writeIntAsUvarint(buf, offset, linesEndOffset-linesStartOffset)
	if hasMapping {
		buf[offset] = 0x1
		offset++

		offset = writeString(buf, offset, string(buildID))
		offset = writeString(buf, offset, string(mappingFile))
		offset = writeUint64(buf, offset, mappingStart)
		offset = writeUint64(buf, offset, mappingLimit-mappingStart)
		offset = writeUint64(buf, offset, mappingOffset)
	} else {
		buf[offset] = 0x0
		offset++
	}

	for i := linesStartOffset; i < linesEndOffset; i++ {
		offset = writeInt64AsUvarint(buf, offset, lineNumber.Value(i))
		offset = writeUint64(buf, offset, columnNumber.Value(i))

		buf[offset] = 0x1
		offset++

		offset = writeInt64AsUvarint(buf, offset, lineFunctionStartLine.Value(i))
		offset = writeString(buf, offset, string(lineFunctionNameDict.Value(int(lineFunctionName.GetValueIndex(i)))))
		offset = writeString(buf, offset, string(lineFunctionSystemNameDict.Value(int(lineFunctionSystemName.GetValueIndex(i)))))

		if lineFunctionFilenameDict.IsValid(lineFunctionFilename.GetPhysicalIndex(i)) {
			offset = writeString(buf, offset, string(lineFunctionFilenameDictValues.Value(int(lineFunctionFilenameDict.GetValueIndex(lineFunctionFilename.GetPhysicalIndex(i))))))
		} else {
			offset = writeString(buf, offset, "")
		}
	}

	return buf
}

func serializedArrowLocationSize(
	address uint64,
	hasMapping bool,
	mappingStart uint64,
	mappingLimit uint64,
	mappingOffset uint64,
	mappingFile []byte,
	buildID []byte,
	linesStartOffset int,
	linesEndOffset int,
	_ *array.List,
	_ *array.Struct,
	lineNumber *array.Int64,
	columnNumber *array.Uint64,
	lineFunctionName *array.Dictionary,
	lineFunctionNameDict *array.Binary,
	lineFunctionSystemName *array.Dictionary,
	lineFunctionSystemNameDict *array.Binary,
	lineFunctionFilename *array.RunEndEncoded,
	lineFunctionFilenameDict *array.Dictionary,
	lineFunctionFilenameDictValues *array.Binary,
	lineFunctionStartLine *array.Int64,
) int {
	size := UvarintSize(address)
	size++ // 1 byte for whether there is a mapping

	size = addSerializedIntAsUvarintSize(size, linesEndOffset-linesStartOffset)

	if hasMapping {
		size = addSerializedStringSize(size, string(buildID))
		size = addSerializedStringSize(size, string(mappingFile))
		size = addSerializedUint64Size(size, mappingStart)
		size = addSerializedUint64Size(size, mappingLimit-mappingStart)
		size = addSerializedUint64Size(size, mappingOffset)
	}

	for i := linesStartOffset; i < linesEndOffset; i++ {
		size = addSerializedInt64AsUvarintSize(size, lineNumber.Value(i))
		size = addSerializedUint64Size(size, columnNumber.Value(i))

		size++ // 1 byte for whether there is a function
		if lineFunctionName.IsValid(i) {
			size = addSerializedInt64AsUvarintSize(size, lineFunctionStartLine.Value(i))
			size = addSerializedStringSize(size, string(lineFunctionNameDict.Value(int(lineFunctionName.GetValueIndex(i)))))
			size = addSerializedStringSize(size, string(lineFunctionSystemNameDict.Value(int(lineFunctionSystemName.GetValueIndex(i)))))

			if lineFunctionFilenameDict.IsValid(lineFunctionFilename.GetPhysicalIndex(i)) {
				size = addSerializedStringSize(size, string(lineFunctionFilenameDictValues.Value(int(lineFunctionFilenameDict.GetValueIndex(lineFunctionFilename.GetPhysicalIndex(i))))))
			} else {
				size = addSerializedStringSize(size, "")
			}
		}
	}

	return size
}

func writeString(buf []byte, offset int, s string) int {
	n := binary.PutUvarint(buf[offset:], uint64(len(s)))
	offset += n

	copy(buf[offset:], s)
	offset += len(s)

	return offset
}

func writeIntAsUvarint(buf []byte, offset, i int) int {
	n := binary.PutUvarint(buf[offset:], uint64(i))
	offset += n

	return offset
}

func writeInt64AsUvarint(buf []byte, offset int, i int64) int {
	n := binary.PutUvarint(buf[offset:], uint64(i))
	offset += n

	return offset
}

func writeUint64(buf []byte, offset int, i uint64) int {
	n := binary.PutUvarint(buf[offset:], i)
	offset += n

	return offset
}

func addSerializedStringSize(size int, s string) int {
	return size + UvarintSize(uint64(len(s))) + len(s)
}

func addSerializedIntAsUvarintSize(size, i int) int {
	return size + UvarintSize(uint64(i))
}

func addSerializedInt64AsUvarintSize(size int, i int64) int {
	return size + UvarintSize(uint64(i))
}

func addSerializedUint64Size(size int, i uint64) int {
	return addUvarintSize(size, i)
}

func addUvarintSize(size int, i uint64) int {
	return size + UvarintSize(i)
}
