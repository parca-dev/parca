// Copyright 2024 The Parca Authors
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

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func EncodePprofLocation(l *pprofpb.Location, m *pprofpb.Mapping, funcs []*pprofpb.Function, stringTable []string) []byte {
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

func serializedPprofLocationSize(l *pprofpb.Location, m *pprofpb.Mapping, funcs []*pprofpb.Function, stringTable []string) int {
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
