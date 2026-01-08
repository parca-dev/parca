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

package symbolizer

import (
	"encoding/binary"

	"github.com/parca-dev/parca/pkg/profile"
)

func encodeLines(lines []profile.LocationLine) []byte {
	size := encodeLinesSize(lines)
	buf := make([]byte, size)

	offset := 0
	offset = writeIntAsUvarint(buf, offset, len(lines))

	for _, line := range lines {
		offset = writeInt64AsUvarint(buf, offset, line.Line)

		if line.Function != nil {
			buf[offset] = 0x1
			offset++

			offset = writeInt64AsUvarint(buf, offset, line.Function.StartLine)
			offset = writeString(buf, offset, line.Function.Name)
			offset = writeString(buf, offset, line.Function.SystemName)
			offset = writeString(buf, offset, line.Function.Filename)
		} else {
			buf[offset] = 0x0
			offset++
		}
	}

	return buf
}

func writeIntAsUvarint(buf []byte, offset, i int) int {
	n := binary.PutUvarint(buf[offset:], uint64(i))
	offset += n

	return offset
}

func writeString(buf []byte, offset int, s string) int {
	n := binary.PutUvarint(buf[offset:], uint64(len(s)))
	offset += n

	copy(buf[offset:], s)
	offset += len(s)

	return offset
}

func writeInt64AsUvarint(buf []byte, offset int, i int64) int {
	n := binary.PutUvarint(buf[offset:], uint64(i))
	offset += n

	return offset
}

func encodeLinesSize(lines []profile.LocationLine) int {
	size := addSerializedIntAsUvarintSize(0, len(lines))
	for _, l := range lines {
		size = addSerializedInt64AsUvarintSize(size, l.Line)

		size++ // 1 byte for whether there is a function
		if l.Function != nil {
			size = addSerializedInt64AsUvarintSize(size, l.Function.StartLine)
			size = addSerializedStringSize(size, l.Function.Name)
			size = addSerializedStringSize(size, l.Function.SystemName)
			size = addSerializedStringSize(size, l.Function.Filename)
		}
	}

	return size
}

func addSerializedInt64AsUvarintSize(size int, i int64) int {
	return size + profile.UvarintSize(uint64(i))
}

func addSerializedStringSize(size int, s string) int {
	return size + profile.UvarintSize(uint64(len(s))) + len(s)
}

func addSerializedIntAsUvarintSize(size, i int) int {
	return size + profile.UvarintSize(uint64(i))
}
