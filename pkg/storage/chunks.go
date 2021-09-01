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

package storage

import (
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
)

type MultiChunksIterator struct {
	chunks     []chunkenc.Chunk
	cit        chunkenc.Iterator
	readChunks uint16
	read       uint16 // read samples, need to track for seeking.

	val    int64
	sparse bool
	err    error
}

func NewMultiChunkIterator(chunks []chunkenc.Chunk) *MultiChunksIterator {
	return &MultiChunksIterator{chunks: chunks}
}

func (it *MultiChunksIterator) Next() bool {
	if it.cit == nil {
		it.cit = it.chunks[it.readChunks].Iterator(it.cit)
		it.readChunks++
	}
	for it.cit.Next() {
		it.val = it.cit.At()
		it.read++
		return true
	}
	if it.cit.Err() != nil {
		it.err = it.cit.Err()
		return false
	}

	if it.readChunks < uint16(len(it.chunks)) {
		it.cit = it.chunks[it.readChunks].Iterator(it.cit)
		it.readChunks++

		// We need to immediately need the next value.
		for it.cit.Next() {
			it.val = it.cit.At()
			it.read++
			return true
		}
		// Rare case were we have an empty next chunk.
		return false
	}

	it.sparse = true

	// We've readChunks everything from all chunks.
	return false
}

func (it *MultiChunksIterator) At() int64 {
	if it.sparse {
		return 0
	}

	return it.val
}

func (it *MultiChunksIterator) Err() error {
	return it.err
}

func (it *MultiChunksIterator) Seek(index uint16) bool {
	if it.err != nil {
		return false
	}

	// If the index is zero we don't do anything,
	// cause Next() will be called before retrieving the first value.
	if index == 0 {
		return true
	}

	for it.read <= index || it.read == 0 {
		if !it.Next() {
			return false
		}
	}
	return true
}

func (it *MultiChunksIterator) Reset(chunks []chunkenc.Chunk) {
	it.readChunks = 0
	it.read = 0
	it.val = 0
	it.sparse = false
	it.err = nil

	it.chunks = chunks
	it.cit = it.chunks[it.readChunks].Iterator(it.cit)
}

// timestampChunk wraps a chunkenc.Chunk to additionally track minTime and maxTime.
type timestampChunk struct {
	minTime int64
	maxTime int64
	chunk   chunkenc.Chunk
}

type timestampChunks []timestampChunk

func (tcs timestampChunks) indexRange(mint, maxt int64) (int, int) {
	start := 0
	end := len(tcs)

	for i, tc := range tcs {
		// The range and thus maxt is before this chunk's minTime.
		if tc.minTime > maxt {
			end = i
			break
		}
		// The range and thus mint is after this chunk's maxTime.
		// The result should be [n:n] for the chunk indexes, resulting in querying no chunks,
		// n being the length of the chunks.
		if tc.maxTime < mint && i == len(tcs)-1 {
			start = i + 1 //TODO: Can this panic for the caller?
			break
		}
		if tc.minTime <= mint {
			start = i
		}
		if tc.maxTime >= maxt {
			end = i + 1
		}
	}

	return start, end
}
