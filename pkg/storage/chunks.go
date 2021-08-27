package storage

import (
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
)

type MultiChunkIterator interface {
	// Next advances the iterator by one possibly across multiple chunks with this iterator.
	Next() bool
	// At returns the current value.
	At() int64
	// Err returns the current error.
	// It should be used only after iterator is exhausted, that is `Next` returns false.
	Err() error
}

type multiChunksIterator struct {
	chunks []chunkenc.Chunk
	cit    chunkenc.Iterator
	read   uint16

	val int64
	err error
}

func (it *multiChunksIterator) Next() bool {
	if it.cit == nil {
		it.cit = it.chunks[it.read].Iterator(it.cit)
		it.read++
	}
	for it.cit.Next() {
		it.val = it.cit.At()
		return true
	}
	if it.cit.Err() != nil {
		it.err = it.cit.Err()
		return false
	}

	if it.read < uint16(len(it.chunks)) {
		it.cit = it.chunks[it.read].Iterator(it.cit)
		it.read++

		// We need to immediately need the next value.
		for it.cit.Next() {
			it.val = it.cit.At()
			return true
		}
		// Rare case were we have an empty next chunk.
		return false
	}

	// We've read everything from all chunks.
	return false
}

func (it *multiChunksIterator) At() int64 {
	return it.val
}

func (it *multiChunksIterator) Err() error {
	return it.err
}
