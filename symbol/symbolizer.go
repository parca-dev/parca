package symbol

import (
	"context"
	"fmt"

	"github.com/google/pprof/profile"
)

type Symbolizer struct {
	c *SymbolServerClient
}

func NewSymbolizer(c *SymbolServerClient) *Symbolizer {
	return &Symbolizer{
		c: c,
	}
}

func (s *Symbolizer) Symbolize(ctx context.Context, p *profile.Profile) error {
	mappings := map[uint64]*profile.Mapping{}
	locationIdxs := map[int]struct{}{}
	for i, location := range p.Location {
		if len(location.Line) == 0 {
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
		locationAddr := fmt.Sprintf("%x", p.Location[locationIdx].Address)
		locationAddrToIdx[locationAddr] = locationIdx
		r.Stacktraces[0].Frames = append(r.Stacktraces[0].Frames, Frame{
			InstructionAddr: locationAddr,
		})
	}

	for _, mapping := range mappings {
		r.Modules = append(r.Modules, Module{
			Type:      "elf",
			CodeID:    mapping.BuildID,
			ImageAddr: fmt.Sprintf("%x", mapping.Start),
		})
	}

	res, err := s.c.Symbolicate(ctx, r)
	if err != nil {
		return err
	}

	functionIdx := map[string]int{}
	for _, stacktrace := range res.Stacktraces {
		for _, frame := range stacktrace.Frames {
			var f *profile.Function
			fIdx, ok := functionIdx[fmt.Sprintf("%s:%s", frame.Function, frame.AbsPath)]
			if !ok {
				f = &profile.Function{
					ID:       uint64(len(p.Function)),
					Name:     frame.Function,
					Filename: frame.AbsPath,
				}
				p.Function = append(p.Function, f)
				fIdx = len(p.Function) - 1
				functionIdx[fmt.Sprintf("%s:%s", frame.Function, frame.AbsPath)] = fIdx
			}
			l := p.Location[locationAddrToIdx[frame.InstructionAddr]]
			l.Line = append(l.Line, profile.Line{
				Function: f,
				Line:     int64(frame.LineNo),
			})
		}
	}

	return nil
}
