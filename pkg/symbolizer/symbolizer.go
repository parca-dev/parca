// Copyright 2022 The Parca Authors
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
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/runutil"
	"github.com/parca-dev/parca/pkg/symbol"
)

type DebuginfoMetadata interface {
	MarkAsNotValidELF(ctx context.Context, buildID string) error
}

type Symbolizer struct {
	logger log.Logger
	// attempts counts the total number of symbolication attempts.
	// It counts per batch.
	attempts prometheus.Counter
	// errors counts the total number of symbolication errors, partitioned by an error reason
	// such as failure to fetch unsymbolized locations.
	// It counts per batch.
	errors *prometheus.CounterVec
	// duration is a histogram to measure how long it takes to finish a symbolication round.
	// Note, a single observation is per batch.
	duration prometheus.Histogram
	// storeDuration is a histogram to measure how long it takes to store the symbolized locations.
	// Note, a single observation is per batch.
	storeDuration prometheus.Histogram

	metastore  pb.MetastoreServiceClient
	symbolizer *symbol.Symbolizer
	debuginfo  DebuginfoFetcher
	metadata   DebuginfoMetadata

	batchSize uint32

	tmpDir string
}

type DebuginfoFetcher interface {
	// Fetch ensures that the debug info for the given build ID is available on
	// a local filesystem and returns a path to it.
	FetchDebuginfo(ctx context.Context, buildID string) (io.ReadCloser, error)
}

func New(
	logger log.Logger,
	reg prometheus.Registerer,
	metadata DebuginfoMetadata,
	metastore pb.MetastoreServiceClient,
	debuginfo DebuginfoFetcher,
	symbolizer *symbol.Symbolizer,
	tmpDir string,
	batchSize uint32,
) *Symbolizer {
	attemptsTotal := promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "symbolizer_symbolication_attempts_total",
			Help: "Total number of symbolication attempts in batches.",
		},
	)
	errorsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "symbolizer_symbolication_errors_total",
			Help: "Total number of symbolication errors in batches, partitioned by an error reason.",
		},
		[]string{"reason"},
	)
	duration := promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "symbolizer_symbolication_duration_seconds",
			Help:    "How long it took in seconds to finish a round of the symbolication cycle in batches.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)
	storeDuration := promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "symbolizer_store_duration_seconds",
			Help:    "How long it took in seconds to store a batch of the symbolized locations.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)

	s := Symbolizer{
		logger:        log.With(logger, "component", "symbolizer"),
		attempts:      attemptsTotal,
		errors:        errorsTotal,
		duration:      duration,
		storeDuration: storeDuration,
		metastore:     metastore,
		symbolizer:    symbolizer,
		debuginfo:     debuginfo,
		tmpDir:        tmpDir,
		batchSize:     batchSize,
		metadata:      metadata,
	}
	return &s
}

func (s *Symbolizer) Run(ctx context.Context, interval time.Duration) error {
	return runutil.Repeat(interval, ctx.Done(), func() error {
		level.Debug(s.logger).Log("msg", "start symbolization cycle")
		s.runSymbolizationCycle(ctx)
		level.Debug(s.logger).Log("msg", "symbolization loop completed")
		return nil
	})
}

