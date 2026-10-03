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
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"
)

// binaryDict builds a Dictionary<Uint32, Binary> whose entry at nullAt is null.
func binaryDict(t *testing.T, mem memory.Allocator, vals []string, nullAt int) (*array.Dictionary, *array.Binary) {
	t.Helper()
	b := array.NewBuilder(mem, &arrow.DictionaryType{
		IndexType: arrow.PrimitiveTypes.Uint32,
		ValueType: arrow.BinaryTypes.Binary,
	}).(*array.BinaryDictionaryBuilder)
	for i, v := range vals {
		if i == nullAt {
			b.AppendNull()
			continue
		}
		require.NoError(t, b.AppendString(v))
	}
	a := b.NewArray().(*array.Dictionary)
	return a, a.Dictionary().(*array.Binary)
}

// EncodeArrowLocation must write exactly what serializedArrowLocationSize
// budgeted.
//
// The writer emitted the hasFunction flag as 0x1 and the function block
// unconditionally, while the sizer budgets that block only when the line's
// function name is valid. For a line whose function name is null the sizer
// allowed 10 bytes and the writer ran past the end of them, panicking in
// writeInt64AsUvarint on the start line.
//
// A line with a number but no function is a representable shape: DecodeInto
// emits precisely that from a pprof Line with FunctionId 0, and the decoders
// already round trip a 0x0 flag.
func TestEncodeArrowLocationMatchesItsBudget(t *testing.T) {
	t.Parallel()
	mem := memory.DefaultAllocator

	lineNumbers := array.NewInt64Builder(mem)
	lineNumbers.Append(42)
	lineNumbers.Append(43)
	lineNumber := lineNumbers.NewArray().(*array.Int64)

	columns := array.NewUint64Builder(mem)
	columns.Append(0)
	columns.Append(7)
	lineColumnNumber := columns.NewArray().(*array.Uint64)

	// Index 0 carries a function, index 1 does not. The dictionary is
	// non-empty either way, so a lookup at index 1 would succeed and the only
	// thing that goes wrong is the length of what gets written.
	fnName, fnNameDict := binaryDict(t, mem, []string{"main.main", "unused"}, 1)
	fnSys, fnSysDict := binaryDict(t, mem, []string{"main.main", "unused"}, 1)

	reeBuilder := array.NewBuilder(mem, arrow.RunEndEncodedOf(
		arrow.PrimitiveTypes.Int32,
		&arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
	)).(*array.RunEndEncodedBuilder)
	reeBuilder.Append(2)
	require.NoError(t, reeBuilder.ValueBuilder().(*array.BinaryDictionaryBuilder).AppendString("/x/main.go"))
	fnFile := reeBuilder.NewArray().(*array.RunEndEncoded)
	fnFileDict := fnFile.Values().(*array.Dictionary)
	fnFileDictValues := fnFileDict.Dictionary().(*array.Binary)

	startLines := array.NewInt64Builder(mem)
	startLines.Append(10)
	startLines.Append(10)
	lineFunctionStartLine := startLines.NewArray().(*array.Int64)

	for _, tc := range []struct {
		name     string
		from, to int
	}{
		{"with a function name", 0, 1},
		{"with a null function name", 1, 2},
		{"both lines", 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			budget := serializedArrowLocationSize(
				0xdeadbeef, false, 0, 0, 0, nil, nil, tc.from, tc.to, nil, nil,
				lineNumber, lineColumnNumber, fnName, fnNameDict, fnSys, fnSysDict,
				fnFile, fnFileDict, fnFileDictValues, lineFunctionStartLine,
			)

			var buf []byte
			require.NotPanics(t, func() {
				buf = EncodeArrowLocation(
					0xdeadbeef, false, 0, 0, 0, nil, nil, tc.from, tc.to, nil, nil,
					lineNumber, lineColumnNumber, fnName, fnNameDict, fnSys, fnSysDict,
					fnFile, fnFileDict, fnFileDictValues, lineFunctionStartLine,
				)
			})

			// The buffer is allocated at the budget, so a writer that wrote
			// fewer bytes than budgeted would leave trailing zeroes rather
			// than fail. Decoding is what proves the bytes are well-formed.
			require.Len(t, buf, budget)

			lw := NewLocationsWriter(mem)
			_, err := DecodeInto(lw, buf, nil)
			require.NoError(t, err)
		})
	}
}
