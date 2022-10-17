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
	"strconv"

	"go.opentelemetry.io/otel/trace"

	metastorev1alpha1 "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateFlamegraphFlat(ctx context.Context, tracer trace.Tracer, p *profile.Profile) (*pb.Flamegraph, error) {
	rootNode := &pb.FlamegraphNode{}
	current := rootNode

	var height int32

	tables := &tableConverter{
		stringsSlice:   []string{},
		stringsIndex:   map[string]int{},
		mappingsSlice:  []*metastorev1alpha1.Mapping{},
		mappingsIndex:  map[string]int{},
		locationsSlice: []*metastorev1alpha1.Location{},
		locationsIndex: map[string]int{},
		functionsSlice: []*metastorev1alpha1.Function{},
		functionsIndex: map[string]int{},
	}

	for _, s := range p.Samples {
		locations := s.Locations
		if int32(len(locations)) > height {
			height = int32(len(locations))
		}

		// Reverse walking the location as stacked location are like 3 > 2 > 1 > 0 where 0 is the root.
		for i := len(locations) - 1; i >= 0; i-- {
			location := locations[i]

			tables.AddMapping(location.Mapping)
			tables.AddLocation(location)

			nodes := locationToTreeNodes(location)
			for j := len(nodes) - 1; j >= 0; j-- {
				node := nodes[j]

				index := sort.Search(len(current.Children), func(i int) bool {
					return current.Children[i].Meta.Location.Id >= node.Meta.Location.Id
				})

				if index < len(current.GetChildren()) && current.Children[index].Meta.Location.Id == node.Meta.Location.Id {
					// Insert onto existing node
					current = current.Children[index]
					current.Cumulative += s.Value
					current.Diff += s.DiffValue
				} else {
					// Insert new node
					node.Cumulative += s.Value
					node.Diff += s.DiffValue

					newChildren := make([]*pb.FlamegraphNode, len(current.Children)+1)
					copy(newChildren, current.Children[:index])

					newChildren[index] = node
					copy(newChildren[index+1:], current.Children[index:])
					current.Children = newChildren

					current = node

					// There is a case where locationToTreeNodes returns the node pointing to its parent,
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

	flamegraph := &pb.Flamegraph{
		Root: &pb.FlamegraphRootNode{
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

	return aggregateByFunction(flamegraph), nil
}

type tableConverter struct {
	stringsSlice   []string
	stringsIndex   map[string]int
	mappingsSlice  []*metastorev1alpha1.Mapping
	mappingsIndex  map[string]int
	locationsSlice []*metastorev1alpha1.Location
	locationsIndex map[string]int
	functionsSlice []*metastorev1alpha1.Function
	functionsIndex map[string]int

	stringsDedup   int
	mappingDedup   int
	locationsDedup int
	functionsDedup int
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

func (c *tableConverter) Locations() []*metastorev1alpha1.Location {
	return c.locationsSlice
}

func (c *tableConverter) Functions() []*metastorev1alpha1.Function {
	return c.functionsSlice
}

// AddString to the string table and return the strings index in the table.
func (c *tableConverter) AddString(s string) int {
	if i, ok := c.stringsIndex[s]; ok {
		c.stringsDedup++
		return i
	}
	c.stringsSlice = append(c.stringsSlice, s)
	c.stringsIndex[s] = len(c.stringsSlice) - 1
	return c.stringsIndex[s]
}

// AddMapping to the mappings table and return the mappings index in the table.
func (c *tableConverter) AddMapping(m *metastorev1alpha1.Mapping) int {
	if m == nil {
		return 0
	}
	if i, ok := c.mappingsIndex[m.Id]; ok {
		return i
	}

	_ = c.AddString(m.File)
	_ = c.AddString(m.BuildId)

	c.mappingsSlice = append(c.mappingsSlice, m)
	c.mappingsIndex[m.Id] = len(c.mappingsSlice) - 1
	return c.mappingsIndex[m.Id]
}

func (c *tableConverter) AddLocation(l *profile.Location) int {
	if i, ok := c.locationsIndex[l.ID]; ok {
		c.locationsDedup++
		return i
	}

	var mid int
	if l.Mapping != nil {
		// TODO: Assumes it's there, might panic
		mid = c.mappingsIndex[l.Mapping.Id]
	}

	lines := make([]*metastorev1alpha1.Line, 0, len(l.Lines))
	for _, line := range l.Lines {
		c.AddFunction(line.Function)
	}

	msl := &metastorev1alpha1.Location{
		// Id Not important for the frontend
		Address:   l.Address,
		MappingId: strconv.Itoa(mid),
		IsFolded:  l.IsFolded,
		Lines:     lines,
	}

	c.locationsSlice = append(c.locationsSlice, msl)
	c.locationsIndex[l.ID] = len(c.locationsSlice) - 1
	return c.locationsIndex[l.ID]
}

func (c *tableConverter) AddFunction(f *metastorev1alpha1.Function) int {
	if i, ok := c.functionsIndex[f.Id]; ok {
		c.functionsDedup++
		return i
	}

	c.AddString(f.Name)
	c.AddString(f.Filename)
	c.AddString(f.SystemName)

	c.functionsSlice = append(c.functionsSlice, f)
	c.functionsIndex[f.Id] = len(c.functionsSlice) - 1
	return c.functionsIndex[f.Id]
}
