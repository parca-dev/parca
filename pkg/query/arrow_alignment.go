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

package query

// Arrow IPC Alignment
//
// This file provides utilities to add padding to Arrow IPC bytes for 8-byte alignment.
// This is required because Arrow's BigInt64Array/BigUint64Array require 8-byte aligned memory,
// but protobuf's bytes field may have arbitrary alignment when deserialized.
//
// Padded format: [1 byte: padding length] [0-7 padding bytes] [Arrow IPC data]
//
// The client extracts the Arrow data by reading the first byte to get the padding length,
// then using subarray(1 + padLen) to get the aligned Arrow IPC data.

// GetAlignedFlamegraphArrowBytes returns padded Arrow IPC bytes for FlamegraphArrow.
func GetAlignedFlamegraphArrowBytes(arrowBytes []byte, total, trimmed int64, unit string, height int32) []byte {
	recordSize := len(arrowBytes)

	// Calculate FlamegraphArrow message size (record + unit + height + trimmed)
	innerMsgSize := 1 + varintSize(uint64(recordSize)) + recordSize // record field
	innerMsgSize += 1 + varintSize(uint64(len(unit))) + len(unit)   // unit field
	if height != 0 {
		innerMsgSize += 1 + varintSize(uint64(height)) // height field
	}
	if trimmed != 0 {
		innerMsgSize += 1 + varintSize(uint64(trimmed)) // trimmed field
	}

	offset := 5 // gRPC-web frame header
	if total != 0 {
		offset += 1 + varintSize(uint64(total)) // QueryResponse.total
	}
	offset += 1                                // oneof field tag
	offset += varintSize(uint64(innerMsgSize)) // FlamegraphArrow message length
	offset += 1                                // record field tag
	offset += varintSize(uint64(recordSize))   // record length

	return padForAlignment(arrowBytes, offset)
}

// GetAlignedTableArrowBytes returns padded Arrow IPC bytes for TableArrow.
func GetAlignedTableArrowBytes(arrowBytes []byte, total int64, unit string) []byte {
	recordSize := len(arrowBytes)

	// Calculate TableArrow message size (record + unit)
	innerMsgSize := 1 + varintSize(uint64(recordSize)) + recordSize // record field
	innerMsgSize += 1 + varintSize(uint64(len(unit))) + len(unit)   // unit field

	offset := 5 // gRPC-web frame header
	if total != 0 {
		offset += 1 + varintSize(uint64(total)) // QueryResponse.total
	}
	offset += 1                                // oneof field tag
	offset += varintSize(uint64(innerMsgSize)) // TableArrow message length
	offset += 1                                // record field tag
	offset += varintSize(uint64(recordSize))   // record length

	return padForAlignment(arrowBytes, offset)
}

// GetAlignedSourceArrowBytes returns padded Arrow IPC bytes for Source.
func GetAlignedSourceArrowBytes(arrowBytes []byte, total int64, source, unit string) []byte {
	recordSize := len(arrowBytes)

	// Calculate Source message size (record + source + unit)
	innerMsgSize := 1 + varintSize(uint64(recordSize)) + recordSize   // record field
	innerMsgSize += 1 + varintSize(uint64(len(source))) + len(source) // source field
	innerMsgSize += 1 + varintSize(uint64(len(unit))) + len(unit)     // unit field

	offset := 5 // gRPC-web frame header
	if total != 0 {
		offset += 1 + varintSize(uint64(total)) // QueryResponse.total
	}
	offset += 1                                // oneof field tag
	offset += varintSize(uint64(innerMsgSize)) // Source message length
	offset += 1                                // record field tag
	offset += varintSize(uint64(recordSize))   // record length

	return padForAlignment(arrowBytes, offset)
}

func padForAlignment(arrowBytes []byte, estimatedOffset int) []byte {
	arrowDataOffset := estimatedOffset + 1 // +1 for the padding length byte
	padLen := (8 - (arrowDataOffset % 8)) % 8

	result := make([]byte, 1+padLen+len(arrowBytes))
	result[0] = byte(padLen)
	copy(result[1+padLen:], arrowBytes)

	return result
}

// varintSize returns the number of bytes needed to encode v as a varint.
func varintSize(v uint64) int {
	size := 1
	for v >= 0x80 {
		v >>= 7
		size++
	}
	return size
}
