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
	"bytes"
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type TreeStackEntry struct {
	nodes        []*querypb.FlamegraphNode
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
	stack TreeStack
}

func NewFlamegraphIterator(fgRoot *querypb.FlamegraphNode) *FlamegraphIterator {
	root := &TreeStackEntry{
		nodes:        []*querypb.FlamegraphNode{fgRoot},
		currentChild: -1,
	}
	return &FlamegraphIterator{
		stack: TreeStack{root},
	}
}

func (fgi *FlamegraphIterator) HasMore() bool {
	return fgi.stack.Size() > 0
}

func (fgi *FlamegraphIterator) NextChild() bool {
	fgi.stack.Peek().currentChild++

	peekNodes := fgi.stack.Peek().nodes
	peekNode := peekNodes[len(peekNodes)-1]
	return len(peekNode.Children) > fgi.stack.Peek().currentChild
}

func (fgi *FlamegraphIterator) At() *querypb.FlamegraphNode {
	peekNodes := fgi.stack.Peek().nodes
	peekNode := peekNodes[len(peekNodes)-1]
	return peekNode.Children[fgi.stack.Peek().currentChild]
}

func (fgi *FlamegraphIterator) StepInto() bool {
	peekNodes := fgi.stack.Peek().nodes
	peekNode := peekNodes[len(peekNodes)-1]
	if len(peekNode.Children) <= fgi.stack.Peek().currentChild {
		return false
	}

	fgi.stack.Push(&TreeStackEntry{
		nodes:        []*querypb.FlamegraphNode{peekNode.Children[fgi.stack.Peek().currentChild]},
		currentChild: -1,
	})

	return true
}

func (fgi *FlamegraphIterator) StepUp() {
	fgi.stack.Pop()
}

type Locations interface {
	GetLocationsByIDs(ctx context.Context, id ...[]byte) (map[string]*pb.Location, [][]byte, error)
}

func GenerateFlamegraph(
	ctx context.Context,
	tracer trace.Tracer,
	metaStore metastore.ProfileMetaStore,
	p InstantProfile,
) (*querypb.Flamegraph, error) {
	fgCtx, fgSpan := tracer.Start(ctx, "generate-flamegraph")
	defer fgSpan.End()

	_, copySpan := tracer.Start(fgCtx, "copy-profile-tree")
	meta := p.ProfileMeta()
	pt := CopyInstantProfileTree(p.ProfileTree())
	copySpan.End()

	locs, err := getLocations(fgCtx, tracer, metaStore, pt)
	if err != nil {
		return nil, fmt.Errorf("get locations: %w", err)
	}

	_, buildSpan := tracer.Start(fgCtx, "build-flamegraph")
	defer buildSpan.End()
	it := pt.Iterator()

	if !it.HasMore() || !it.NextChild() {
		return nil, nil
	}

	n := it.At()
	loc := n.LocationID()
	if loc != uuid.Nil {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	rootNode := &querypb.FlamegraphNode{}

	flamegraph := &querypb.Flamegraph{
		Root: &querypb.FlamegraphRootNode{},
		Unit: meta.SampleType.Unit,
	}

	flamegraphStack := TreeStack{{nodes: []*querypb.FlamegraphNode{rootNode}}}
	steppedInto := it.StepInto()
	if !steppedInto {
		return flamegraph, nil
	}
	flamegraph.Height = int32(1)

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()
			id := child.LocationID()
			l, found := locs[string(id[:])]
			if !found {
				return nil, fmt.Errorf("could not find location with ID %d", id)
			}

			nodes := locationToTreeNodes(l)

			peekNodes := flamegraphStack.Peek().nodes
			peekNode := peekNodes[len(peekNodes)-1]
			peekNode.Children = append(peekNode.Children, nodes[0])

			steppedInto := it.StepInto()
			if steppedInto {
				flamegraphStack.Push(&TreeStackEntry{
					nodes: nodes,
				})
				if int32(len(flamegraphStack)) > flamegraph.Height {
					flamegraph.Height = int32(len(flamegraphStack))
				}
			}

			for _, n := range child.FlatValues() {
				if n.Value == 0 {
					continue
				}
				for _, entry := range flamegraphStack {
					for _, node := range entry.nodes {
						node.Cumulative += n.Value
					}
				}
			}
			for _, n := range child.FlatDiffValues() {
				if n.Value == 0 {
					continue
				}
				for _, entry := range flamegraphStack {
					for _, node := range entry.nodes {
						node.Diff += n.Value
					}
				}
			}

			continue
		}

		it.StepUp()
		flamegraphStack.Pop()
	}

	flamegraph.Total = rootNode.Cumulative
	flamegraph.Root.Cumulative = rootNode.Cumulative
	flamegraph.Root.Diff = rootNode.Diff
	flamegraph.Root.Children = rootNode.Children

	return aggregateByFunction(flamegraph), nil
}

