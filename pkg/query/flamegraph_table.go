// Copyright 2022 The Parca Authors
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

package query

import (
	"context"
	"sort"

	"go.opentelemetry.io/otel/trace"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	querypb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateFlamegraphTable(ctx context.Context, tracer trace.Tracer, p *profile.Profile, disableTriming bool) (*querypb.Flamegraph, error) {
	rootNode := &querypb.FlamegraphNode{}
	current := rootNode

	var height int32

	tables := &tableConverter{
		stringsSlice:   []string{},
		stringsIndex:   map[string]uint32{},
		mappingsSlice:  []*metastorev1alpha1.Mapping{},
		mappingsIndex:  map[string]uint32{},
		locationsSlice: []*metastorev1alpha1.Location{},
		locationsIndex: map[string]uint32{},
		functionsSlice: []*metastorev1alpha1.Function{},
		functionsIndex: map[string]uint32{},
	}

	tables.AddString("") // Add empty string to the string table.

	for _, s := range p.Samples {
		locations := s.Locations
		if int32(len(locations)) > height {
			height = int32(len(locations))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(locations) - 1; i >= 0; i-- {
			tables.AddMapping(locations[i].Mapping)
			li := tables.AddLocation(locations[i])
			location := tables.locationsSlice[li-1]

			nodes := tableLocationToTreeNodes(location, li)
			for j := len(nodes) - 1; j >= 0; j-- {
				node := nodes[j]

				// Find the index of a child that has the same location by index.
				index := sort.Search(len(current.Children), func(i int) bool {
					return current.Children[i].Meta.LocationIndex >= node.Meta.LocationIndex
				})

				if index < len(current.GetChildren()) && current.Children[index].Meta.LocationIndex == node.Meta.LocationIndex {
					// Insert onto existing node
					current = current.Children[index]
					current.Cumulative += s.Value
					current.Diff += s.DiffValue
				} else {
					// Insert new node
					node.Cumulative += s.Value
					node.Diff += s.DiffValue

					newChildren := make([]*querypb.FlamegraphNode, len(current.Children)+1)
					copy(newChildren, current.Children[:index])

					newChildren[index] = node
					copy(newChildren[index+1:], current.Children[index:])
					current.Children = newChildren

					current = node

					// There is a case where tableLocationToTreeNodes returns the node pointing to its parent,
					// resulting in an endless loop. We remove all possible children and add them later ourselves.
					current.Children = nil
				}
			}
		}

		// Sum up the value to the cumulative value of the root
		rootNode.Cumulative += s.Value
		rootNode.Diff += s.DiffValue

		// For next sample start at the root again
		current = rootNode
	}

	flamegraph := &querypb.Flamegraph{
		Root: &querypb.FlamegraphRootNode{
			Cumulative: rootNode.Cumulative,
			Diff:       rootNode.Diff,
			Children:   rootNode.Children,
		},
		Total:  rootNode.Cumulative,
		Unit:   p.Meta.SampleType.Unit,
		Height: height + 1, // add one for the root

		StringTable: tables.Strings(),
		Mapping:     tables.Mappings(),
		Locations:   tables.Locations(),
		Function:    tables.Functions(),
	}

	for _, f := range flamegraph.Function {
		// At this point we don't need the function's ID anymore, so we don't
		// need to transfer it over the wire.
		f.Id = ""
	}

	aggregatedFlamegraph := aggregateByFunctionTable(tables, flamegraph)

	if disableTriming {
		return aggregatedFlamegraph, nil
	}

	return TrimFlamegraph(ctx, tracer, aggregatedFlamegraph, NodeCutOffFraction), nil
}

type tableConverter struct {
	stringsSlice   []string
	stringsIndex   map[string]uint32
	mappingsSlice  []*metastorev1alpha1.Mapping
	mappingsIndex  map[string]uint32
	locationsSlice []*metastorev1alpha1.Location
	locationsIndex map[string]uint32
	functionsSlice []*metastorev1alpha1.Function
	functionsIndex map[string]uint32
}

// Strings return the table, slice more specifically, of all strings.
func (c *tableConverter) Strings() []string {
	return c.stringsSlice
}

// Mappings return the table, slice more specifically, of all mappings.
func (c *tableConverter) Mappings() []*metastorev1alpha1.Mapping {
	for _, m := range c.mappingsSlice {
		// Set all for unnecessary fields, for the frontend, to empty strings.
		m.Id = ""
	}
	return c.mappingsSlice
}

// GetLocation by its index. Returns nil if index doesn't exist.
func (c *tableConverter) GetLocation(index uint32) *metastorev1alpha1.Location {
	if index == 0 {
		return nil
	}

	if uint32(len(c.locationsSlice)) <= (index - 1) {
		return nil
	}

	return c.locationsSlice[index-1]
}

// Locations returns all the locations deduplicated by their ID.
func (c *tableConverter) Locations() []*metastorev1alpha1.Location {
	return c.locationsSlice
}

// GetFunction by its index. Returns nil if index doesn't exist.
func (c *tableConverter) GetFunction(index uint32) *metastorev1alpha1.Function {
	if index == 0 {
		return nil
	}

	if uint32(len(c.functionsSlice)) <= (index - 1) {
		return nil
	}
	return c.functionsSlice[index-1]
}

// Functions returns all the functions deduplicated by their ID.
func (c *tableConverter) Functions() []*metastorev1alpha1.Function {
	return c.functionsSlice
}

// AddString to the string table and return the strings index in the table.
func (c *tableConverter) AddString(s string) uint32 {
	if i, ok := c.stringsIndex[s]; ok {
		return i
	}
	c.stringsSlice = append(c.stringsSlice, s)
	c.stringsIndex[s] = uint32(len(c.stringsSlice) - 1)
	return c.stringsIndex[s]
}

// AddMapping by its ID and only add it if it's not yet in the table.
// Returns the mapping's index in the table.
func (c *tableConverter) AddMapping(m *metastorev1alpha1.Mapping) uint32 {
	if m == nil {
		return 0
	}
	if i, ok := c.mappingsIndex[m.Id]; ok {
		return i
	}

	// Reference strings in the string table
	m.FileStringIndex = c.AddString(m.File)
	m.BuildIdStringIndex = c.AddString(m.BuildId)
	m.File = ""
	m.BuildId = ""

	c.mappingsSlice = append(c.mappingsSlice, m)
	c.mappingsIndex[m.Id] = uint32(len(c.mappingsSlice))
	return c.mappingsIndex[m.Id]
}

// AddLocation by its ID and only add it if it's not yet in the table.
// Returns the locations's index in the table.
func (c *tableConverter) AddLocation(l *profile.Location) uint32 {
	if i, ok := c.locationsIndex[l.ID]; ok {
		return i
	}

	var mid uint32
	if l.Mapping != nil {
		// TODO: Assumes it's there, might panic
		mid = c.mappingsIndex[l.Mapping.Id]
	}

	lines := make([]*metastorev1alpha1.Line, 0, len(l.Lines))
	for _, line := range l.Lines {
		lines = append(lines, &metastorev1alpha1.Line{
			Line:          line.Line,
			FunctionIndex: c.AddFunction(line.Function),
		})
	}

	msl := &metastorev1alpha1.Location{
		// Id Not important for the frontend
		Address:      l.Address,
		MappingIndex: mid,
		IsFolded:     l.IsFolded,
		Lines:        lines,
	}

	c.locationsSlice = append(c.locationsSlice, msl)
	c.locationsIndex[l.ID] = uint32(len(c.locationsSlice))
	return c.locationsIndex[l.ID]
}

// AddFunction by its ID and only add it if it's not yet in the table.
// Returns the function's index in the table.
func (c *tableConverter) AddFunction(f *metastorev1alpha1.Function) uint32 {
	if i, ok := c.functionsIndex[f.Id]; ok {
		return i
	}

	// Reference strings in the string table
	f.NameStringIndex = c.AddString(f.Name)
	f.FilenameStringIndex = c.AddString(f.Filename)
	f.SystemNameStringIndex = c.AddString(f.SystemName)
	f.Name = ""
	f.Filename = ""
	f.SystemName = ""

	c.functionsSlice = append(c.functionsSlice, f)
	c.functionsIndex[f.Id] = uint32(len(c.functionsSlice))
	return c.functionsIndex[f.Id]
}

// tableLocationToTreeNodes converts a location to its tree nodes, if the location
// has multiple inlined functions it creates multiple nodes for each inlined
// function.
func tableLocationToTreeNodes(location *metastorev1alpha1.Location, locationIndex uint32) []*querypb.FlamegraphNode {
	if len(location.Lines) == 0 {
		return []*querypb.FlamegraphNode{{
			Meta: &querypb.FlamegraphNodeMeta{
				LocationIndex: locationIndex,
			},
		}}
	}
	nodes := make([]*querypb.FlamegraphNode, len(location.Lines))
	for i := range location.Lines {
		nodes[i] = &querypb.FlamegraphNode{
			Meta: &querypb.FlamegraphNodeMeta{
				LocationIndex: locationIndex,
				LineIndex:     uint32(i),
			},
		}
	}
	return nodes
}

type TableGetter interface {
	Strings() []string
	GetLocation(index uint32) *metastorev1alpha1.Location
	GetFunction(index uint32) *metastorev1alpha1.Function
}

func aggregateByFunctionTable(tables TableGetter, fg *querypb.Flamegraph) *querypb.Flamegraph {
	oldRootNode := &querypb.FlamegraphNode{
		Cumulative: fg.Root.Cumulative,
		Diff:       fg.Root.Diff,
		Children:   fg.Root.Children,
	}
	mergeChildrenTable(tables, oldRootNode, compareByNameTable, equalsByNameTable)

	it := NewFlamegraphIterator(oldRootNode)
	tree := &querypb.Flamegraph{
		Total:  fg.Total,
		Height: fg.Height,
		Root: &querypb.FlamegraphRootNode{
			Cumulative: fg.Root.Cumulative,
			Diff:       fg.Root.Diff,
		},
		Unit:        fg.Unit,
		StringTable: fg.StringTable,
		Locations:   fg.Locations,
		Mapping:     fg.Mapping,
		Function:    fg.Function,
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
			mergeChildrenTable(tables, node, compareByNameTable, equalsByNameTable)
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

// mergeChildrenTable sorts and merges the children of the given node if they are equals (in-place).
// compare function used for sorting and equals function used for comparing two nodes before merging.
func mergeChildrenTable(
	tables TableGetter,
	node *querypb.FlamegraphNode,
	compare, equals func(tables TableGetter, a, b *querypb.FlamegraphNode) bool,
) {
	if len(node.Children) < 2 {
		return
	}

	// Even though we stably sort them, we might be messing the call order?
	sort.SliceStable(node.Children, func(i, j int) bool {
		return compare(tables, node.Children[i], node.Children[j])
	})

	var cumulative int64

	i, j := 0, 1
	for i < len(node.Children)-1 {
		current, next := node.Children[i], node.Children[j]
		if equals(tables, current, next) {
			// Merge children into the first one
			current.Meta.Line = nil
			currLoc := tables.GetLocation(current.Meta.LocationIndex)
			nextLoc := tables.GetLocation(next.Meta.LocationIndex)
			if currLoc != nil && nextLoc != nil && currLoc.MappingIndex != nextLoc.MappingIndex {
				currLoc.MappingIndex = 0
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

func compareByNameTable(tables TableGetter, a, b *querypb.FlamegraphNode) bool {
	aLocation := tables.GetLocation(a.Meta.LocationIndex)
	bLocation := tables.GetLocation(b.Meta.LocationIndex)

	if aLocation == nil && bLocation != nil {
		return true
	}
	if aLocation != nil && bLocation == nil {
		return false
	}
	if aLocation == nil && bLocation == nil {
		return false
	}
	if len(aLocation.Lines) == 0 && len(bLocation.Lines) > 0 {
		return true
	}
	if len(aLocation.Lines) > 0 && len(bLocation.Lines) == 0 {
		return false
	}

	var aFunction, bFunction *metastorev1alpha1.Function
	if uint32(len(aLocation.Lines)) > a.Meta.LineIndex {
		aFunction = tables.GetFunction(aLocation.Lines[a.Meta.LineIndex].FunctionIndex)
	}
	if uint32(len(bLocation.Lines)) > b.Meta.LineIndex {
		bFunction = tables.GetFunction(bLocation.Lines[b.Meta.LineIndex].FunctionIndex)
	}

	if aFunction != nil && bFunction == nil {
		return false
	}
	if aFunction == nil && bFunction != nil {
		return true
	}
	if aFunction == nil && bFunction == nil {
		return aLocation.Address <= bLocation.Address
	}

	strings := tables.Strings()
	return strings[aFunction.NameStringIndex] <= strings[bFunction.NameStringIndex]
}

func equalsByNameTable(tables TableGetter, a, b *querypb.FlamegraphNode) bool {
	aLocation := tables.GetLocation(a.Meta.LocationIndex)
	bLocation := tables.GetLocation(b.Meta.LocationIndex)

	if aLocation == nil || bLocation == nil {
		return false
	}
	if a.Meta.LineIndex >= uint32(len(aLocation.Lines)) || b.Meta.LineIndex >= uint32(len(bLocation.Lines)) {
		return false
	}

	aFunction := tables.GetFunction(aLocation.Lines[a.Meta.LineIndex].FunctionIndex)
	bFunction := tables.GetFunction(bLocation.Lines[b.Meta.LineIndex].FunctionIndex)

	if aFunction != nil && bFunction == nil {
		return false
	}
	if aFunction == nil && bFunction != nil {
		return false
	}
	if aFunction == nil && bFunction == nil {
		return aLocation.Address == bLocation.Address
	}

	strings := tables.Strings()
	return strings[aFunction.NameStringIndex] == strings[bFunction.NameStringIndex]
}

type FlamegraphChildren []*querypb.FlamegraphNode

func (n FlamegraphChildren) Cumulative() int64 {
	cumulative := int64(0)
	for _, child := range n {
		cumulative += child.Cumulative
	}
	return cumulative
}

func (n FlamegraphChildren) Diff() int64 {
	diff := int64(0)
	for _, child := range n {
		diff += child.Diff
	}
	return diff
}

func TrimFlamegraph(ctx context.Context, tracer trace.Tracer, graph *querypb.Flamegraph, thresholdRate float64) *querypb.Flamegraph {
	ctx, span := tracer.Start(ctx, "trimFlamegraph")
	defer span.End()
	if graph == nil {
		return nil
	}
	total := graph.Total

	threshold := int64(thresholdRate * float64(total))
	var children FlamegraphChildren = trimFlamegraphNodes(ctx, tracer, graph.Root.Children, threshold)
	newTotal := int64(0)
	newDiff := int64(0)
	if len(graph.Root.Children) > 0 {
		newTotal = children.Cumulative()
		newDiff = children.Diff()
	} else {
		newTotal = graph.Root.Cumulative
		newDiff = graph.Root.Diff
	}

	trimmedGraph := &querypb.Flamegraph{
		Root: &querypb.FlamegraphRootNode{
			Children:   children,
			Cumulative: newTotal,
			Diff:       newDiff,
		},
		Total:       newTotal,
		Unit:        graph.Unit,
		Height:      graph.Height,
		StringTable: graph.StringTable,
		Locations:   graph.Locations,
		Mapping:     graph.Mapping,
		Function:    graph.Function,
	}

	return trimmedGraph
}

func trimFlamegraphNodes(ctx context.Context, tracer trace.Tracer, nodes []*querypb.FlamegraphNode, threshold int64) []*querypb.FlamegraphNode {
	var trimmedNodes []*querypb.FlamegraphNode
	for _, node := range nodes {
		if node.Cumulative < threshold {
			continue
		}
		var children FlamegraphChildren = trimFlamegraphNodes(ctx, tracer, node.Children, threshold)
		newCum := int64(0)
		newDiff := int64(0)
		if len(node.Children) > 0 {
			newCum = children.Cumulative()
			newDiff = children.Diff()
		} else {
			newCum = node.Cumulative
			newDiff = node.Diff
		}
		trimmedNodes = append(trimmedNodes, &querypb.FlamegraphNode{
			Meta:       node.Meta,
			Cumulative: newCum,
			Diff:       newDiff,
			Children:   children,
		})
	}
	return trimmedNodes
}
