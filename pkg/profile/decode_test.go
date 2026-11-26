// Copyright 2026 The Parca Authors
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
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

// A truncated location must return an error, not panic.
//
// DecodeInto parses bytes read back from storage on the query path
// (pkg/parcacol), and there is no recover() anywhere in the server, so an
// unchecked index here takes the process down on a corrupt or truncated
// location rather than failing the query that touched it.
//
// Every prefix of a real encoded record is fed in, so this covers running out
// mid-varint, mid-string and exactly on a boundary, at every field in the
// layout rather than at a hand-picked few. Each prefix is sliced encoded[:i:i]
// because a two-index slice drops the original cap, and Go does not panic
// reading past len while still inside cap -- the cheaper spelling silently
// passes on inputs that really do over-read. That is not academic: these bytes
// usually arrive as an Arrow value, and array.Binary.Value slices with cap
// running to the end of the whole buffer, so an over-read returns the next
// location's bytes instead of faulting.
func TestDecodeIntoSurvivesTruncation(t *testing.T) {
	t.Parallel()

	stringTable := []string{"", "main.main", "main.main", "/x/main.go", "build-id", "/bin/svc"}
	funcs := []*pprofpb.Function{{Id: 1, Name: 1, SystemName: 2, Filename: 3, StartLine: 10}}
	mapping := &pprofpb.Mapping{
		Id: 1, BuildId: 4, Filename: 5,
		MemoryStart: 0x1000, MemoryLimit: 0x2000, FileOffset: 8,
	}
	withFunc := []*pprofpb.Line{{FunctionId: 1, Line: 42}}

	for _, tc := range []struct {
		name    string
		mapping *pprofpb.Mapping
		lines   []*pprofpb.Line
	}{
		{"mapping and function", mapping, withFunc},
		{"no mapping", nil, withFunc},
		{"no function", mapping, []*pprofpb.Line{{FunctionId: 0, Line: 42}}},
		{"no lines", mapping, nil},
		{"several lines", mapping, []*pprofpb.Line{
			{FunctionId: 1, Line: 42}, {FunctionId: 1, Line: 43},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			loc := &pprofpb.Location{Id: 1, Address: 0xdeadbeef, Line: tc.lines}
			if tc.mapping != nil {
				loc.MappingId = tc.mapping.Id
			}
			encoded := EncodePprofLocation(loc, tc.mapping, funcs, stringTable)
			require.NotEmpty(t, encoded)

			for i := 0; i <= len(encoded); i++ {
				require.NotPanicsf(t, func() {
					lw := NewLocationsWriter(memory.DefaultAllocator)
					_, _ = DecodeInto(lw, encoded[:i:i], nil)
				}, "panicked on the first %d of %d bytes", i, len(encoded))
			}

			// A short record is reported, not swallowed.
			if len(encoded) > 1 {
				lw := NewLocationsWriter(memory.DefaultAllocator)
				_, err := DecodeInto(lw, encoded[:len(encoded)-1:len(encoded)-1], nil)
				require.Error(t, err, "a truncated record must be reported")
			}

			// The whole record still decodes, so the bounds checks did not cost
			// a field.
			lw := NewLocationsWriter(memory.DefaultAllocator)
			res, err := DecodeInto(lw, encoded, nil)
			require.NoError(t, err)
			require.Equal(t, len(tc.lines) > 0, res.WroteLines)
		})
	}
}

// A length prefix larger than the record must be refused, not panic. The length
// is unsigned: one above MaxInt converts to a negative int, so
// offset+int(length) lands below offset, satisfying a naive upper-bound check
// before panicking on a slice whose high bound is below its low one.
func TestDecodeIntoRejectsOversizedLength(t *testing.T) {
	t.Parallel()

	for _, length := range []uint64{1 << 20, 1 << 63, ^uint64(0)} {
		// A location with a mapping, whose build ID claims more bytes than the
		// record holds.
		var buf []byte
		buf = binary.AppendUvarint(buf, 0xdeadbeef) // address
		buf = binary.AppendUvarint(buf, 1)          // numLines
		buf = append(buf, 0x1)                      // hasMapping = true
		buf = binary.AppendUvarint(buf, length)     // build ID length
		buf = append(buf, []byte("short")...)       // fewer bytes than claimed

		require.NotPanics(t, func() {
			lw := NewLocationsWriter(memory.DefaultAllocator)
			_, err := DecodeInto(lw, buf, nil)
			require.Error(t, err)
		})
	}
}
