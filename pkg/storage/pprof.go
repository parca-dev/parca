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

	"github.com/google/pprof/profile"
)

type LocationStack []*profile.Location

func (s *LocationStack) Push(e *profile.Location) {
	*s = append(*s, e)
}

func (s *LocationStack) Pop() (*profile.Location, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *LocationStack) Peek() *profile.Location {
	return (*s)[len(*s)-1]
}

func (s *LocationStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *LocationStack) Size() int {
	return len(*s)
}

func (s *LocationStack) ToLocationStacktrace() []*profile.Location {
	a := make([]*profile.Location, len(*s))
	copy(a, *s)

	// Reverse it because the leaf needs to be first in the pprof profile.
	for i, j := 0, len(a)-1; i < j; i, j = i+1, j-1 {
		a[i], a[j] = a[j], a[i]
	}

	return a
}

func generatePprof(ctx context.Context, locationStore Locations, ip InstantProfile) (*profile.Profile, error) {
	meta := ip.ProfileMeta()

	mappingByID := map[uint64]*profile.Mapping{}
	functionByID := map[uint64]*profile.Function{}
	locationByID := map[uint64]*profile.Location{}
	p := &profile.Profile{
		PeriodType:    &profile.ValueType{Type: meta.PeriodType.Type, Unit: meta.PeriodType.Unit},
		SampleType:    []*profile.ValueType{{Type: meta.SampleType.Type, Unit: meta.SampleType.Unit}},
		TimeNanos:     meta.Timestamp * 1000000, // We store timestamps in millisecond not nanoseconds.
		DurationNanos: meta.Duration,
		Period:        meta.Period,
	}

	pt := CopyInstantProfileTree(ip.ProfileTree())
	skippedFirst := false
	err := WalkProfileTree(pt, func(n InstantProfileTreeNode) error {
		if !skippedFirst {
			// Need to do this to skip the first tree node which is the root as
			// it only has an artificial Location ID.
			skippedFirst = true
			return nil
		}
		id := n.LocationID()
		_, seenLocation := locationByID[id]
		if !seenLocation {
			loc, err := locationStore.GetLocationByID(ctx, id)
			if err != nil {
				return err
			}

			var mapping *profile.Mapping
			if loc.Mapping != nil {
				var seenMapping bool
				mapping, seenMapping = mappingByID[loc.Mapping.ID]
				if !seenMapping {
					mapping = &profile.Mapping{
						ID:              uint64(len(p.Mapping) + 1),
						Start:           loc.Mapping.Start,
						Limit:           loc.Mapping.Limit,
						Offset:          loc.Mapping.Offset,
						File:            loc.Mapping.File,
						BuildID:         loc.Mapping.BuildID,
						HasFunctions:    loc.Mapping.HasFunctions,
						HasFilenames:    loc.Mapping.HasFilenames,
						HasLineNumbers:  loc.Mapping.HasLineNumbers,
						HasInlineFrames: loc.Mapping.HasInlineFrames,
					}
					p.Mapping = append(p.Mapping, mapping)
					mappingByID[loc.Mapping.ID] = mapping
				}
			}

			var lines []profile.Line
			for _, line := range loc.Line {
				if line.Function != nil {
					function, seenFunction := functionByID[line.Function.ID]
					if !seenFunction {
						function = &profile.Function{
							ID:         uint64(len(p.Function) + 1),
							Name:       line.Function.Name,
							SystemName: line.Function.SystemName,
							Filename:   line.Function.Filename,
							StartLine:  line.Function.StartLine,
						}
						p.Function = append(p.Function, function)
						functionByID[line.Function.ID] = function
					}
					lines = append(lines, profile.Line{
						Function: function,
						Line:     line.Line,
					})
				}
			}

			location := &profile.Location{
				ID:      uint64(len(p.Location) + 1),
				Mapping: mapping,
				// TODO: Is this right?
				Address:  loc.Address + mapping.Offset,
				Line:     lines,
				IsFolded: loc.IsFolded,
			}
			p.Location = append(p.Location, location)
			locationByID[id] = location
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	it := pt.Iterator()

	if !it.HasMore() || !it.NextChild() {
		return nil, nil
	}

	n := it.At()
	loc := n.LocationID()
	if loc != uint64(0) {
		return nil, errors.New("expected root node to be first node returned by iterator")
	}

	it.StepInto()

	stack := LocationStack{}
	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			id := node.LocationID()
			l, found := locationByID[id]
			if !found {
				return nil, fmt.Errorf("unknown location ID %v", id)
			}
			if l == nil {
				return nil, fmt.Errorf("nil location for ID %v", id)
			}

			stack.Push(l)
			flatValues := node.FlatValues()
			var stacktrace []*profile.Location
			if len(flatValues) > 0 {
				// Only calculate once per flat value.
				stacktrace = stack.ToLocationStacktrace()
			}

			for _, flatValue := range flatValues {
				p.Sample = append(p.Sample, &profile.Sample{
					Location: stacktrace,
					Value:    []int64{flatValue.Value},
					Label:    flatValue.Label,
					NumLabel: flatValue.NumLabel,
					NumUnit:  flatValue.NumUnit,
				})
			}

			it.StepInto()
			continue
		}
		it.StepUp()
		stack.Pop()
	}

	err = p.CheckValid()
	if err != nil {
		return nil, err
	}

	return p, nil
}
