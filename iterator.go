package storage

type ProfileTreeStackEntry struct {
	node  *ProfileTreeNode
	child int
}

type ProfileTreeStack []*ProfileTreeStackEntry

func (s *ProfileTreeStack) Push(e *ProfileTreeStackEntry) {
	*s = append(*s, e)
}

func (s *ProfileTreeStack) Pop() (*ProfileTreeStackEntry, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *ProfileTreeStack) Peek() *ProfileTreeStackEntry {
	return (*s)[len(*s)-1]
}

func (s *ProfileTreeStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *ProfileTreeStack) Size() int {
	return len(*s)
}

type ProfileTreeIterator struct {
	tree  *ProfileTree
	stack ProfileTreeStack
	cur   *ProfileTreeStackEntry
}

func NewProfileTreeIterator(t *ProfileTree) *ProfileTreeIterator {
	root := &ProfileTreeStackEntry{
		node:  &ProfileTreeNode{Children: []*ProfileTreeNode{t.Roots}},
		child: -1,
	}
	return &ProfileTreeIterator{
		tree:  t,
		stack: ProfileTreeStack{root},
		cur:   root,
	}
}

func (i *ProfileTreeIterator) HasMore() bool {
	return i.stack.Size() > 0
}

func (i *ProfileTreeIterator) NextChild() bool {
	i.stack.Peek().child++

	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	return true
}

func (i *ProfileTreeIterator) At() InstantProfileTreeNode {
	return i.stack.Peek().node.Children[i.stack.Peek().child]
}

func (i *ProfileTreeIterator) StepInto() bool {
	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	i.stack.Push(&ProfileTreeStackEntry{
		node:  i.stack.Peek().node.Children[i.stack.Peek().child],
		child: -1,
	})

	return true
}

func (i *ProfileTreeIterator) StepUp() {
	i.stack.Pop()
}

type MemSeriesTreeStackEntry struct {
	node  *MemSeriesTreeNode
	child int
}

type MemSeriesTreeStack []*MemSeriesTreeStackEntry

func (s *MemSeriesTreeStack) Push(e *MemSeriesTreeStackEntry) {
	*s = append(*s, e)
}

func (s *MemSeriesTreeStack) Pop() (*MemSeriesTreeStackEntry, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *MemSeriesTreeStack) Peek() *MemSeriesTreeStackEntry {
	return (*s)[len(*s)-1]
}

func (s *MemSeriesTreeStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *MemSeriesTreeStack) Size() int {
	return len(*s)
}

type MemSeriesTreeIterator struct {
	tree  *MemSeriesTree
	stack MemSeriesTreeStack
}

func NewMemSeriesTreeIterator(t *MemSeriesTree) *MemSeriesTreeIterator {
	root := &MemSeriesTreeStackEntry{
		node:  &MemSeriesTreeNode{Children: []*MemSeriesTreeNode{t.Roots}},
		child: -1,
	}
	return &MemSeriesTreeIterator{
		tree:  t,
		stack: MemSeriesTreeStack{root},
	}
}

func (i *MemSeriesTreeIterator) HasMore() bool {
	return i.stack.Size() > 0
}

func (i *MemSeriesTreeIterator) NextChild() bool {
	i.stack.Peek().child++

	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	return true
}

func (i *MemSeriesTreeIterator) At() *MemSeriesTreeNode {
	return i.stack.Peek().node.Children[i.stack.Peek().child]
}

func (i *MemSeriesTreeIterator) ChildIndex() int {
	return i.stack.Peek().child
}

func (i *MemSeriesTreeIterator) Node() *MemSeriesTreeNode {
	return i.stack.Peek().node
}

func (i *MemSeriesTreeIterator) StepInto() bool {
	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	i.stack.Push(&MemSeriesTreeStackEntry{
		node:  i.stack.Peek().node.Children[i.stack.Peek().child],
		child: -1,
	})

	return true
}

func (i *MemSeriesTreeIterator) StepUp() {
	i.stack.Pop()
}

// ####################

type MemSeriesIteratorTreeStackEntry struct {
	node  *MemSeriesIteratorTreeNode
	child int
}

type MemSeriesIteratorTreeStack []*MemSeriesIteratorTreeStackEntry

func (s *MemSeriesIteratorTreeStack) Push(e *MemSeriesIteratorTreeStackEntry) {
	*s = append(*s, e)
}

func (s *MemSeriesIteratorTreeStack) Pop() (*MemSeriesIteratorTreeStackEntry, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *MemSeriesIteratorTreeStack) Peek() *MemSeriesIteratorTreeStackEntry {
	return (*s)[len(*s)-1]
}

func (s *MemSeriesIteratorTreeStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *MemSeriesIteratorTreeStack) Size() int {
	return len(*s)
}

type MemSeriesIteratorTreeIterator struct {
	tree  *MemSeriesIteratorTree
	stack MemSeriesIteratorTreeStack
}

func NewMemSeriesIteratorTreeIterator(t *MemSeriesIteratorTree) *MemSeriesIteratorTreeIterator {
	root := &MemSeriesIteratorTreeStackEntry{
		node:  t.Roots,
		child: -1,
	}
	return &MemSeriesIteratorTreeIterator{
		tree:  t,
		stack: MemSeriesIteratorTreeStack{root},
	}
}

func (i *MemSeriesIteratorTreeIterator) HasMore() bool {
	return i.stack.Size() > 0
}

func (i *MemSeriesIteratorTreeIterator) NextChild() bool {
	i.stack.Peek().child++

	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	return true
}

func (i *MemSeriesIteratorTreeIterator) At() InstantProfileTreeNode {
	return i.stack.Peek().node.Children[i.stack.Peek().child]
}

func (i *MemSeriesIteratorTreeIterator) StepInto() bool {
	if len(i.stack.Peek().node.Children) <= i.stack.Peek().child {
		return false
	}

	i.stack.Push(&MemSeriesIteratorTreeStackEntry{
		node:  i.stack.Peek().node.Children[i.stack.Peek().child],
		child: -1,
	})

	return true
}

func (i *MemSeriesIteratorTreeIterator) StepUp() {
	i.stack.Pop()
}
