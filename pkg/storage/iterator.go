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
}

func NewProfileTreeIterator(t *ProfileTree) *ProfileTreeIterator {
	root := &ProfileTreeStackEntry{
		node:  &ProfileTreeNode{Children: []*ProfileTreeNode{t.Roots.ProfileTreeNode}},
		child: -1,
	}
	return &ProfileTreeIterator{
		tree:  t,
		stack: ProfileTreeStack{root},
	}
}

func (i *ProfileTreeIterator) HasMore() bool {
	return i.stack.Size() > 0
}

func (i *ProfileTreeIterator) NextChild() bool {
	i.stack.Peek().child++

	return len(i.stack.Peek().node.Children) > i.stack.Peek().child
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
