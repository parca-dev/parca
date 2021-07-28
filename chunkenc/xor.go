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
	"math/bits"
)

const (
	chunkCompactCapacityThreshold = 32
)

// XORChunk holds XOR encoded sample data.
type XORChunk struct {
	b bstream
}

// NewXORChunk returns a new chunk with XOR encoding of the given size.
func NewXORChunk() *XORChunk {
	b := make([]byte, 2, 128)
	return &XORChunk{b: bstream{stream: b, count: 0}}
}

// Encoding returns the encoding type.
func (c *XORChunk) Encoding() Encoding {
	return EncXOR
}

// Bytes returns the underlying byte slice of the chunk.
func (c *XORChunk) Bytes() []byte {
	return c.b.bytes()
}

// NumSamples returns the number of samples in the chunk.
func (c *XORChunk) NumSamples() int {
	return int(binary.BigEndian.Uint16(c.Bytes()))
}

func (c *XORChunk) Compact() {
	if l := len(c.b.stream); cap(c.b.stream) > l+chunkCompactCapacityThreshold {
		buf := make([]byte, l)
		copy(buf, c.b.stream)
		c.b.stream = buf
	}
}

// Appender implements the Chunk interface.
func (c *XORChunk) Appender() (Appender, error) {
	it := c.iterator(nil)

	// To get an appender we must know the state it would have if we had
	// appended all existing data from scratch.
	// We iterate through the end and populate via the iterator's state.
	for it.Next() {
	}
	if err := it.Err(); err != nil {
		return nil, err
	}

	a := &xorAppender{
		b:        &c.b,
		v:        it.val,
		leading:  it.leading,
		trailing: it.trailing,
	}
	if binary.BigEndian.Uint16(a.b.bytes()) == 0 {
		a.leading = 0xff
	}
	return a, nil
}

func (c *XORChunk) iterator(it Iterator) *xorIterator {
	// Should iterators guarantee to act on a copy of the data so it doesn't lock append?
	// When using striped locks to guard access to chunks, probably yes.
	// Could only copy data if the chunk is not completed yet.
	if xorIter, ok := it.(*xorIterator); ok {
		xorIter.Reset(c.b.bytes())
		return xorIter
	}
	return &xorIterator{
		// The first 2 bytes contain chunk headers.
		// We skip that for actual samples.
		br:       newBReader(c.b.bytes()[2:]),
		numTotal: binary.BigEndian.Uint16(c.b.bytes()),
	}
}

// Iterator implements the Chunk interface.
func (c *XORChunk) Iterator(it Iterator) Iterator {
	return c.iterator(it)
}

type xorAppender struct {
	b *bstream

	v int64

	leading  uint8
	trailing uint8
}

func (a *xorAppender) Append(v int64) {
	num := binary.BigEndian.Uint16(a.b.bytes())

	if num == 0 {
		a.b.writeBits(uint64(v), 64)
	} else {
		a.writeVDelta(v)
	}

	a.v = v
	binary.BigEndian.PutUint16(a.b.bytes(), num+1)
}

func (a *xorAppender) AppendAt(index uint16, v int64) {
	num := binary.BigEndian.Uint16(a.b.bytes())
	// TODO(metalmatze): We should be able to write sequence of zeros to the stream directly (no loops)
	for i := num; i < index; i++ {
		a.Append(0)
	}
	a.Append(v)
}

