package storage

import (
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
)

type multiChunksIterator struct {
	chunks     []chunkenc.Chunk
	cit        chunkenc.Iterator
	readChunks uint16
	read       uint16 // read samples, need to track for seeking.

	val    int64
	sparse bool
	err    error
}

func NewMultiChunkIterator(chunks []chunkenc.Chunk) chunkenc.Iterator {
	return &multiChunksIterator{chunks: chunks}
}

func (it *multiChunksIterator) Next() bool {
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

func (it *multiChunksIterator) At() int64 {
	if it.sparse {
		return 0
	}

	return it.val
}

func (it *multiChunksIterator) Err() error {
	return it.err
}

func (it *multiChunksIterator) Seek(index uint16) bool {
	if it.err != nil {
		return false
	}

	if it.read == index {
		return true
	}

	for it.read <= index || it.read == 0 {
		if !it.Next() {
			return false
		}
	}
	return true
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
