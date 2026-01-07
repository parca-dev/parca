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
	"github.com/dennwc/varint"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func decodeLines(data []byte) []profile.LocationLine {
	offset := 0
	numberOfLines, n := varint.Uvarint(data[offset:])
	offset += n

	res := make([]profile.LocationLine, 0, numberOfLines)

	if numberOfLines > 0 {
		for i := uint64(0); i < numberOfLines; i++ {
			line, n := varint.Uvarint(data[offset:])
			offset += n

			hasFunction := data[offset] == 0x1
			offset++

			var f *pb.Function
			if hasFunction {
				startLine, n := varint.Uvarint(data[offset:])
				offset += n

				name, n := decodeString(data[offset:])
				offset += n

				systemName, n := decodeString(data[offset:])
				offset += n

				filename, n := decodeString(data[offset:])
				offset += n

				f = &pb.Function{
					StartLine:  int64(startLine),
					Name:       string(name),
					SystemName: string(systemName),
					Filename:   string(filename),
				}
			}

			res = append(res, profile.LocationLine{
				Line:     int64(line),
				Function: f,
			})
		}
	}

	return res
}

func decodeString(data []byte) ([]byte, int) {
	length, n := varint.Uvarint(data)
	return data[n : n+int(length)], n + int(length)
}
