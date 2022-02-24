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
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/cespare/xxhash/v2"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/goburrow/cache"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/metastore"
	"github.com/parca-dev/parca/pkg/symbol/addr2line"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var ErrLinerFailedBefore = errors.New("failed to initialize liner")

type Symbolizer struct {
	logger    log.Logger
	demangler *demangle.Demangler

	cache     cache.Cache
	cacheOpts []cache.Option

	failed           map[string]struct{}
	attemptThreshold int
}

type liner interface {
	PCToLines(pc uint64) ([]metastore.LocationLine, error)
}

func NewSymbolizer(logger log.Logger, opts ...Option) (*Symbolizer, error) {
	log.With(logger, "component", "symbolizer")

	const (
		defaultDemangleMode     = "simple"
		defaultCacheSize        = 1000
		defaultCacheItemTTL     = time.Minute
		defaultAttemptThreshold = 3
	)

	sym := &Symbolizer{
		logger:    logger,
		demangler: demangle.NewDemangler(defaultDemangleMode, false),

		// e.g: Parca binary compressed DWARF data size ~8mb as of 10.2021
		cacheOpts: []cache.Option{
			cache.WithMaximumSize(defaultCacheSize),
			cache.WithExpireAfterAccess(defaultCacheItemTTL),
		},

		failed:           map[string]struct{}{},
		attemptThreshold: defaultAttemptThreshold,
	}
	for _, opt := range opts {
		opt(sym)
	}
	sym.cache = cache.New(sym.cacheOpts...)

	return sym, nil
}

func (s *Symbolizer) NewLiner(m *pb.Mapping, path string) (liner, error) {
	h, err := hash(path)
	if err != nil {
		level.Warn(s.logger).Log("msg", "failed to generate cache key", "err", err)
		h = path
	}

	// Check if we already attempt to build a liner for this path.
	if _, failedBefore := s.failed[h]; failedBefore {
		level.Debug(s.logger).Log("msg", "already failed to create liner for this debug info file, skipping")
		return nil, ErrLinerFailedBefore
	}

	if val, ok := s.cache.GetIfPresent(h); ok {
		level.Debug(s.logger).Log("msg", "using cached liner to resolve symbols", "file", path)
		return val.(liner), nil
	}

	lnr, err := s.newLiner(m, path)
	if err != nil {
		s.failed[h] = struct{}{}
		s.cache.Invalidate(h)
		return nil, err
	}

	level.Debug(s.logger).Log("msg", "liner cached", "file", path)
	s.cache.Put(h, lnr)
	return lnr, nil
}

func (s *Symbolizer) Close() error {
	return s.cache.Close()
}

func (s *Symbolizer) newLiner(m *pb.Mapping, path string) (liner, error) {
	logger := log.With(s.logger, "file", path, "buildid", m.BuildId)
	hasDWARF, err := elfutils.HasDWARF(path)
	if err != nil {
		level.Debug(logger).Log("msg", "failed to determine if binary has DWARF info", "err", err)
	}
	if hasDWARF {
		level.Debug(logger).Log("msg", "using DWARF liner to resolve symbols")
		lnr, err := addr2line.DWARF(s.logger, path, m, s.demangler, s.attemptThreshold)
		if err != nil {
			level.Error(logger).Log(
				"msg", "failed to open object file",
				"start", m.Start,
				"limit", m.Limit,
				"offset", m.Offset,
				"err", err,
			)
			return nil, err
		}
		return lnr, nil
	}

	// Go binaries has a special case. They use ".gopclntab" section to symbolize addresses.
	// Keep that section and other identifying sections in the debug information file.
	isGo, err := elfutils.IsSymbolizableGoObjFile(path)
	if err != nil {
		level.Debug(s.logger).Log("msg", "failed to determine if binary is a Go binary", "err", err)
	}
	if isGo {
		// Right now, this uses "debug/gosym" package, and it won't work for inlined functions,
		// so this is just a best-effort implementation, in case we don't have DWARF.
		level.Debug(s.logger).Log("msg", "symbolizing a Go binary", "file")
		lnr, err := addr2line.Go(s.logger, path)
		if err == nil {
			level.Debug(s.logger).Log("msg", "using go liner to resolve symbols", "file")
			return lnr, nil
		}
		level.Error(s.logger).Log("msg", "failed to create go liner, falling back to binary liner", "err", err)
	}

	return nil, errors.New("cannot create a liner from given object file")
}

func hash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", fmt.Errorf("failed to hash debug info file: %w", err)
	}
	return string(h.Sum(nil)), nil
}
