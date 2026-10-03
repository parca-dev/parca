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

package profile

import (
	"fmt"

	"github.com/dennwc/varint"
)

type Demangler interface {
	Demangle(name []byte) string
}

type SymbolizationInfo struct {
	Addr    uint64
	BuildID []byte
	Mapping Mapping
}

func DecodeSymbolizationInfo(data []byte) (SymbolizationInfo, uint64) {
	offset := 0
	addr, n := varint.Uvarint(data) // we need to know the address size to read the build ID
	offset += n

	numberOfLines, n := varint.Uvarint(data[offset:])
	offset += n

	hasMapping := data[offset] == 0x1
	offset++

	if hasMapping {
		buildID, n := decodeString(data[offset:])
		offset += n

		file, n := decodeString(data[offset:])
		offset += n

		memoryStart, n := varint.Uvarint(data[offset:])
		offset += n

		memoryLength, n := varint.Uvarint(data[offset:])
		offset += n

		mappingOffset, _ := varint.Uvarint(data[offset:])

		return SymbolizationInfo{
			Addr:    addr,
			BuildID: buildID,
			Mapping: Mapping{
				StartAddr: memoryStart,
				EndAddr:   memoryStart + memoryLength,
				Offset:    mappingOffset,
				File:      string(file),
			},
		}, numberOfLines
	}

	return SymbolizationInfo{
		Addr: addr,
	}, numberOfLines
}

type DecodeResult struct {
	WroteLines bool
	BuildID    []byte
	Addr       uint64
	Mapping    Mapping
}

// DecodeInto decodes an encoded location into lw.
//
// Every read is bounds-checked and a record that runs out returns an error
// rather than panicking. This parses bytes read back from storage on the query
// path, and there is no recover() anywhere in the server, so an unchecked index
// here would take the process down on a corrupt or truncated location instead
// of failing the query that touched it.
//
// Reading past the end is not even reliably a crash: these bytes usually arrive
// as an Arrow value, and array.Binary.Value slices with cap running to the end
// of the whole buffer, so an over-read can silently return the next location's
// bytes and decode them as this location's own.
func DecodeInto(lw LocationsWriter, data []byte, demangler Demangler) (DecodeResult, error) {
	var (
		buildID       []byte
		memoryStart   uint64
		memoryLength  uint64
		mappingOffset uint64
	)

	offset := 0

	// uvarint reports false when the record has run out or the varint is
	// malformed; Uvarint returns n <= 0 for both.
	uvarint := func() (uint64, bool) {
		if offset >= len(data) {
			return 0, false
		}
		v, n := varint.Uvarint(data[offset:])
		if n <= 0 {
			return 0, false
		}
		offset += n
		return v, true
	}
	// str reads a length-prefixed string. The length is unsigned, so one above
	// MaxInt converts to a negative int and offset+int(length) lands BELOW
	// offset -- which slips past a naive upper-bound check straight into a
	// panicking slice. Compare in the space the length was read in.
	str := func() ([]byte, bool) {
		length, ok := uvarint()
		if !ok || length > uint64(len(data)-offset) {
			return nil, false
		}
		v := data[offset : offset+int(length)]
		offset += int(length)
		return v, true
	}
	flag := func() (bool, bool) {
		if offset >= len(data) {
			return false, false
		}
		v := data[offset] == 0x1
		offset++
		return v, true
	}
	truncated := func(field string) error {
		return fmt.Errorf("malformed location: ran out of bytes reading %s at offset %d of %d", field, offset, len(data))
	}

	addr, ok := uvarint()
	if !ok {
		return DecodeResult{}, truncated("address")
	}

	lineNumber, ok := uvarint()
	if !ok {
		return DecodeResult{}, truncated("number of lines")
	}

	hasMapping, ok := flag()
	if !ok {
		return DecodeResult{}, truncated("has-mapping flag")
	}
	if hasMapping {
		buildID, ok = str()
		if !ok {
			return DecodeResult{}, truncated("mapping build ID")
		}

		if err := lw.MappingBuildID.Append(buildID); err != nil {
			return DecodeResult{}, fmt.Errorf("append mapping build id: %w", err)
		}

		filename, ok := str()
		if !ok {
			return DecodeResult{}, truncated("mapping filename")
		}

		if err := lw.MappingFile.Append(filename); err != nil {
			return DecodeResult{}, fmt.Errorf("append mapping filename: %w", err)
		}

		memoryStart, ok = uvarint()
		if !ok {
			return DecodeResult{}, truncated("mapping memory start")
		}

		lw.MappingStart.Append(memoryStart)

		memoryLength, ok = uvarint()
		if !ok {
			return DecodeResult{}, truncated("mapping memory length")
		}

		lw.MappingLimit.Append(memoryStart + memoryLength)

		mappingOffset, ok = uvarint()
		if !ok {
			return DecodeResult{}, truncated("mapping offset")
		}

		lw.MappingOffset.Append(mappingOffset)
	} else {
		lw.MappingStart.AppendNull()
		lw.MappingLimit.AppendNull()
		lw.MappingOffset.AppendNull()
		lw.MappingFile.AppendNull()
		lw.MappingBuildID.AppendNull()
	}

	if lineNumber > 0 {
		lw.Lines.Append(true)

		for i := uint64(0); i < lineNumber; i++ {
			lw.Line.Append(true)

			line, ok := uvarint()
			if !ok {
				return DecodeResult{}, truncated("line number")
			}

			lw.LineNumber.Append(int64(line))

			column, ok := uvarint()
			if !ok {
				return DecodeResult{}, truncated("column")
			}
			lw.ColumnNumber.Append(column)

			hasFunction, ok := flag()
			if !ok {
				return DecodeResult{}, truncated("has-function flag")
			}

			if hasFunction {
				startLine, ok := uvarint()
				if !ok {
					return DecodeResult{}, truncated("function start line")
				}

				lw.FunctionStartLine.Append(int64(startLine))

				name, ok := str()
				if !ok {
					return DecodeResult{}, truncated("function name")
				}

				systemName, ok := str()
				if !ok {
					return DecodeResult{}, truncated("function system name")
				}

				// Data written by the v2 ingest path before it populated the
				// name only carries system_name.
				if len(name) == 0 {
					name = systemName
				}

				if demangler != nil {
					name = []byte(demangler.Demangle(name))
				}

				if err := lw.FunctionName.Append([]byte(name)); err != nil {
					return DecodeResult{}, fmt.Errorf("append function name: %w", err)
				}

				if err := lw.FunctionSystemName.Append(systemName); err != nil {
					return DecodeResult{}, fmt.Errorf("append function system name: %w", err)
				}

				filename, ok := str()
				if !ok {
					return DecodeResult{}, truncated("function filename")
				}

				if err := lw.FunctionFilename.Append(filename); err != nil {
					return DecodeResult{}, fmt.Errorf("append function filename: %w", err)
				}
			} else {
				lw.FunctionStartLine.AppendNull()
				lw.FunctionName.AppendNull()
				lw.FunctionSystemName.AppendNull()
				lw.FunctionFilename.AppendNull()
			}
		}

		return DecodeResult{
			WroteLines: true,
		}, nil
	}

	return DecodeResult{
		WroteLines: false,
		BuildID:    buildID,
		Addr:       addr,
		Mapping: Mapping{
			StartAddr: memoryStart,
			EndAddr:   memoryStart + memoryLength,
			Offset:    mappingOffset,
		},
	}, nil
}

