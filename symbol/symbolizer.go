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
	"fmt"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/google/pprof/profile"
)

type Symbolizer struct {
	logger log.Logger
	c      *SymbolServerClient
}

func NewSymbolizer(logger log.Logger, c *SymbolServerClient) *Symbolizer {
	return &Symbolizer{
		logger: logger,
		c:      c,
	}
}

func (s *Symbolizer) Symbolize(ctx context.Context, p *profile.Profile) error {
	mappings := map[uint64]*profile.Mapping{}
	locationIdxs := map[int]struct{}{}
	for i, location := range p.Location {
		if len(location.Line) == 0 && location.Mapping != nil {
			locationIdxs[i] = struct{}{}
			mappings[location.Mapping.ID] = location.Mapping
		}
	}

	if len(locationIdxs) == 0 {
		// Nothing to symbolize.
		return nil
	}

	r := &SymbolicateRequest{
		Stacktraces: []Stacktrace{{
			Frames: make([]Frame, 0, len(locationIdxs)),
		}},
		Modules: make([]Module, 0, len(mappings)),
	}

	locationAddrToIdx := map[string]int{}
	for locationIdx := range locationIdxs {
		locationAddr := fmt.Sprintf("0x%x", p.Location[locationIdx].Address)
		locationAddrToIdx[locationAddr] = locationIdx
		r.Stacktraces[0].Frames = append(r.Stacktraces[0].Frames, Frame{
			InstructionAddr: locationAddr,
		})
	}

	for _, mapping := range mappings {
		r.Modules = append(r.Modules, Module{
			Type:      "elf",
			CodeID:    mapping.BuildID,
			ImageAddr: fmt.Sprintf("0x%x", mapping.Start),
		})
	}

	level.Debug(s.logger).Log("msg", "remote symbolization request")
	res, err := s.c.Symbolicate(ctx, r)
	if err != nil {
		return err
	}

	functionIdx := map[string]int{}
	for _, stacktrace := range res.Stacktraces {
		for _, frame := range stacktrace.Frames {
			var f *profile.Function
			fIdx, ok := functionIdx[frame.Function+":"+frame.AbsPath]
			if !ok {
				f = &profile.Function{
					ID:       uint64(len(p.Function)) + 1,
					Name:     frame.Function,
					Filename: frame.AbsPath,
				}
				p.Function = append(p.Function, f)
				fIdx = len(p.Function) - 1
				functionIdx[frame.Function+":"+frame.AbsPath] = fIdx
			}
			f = p.Function[fIdx]
			l := p.Location[locationAddrToIdx[frame.InstructionAddr]]
			l.Line = append(l.Line, profile.Line{
				Function: f,
				Line:     int64(frame.LineNo),
			})
		}
	}

	return nil
}
