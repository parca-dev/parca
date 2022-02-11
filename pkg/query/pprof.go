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

package query

import (
	"context"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/metastore"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
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

func GenerateFlatPprof(ctx context.Context, metaStore metastore.ProfileMetaStore, ip *parcaprofile.StacktraceSamples) (*profile.Profile, error) {
	meta := ip.Meta

	mappingByID := map[string]*profile.Mapping{}
	functionByID := map[string]*profile.Function{}
	locationByID := map[string]*profile.Location{}

	p := &profile.Profile{
		PeriodType:    &profile.ValueType{Type: meta.PeriodType.Type, Unit: meta.PeriodType.Unit},
		SampleType:    []*profile.ValueType{{Type: meta.SampleType.Type, Unit: meta.SampleType.Unit}},
		TimeNanos:     meta.Timestamp * 1000000, // We store timestamps in millisecond not nanoseconds.
		DurationNanos: meta.Duration,
		Period:        meta.Period,
	}

	for _, s := range ip.Samples {
		locations := make([]*profile.Location, 0, len(s.Location))
		for _, l := range s.Location {
			if loc, ok := locationByID[string(l.ID[:])]; ok {
				locations = append(locations, loc)
				continue
			}

			var (
				pm *profile.Mapping
				ok bool
			)
			if l.Mapping != nil {
				if pm, ok = mappingByID[string(l.Mapping.Id)]; !ok {
					lm := l.Mapping
					pm := &profile.Mapping{
						ID:              0, // set later
						Start:           lm.Start,
						Limit:           lm.Limit,
						Offset:          lm.Offset,
						File:            lm.File,
						BuildID:         lm.BuildId,
						HasFunctions:    lm.HasFunctions,
						HasFilenames:    lm.HasFilenames,
						HasLineNumbers:  lm.HasLineNumbers,
						HasInlineFrames: lm.HasInlineFrames,
					}
					mappingByID[string(lm.Id)] = pm
				}
			}

			lines := make([]profile.Line, 0, len(l.Lines))
			for _, line := range l.Lines {
				var (
					f  *profile.Function
					ok bool
				)
				if line.Function != nil {
					lf := line.Function
					if f, ok = functionByID[string(lf.Id)]; !ok {
						f = &profile.Function{
							ID:         0, // set later
							Name:       lf.Name,
							SystemName: lf.SystemName,
							Filename:   lf.Filename,
							StartLine:  lf.StartLine,
						}
						functionByID[string(lf.Id)] = f
					}
				}
				lines = append(lines, profile.Line{
					Function: f,
					Line:     line.Line,
				})
			}

			addr := l.Address
			if pm != nil {
				addr += pm.Offset
			}

			pl := &profile.Location{
				ID:       0,
				Mapping:  pm,
				Address:  addr,
				Line:     lines,
				IsFolded: l.IsFolded,
			}
			locationByID[string(l.ID[:])] = pl
			locations = append(locations, pl)
		}

		p.Sample = append(p.Sample, &profile.Sample{
			Value:    []int64{s.Value},
			Location: locations,
			Label:    s.Label,
			NumLabel: s.NumLabel,
			NumUnit:  s.NumUnit,
		})
	}

	mappings := make([]*profile.Mapping, 0, len(mappingByID))
	mi := uint64(1)
	for _, m := range mappingByID {
		m.ID = mi
		mappings = append(mappings, m)
		mi++
	}
	p.Mapping = mappings

	functions := make([]*profile.Function, 0, len(functionByID))
	fi := uint64(1)
	for _, f := range functionByID {
		f.ID = fi
		functions = append(functions, f)
		fi++
	}
	p.Function = functions

	locations := make([]*profile.Location, 0, len(locationByID))
	li := uint64(1)
	for _, l := range locationByID {
		l.ID = li
		locations = append(locations, l)
		li++
	}
	p.Location = locations

	for _, s := range p.Sample {
		for _, l := range s.Location {
			if l.ID == 0 {
				panic("location id is 0")
			}
		}
	}

	return p, nil
}
