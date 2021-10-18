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

package symbolizer

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/hashicorp/go-multierror"

	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/runutil"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type Symbolizer struct {
	logger    log.Logger
	locations metastore.LocationStore
	debugInfo *debuginfo.Store
}

func NewSymbolizer(logger log.Logger, loc metastore.LocationStore, info *debuginfo.Store) *Symbolizer {
	return &Symbolizer{
		logger:    log.With(logger, "component", "symbolizer"),
		locations: loc,
		debugInfo: info,
	}
}

func (s *Symbolizer) Run(ctx context.Context, interval time.Duration) error {
	return runutil.Repeat(interval, ctx.Done(), func() error {
		locations, err := s.locations.GetSymbolizableLocations(ctx)
		if err != nil {
			return err
		}
		if len(locations) == 0 {
			// Nothing to symbolize.
			return nil
		}

		err = s.symbolize(ctx, locations)
		if err != nil {
			level.Error(s.logger).Log("msg", "symbolization attempt failed", "err", err)
		}
		return nil
	})
}

func (s *Symbolizer) symbolize(ctx context.Context, locations []*profile.Location) error {
	// Aggregate locations per mapping to get prepared for batch request.
	mappings := map[uint64]*profile.Mapping{}
	mappingLocations := map[uint64][]*profile.Location{}
	for _, loc := range locations {
		// If Mapping or Mapping.BuildID is empty, we cannot associate an object file with functions.
		if loc.Mapping == nil || len(loc.Mapping.BuildID) == 0 || loc.Mapping.Unsymbolizable() {
			level.Debug(s.logger).Log("msg", "mapping of location is empty, skipping")
			continue
		}
		// Already symbolized!
		if len(loc.Line) > 0 {
			level.Debug(s.logger).Log("msg", "location already symbolized, skipping")
			continue
		}
		mappings[loc.Mapping.ID] = loc.Mapping
		mappingLocations[loc.Mapping.ID] = append(mappingLocations[loc.Mapping.ID], loc)
	}

	var result *multierror.Error
	for id, mapping := range mappings {
		level.Debug(s.logger).Log("msg", "storage symbolization request started", "buildid", mapping.BuildID)
		// TODO(kakkoyun): Cache failed symbolization attempts per location.
		symbolizedLines, err := s.debugInfo.Symbolize(ctx, mapping, mappingLocations[id]...)
		if err != nil {
			// It's ok if we don't have the symbols for given BuildID, it happens too often.
			if errors.Is(err, debuginfo.ErrDebugInfoNotFound) {
				level.Debug(s.logger).Log("msg", "failed to find the debug info in storage", "buildid", mapping.BuildID)
				continue
			}
			result = multierror.Append(result, fmt.Errorf("storage symbolization request failed: %w", err))
			continue
		}
		level.Debug(s.logger).Log("msg", "storage symbolization request done", "buildid", mapping.BuildID)

		for loc, lines := range symbolizedLines {
			loc.Line = lines
			// Only creates lines for given location.
			if err := s.locations.Symbolize(ctx, loc); err != nil {
				result = multierror.Append(result, fmt.Errorf("failed to update location %d: %w", loc.ID, err))
				continue
			}
		}
	}

	return result.ErrorOrNil()
}
