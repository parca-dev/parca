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

// GZipChunk holds GZip encoded sample data.
type GZipChunk struct {
	b []byte
}

// NewGZipChunk returns a new chunk with GZip encoding of the given size.
func NewGZipChunk() *GZipChunk {
	b := make([]byte, 2, 128)
	return &GZipChunk{b: b}
}

// Encoding returns the encoding type.
func (c *GZipChunk) Encoding() Encoding {
	return EncGZip
}

// Bytes returns the underlying byte slice of the chunk.
func (c *GZipChunk) Bytes() []byte {
	return c.b
}

// NumSamples returns the number of samples in the chunk.
func (c *GZipChunk) NumSamples() int {
	num := int(binary.BigEndian.Uint16(c.b))
	return num
}

// Appender implements the Chunk interface.
func (c *GZipChunk) Appender() (Appender, error) {
	return &gzipAppender{
		c: c,
	}, nil
}

func (c *GZipChunk) iterator() *gzipIterator {
	// Should iterators guarantee to act on a copy of the data so it doesn't lock append?
	// When using striped locks to guard access to chunks, probably yes.
	// Could only copy data if the chunk is not completed yet.
	return &gzipIterator{
		br:       bytes.NewReader(c.b[2:]),
		numTotal: binary.BigEndian.Uint16(c.b),
	}
}

// Iterator implements the Chunk interface.
func (c *GZipChunk) Iterator() Iterator {
	return c.iterator()
}

type gzipAppender struct {
	c *GZipChunk
}

func (a *gzipAppender) Append(t int64, v []byte) {
	num := binary.BigEndian.Uint16(a.c.b)

	buf := make([]byte, binary.MaxVarintLen64)
	// append timestamp as varint
	for _, b := range buf[:binary.PutVarint(buf, t)] {
		a.c.b = append(a.c.b, b)
	}
	buf = make([]byte, binary.MaxVarintLen64)
	// append size of the sample's byte slice as varint
	for _, b := range buf[:binary.PutVarint(buf, int64(len(v)))] {
		a.c.b = append(a.c.b, b)
	}
	// append the sample's bytes
	for _, b := range v {
		a.c.b = append(a.c.b, b)
	}

	binary.BigEndian.PutUint16(a.c.b, num+1)
}

type gzipIterator struct {
	br       *bytes.Reader
	numTotal uint16
	numRead  uint16

	t   int64
	val []byte

	err error
}

func (it *gzipIterator) At() (int64, []byte) {
	return it.t, it.val
}

func (it *gzipIterator) Err() error {
	return it.err
}

func (it *gzipIterator) Next() bool {
	if it.err != nil || it.numRead == it.numTotal {
		return false
	}

	var err error
	it.t, err = binary.ReadVarint(it.br)
	if err != nil {
		it.err = err
		return false
	}

	sampleLen, err := binary.ReadVarint(it.br)
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

	it.numRead++
	return true
}
