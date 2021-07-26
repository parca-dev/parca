package chunk

type ChunkIterator interface {
	Next() bool
	At() int64
	Err() error
}

type Chunk interface {
	AppendAt(i int, v int64) error
	Iterator() ChunkIterator
}

func NewFakeChunk() *FakeChunk {
	return &FakeChunk{}
}

func MustFakeChunk(v ...int64) *FakeChunk {
	return &FakeChunk{
		values: v,
	}
}

type FakeChunk struct {
	values []int64
}

func (c *FakeChunk) AppendAt(i int, v int64) error {
	for len(c.values) < i {
		c.values = append(c.values, 0)
	}
	c.values = append(c.values, v)
	return nil
}

type FakeChunkIterator struct {
	i      int
	values []int64
}

func (c *FakeChunk) Iterator() ChunkIterator {
	return &FakeChunkIterator{
		values: c.values,
		i:      -1,
	}
}

func (c *FakeChunkIterator) Next() bool {
	c.i++

	return len(c.values) > c.i
}

func (c *FakeChunkIterator) At() int64 {
	return c.values[c.i]
}

func (c *FakeChunkIterator) Err() error {
	return nil
}
