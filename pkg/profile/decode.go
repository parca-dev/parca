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

func DecodeInto(lw LocationsWriter, data []byte) (DecodeResult, error) {
	var (
		n             int
		buildID       []byte
		memoryStart   uint64
		memoryLength  uint64
		mappingOffset uint64
	)

	addr, offset := varint.Uvarint(data)

	lineNumber, n := varint.Uvarint(data[offset:])
	offset += n

	hasMapping := data[offset] == 0x1
	offset++
	if hasMapping {
		buildID, n = decodeString(data[offset:])
		offset += n

		if err := lw.MappingBuildID.Append(buildID); err != nil {
			return DecodeResult{}, fmt.Errorf("append mapping build id: %w", err)
		}

		filename, n := decodeString(data[offset:])
		offset += n

		if err := lw.MappingFile.Append(filename); err != nil {
			return DecodeResult{}, fmt.Errorf("append mapping filename: %w", err)
		}

		memoryStart, n = varint.Uvarint(data[offset:])
		offset += n

		lw.MappingStart.Append(memoryStart)

		memoryLength, n = varint.Uvarint(data[offset:])
		offset += n

		lw.MappingLimit.Append(memoryStart + memoryLength)

		mappingOffset, n = varint.Uvarint(data[offset:])
		offset += n

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

			line, n := varint.Uvarint(data[offset:])
			offset += n

			lw.LineNumber.Append(int64(line))

			hasFunction := data[offset] == 0x1
			offset++

			if hasFunction {
				startLine, n := varint.Uvarint(data[offset:])
				offset += n

				lw.FunctionStartLine.Append(int64(startLine))

				name, n := decodeString(data[offset:])
				offset += n

				if err := lw.FunctionName.Append(name); err != nil {
					return DecodeResult{}, fmt.Errorf("append function name: %w", err)
				}

				systemName, n := decodeString(data[offset:])
				offset += n

				if err := lw.FunctionSystemName.Append(systemName); err != nil {
					return DecodeResult{}, fmt.Errorf("append function system name: %w", err)
				}

				filename, n := decodeString(data[offset:])
				offset += n

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
	} else {
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

			hasFunction := data[offset] == 0x1
			offset++

			if hasFunction {
				// startLine
				_, n = varint.Uvarint(data[offset:])
				offset += n

				name, _ := decodeString(data[offset:])
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
