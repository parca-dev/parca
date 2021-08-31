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
	"context"
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
		logger:    logger,
		locations: loc,
		debugInfo: info,
	}
}

func (s *Symbolizer) Run(ctx context.Context, interval time.Duration) error {
	return runutil.Repeat(interval, ctx.Done(), func() error {
		// Get all unsymbolized locations.
		locations, err := s.locations.GetUnsymbolizedLocations()
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
	mappings := map[*profile.Mapping][]*profile.Location{}
	for _, loc := range locations {
		// If Mapping or Mapping.BuildID is empty, we cannot associate an object file with functions.
		if loc.Mapping == nil || len(loc.Mapping.BuildID) == 0 {
			level.Debug(s.logger).Log("msg", "mapping of location is empty, skipping")
			continue
		}
		// Already symbolized!
		if len(loc.Line) > 0 {
			level.Debug(s.logger).Log("msg", "location already symbolized, skipping")
			continue
		}
		mappings[loc.Mapping] = append(mappings[loc.Mapping], loc)
	}

	var result *multierror.Error
	for mapping, locations := range mappings {
		// Symbolize Locations using DebugInfoStore.
		level.Debug(s.logger).Log("msg", "storage symbolization request started")
		symbolizedLines, err := s.debugInfo.Symbolize(ctx, mapping, locations...)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("storage symbolization request failed: %w", err))
			continue
		}

		// Update LocationStore with found symbols.
		for loc, lines := range symbolizedLines {
			loc.Line = lines
			err := s.locations.UpdateLocation(loc)
			if err != nil {
				result = multierror.Append(result, fmt.Errorf("failed to update location: %w", err))
				continue
			}
		}
	}

	return result.ErrorOrNil()
}
