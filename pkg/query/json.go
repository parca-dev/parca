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

package query

import (
	"math/bits"
	"reflect"
	"unsafe"
)

func MarshalStringMap(buf []byte, m map[string]string) []byte {
	buf = buf[:0]
	buf = append(buf, '{')

	for k, v := range m {
		buf = appendString(buf, k)
		buf = append(buf, ':')
		buf = appendString(buf, v)
		buf = append(buf, ',')
	}

	buf[len(buf)-1] = '}'
	return buf
}

const (
	lsb = 0x0101010101010101
	msb = 0x8080808080808080
)

var hex = "0123456789abcdef"

var needEscape = [256]bool{
	'"':  true,
	'\\': true,
	0x00: true,
	0x01: true,
	0x02: true,
	0x03: true,
	0x04: true,
	0x05: true,
	0x06: true,
	0x07: true,
	0x08: true,
	0x09: true,
	0x0a: true,
	0x0b: true,
	0x0c: true,
	0x0d: true,
	0x0e: true,
	0x0f: true,
	0x10: true,
	0x11: true,
	0x12: true,
	0x13: true,
	0x14: true,
	0x15: true,
	0x16: true,
	0x17: true,
	0x18: true,
	0x19: true,
	0x1a: true,
	0x1b: true,
	0x1c: true,
	0x1d: true,
	0x1e: true,
	0x1f: true,
	/* 0x20 - 0xff */
}

// Taken from goccy/go-json.
func appendString(buf []byte, s string) []byte {
	valLen := len(s)
	if valLen == 0 {
		return append(buf, `""`...)
	}
	buf = append(buf, '"')
	var i, j int
	if valLen >= 8 {
		chunks := stringToUint64Slice(s)
		for _, n := range chunks {
			// combine masks before checking for the MSB of each byte. We include
			// `n` in the mask to check whether any of the *input* byte MSBs were
			// set (i.e. the byte was outside the ASCII range).
			mask := n | (n - (lsb * 0x20)) |
				((n ^ (lsb * '"')) - lsb) |
				((n ^ (lsb * '\\')) - lsb)
			if (mask & msb) != 0 {
				j = bits.TrailingZeros64(mask&msb) / 8
				goto ESCAPE_END
			}
		}
		valLen := len(s)
		for i := len(chunks) * 8; i < valLen; i++ {
			if needEscape[s[i]] {
				j = i
				goto ESCAPE_END
			}
		}
		return append(append(buf, s...), '"')
	}
ESCAPE_END:
	for j < valLen {
		c := s[j]

		if !needEscape[c] {
			// fast path: most of the time, printable ascii characters are used
			j++
			continue
		}

		switch c {
		case '\\', '"':
			buf = append(buf, s[i:j]...)
			buf = append(buf, '\\', c)
			i = j + 1
			j = j + 1
			continue

		case '\n':
			buf = append(buf, s[i:j]...)
			buf = append(buf, '\\', 'n')
			i = j + 1
			j = j + 1
			continue

		case '\r':
			buf = append(buf, s[i:j]...)
			buf = append(buf, '\\', 'r')
			i = j + 1
			j = j + 1
			continue

		case '\t':
			buf = append(buf, s[i:j]...)
			buf = append(buf, '\\', 't')
			i = j + 1
			j = j + 1
			continue

		case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x0B, 0x0C, 0x0E, 0x0F, // 0x00-0x0F
			0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1A, 0x1B, 0x1C, 0x1D, 0x1E, 0x1F: // 0x10-0x1F
			buf = append(buf, s[i:j]...)
			buf = append(buf, `\u00`...)
			buf = append(buf, hex[c>>4], hex[c&0xF])
			i = j + 1
			j = j + 1
			continue
		}
		j++
	}

	return append(append(buf, s[i:]...), '"')
}

func stringToUint64Slice(s string) []uint64 {
	return *(*[]uint64)(unsafe.Pointer(&reflect.SliceHeader{ //nolint:govet
		Data: ((*reflect.StringHeader)(unsafe.Pointer(&s))).Data,
		Len:  len(s) / 8,
		Cap:  len(s) / 8,
	}))
}
