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

package chunkenc

import (
	"encoding/binary"
)

// RLEChunk implements a run-length-encoding chunk that's useful when there are lots of repetitive values stored.
type RLEChunk struct {
	b bstream
}

func NewRLEChunk() *RLEChunk {
	return &RLEChunk{
		b: bstream{
			stream: make([]byte, 2, 128),
			count:  0,
		},
	}
}

// Encoding returns the encoding type.
func (c *RLEChunk) Encoding() Encoding {
	return EncRLE
}

// Bytes returns the underlying byte slice of the chunk.
func (c *RLEChunk) Bytes() []byte {
	return c.b.bytes()
}

// NumSamples returns the number of samples in the chunk.
func (c *RLEChunk) NumSamples() int {
	return int(binary.BigEndian.Uint16(c.Bytes()))
}

func (c *RLEChunk) Compact() {
	return
}

func (c *RLEChunk) Appender() (Appender, error) {
	return &rleAppender{
		b: &c.b,
	}, nil
}

type rleAppender struct {
	b *bstream
	v int64
}

func (a *rleAppender) Append(v int64) {
	num := binary.BigEndian.Uint16(a.b.bytes())

	// Only if values differ we need to write the next new value.
	// Otherwise, simply increase the count of the current value.
	if a.v != v {
		buf := make([]byte, binary.MaxVarintLen64)
		for _, b := range buf[:binary.PutVarint(buf, v)] {
			a.b.writeByte(b)
		}

		buf = make([]byte, 2)
		binary.BigEndian.PutUint16(buf, 1)
		for _, b := range buf {
			a.b.writeByte(b)
		}
	} else {
		b := a.b.bytes()
		// Read the last 3 bytes as that's the current count as uint16
		count := binary.BigEndian.Uint16(b[len(b)-3:])
		binary.BigEndian.PutUint16(a.b.bytes()[len(b)-3:], count+1)
	}

	a.v = v
	binary.BigEndian.PutUint16(a.b.bytes(), num+1)
}

func (a *rleAppender) AppendAt(index uint16, v int64) {
	panic("implement me")
}

func (c *RLEChunk) Iterator(it Iterator) Iterator {
	return c.iterator(it)
}

type rleIterator struct {
	br bstreamReader

	read  uint16
	total uint16

	length     uint16
	lengthRead uint16

	v   int64
	err error
}

func (c *RLEChunk) iterator(it Iterator) *rleIterator {
	return &rleIterator{
		br:    newBReader(c.b.bytes()[2:]),
		total: binary.BigEndian.Uint16(c.b.bytes()),
	}
}

func (it *rleIterator) Next() bool {
	if it.err != nil || it.read == it.total {
		return false
	}

	if it.lengthRead >= it.length {
		v, err := binary.ReadVarint(&it.br)
		if err != nil {
			it.err = err
			return false
		}
		it.v = v

		lengthBytes := make([]byte, 2)
		for i := 0; i < 2; i++ {
			b, err := it.br.ReadByte()
			if err != nil {
				it.err = err
				return false
			}
			lengthBytes[i] = b
		}
		it.length = binary.BigEndian.Uint16(lengthBytes)
		it.lengthRead = it.length - 1 // we've already read the first one
	} else {
		it.lengthRead--
	}

	it.read++

	return true
}

func (it *rleIterator) Seek(index uint16) bool {
	panic("implement me")
}

func (it *rleIterator) At() int64 {
	return it.v
}

func (it *rleIterator) Err() error {
	return it.err
}
