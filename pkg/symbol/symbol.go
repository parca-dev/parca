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
	"fmt"
	"io"
	"os"

	"github.com/cespare/xxhash/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	lru "github.com/hashicorp/golang-lru"
	"github.com/hashicorp/golang-lru/simplelru"
	"github.com/parca-dev/parca/pkg/symbol/addr2line"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

type Symbolizer struct {
	logger    log.Logger
	cache     simplelru.LRUCache
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
	var cache simplelru.LRUCache
	// e.g: Parca binary compressed DWARF data size ~8mb as of 10.2021
	cache, err := lru.New(50) // Totally arbitrary.
	if err != nil {
		level.Error(logger).Log("msg", "failed to initialize liner cache", "err", err)
		cache = noopLinerCache{}
	}
	return &Symbolizer{
		logger:    log.With(logger, "component", "symbolizer"),
		cache:     cache,
		demangler: demangle.NewDemangler(dm, false),
	}
}

func (s *Symbolizer) NewLiner(m *profile.Mapping, path string) (liner, error) {
	hasDWARF, err := elfutils.HasDWARF(path)
	if err != nil {
		level.Debug(s.logger).Log(
			"msg", "failed to determine if binary has DWARF info",
			"file", path,
			"err", err,
		)
	}
	cacheKey, err := cacheKey(path)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to generate cache key", "err", err)
		cacheKey = path
	}
	if hasDWARF {
		level.Debug(s.logger).Log("msg", "using DWARF to resolve symbols", "file", path)
		if val, ok := s.cache.Get(cacheKey); ok {
			return val.(liner), nil
		}
		lnr, err := addr2line.DWARF(s.demangler, m, path)
		if err != nil {
			level.Error(s.logger).Log(
				"msg", "failed to open object file",
				"file", path,
				"start", m.Start,
				"limit", m.Limit,
				"offset", m.Offset,
				"err", err,
			)
			return nil, err
		}
		s.cache.Add(cacheKey, lnr)
		return lnr, nil
	}

	// Go binaries has a special case. They use ".gopclntab" section to symbolize addresses.
	// Keep that section and other identifying sections in the debug information file.
	isGo, err := elfutils.IsSymbolizableGoObjFile(path)
	if err != nil {
		level.Debug(s.logger).Log(
			"msg", "failed to determine if binary is a Go binary",
			"file", path,
			"err", err,
		)
	}
	if isGo {
		// Right now, this uses "debug/gosym" package, and it won't work for inlined functions,
		// so this is just a best-effort implementation, in case we don't have DWARF.
		level.Debug(s.logger).Log("msg", "symbolizing a Go binary", "file", path)
		if val, ok := s.cache.Get(cacheKey); ok {
			return val.(liner), nil
		}
		f, err := addr2line.Go(path)
		if err == nil {
			level.Debug(s.logger).Log("msg", "using go liner to resolve symbols", "file", path)
			s.cache.Add(cacheKey, funcLiner(f))
			return funcLiner(f), nil
		}
		level.Error(s.logger).Log(
			"msg", "failed to create go liner, falling back to binary liner",
			"file", path,
			"err", err,
		)
	}

	// Just in case, underlying DWARF can symbolize addresses.
	level.Debug(s.logger).Log("msg", "falling back to DWARF liner resolve symbols", "file", path)
	if val, ok := s.cache.Get(cacheKey); ok {
		return val.(liner), nil
	}
	lnr, err := addr2line.DWARF(s.demangler, m, path)
	if err != nil {
		level.Error(s.logger).Log("msg", "failed to open object file",
			"file", path,
			"start", m.Start,
			"limit", m.Limit,
			"offset", m.Offset,
			"err", err,
		)
		return nil, err
	}
	s.cache.Add(cacheKey, lnr)
	return lnr, nil
}

func cacheKey(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash debug info file: %w", err)
	}
	return string(h.Sum(nil)), nil
}

type noopLinerCache struct {
}

func (n noopLinerCache) Add(key, value interface{}) bool {
	return false
}

func (n noopLinerCache) Get(key interface{}) (value interface{}, ok bool) {
	return nil, false
}

func (n noopLinerCache) Contains(key interface{}) (ok bool) {
	return false
}

func (n noopLinerCache) Peek(key interface{}) (value interface{}, ok bool) {
	return nil, false
}

func (n noopLinerCache) Remove(key interface{}) bool {
	return false
}

func (n noopLinerCache) RemoveOldest() (interface{}, interface{}, bool) {
	return nil, nil, false
}

func (n noopLinerCache) GetOldest() (interface{}, interface{}, bool) {
	return nil, nil, false
}

func (n noopLinerCache) Keys() []interface{} {
	return nil
}

func (n noopLinerCache) Len() int {
	return 0
}

func (n noopLinerCache) Purge() {
}

func (n noopLinerCache) Resize(i int) int {
	return 0
}
