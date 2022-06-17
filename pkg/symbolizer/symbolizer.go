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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/hashicorp/go-multierror"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/runutil"
)

type Symbolizer struct {
	logger log.Logger

	metastore pb.MetastoreServiceClient
	debugInfo *debuginfo.Store
}

func New(logger log.Logger, metastore pb.MetastoreServiceClient, info *debuginfo.Store) *Symbolizer {
	return &Symbolizer{
		logger:    log.With(logger, "component", "symbolizer"),
		metastore: metastore,
		debugInfo: info,
	}
}

func (s *Symbolizer) Run(ctx context.Context, interval time.Duration) error {
	return runutil.Repeat(interval, ctx.Done(), func() error {
		lres, err := s.metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{})
		if err != nil {
			return err
		}
		if len(lres.Locations) == 0 {
			// Nothing to symbolize.
			return nil
		}

		err = s.symbolize(ctx, lres.Locations)
		if err != nil {
			level.Error(s.logger).Log("msg", "symbolization attempt finished with errors", "err", err)
		}
		return nil
	})
}

// UnsymbolizableMapping returns true if a mapping points to a binary for which
// locations can't be symbolized in principle, at least now. Examples are
// "[vdso]", [vsyscall]" and some others, see the code.
func UnsymbolizableMapping(m *pb.Mapping) bool {
	name := filepath.Base(m.File)
	return strings.HasPrefix(name, "[") || strings.HasPrefix(name, "linux-vdso") || strings.HasPrefix(m.File, "/dev/dri/")
}

type MappingLocations struct {
	Mapping   *pb.Mapping
	Locations []*pb.Location

	// LocationsLines is a list of lines per location.
	LocationsLines [][]profile.LocationLine
}

func (s *Symbolizer) symbolize(ctx context.Context, locations []*pb.Location) error {
	mappingsIndex := map[string]int{}
	mappingIDs := []string{}
	for _, loc := range locations {
		if _, ok := mappingsIndex[loc.MappingId]; !ok {
			mappingIDs = append(mappingIDs, loc.MappingId)
			mappingsIndex[loc.MappingId] = len(mappingIDs) - 1
		}
	}

	mres, err := s.metastore.Mappings(ctx, &pb.MappingsRequest{MappingIds: mappingIDs})
	if err != nil {
		return err
	}

	// Aggregate locations per mapping to get prepared for batch request.
	locationsByMappings := make([]*MappingLocations, len(mres.Mappings))
	for i, m := range mres.Mappings {
		locationsByMappings[i] = &MappingLocations{Mapping: m}
	}

	for _, loc := range locations {
		locationsByMapping := locationsByMappings[mappingsIndex[loc.MappingId]]
		mapping := locationsByMapping.Mapping
		// If Mapping or Mapping.BuildID is empty, we cannot associate an object file with functions.
		if mapping == nil || len(mapping.BuildId) == 0 || UnsymbolizableMapping(mapping) {
			level.Debug(s.logger).Log("msg", "mapping of location is empty, skipping")
			continue
		}
		// Already symbolized!
		if loc.Lines != nil && len(loc.Lines.Entries) > 0 {
			level.Debug(s.logger).Log("msg", "location already symbolized, skipping")
			continue
		}
		locationsByMapping.Locations = append(locationsByMapping.Locations, loc)
	}

	var result *multierror.Error
	for _, locationsByMapping := range locationsByMappings {
		mapping := locationsByMapping.Mapping
		locations := locationsByMapping.Locations
		logger := log.With(s.logger, "buildid", mapping.BuildId)
		level.Debug(logger).Log("msg", "storage symbolization request started")

		// Symbolize returns a list of lines per location passed to it.
		locationsByMapping.LocationsLines, err = s.debugInfo.Symbolize(ctx, mapping, locations)
		if err != nil {
			result = multierror.Append(result, fmt.Errorf("storage symbolization request failed: %w", err))
			continue
		}
		level.Debug(logger).Log("msg", "storage symbolization request done")
	}
	err = result.ErrorOrNil()
	if err != nil {
		return err
	}

	numFunctions := 0
	for _, locationsByMapping := range locationsByMappings {
		for _, locationLines := range locationsByMapping.LocationsLines {
			numFunctions += len(locationLines)
		}
	}

	functions := make([]*pb.Function, numFunctions)
	i := 0
	for _, locationsByMapping := range locationsByMappings {
		for _, locationLines := range locationsByMapping.LocationsLines {
			for _, line := range locationLines {
				functions[i] = line.Function
				i++
			}
		}
	}

	fres, err := s.metastore.GetOrCreateFunctions(ctx, &pb.GetOrCreateFunctionsRequest{Functions: functions})
	if err != nil {
		return err
	}

	i = 0
	for _, locationsByMapping := range locationsByMappings {
		for j, locationLines := range locationsByMapping.LocationsLines {
			lines := make([]*pb.Line, 0, len(locationLines))
			for _, line := range locationLines {
				lines = append(lines, &pb.Line{
					FunctionId: fres.Functions[i].Id,
					Line:       line.Line,
				})

				i++
			}
			// Update the location with the lines in-place so that in the next
			// step we can just reuse the same locations as were originally
			// passed in.
			locationsByMapping.Locations[j].Lines = &pb.LocationLines{Entries: lines}
		}
	}

	// At this point the locations are symbolized in-place and we can send them to the metastore.
	_, err = s.metastore.CreateLocationLines(ctx, &pb.CreateLocationLinesRequest{
		Locations: locations,
	})

	return err
}
