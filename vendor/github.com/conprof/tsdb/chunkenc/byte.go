// Copyright 2017 The Prometheus Authors
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

// The code in this file was largely written by Damian Gryski as part of
// https://github.com/dgryski/go-tsz and published under the license below.
// It was modified to accommodate reading from byte slices without modifying
// the underlying bytes, which would panic when reading from mmaped
// read-only byte slices.

// Copyright (c) 2015,2016 Damian Gryski <damian@gryski.com>
// All rights reserved.

// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:

// * Redistributions of source code must retain the above copyright notice,
// this list of conditions and the following disclaimer.
//
// * Redistributions in binary form must reproduce the above copyright notice,
// this list of conditions and the following disclaimer in the documentation
// and/or other materials provided with the distribution.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
// ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE
// FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL
// DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR
// SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER
// CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY,
// OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package chunkenc

import (
	"bytes"
	"encoding/binary"
)

// BytesChunk holds Bytes encoded sample data.
type BytesChunk struct {
	b []byte
}

// NewBytesChunk returns a new chunk with Bytes encoding of the given size.
func NewBytesChunk() *BytesChunk {
	// Each chunk holds arround 120 samples.
	// 2 bytes are used for the Sumples count.
	// All timestamps occupy arround 130-150 bytes leaving 4850bytes for the samples.
	// This is arround 40bytes per sample.
	// If the appended samples require more space can increase this array size.
	b := make([]byte, 2, 5000)
	return &BytesChunk{b: b}
}

// Encoding returns the encoding type.
func (c *BytesChunk) Encoding() Encoding {
	return EncBytes
}

// Bytes returns the underlying byte slice of the chunk.
func (c *BytesChunk) Bytes() []byte {
	return c.b
}

// NumSamples returns the number of samples in the chunk.
func (c *BytesChunk) NumSamples() int {
	return int(binary.BigEndian.Uint16(c.Bytes()))
}

// Appender implements the Chunk interface.
func (c *BytesChunk) Appender() (Appender, error) {
	// it := c.iterator()

	// // To get an appender we must know the state it would have if we had
	// // appended all existing data from scratch.
	// // We iterate through the end and populate via the iterator's state.
	// for it.Next() {
	// }
	// if err := it.Err(); err != nil {
	// 	return nil, err
	// }

	a := &bytesAppender{
		b: c,
	}
	return a, nil
}

// func (c *BytesChunk) iterator() *bytesIterator {
// 	// Should iterators guarantee to act on a copy of the data so it doesn't lock append?
// 	// When using striped locks to guard access to chunks, probably yes.
// 	// Could only copy data if the chunk is not completed yet.
// 	return &bytesIterator{
// 		br:       bytes.NewReader(c.b[2:]),
// 		numTotal: binary.BigEndian.Uint16(c.b),
// 	}
// }

// Iterator implements the Chunk interface.
func (c *BytesChunk) Iterator() Iterator {
	// Should iterators guarantee to act on a copy of the data so it doesn't lock append?
	// When using striped locks to guard access to chunks, probably yes.
	// Could only copy data if the chunk is not completed yet.
	return &bytesIterator{
		br:       bytes.NewReader(c.b[2:]),
		numTotal: binary.BigEndian.Uint16(c.b),
	}
}

type bytesAppender struct {
	b *BytesChunk

	t      int64
	tDelta uint64
}

func (a *bytesAppender) Append(t int64, v []byte) {
	var tDelta uint64
	var tt uint64
	num := binary.BigEndian.Uint16(a.b.b)

	if num == 0 {
		tt = uint64(t)

	} else if num == 1 {
		tDelta = uint64(t - a.t)
		tt = tDelta

	} else {
		tDelta = uint64(t - a.t)
		tt = tDelta - a.tDelta
	}

	// Append the time.
	buf := make([]byte, binary.MaxVarintLen64)
	time := buf[:binary.PutUvarint(buf, tt)]
	a.b.b = append(a.b.b, time...)

	// When adding empty samples we still need to create a non nil byte array to avoid EOF errors.
	if len(v) == 0 {
		v = []byte(" ")
	}
	// Append size of the sample's byte slice.
	size := buf[:binary.PutUvarint(buf, uint64(len(v)))]
	a.b.b = append(a.b.b, size...)

	// Append the sample's bytes.
	a.b.b = append(a.b.b, v...)

	a.t = t
	binary.BigEndian.PutUint16(a.b.b, num+1)

	a.tDelta = tDelta

}

type bytesIterator struct {
	br       *bytes.Reader
	numTotal uint16
	numRead  uint16

	t   int64
	val []byte

	tDelta uint64
	err    error
}

func (it *bytesIterator) At() (int64, []byte) {
	return it.t, it.val
}

func (it *bytesIterator) Err() error {
	return it.err
}

func (it *bytesIterator) Next() bool {
	if it.err != nil || it.numRead == it.numTotal {
		return false
	}
	t, err := binary.ReadUvarint(it.br)
	if err != nil {
		it.err = err
		return false
	}

	if it.numRead == 0 {
		it.t = int64(t)
	} else if it.numRead == 1 {
		it.tDelta = t
		it.t = it.t + int64(it.tDelta)
	} else {
		it.tDelta = uint64(int64(it.tDelta) + int64(t))
		it.t = it.t + int64(it.tDelta)
	}

	sampleLen, err := binary.ReadUvarint(it.br)
	if err != nil {
		it.err = err
		return false
	}

	it.val = make([]byte, sampleLen)
	_, err = it.br.Read(it.val)
	if err != nil {
		it.err = err
		return false
	}

	// Convert an empty sample value to a nil array as this is what the reader will expect.
	if bytes.Equal(it.val, []byte(" ")) {
		it.val = nil
	}

	it.numRead++
	return true

}
