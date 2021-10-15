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

package symbol

import (
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/symbol/addr2line"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

type Symbolizer struct {
	logger log.Logger

	demangler *demangle.Demangler
}

type liner interface {
	PCToLines(pc uint64) ([]profile.Line, error)
}

type funcLiner func(addr uint64) ([]profile.Line, error)

func (f funcLiner) PCToLines(pc uint64) ([]profile.Line, error) { return f(pc) }

func NewSymbolizer(logger log.Logger, demangleMode ...string) *Symbolizer {
	var dm string
	if len(demangleMode) > 0 {
		dm = demangleMode[0]
	}
	return &Symbolizer{
		logger:    log.With(logger, "component", "symbolizer"),
		demangler: demangle.NewDemangler(dm, false),
	}
}

// TODO(kakkoyun): Do we still need mapping? What is the actual usecase?
func (s *Symbolizer) NewLiner(m *profile.Mapping, file string) (liner, error) {
	hasDWARF, err := elfutils.HasDWARF(file)
	if err != nil {
		level.Debug(s.logger).Log(
			"msg", "failed to determine if binary has DWARF info",
			"file", file,
			"err", err,
		)
	}
	if hasDWARF {
		level.Debug(s.logger).Log("msg", "using DWARF to resolve symbols", "file", file)
		f, err := addr2line.DWARF(s.demangler, m, file)
		if err != nil {
			level.Error(s.logger).Log(
				"msg", "failed to open object file",
				"file", file,
				"start", m.Start,
				"limit", m.Limit,
				"offset", m.Offset,
				"err", err,
			)
			return nil, err
		}
		return funcLiner(f), nil
	}

	// Go binaries has a special case. They use ".gopclntab" section to symbolize addresses.
	// Keep that section and other identifying sections in the debug information file.
	isGo, err := elfutils.IsSymbolizableGoObjFile(file)
	if err != nil {
		level.Debug(s.logger).Log(
			"msg", "failed to determine if binary is a Go binary",
			"file", file,
			"err", err,
		)
	}
	if isGo {
		// Right now, this uses "debug/gosym" package, and it won't work for inlined functions,
		// so this is just a best-effort implementation, in case we don't have DWARF.
		level.Debug(s.logger).Log("msg", "symbolizing a Go binary", "file", file)
		f, err := addr2line.Go(file)
		if err == nil {
			level.Debug(s.logger).Log("msg", "using go liner to resolve symbols", "file", file)
			return funcLiner(f), nil
		}

		level.Error(s.logger).Log(
			"msg", "failed to create go liner, falling back to binary liner",
			"file", file,
			"err", err,
		)
	}

	// Just in case, underlying DWARF can symbolize addresses.
	level.Debug(s.logger).Log("msg", "falling back to DWARF liner resolve symbols", "file", file)
	f, err := addr2line.DWARF(s.demangler, m, file)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to open object file",
			"file", file,
			"start", m.Start,
			"limit", m.Limit,
			"offset", m.Offset,
			"err", err,
		)
		return nil, err
	}
	return funcLiner(f), nil
}