// DecodeFunctionName is a fork of DecodeInto that only tries to find a function name and returns it.
// It returns "" if no function name is found.
func DecodeFunctionName(data []byte) ([]byte, error) {
	var n int

	// addr
	_, offset := varint.Uvarint(data)

	lineNumber, n := varint.Uvarint(data[offset:])
	offset += n

	hasMapping := data[offset] == 0x1
	offset++
	if hasMapping {
		// buildID
		_, n = decodeString(data[offset:])
		offset += n

		// filename
		_, n := decodeString(data[offset:])
		offset += n

		// memoryStart
		_, n = varint.Uvarint(data[offset:])
		offset += n

		// memoryLength
		_, n = varint.Uvarint(data[offset:])
		offset += n

		// mappingOffset
		_, n = varint.Uvarint(data[offset:])
		offset += n
	}

	if lineNumber > 0 {
		for i := uint64(0); i < lineNumber; i++ {
			// line
			_, n = varint.Uvarint(data[offset:])
			offset += n

			_, n = varint.Uvarint(data[offset:])
			offset += n

			hasFunction := data[offset] == 0x1
			offset++

			if hasFunction {
				// startLine
				_, n = varint.Uvarint(data[offset:])
				offset += n

				name, n := decodeString(data[offset:])
				if len(name) == 0 {
					// Fall back to system_name, matching DecodeInto.
					name, _ = decodeString(data[offset+n:])
				}
				return name, nil
			}
		}

		return []byte{}, nil
	} else {
		return []byte{}, nil
	}
}

func decodeString(data []byte) ([]byte, int) {
	length, n := varint.Uvarint(data)
	return data[n : n+int(length)], n + int(length)
}
