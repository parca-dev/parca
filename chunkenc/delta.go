// Copyright 2021 The Parca Authors
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
// the underlying bytes, which would panic when reading from mmap'd
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
	"encoding/binary"
)

// DeltaChunk holds double delta encoded sample data.
type DeltaChunk struct {
	b bstream
}

func NewDeltaChunk() *DeltaChunk {
	b := make([]byte, 2, 128)
	return &DeltaChunk{b: bstream{stream: b, count: 0}}
}

// Encoding returns the encoding type.
func (c *DeltaChunk) Encoding() Encoding {
	return EncDelta
}

// Bytes returns the underlying byte slice of the chunk.
func (c *DeltaChunk) Bytes() []byte {
	return c.b.bytes()
}

// NumSamples returns the number of samples in the chunk.
func (c *DeltaChunk) NumSamples() int {
	return int(binary.BigEndian.Uint16(c.Bytes()))
}

func (c *DeltaChunk) Compact() {
	if l := len(c.b.stream); cap(c.b.stream) > l+chunkCompactCapacityThreshold {
		buf := make([]byte, l)
		copy(buf, c.b.stream)
		c.b.stream = buf
	}
}

// Appender implements the Chunk interface.
func (c *DeltaChunk) Appender() (Appender, error) {
	// To get an appender we must know the state it would have if we had
	// appended all existing data from scratch.
	// We iterate through the end and populate via the iterator's state.
	it := c.iterator(nil)
	for it.Next() {
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	return &deltaAppender{
		b:     &c.b,
		v:     it.v,
		delta: it.delta,
	}, nil
}

type deltaAppender struct {
	b *bstream

	v     int64
	delta uint64
}

func (a *deltaAppender) Append(v int64) {
	num := binary.BigEndian.Uint16(a.b.bytes())

	var delta uint64
	if num == 0 {
		buf := make([]byte, binary.MaxVarintLen64)
		for _, b := range buf[:binary.PutVarint(buf, v)] {
			a.b.writeByte(b)
		}
	} else if num == 1 {
		delta = uint64(v - a.v)

		buf := make([]byte, binary.MaxVarintLen64)
		for _, b := range buf[:binary.PutUvarint(buf, delta)] {
			a.b.writeByte(b)
		}
	} else {
		delta = uint64(v - a.v)
		double := int64(delta - a.delta)

		switch {
		case double == 0:
			a.b.writeBit(zero)
		case bitRange(double, 14):
			a.b.writeBits(0b10, 2)
			a.b.writeBits(uint64(double), 14)
		case bitRange(double, 17):
			a.b.writeBits(0b110, 3)
			a.b.writeBits(uint64(double), 17)
		case bitRange(double, 20):
			a.b.writeBits(0b1110, 4)
			a.b.writeBits(uint64(double), 20)
		default:
			a.b.writeBits(0b1111, 4)
			a.b.writeBits(uint64(double), 64)
		}
	}

	a.v = v
	a.delta = delta
	binary.BigEndian.PutUint16(a.b.bytes(), num+1)
}

func bitRange(x int64, nbits uint8) bool {
	return -((1<<(nbits-1))-1) <= x && x <= 1<<(nbits-1)
}

func (a *deltaAppender) AppendAt(index uint16, v int64) {
	num := binary.BigEndian.Uint16(a.b.bytes())
	// TODO(metalmatze): We should be able to write sequence of zeros to the stream directly (no loops)
	for i := num; i < index; i++ {
		a.Append(0)
	}
	a.Append(v)
}

// Iterator implements the Chunk interface.
func (c *DeltaChunk) Iterator(it Iterator) Iterator {
	return c.iterator(it)
}

func (c *DeltaChunk) iterator(it Iterator) *deltaIterator {
	if deltaIt, ok := it.(*deltaIterator); ok {
		deltaIt.Reset(c.b.bytes())
		return deltaIt
	}

	return &deltaIterator{
		br:    newBReader(c.b.bytes()[2:]),
		total: binary.BigEndian.Uint16(c.b.bytes()),
	}
}

type deltaIterator struct {
	br    bstreamReader
	total uint16
	read  uint16

	v int64

	delta uint64
	err   error
}

func (it *deltaIterator) At() int64 {
	return it.v
}

func (it *deltaIterator) Err() error {
	return it.err
}

func (it *deltaIterator) Seek(index uint16) bool {
	if it.err != nil {
		return false
	}

	for it.read <= index || it.read == 0 {
		if !it.Next() {
			return false
		}
	}
	return true
}

func (it *deltaIterator) Reset(b []byte) {
	// The first 2 bytes contain chunk headers.
	// We skip that for actual samples.
	it.br = newBReader(b[2:])
	it.total = binary.BigEndian.Uint16(b)
	it.read = 0

	it.v = 0

	it.delta = 0
	it.err = nil
}

func (it *deltaIterator) Next() bool {
	if it.err != nil || it.read == it.total {
		return false
	}

	if it.read == 0 {
		v, err := binary.ReadVarint(&it.br)
		if err != nil {
			it.err = err
			return false
		}
		it.v = v
		it.read++
		return true
	}
	if it.read == 1 {
		delta, err := binary.ReadUvarint(&it.br)
		if err != nil {
			it.err = err
			return false
		}
		it.delta = delta
		it.v = it.v + int64(it.delta)
		it.read++
		return true
	}

	var d byte
	// read delta-of-delta
	for i := 0; i < 4; i++ {
		d <<= 1
		bit, err := it.br.readBitFast()
		if err != nil {
			bit, err = it.br.readBit()
		}
		if err != nil {
			it.err = err
			return false
		}
		if bit == zero {
			break
		}
		d |= 1
	}

	var sz uint8
	var dod int64
	switch d {
	case 0b0:
		// dod == 0
	case 0b10:
		sz = 14
	case 0b110:
		sz = 17
	case 0b1110:
		sz = 20
	case 0b1111:
		// Do not use fast because it's very unlikely it will succeed.
		bits, err := it.br.readBits(64)
		if err != nil {
			it.err = err
			return false
		}
		dod = int64(bits)
	}

	if sz != 0 {
		bits, err := it.br.readBitsFast(sz)
		if err != nil {
			bits, err = it.br.readBits(sz)
		}
		if err != nil {
			it.err = err
			return false
		}
		if bits > (1 << (sz - 1)) {
			// or something
			bits = bits - (1 << sz)
		}
		dod = int64(bits)
	}

	it.delta = uint64(int64(it.delta) + dod)
	it.v = it.v + int64(it.delta)
	it.read++

	return true
}
