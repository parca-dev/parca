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
	"sort"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/storage/metastore"
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
	stack TreeStack
}

func NewFlamegraphIterator(fgRoot *pb.FlamegraphNode) *FlamegraphIterator {
	root := &TreeStackEntry{
		node:         fgRoot,
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
	GetLocationsByIDs(ctx context.Context, id ...uuid.UUID) (map[uuid.UUID]*metastore.Location, error)
}

func GenerateFlamegraph(
	ctx context.Context,
	tracer trace.Tracer,
	locations Locations,
	p InstantProfile,
) (*pb.Flamegraph, error) {
	fgCtx, fgSpan := tracer.Start(ctx, "generate-flamegraph")
	defer fgSpan.End()

	_, copySpan := tracer.Start(fgCtx, "copy-profile-tree")
	meta := p.ProfileMeta()
	pt := CopyInstantProfileTree(p.ProfileTree())
	copySpan.End()

	locs, err := getLocations(fgCtx, tracer, locations, pt)
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
	emptyUUID := uuid.UUID{}
	if loc != emptyUUID {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	rootNode := &pb.FlamegraphNode{}

	flamegraph := &pb.Flamegraph{
		Root: &pb.FlamegraphRootNode{},
		Unit: meta.SampleType.Unit,
	}

	flamegraphStack := TreeStack{{node: rootNode}}
	steppedInto := it.StepInto()
	if !steppedInto {
		return flamegraph, nil
	}
	flamegraph.Height = int32(1)

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()
			id := child.LocationID()
			l, found := locs[id]
			if !found {
				return nil, fmt.Errorf("could not find location with ID %d", id)
			}

			outerMost, innerMost := locationToTreeNodes(l, 0, 0)

			flamegraphStack.Peek().node.Children = append(flamegraphStack.Peek().node.Children, outerMost)

			steppedInto := it.StepInto()
			if steppedInto {
				flamegraphStack.Push(&TreeStackEntry{
					node: innerMost,
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
					entry.node.Cumulative += n.Value
				}
			}
			for _, n := range child.FlatDiffValues() {
				if n.Value == 0 {
					continue
				}
				for _, entry := range flamegraphStack {
					entry.node.Diff += n.Value
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

func getLocations(ctx context.Context, tracer trace.Tracer, locations Locations, pt InstantProfileTree) (map[uuid.UUID]*metastore.Location, error) {
	ctx, locationsSpan := tracer.Start(ctx, "get-locations")
	defer locationsSpan.End()

	locationIDs := []uuid.UUID{}
	locationIDsSeen := map[uuid.UUID]struct{}{}
	err := WalkProfileTree(pt, func(n InstantProfileTreeNode) error {
		id := n.LocationID()
		if _, seen := locationIDsSeen[id]; !seen {
			locationIDs = append(locationIDs, id)
			locationIDsSeen[id] = struct{}{}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk profile tree: %w", err)
	}

	locs, err := locations.GetLocationsByIDs(ctx, locationIDs...)
	if err != nil {
		return nil, fmt.Errorf("get locations by ids: %w", err)
	}

	return locs, nil
}

func aggregateByFunction(fg *pb.Flamegraph) *pb.Flamegraph {
	oldRootNode := &pb.FlamegraphNode{
		Cumulative: fg.Root.Cumulative,
		Diff:       fg.Root.Diff,
		Children:   fg.Root.Children,
	}
	mergeChildren(oldRootNode, compareByName, equalsByName)

	it := NewFlamegraphIterator(oldRootNode)
	tree := &pb.Flamegraph{
		Total:  fg.Total,
		Height: fg.Height,
		Root: &pb.FlamegraphRootNode{
			Cumulative: fg.Root.Cumulative,
			Diff:       fg.Root.Diff,
		},
		Unit: fg.Unit,
	}
	if !it.HasMore() {
		return tree
	}

	newRootNode := &pb.FlamegraphNode{
		Cumulative: fg.Root.Cumulative,
		Diff:       fg.Root.Diff,
	}
	stack := TreeStack{{node: newRootNode}}

	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			cur := &pb.FlamegraphNode{
				Meta:       node.Meta,
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

	tree.Root.Children = newRootNode.Children

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

	var cumulative int64

	i, j := 0, 1
	for i < len(node.Children)-1 {
		current, next := node.Children[i], node.Children[j]
		if equals(current, next) {
			// Merge children into the first one
			current.Meta.Line = nil
			if current.Meta.Mapping != nil && next.Meta.Mapping != nil && current.Meta.Mapping.Id != next.Meta.Mapping.Id {
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

func compareByName(a, b *pb.FlamegraphNode) bool {
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

func equalsByName(a, b *pb.FlamegraphNode) bool {
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

func locationToTreeNodes(location *metastore.Location, value, diff int64) (outerMost *pb.FlamegraphNode, innerMost *pb.FlamegraphNode) {
	mappingId := uuid.UUID{}
	var mapping *pb.Mapping
	if location.Mapping != nil {
		mappingId = location.Mapping.ID
		mapping = &pb.Mapping{
			Id:      location.Mapping.ID.String(),
			Start:   location.Mapping.Start,
			Limit:   location.Mapping.Limit,
			Offset:  location.Mapping.Offset,
			File:    location.Mapping.File,
			BuildId: location.Mapping.BuildID,
		}
	}

	if len(location.Lines) > 0 {
		outerMost, innerMost = linesToTreeNodes(
			location,
			mappingId,
			mapping,
			location.Lines,
			value,
			diff,
		)
		return outerMost, innerMost
	}

	n := &pb.FlamegraphNode{
		Meta: &pb.FlamegraphNodeMeta{
			Location: &pb.Location{
				Id:        location.ID.String(),
				MappingId: mappingId.String(),
				Address:   location.Address,
				IsFolded:  location.IsFolded,
			},
			Mapping: mapping,
		},
		Cumulative: value,
		Diff:       diff,
	}
	return n, n
}

// linesToTreeNodes turns inlined `lines` into a stack of TreeNode items and
// returns the outerMost and innerMost items.
func linesToTreeNodes(
	location *metastore.Location,
	mappingId uuid.UUID,
	mapping *pb.Mapping,
	lines []metastore.LocationLine,
	value int64,
	diff int64,
) (outerMost *pb.FlamegraphNode, innerMost *pb.FlamegraphNode) {
	for i, line := range lines {
		var children []*pb.FlamegraphNode = nil
		if i > 0 {
			children = []*pb.FlamegraphNode{outerMost}
		}
		outerMost = &pb.FlamegraphNode{
			Meta: &pb.FlamegraphNodeMeta{
				Location: &pb.Location{
					Id:        location.ID.String(),
					MappingId: mappingId.String(),
					Address:   location.Address,
					IsFolded:  location.IsFolded,
				},
				Function: &pb.Function{
					Id:         line.Function.ID.String(),
					Name:       line.Function.Name,
					SystemName: line.Function.SystemName,
					Filename:   line.Function.Filename,
					StartLine:  line.Function.StartLine,
				},
				Line: &pb.Line{
					LocationId: location.ID.String(),
					FunctionId: line.Function.ID.String(),
					Line:       line.Line,
				},
				Mapping: mapping,
			},
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