func (s *Symbolizer) runSymbolizationCycle(ctx context.Context) {
	var begin time.Time
	prevMaxKey := ""
	for {
		begin = time.Now()
		s.attempts.Inc()

		lres, err := s.metastore.UnsymbolizedLocations(ctx, &pb.UnsymbolizedLocationsRequest{
			Limit:  s.batchSize,
			MinKey: prevMaxKey,
		})
		if err != nil {
			level.Error(s.logger).Log("msg", "failed to fetch unsymbolized locations", "err", err)
			s.errors.WithLabelValues("fetch_unsymbolized_locations").Inc()
			s.duration.Observe(time.Since(begin).Seconds())
			// Try again on the next cycle.
			return
		}
		if len(lres.Locations) == 0 {
			level.Debug(s.logger).Log("msg", "no locations to symbolize")
			s.duration.Observe(time.Since(begin).Seconds())
			// Nothing to symbolize.
			return
		}
		prevMaxKey = lres.MaxKey

		level.Debug(s.logger).Log("msg", "attempting to symbolize locations", "count", len(lres.Locations))
		err = s.Symbolize(ctx, lres.Locations)
		if err != nil {
			level.Warn(s.logger).Log("msg", "symbolization attempt finished with errors")
			level.Debug(s.logger).Log("msg", "errors occurred during symbolization", "err", err)
		}
		s.duration.Observe(time.Since(begin).Seconds())

		if s.batchSize == 0 {
			// If batch size is 0 we won't continue with the next batch as we
			// should have already processed everything.
			return
		}
	}
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

func (s *Symbolizer) Symbolize(ctx context.Context, locations []*pb.Location) error {
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
		s.errors.WithLabelValues("get_mappings").Inc()
		return fmt.Errorf("get mappings: %w", err)
	}

	// Aggregate locations per mapping to get prepared for batch request.
	locationsByMappings := make([]*MappingLocations, len(mres.Mappings))
	for i, m := range mres.Mappings {
		locationsByMappings[i] = &MappingLocations{Mapping: m}
	}

	for _, loc := range locations {
		locationsByMapping := locationsByMappings[mappingsIndex[loc.MappingId]]
		// Already symbolized!
		if loc.Lines != nil && len(loc.Lines) > 0 {
			level.Debug(s.logger).Log("msg", "location already symbolized, skipping")
			continue
		}
		locationsByMapping.Locations = append(locationsByMapping.Locations, loc)
	}

	for _, locationsByMapping := range locationsByMappings {
		mapping := locationsByMapping.Mapping

		// If Mapping or Mapping.BuildID is empty, we cannot associate an object file with functions.
		if mapping == nil || len(mapping.BuildId) == 0 || UnsymbolizableMapping(mapping) {
			level.Debug(s.logger).Log("msg", "mapping of location is empty, skipping")
			continue
		}
		logger := log.With(s.logger, "buildid", mapping.BuildId)

		locations := locationsByMapping.Locations
		level.Debug(logger).Log("msg", "storage symbolization request started", "build_id_length", len(mapping.BuildId))
		// Symbolize returns a list of lines per location passed to it.
		locationsByMapping.LocationsLines, err = s.symbolizeLocationsForMapping(ctx, mapping, locations)
		if err != nil {
			level.Debug(logger).Log("msg", "storage symbolization request failed", "err", err)
			continue
		}
		level.Debug(logger).Log("msg", "storage symbolization request done")
	}

	numFunctions := 0
	for _, locationsByMapping := range locationsByMappings {
		for _, locationLines := range locationsByMapping.LocationsLines {
			numFunctions += len(locationLines)
		}
	}
	if numFunctions == 0 {
		level.Debug(s.logger).Log("msg", "nothing to store after symbolization")
		return nil
	}
	level.Debug(s.logger).Log("msg", "storing found symbols")

	functions := make([]*pb.Function, numFunctions)
	numLocations := 0
	i := 0
	for _, locationsByMapping := range locationsByMappings {
		for _, locationLines := range locationsByMapping.LocationsLines {
			if len(locationLines) == 0 {
				continue
			}
			numLocations++
			for _, line := range locationLines {
				functions[i] = line.Function
				i++
			}
		}
	}

	fres, err := s.metastore.GetOrCreateFunctions(ctx, &pb.GetOrCreateFunctionsRequest{Functions: functions})
	if err != nil {
		s.errors.WithLabelValues("get_or_create_functions").Inc()
		return fmt.Errorf("get or create functions: %w", err)
	}

	locations = make([]*pb.Location, 0, numLocations)
	i = 0
	for _, locationsByMapping := range locationsByMappings {
		for j, locationLines := range locationsByMapping.LocationsLines {
			if len(locationLines) == 0 {
				continue
			}
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
			locations = append(locations, locationsByMapping.Locations[j])
			locationsByMapping.Locations[j].Lines = lines
		}
	}

	// At this point the locations are symbolized in-place and we can send them to the metastore.
	defer func(begin time.Time) {
		s.storeDuration.Observe(time.Since(begin).Seconds())
	}(time.Now())
	_, err = s.metastore.CreateLocationLines(ctx, &pb.CreateLocationLinesRequest{
		Locations: locations,
	})
	if err != nil {
		s.errors.WithLabelValues("create_location_lines").Inc()
		return fmt.Errorf("create location lines: %w", err)
	}

	return nil
}

// symbolizeLocationsForMapping fetches the debug info for a given build ID and symbolizes it the
// given location.
func (s *Symbolizer) symbolizeLocationsForMapping(ctx context.Context, m *pb.Mapping, locations []*pb.Location) ([][]profile.LocationLine, error) {
	logger := log.With(s.logger, "buildid", m.BuildId)

	// Fetch the debug info for the build ID.
	rc, err := s.debuginfo.FetchDebuginfo(ctx, m.BuildId)
	if err != nil {
		return nil, fmt.Errorf("fetch debuginfo (BuildID: %q): %w", m.BuildId, err)
	}

	f, err := os.CreateTemp(s.tmpDir, "parca-symbolizer-*")
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	_, err = io.Copy(f, rc)
	if err != nil {
		return nil, fmt.Errorf("copy debuginfo to temp file: %w", err)
	}

	if err := rc.Close(); err != nil {
		return nil, fmt.Errorf("close debuginfo reader: %w", err)
	}

	if err := f.Close(); err != nil {
		return nil, fmt.Errorf("close temp file: %w", err)
	}

	e, err := elf.Open(f.Name())
	if err != nil {
		if err := s.metadata.MarkAsNotValidELF(ctx, m.BuildId); err != nil {
			level.Error(logger).Log("msg", "failed to mark build ID as not ELF", "err", err)
		}
		return nil, fmt.Errorf("open temp file as ELF: %w", err)
	}

	if err := e.Close(); err != nil {
		return nil, fmt.Errorf("close debuginfo file: %w", err)
	}

	// At this point we have the best version of the debug information file that we could find.
	// Let's symbolize it.
	lines, err := s.symbolizer.Symbolize(ctx, m, locations, f.Name())
	if err != nil {
		if errors.Is(err, symbol.ErrLinerCreationFailedBefore) {
			level.Debug(logger).Log("msg", "failed to symbolize before", "err", err)
			return nil, nil
		}

		return nil, fmt.Errorf("failed to symbolize locations for mapping: %w", err)
	}
	return lines, nil
}
