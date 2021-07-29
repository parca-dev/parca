package chunk

type ChunkIterator interface {
	Next() bool
	At() int64
	Err() error
}

type Chunk interface {
	AppendAt(i uint16, v int64) error
	Iterator() ChunkIterator
}

func NewFakeChunk() *FakeChunk {
	return &FakeChunk{}
}

func MustFakeChunk(v ...int64) *FakeChunk {
	return &FakeChunk{
		Values: v,
	}
}

type FakeChunk struct {
	Values []int64
}

func (c *FakeChunk) AppendAt(i uint16, v int64) error {
	for uint16(len(c.Values)) < i {
		c.Values = append(c.Values, 0)
	}
	c.Values = append(c.Values, v)
	return nil
}

type FakeChunkIterator struct {
	i      int
	values []int64
}

func (c *FakeChunk) Iterator() ChunkIterator {
	return &FakeChunkIterator{
		values: c.Values,
		i:      -1,
	}
}

func (c *FakeChunkIterator) Next() bool {
	c.i++

	return true
}

// At returns the current value of the iterator. It will continue to return 0s
// for indices greater than the size of values known. This is important because
// values for stack traces can be sparse, and the series tracks how many
// samples have truly been written, and 0 values in profiles are a no-op value,
// it's equivalent to the stack trace not existing.
func (c *FakeChunkIterator) At() int64 {
	if len(c.values) <= c.i {
		return int64(0)
	}
	return c.values[c.i]
}

func (c *FakeChunkIterator) Err() error {
	return nil
}