func getLocations(ctx context.Context, tracer trace.Tracer, metaStore metastore.ProfileMetaStore, pt InstantProfileTree) (map[string]*metastore.Location, error) {
	ctx, locationsSpan := tracer.Start(ctx, "get-locations")
	defer locationsSpan.End()

	locationIDs := [][]byte{}
	locationIDsSeen := map[string]struct{}{}
	err := WalkProfileTree(pt, func(n InstantProfileTreeNode) error {
		id := n.LocationID()
		if _, seen := locationIDsSeen[string(id[:])]; !seen {
			locationIDs = append(locationIDs, id[:])
			locationIDsSeen[string(id[:])] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk profile tree: %w", err)
	}

	// First ID is the root node, which is an empty UUID and we won't find it
	// in the locations.
	locs, err := metastore.GetLocationsByIDs(ctx, metaStore, locationIDs[1:]...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	return locs, nil
}

func aggregateByFunction(fg *querypb.Flamegraph) *querypb.Flamegraph {
	oldRootNode := &querypb.FlamegraphNode{
		Cumulative: fg.Root.Cumulative,
		Diff:       fg.Root.Diff,
		Children:   fg.Root.Children,
	}
	mergeChildren(oldRootNode, compareByName, equalsByName)

	it := NewFlamegraphIterator(oldRootNode)
	tree := &querypb.Flamegraph{
		Total:  fg.Total,
		Height: fg.Height,
		Root: &querypb.FlamegraphRootNode{
			Cumulative: fg.Root.Cumulative,
			Diff:       fg.Root.Diff,
		},
		Unit: fg.Unit,
	}
	if !it.HasMore() {
		return tree
	}

	newRootNode := &querypb.FlamegraphNode{
		Cumulative: fg.Root.Cumulative,
		Diff:       fg.Root.Diff,
	}
	stack := TreeStack{{nodes: []*querypb.FlamegraphNode{newRootNode}}}

	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			cur := &querypb.FlamegraphNode{
				Meta:       node.Meta,
				Cumulative: node.Cumulative,
				Diff:       node.Diff,
			}
			mergeChildren(node, compareByName, equalsByName)
			peekNodes := stack.Peek().nodes
			peekNode := peekNodes[len(peekNodes)-1]
			peekNode.Children = append(peekNode.Children, cur)

			steppedInto := it.StepInto()
			if steppedInto {
				stack.Push(&TreeStackEntry{
					nodes: []*querypb.FlamegraphNode{cur},
				})
			}
			continue
		}
		it.StepUp()
		stack.Pop()
	}

	tree.Root.Children = newRootNode.Children

	return tree
}

// mergeChildren sorts and merges the children of the given node if they are equals (in-place).
// compare function used for sorting and equals function used for comparing two nodes before merging.
func mergeChildren(node *querypb.FlamegraphNode, compare, equals func(a, b *querypb.FlamegraphNode) bool) {
	if len(node.Children) < 2 {
		return
	}

	// Even though we stably sort them, we might be messing the call order?
	sort.SliceStable(node.Children, func(i, j int) bool {
		return compare(node.Children[i], node.Children[j])
	})

	var cumulative int64

	i, j := 0, 1
	for i < len(node.Children)-1 {
		current, next := node.Children[i], node.Children[j]
		if equals(current, next) {
			// Merge children into the first one
			current.Meta.Line = nil
			if current.Meta.Mapping != nil && next.Meta.Mapping != nil && !bytes.Equal(current.Meta.Mapping.Id, next.Meta.Mapping.Id) {
				current.Meta.Mapping = &pb.Mapping{}
			}

			cumulative += next.Cumulative
			current.Cumulative += next.Cumulative
			current.Diff += next.Diff
			current.Children = append(current.Children, next.Children...)
			// Delete merged child
			node.Children = append(node.Children[:j], node.Children[j+1:]...)
			continue
		}
		i, j = i+1, j+1
	}

	// TODO: This is just a safeguard and should be properly fixed before this function.
	if node.Cumulative < cumulative {
		node.Cumulative = cumulative
	}
}

func compareByName(a, b *querypb.FlamegraphNode) bool {
	if a.Meta.Function != nil && b.Meta.Function == nil {
		return false
	}

	if a.Meta.Function == nil && b.Meta.Function != nil {
		return true
	}

	if a.Meta.Function == nil && b.Meta.Function == nil {
		return a.Meta.Location.Address <= b.Meta.Location.Address
	}

	return a.Meta.Function.Name <= b.Meta.Function.Name
}

func equalsByName(a, b *querypb.FlamegraphNode) bool {
	if a.Meta.Function != nil && b.Meta.Function == nil {
		return false
	}

	if a.Meta.Function == nil && b.Meta.Function != nil {
		return false
	}

	if a.Meta.Function == nil && b.Meta.Function == nil {
		return a.Meta.Location.Address == b.Meta.Location.Address
	}

	return a.Meta.Function.Name == b.Meta.Function.Name
}

// locationToTreeNodes converts a location to its tree nodes, if the location
// has multiple inlined functions it creates multiple nodes for each inlined
// function.
func locationToTreeNodes(location *metastore.Location) []*querypb.FlamegraphNode {
	if len(location.Lines) > 0 {
		return linesToTreeNodes(
			location,
			location.Mapping,
			location.Lines,
		)
	}

	var mappingId []byte
	if location.Mapping != nil {
		mappingId = location.Mapping.Id
	}
	return []*querypb.FlamegraphNode{{
		Meta: &querypb.FlamegraphNodeMeta{
			Location: &pb.Location{
				Id:        location.ID[:],
				MappingId: mappingId,
				Address:   location.Address,
				IsFolded:  location.IsFolded,
			},
			Mapping: location.Mapping,
		},
	}}
}

// linesToTreeNodes turns inlined `lines` into a stack of TreeNode items and
// returns the slice of items in order from outer-most to inner-most.
func linesToTreeNodes(
	location *metastore.Location,
	mapping *pb.Mapping,
	lines []metastore.LocationLine,
) []*querypb.FlamegraphNode {
	if len(lines) == 0 {
		return nil
	}

	res := make([]*querypb.FlamegraphNode, len(lines))
	var prev *querypb.FlamegraphNode

	// Same as locations, lines are in order from deepest to highest in the
	// stack. Therefore we start with the innermost, and work ourselves
	// outwards. We want the result to be from higest to deepest to be inserted
	// into our flamegraph at our "current" position that's calling
	// linesToTreeNodes.
	for i := 0; i < len(lines); i++ {
		node := lineToTreeNode(
			location,
			mapping,
			lines[i],
			prev,
		)
		res[len(lines)-1-i] = node
		prev = node
	}

	return res
}

func lineToTreeNode(
	location *metastore.Location,
	mapping *pb.Mapping,
	line metastore.LocationLine,
	child *querypb.FlamegraphNode,
) *querypb.FlamegraphNode {
	var children []*querypb.FlamegraphNode
	if child != nil {
		children = []*querypb.FlamegraphNode{child}
	}
	var mappingId []byte
	if mapping != nil {
		mappingId = mapping.Id
	}
	return &querypb.FlamegraphNode{
		Meta: &querypb.FlamegraphNodeMeta{
			Location: &pb.Location{
				Id:        location.ID[:],
				MappingId: mappingId,
				Address:   location.Address,
				IsFolded:  location.IsFolded,
			},
			Function: line.Function,
			Line: &pb.Line{
				FunctionId: line.Function.Id,
				Line:       line.Line,
			},
			Mapping: mapping,
		},
		Children: children,
	}
}
