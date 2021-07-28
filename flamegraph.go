package storage

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/pprof/profile"
)

type TreeNode struct {
	Name      string      `json:"n"`
	FullName  string      `json:"f"`
	Cum       int64       `json:"v"`
	CumFormat string      `json:"l"`
	Percent   string      `json:"p"`
	Children  []*TreeNode `json:"c"`
	// TODO: add mapping for coloration
}

func (n *TreeNode) AddChild(c *TreeNode) {
	n.Children = append(n.Children, c)
}

type TreeStack []*TreeNode

func (s *TreeStack) Push(e *TreeNode) {
	*s = append(*s, e)
}

func (s *TreeStack) Peek() *TreeNode {
	return (*s)[len(*s)-1]
}

func (s *TreeStack) Pop() (*TreeNode, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *TreeStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *TreeStack) Size() int {
	return len(*s)
}

type Locations interface {
	GetByID(id uint64) (*profile.Location, error)
}

func generateFlamegraph(locations Locations, it InstantProfileTreeIterator) (*TreeNode, error) {
	if !it.HasMore() || !it.NextChild() {
		return nil, nil
	}

	n := it.At()
	loc := n.LocationID()
	if loc != uint64(0) {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	flamegraph := &TreeNode{
		Name: "root",
		Cum:  n.CumulativeValue(),
	}

	flamegraphStack := TreeStack{flamegraph}
	it.StepInto()

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()
			cumulative := child.CumulativeValue()
			if cumulative > 0 {
				l, err := locations.GetByID(child.LocationID())
				if err != nil {
					return nil, err
				}
				outerMost, innerMost := locationToTreeNodes(l, cumulative)
				flamegraphStack.Peek().AddChild(outerMost)
				flamegraphStack.Push(innerMost)
				it.StepInto()
			}
			continue
		}
		it.StepUp()
		flamegraphStack.Pop()
	}
	return flamegraph, nil
}

func locationToTreeNodes(location *profile.Location, value int64) (outerMost *TreeNode, innerMost *TreeNode) {
	nameParts := []string{}

	if len(location.Line) > 0 {
		outerMost, innerMost = linesToTreeNodes(nameParts, location.Line, value)
		return outerMost, innerMost
	}

	if location.Address != 0 {
		nameParts = append(nameParts, fmt.Sprintf("%016x", location.Address))
	}

	if location.Mapping != nil {
		nameParts = append(nameParts, "["+filepath.Base(location.Mapping.File)+"]")
	}

	fullName := strings.Join(nameParts, " ")
	n := &TreeNode{
		Name:     ShortenFunctionName(fullName),
		FullName: fullName,
		Cum:      value,
	}
	return n, n
}

// linesToTreeNodes turns inlined `lines` into a stack of TreeNode items and
// returns the outerMost and innerMost items.
func linesToTreeNodes(nameParts []string, lines []profile.Line, value int64) (outerMost *TreeNode, innerMost *TreeNode) {
	for i := 0; i < len(lines); i++ {
		functionNameParts := append(nameParts, lines[i].Function.Name)
		functionNameParts = append(functionNameParts, fmt.Sprintf("%s:%d", lines[i].Function.Filename, lines[i].Line))

		var children []*TreeNode = nil
		if i > 0 {
			children = []*TreeNode{outerMost}
		}
		fullName := strings.Join(functionNameParts, " ")
		outerMost = &TreeNode{
			Name:     ShortenFunctionName(fullName),
			FullName: fullName,
			Children: children,
			Cum:      value,
		}
		if i == 0 {
			innerMost = outerMost
		}
	}

	return outerMost, innerMost
}
