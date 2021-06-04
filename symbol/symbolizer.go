// Copyright 2021 The conprof Authors
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

package symbol

import (
	"context"

	"github.com/conprof/conprof/pkg/store/storepb"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/profile"
)

type Symbolizer struct {
	logger log.Logger
	c      SymbolizeClient
}

type SymbolizeClient interface {
	Symbolize(context.Context, *storepb.SymbolizeRequest) (*storepb.SymbolizeResponse, error)
}

func NewSymbolizer(logger log.Logger, c SymbolizeClient) *Symbolizer {
	return &Symbolizer{
		logger: logger,
		c:      c,
	}
}

func (s *Symbolizer) Symbolize(ctx context.Context, p *profile.Profile) error {
	mappingIndices := map[string]int{}
	mappings := []*storepb.Mapping{}
	for _, location := range p.Location {
		if len(location.Line) == 0 && location.Mapping != nil && len(location.Mapping.BuildID) > 0 {
			mappingIdx, ok := mappingIndices[location.Mapping.BuildID]
			if !ok {
				mappingIdx = len(mappings)
				mappingIndices[location.Mapping.BuildID] = mappingIdx
				mappings = append(mappings, &storepb.Mapping{
					BuildId:     location.Mapping.BuildID,
					MemoryStart: location.Mapping.Start,
					MemoryLimit: location.Mapping.Limit,
					FileOffset:  location.Mapping.Offset,
				})
			}
			mapping := mappings[mappingIdx]
			mapping.Locations = append(mapping.Locations, &storepb.Location{
				Address: location.Address,
			})
		}
	}

	if len(mappings) == 0 {
		// Nothing to symbolize.
		return nil
	}

	level.Debug(s.logger).Log("msg", "remote symbolization request")
	res, err := s.c.Symbolize(ctx, &storepb.SymbolizeRequest{
		Mappings: mappings,
	})
	if err != nil {
		return err
	}

	mappingLocationIdx := map[string]map[uint64]int{}
	for _, mapping := range res.Mappings {
		locationIdx := map[uint64]int{}
		for i, location := range mapping.Locations {
			locationIdx[location.Address] = i
		}
		mappingLocationIdx[mapping.BuildId] = locationIdx
	}

	functionIdx := map[string]int{}
	for _, location := range p.Location {
		if len(location.Line) == 0 && location.Mapping != nil && len(location.Mapping.BuildID) > 0 {
			m := res.Mappings[mappingIndices[location.Mapping.BuildID]]
			locationIndices := mappingLocationIdx[location.Mapping.BuildID]

			l := m.Locations[locationIndices[location.Address]]
			for _, line := range l.Lines {
				var f *profile.Function
				fIdx, ok := functionIdx[line.Function.Name+":"+line.Function.Filename]
				if !ok {
					f = &profile.Function{
						ID:       uint64(len(p.Function)) + 1,
						Name:     line.Function.Name,
						Filename: line.Function.Filename,
					}
					p.Function = append(p.Function, f)
					fIdx = len(p.Function) - 1
					functionIdx[line.Function.Name+":"+line.Function.Filename] = fIdx
				}
				f = p.Function[fIdx]
				location.Line = append(location.Line, profile.Line{
					Function: f,
					Line:     line.Line,
				})
			}
		}
	}

	return nil
}
