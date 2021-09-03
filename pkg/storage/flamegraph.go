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
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
	"go.opentelemetry.io/otel"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
)

type TreeStackEntry struct {
	node         *pb.FlamegraphNode
	currentChild int
}

type TreeStack []*TreeStackEntry

func (s *TreeStack) Push(e *TreeStackEntry) {
	*s = append(*s, e)
}

func (s *TreeStack) Peek() *TreeStackEntry {
	return (*s)[len(*s)-1]
}

func (s *TreeStack) Pop() (*TreeStackEntry, bool) {
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

type FlamegraphIterator struct {
	tree  *pb.Flamegraph
	stack TreeStack
}

func NewFlamegraphIterator(fg *pb.Flamegraph) *FlamegraphIterator {
	root := &TreeStackEntry{
		node: &pb.FlamegraphNode{
			Name:       fg.Root.Name,
			FullName:   fg.Root.FullName,
			Cumulative: fg.Root.Cumulative,
			Diff:       fg.Root.Diff,
			Children:   fg.Root.Children,
		},
		currentChild: -1,
	}
	return &FlamegraphIterator{
		tree:  fg,
		stack: TreeStack{root},
	}
}

func (fgi *FlamegraphIterator) HasMore() bool {
	return fgi.stack.Size() > 0
}

func (fgi *FlamegraphIterator) NextChild() bool {
	fgi.stack.Peek().currentChild++

	return len(fgi.stack.Peek().node.Children) > fgi.stack.Peek().currentChild
}

func (fgi *FlamegraphIterator) At() *pb.FlamegraphNode {
	return fgi.stack.Peek().node.Children[fgi.stack.Peek().currentChild]
}

func (fgi *FlamegraphIterator) StepInto() bool {
	if len(fgi.stack.Peek().node.Children) <= fgi.stack.Peek().currentChild {
		return false
	}

	fgi.stack.Push(&TreeStackEntry{
		node:         fgi.stack.Peek().node.Children[fgi.stack.Peek().currentChild],
		currentChild: -1,
	})

	return true
}

func (fgi *FlamegraphIterator) StepUp() {
	fgi.stack.Pop()
}

type Locations interface {
	GetLocationsByIDs(ctx context.Context, id ...uint64) (map[uint64]*profile.Location, error)
}

var tracer = otel.Tracer("flamegraph")

func GenerateFlamegraph(ctx context.Context, locations Locations, p InstantProfile) (*pb.Flamegraph, error) {
	fgCtx, fgSpan := tracer.Start(ctx, "generate-flamegraph")
	defer fgSpan.End()

	_, copySpan := tracer.Start(fgCtx, "copy-profile-tree")
	meta := p.ProfileMeta()
	pt := CopyInstantProfileTree(p.ProfileTree())
	copySpan.End()

	locs, err := getLocations(fgCtx, locations, pt)
	if err != nil {
		return nil, err
	}

	_, buildSpan := tracer.Start(fgCtx, "build-flamegraph")
	defer buildSpan.End()
	it := pt.Iterator()

	if !it.HasMore() || !it.NextChild() {
		return nil, nil
	}

	n := it.At()
	loc := n.LocationID()
	if loc != uint64(0) {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	flamegraphRoot := &pb.FlamegraphNode{
		Name:       "root",
		Cumulative: n.CumulativeValue(),
		Diff:       n.CumulativeDiffValue(),
	}

	flamegraph := &pb.Flamegraph{
		Root:  flamegraphRoot,
		Total: flamegraphRoot.Cumulative,
		Unit:  meta.SampleType.Unit,
	}

	flamegraphStack := TreeStack{{node: flamegraphRoot}}
	steppedInto := it.StepInto()
	if !steppedInto {
		return flamegraph, nil
	}

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()
			cumulative := child.CumulativeValue()
			if cumulative > 0 {
				id := child.LocationID()
				l, found := locs[id]
				if !found {
					return nil, fmt.Errorf("could not find location with ID %d", id)
				}
				outerMost, innerMost := locationToTreeNodes(l, cumulative, child.CumulativeDiffValue())

				flamegraphStack.Peek().node.Children = append(flamegraphStack.Peek().node.Children, outerMost)
				flamegraphStack.Push(&TreeStackEntry{
					node: innerMost,
				})
				it.StepInto()
			}
			continue
		}

		it.StepUp()
		flamegraphStack.Pop()
	}
	return flamegraph, nil
	//return aggregateByFunctionName(flamegraph), nil
}

