package chunk

type Chunk interface {
	AppendAt(i int, v int64) error
	Values() []int64
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

func (c *FakeChunk) Values() []int64 {
	return c.values
}