func (a *xorAppender) writeVDelta(v int64) {
	vDelta := uint64(v) ^ uint64(a.v)

	if vDelta == 0 {
		a.b.writeBit(zero)
		return
	}
	a.b.writeBit(one)

	leading := uint8(bits.LeadingZeros64(vDelta))
	trailing := uint8(bits.TrailingZeros64(vDelta))

	// Clamp number of leading zeros to avoid overflow when encoding.
	if leading >= 32 {
		leading = 31
	}

	if a.leading != 0xff && leading >= a.leading && trailing >= a.trailing {
		a.b.writeBit(zero)
		a.b.writeBits(vDelta>>a.trailing, 64-int(a.leading)-int(a.trailing))
	} else {
		a.leading, a.trailing = leading, trailing

		a.b.writeBit(one)
		a.b.writeBits(uint64(leading), 5)

		// Note that if leading == trailing == 0, then sigbits == 64.  But that value doesn't actually fit into the 6 bits we have.
		// Luckily, we never need to encode 0 significant bits, since that would put us in the other case (vdelta == 0).
		// So instead we write out a 0 and adjust it back to 64 on unpacking.
		sigbits := 64 - leading - trailing
		a.b.writeBits(uint64(sigbits), 6)
		a.b.writeBits(vDelta>>trailing, int(sigbits))
	}
}

type xorIterator struct {
	br       bstreamReader
	numTotal uint16
	numRead  uint16

	val int64

	leading  uint8
	trailing uint8

	err error
}

func (it *xorIterator) Seek(index uint16) bool {
	if it.err != nil {
		return false
	}

	for it.numRead <= index || it.numRead == 0 {
		if !it.Next() {
			return false
		}
	}
	return true
}

func (it *xorIterator) At() int64 {
	return it.val
}

func (it *xorIterator) Err() error {
	return it.err
}

func (it *xorIterator) Reset(b []byte) {
	// The first 2 bytes contain chunk headers.
	// We skip that for actual samples.
	it.br = newBReader(b[2:])
	it.numTotal = binary.BigEndian.Uint16(b)

	it.numRead = 0
	it.val = 0
	it.leading = 0
	it.trailing = 0
	it.err = nil
}

func (it *xorIterator) Next() bool {
	if it.err != nil || it.numRead == it.numTotal {
		return false
	}

	if it.numRead == 0 {
		v, err := it.br.readBits(64)
		if err != nil {
			it.err = err
			return false
		}
		it.val = int64(v) // internally XORed as uint64 but when reading back we want to interpret as int64

		it.numRead++
		return true
	}

	return it.readValue()
}

func (it *xorIterator) readValue() bool {
	bit, err := it.br.readBitFast()
	if err != nil {
		bit, err = it.br.readBit()
	}
	if err != nil {
		it.err = err
		return false
	}

	if bit == zero {
		// it.val = it.val
	} else {
		bit, err := it.br.readBitFast()
		if err != nil {
			bit, err = it.br.readBit()
		}
		if err != nil {
			it.err = err
			return false
		}
		if bit == zero {
			// reuse leading/trailing zero bits
			// it.leading, it.trailing = it.leading, it.trailing
		} else {
			bits, err := it.br.readBitsFast(5)
			if err != nil {
				bits, err = it.br.readBits(5)
			}
			if err != nil {
				it.err = err
				return false
			}
			it.leading = uint8(bits)

			bits, err = it.br.readBitsFast(6)
			if err != nil {
				bits, err = it.br.readBits(6)
			}
			if err != nil {
				it.err = err
				return false
			}
			mbits := uint8(bits)
			// 0 significant bits here means we overflowed and we actually need 64; see comment in encoder
			if mbits == 0 {
				mbits = 64
			}
			it.trailing = 64 - it.leading - mbits
		}

		mbits := 64 - it.leading - it.trailing
		bits, err := it.br.readBitsFast(mbits)
		if err != nil {
			bits, err = it.br.readBits(mbits)
		}
		if err != nil {
			it.err = err
			return false
		}
		vbits := uint64(it.val)
		vbits ^= bits << it.trailing
		it.val = int64(vbits)
	}

	it.numRead++
	return true
}

// FromValuesXOR takes a sequence of values and returns a new populated Chunk.
// This is mostly helpful in tests.
func FromValuesXOR(values ...int64) Chunk {
	c := NewXORChunk()
	app, _ := c.Appender()
	for _, v := range values {
		app.Append(v)
	}
	return c
}