func getLocations(ctx context.Context, locations Locations, pt InstantProfileTree) (map[uint64]*profile.Location, error) {
	ctx, locationsSpan := tracer.Start(ctx, "get-locations")
	defer locationsSpan.End()

	locationIDs := []uint64{}
	locationIDsSeen := map[uint64]struct{}{}
	err := WalkProfileTree(pt, func(n InstantProfileTreeNode) error {
		id := n.LocationID()
		if _, seen := locationIDsSeen[id]; !seen {
			locationIDs = append(locationIDs, id)
			locationIDsSeen[id] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	locs, err := locations.GetLocationsByIDs(ctx, locationIDs...)
	if err != nil {
		return nil, err
	}

	return locs, nil
}

func aggregateByFunctionName(fg *pb.Flamegraph) *pb.Flamegraph {
	it := NewFlamegraphIterator(fg)
	tree := &pb.Flamegraph{
		Total: fg.Total,
		Root: &pb.FlamegraphNode{
			Name:       fg.Root.Name,
			FullName:   fg.Root.FullName,
			Cumulative: fg.Root.Cumulative,
			Diff:       fg.Root.Diff,
		},
	}
	if !it.HasMore() {
		return tree
	}
	stack := TreeStack{{node: tree.Root}}

	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			cur := &pb.FlamegraphNode{
				Name:       node.Name,
				FullName:   node.FullName,
				Cumulative: node.Cumulative,
				Diff:       node.Diff,
			}
			mergeChildren(node, compareByName, equalsByName)
			stack.Peek().node.Children = append(stack.Peek().node.Children, cur)

			steppedInto := it.StepInto()
			if steppedInto {
				stack.Push(&TreeStackEntry{
					node: cur,
				})
			}
			continue
		}
		it.StepUp()
		stack.Pop()
	}

	return tree
}

// mergeChildren sorts and merges the children of the given node if they are equals (in-place).
// compare function used for sorting and equals function used for comparing two nodes before merging.
func mergeChildren(node *pb.FlamegraphNode, compare, equals func(a, b *pb.FlamegraphNode) bool) {
	if len(node.Children) < 2 {
		return
	}

	// Even though we stably sort them, we might be messing the call order?
	sort.SliceStable(node.Children, func(i, j int) bool {
		return compare(node.Children[i], node.Children[j])
	})

	i, j := 0, 1
	for i < len(node.Children)-1 {
		current, next := node.Children[i], node.Children[j]
		if equals(current, next) {
			// Merge children into the first one
			aggregatedName := strings.Split(current.Name, " ")[0]
			current.Name = fmt.Sprintf("%s :0", aggregatedName)
			current.Cumulative += next.Cumulative
			current.Diff += next.Diff
			current.Children = append(current.Children, next.Children...)
			// Delete merged child
			node.Children = append(node.Children[:j], node.Children[j+1:]...)
			continue
		}
		i, j = i+1, j+1
	}
}

func compareByName(a, b *pb.FlamegraphNode) bool {
	// e.g Name: alertmanager.(*Operator).sync .../pkg/alertmanager/operator.go:663
	return strings.Split(a.Name, " ")[0] <= strings.Split(b.Name, " ")[0]
}

func equalsByName(a, b *pb.FlamegraphNode) bool {
	// e.g Name: alertmanager.(*Operator).sync .../pkg/alertmanager/operator.go:663
	return strings.Split(a.Name, " ")[0] == strings.Split(b.Name, " ")[0]
}

func locationToTreeNodes(location *profile.Location, value, diff int64) (outerMost *pb.FlamegraphNode, innerMost *pb.FlamegraphNode) {
	if len(location.Line) > 0 {
		outerMost, innerMost = linesToTreeNodes(location.Line, value, diff)
		return outerMost, innerMost
	}

	short, full := locationToFuncName(location)
	n := &pb.FlamegraphNode{
		Name:       short,
		FullName:   full,
		Cumulative: value,
		Diff:       diff,
	}
	return n, n
}

func locationToFuncName(location *profile.Location) (string, string) {
	nameParts := []string{}
	if location.Address != 0 {
		nameParts = append(nameParts, fmt.Sprintf("%016x", location.Address))
	}

	if location.Mapping != nil {
		nameParts = append(nameParts, "["+filepath.Base(location.Mapping.File)+"]")
	}

	fullName := strings.Join(nameParts, " ")
	return ShortenFunctionName(fullName), fullName
}

// linesToTreeNodes turns inlined `lines` into a stack of TreeNode items and
// returns the outerMost and innerMost items.
func linesToTreeNodes(lines []profile.Line, value, diff int64) (outerMost *pb.FlamegraphNode, innerMost *pb.FlamegraphNode) {
	nameParts := []string{}
	for i := 0; i < len(lines); i++ {
		functionNameParts := append(nameParts, lines[i].Function.Name)
		functionNameParts = append(functionNameParts, fmt.Sprintf("%s:%d", lines[i].Function.Filename, lines[i].Line))

		var children []*pb.FlamegraphNode = nil
		if i > 0 {
			children = []*pb.FlamegraphNode{outerMost}
		}
		fullName := strings.Join(functionNameParts, " ")
		outerMost = &pb.FlamegraphNode{
			Name:       ShortenFunctionName(fullName),
			FullName:   fullName,
			Children:   children,
			Cumulative: value,
			Diff:       diff,
		}
		if i == 0 {
			innerMost = outerMost
		}
	}

	return outerMost, innerMost
}
