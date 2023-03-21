// Copyright 2022-2023 The Parca Authors
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

	debuginfopb "github.com/parca-dev/parca/gen/proto/go/parca/debuginfo/v1alpha1"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/debuginfo"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/runutil"
	"github.com/parca-dev/parca/pkg/symbol/addr2line"
	"github.com/parca-dev/parca/pkg/symbol/demangle"
	"github.com/parca-dev/parca/pkg/symbol/elfutils"
)

var (
	ErrNotValidElf = errors.New("not a valid ELF file")
	ErrNoDebuginfo = errors.New("no debug info found")
	ErrLinerFailed = errors.New("liner creation failed")
)

type DebuginfoMetadata interface {
	SetQuality(ctx context.Context, buildID string, quality *debuginfopb.DebuginfoQuality) error
	Fetch(ctx context.Context, buildID string) (*debuginfopb.Debuginfo, error)
}

// liner is the interface implemented by symbolizers
// which read an object file (symbol table or debug information) and return
// source code lines by a given memory address.
type liner interface {
	PCToLines(pc uint64) ([]profile.LocationLine, error)
	PCRange() ([2]uint64, error)
	Close() error
	File() string
}

type Option func(*Symbolizer)

func WithAttemptThreshold(t int) Option {
	return func(s *Symbolizer) {
		s.attemptThreshold = t
	}
}

func WithDemangleMode(mode string) Option {
	return func(s *Symbolizer) {
		s.demangler = demangle.NewDemangler(mode, false)
	}
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

	metastore pb.MetastoreServiceClient
	debuginfo DebuginfoFetcher
	metadata  DebuginfoMetadata

	demangler        *demangle.Demangler
	attemptThreshold int

	linerCreationFailed   map[string]struct{}
	symbolizationAttempts map[string]map[uint64]int
	symbolizationFailed   map[string]map[uint64]struct{}
	pcRanges              map[string][2]uint64
	linerCache            map[string]liner

	batchSize uint32

	tmpDir string
}

type DebuginfoFetcher interface {
	// Fetch ensures that the debug info for the given build ID is available on
	// a local filesystem and returns a path to it.
	FetchDebuginfo(ctx context.Context, dbginfo *debuginfopb.Debuginfo) (io.ReadCloser, error)
}

func New(
	logger log.Logger,
	reg prometheus.Registerer,
	metadata DebuginfoMetadata,
	metastore pb.MetastoreServiceClient,
	debuginfo DebuginfoFetcher,
	tmpDir string,
	batchSize uint32,
	opts ...Option,
) *Symbolizer {
	attemptsTotal := promauto.With(reg).NewCounter(
		prometheus.CounterOpts{
			Name: "parca_symbolizer_symbolication_attempts_total",
			Help: "Total number of symbolication attempts in batches.",
		},
	)
	errorsTotal := promauto.With(reg).NewCounterVec(
		prometheus.CounterOpts{
			Name: "parca_symbolizer_symbolication_errors_total",
			Help: "Total number of symbolication errors in batches, partitioned by an error reason.",
		},
		[]string{"reason"},
	)
	duration := promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "parca_symbolizer_symbolication_duration_seconds",
			Help:    "How long it took in seconds to finish a round of the symbolication cycle in batches.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)
	storeDuration := promauto.With(reg).NewHistogram(
		prometheus.HistogramOpts{
			Name:    "parca_symbolizer_store_duration_seconds",
			Help:    "How long it took in seconds to store a batch of the symbolized locations.",
			Buckets: []float64{0.001, 0.01, 0.1, 0.3, 0.6, 1, 3, 6, 9, 20, 30, 60, 90, 120},
		},
	)

	const (
		defaultDemangleMode     = "simple"
		defaultAttemptThreshold = 3
	)

	s := &Symbolizer{
		logger:           log.With(logger, "component", "symbolizer"),
		attempts:         attemptsTotal,
		errors:           errorsTotal,
		duration:         duration,
		storeDuration:    storeDuration,
		metastore:        metastore,
		debuginfo:        debuginfo,
		tmpDir:           tmpDir,
		batchSize:        batchSize,
		metadata:         metadata,
		demangler:        demangle.NewDemangler(defaultDemangleMode, false),
		attemptThreshold: defaultAttemptThreshold,

		linerCreationFailed:   map[string]struct{}{},
		symbolizationAttempts: map[string]map[uint64]int{},
		symbolizationFailed:   map[string]map[uint64]struct{}{},
		pcRanges:              map[string][2]uint64{},
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
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
			s.duration.Observe(time.Since(begin).Seconds())
			// Nothing to symbolize.
			return
		}
		prevMaxKey = lres.MaxKey

		err = s.Symbolize(ctx, lres.Locations)
		if err != nil {
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

	newLinerCache := map[string]liner{}
	for _, locationsByMapping := range locationsByMappings {
		mapping := locationsByMapping.Mapping

		// If Mapping or Mapping.BuildID is empty, we cannot associate an object file with functions.
		if mapping == nil || len(mapping.BuildId) == 0 || UnsymbolizableMapping(mapping) {
			level.Debug(s.logger).Log("msg", "mapping of location is empty, skipping")
			continue
		}
		logger := log.With(s.logger, "buildid", mapping.BuildId)

		var liner liner
		locations := locationsByMapping.Locations
		// Symbolize returns a list of lines per location passed to it.
		locationsByMapping.LocationsLines, liner, err = s.symbolizeLocationsForMapping(ctx, mapping, locations)
		if err != nil {
			level.Debug(logger).Log("msg", "storage symbolization request failed", "err", err)
			continue
		}
		if liner != nil {
			newLinerCache[mapping.BuildId] = liner
		}
	}
	for k := range newLinerCache {
		delete(s.linerCache, k)
	}
	for _, liner := range s.linerCache {
		// These are liners that didn't show up in the latest iteration.
		if err := liner.Close(); err != nil {
			level.Error(s.logger).Log("msg", "failed to close liner", "err", err)
		}
		if err := os.Remove(liner.File()); err != nil {
			level.Error(s.logger).Log("msg", "failed to remove liner file", "err", err)
		}
	}
	s.linerCache = newLinerCache

	numFunctions := 0
	for _, locationsByMapping := range locationsByMappings {
		for _, locationLines := range locationsByMapping.LocationsLines {
			numFunctions += len(locationLines)
		}
	}
	if numFunctions == 0 {
		return nil
	}

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
func (s *Symbolizer) symbolizeLocationsForMapping(ctx context.Context, m *pb.Mapping, locations []*pb.Location) ([][]profile.LocationLine, liner, error) {
	dbginfo, err := s.metadata.Fetch(ctx, m.BuildId)
	if err != nil {
		return nil, nil, fmt.Errorf("fetching metadata: %w", err)
	}

	if dbginfo.Quality != nil {
		if dbginfo.Quality.NotValidElf {
			return nil, nil, ErrNotValidElf
		}
		if !dbginfo.Quality.HasDwarf && !dbginfo.Quality.HasGoPclntab && !(dbginfo.Quality.HasSymtab || dbginfo.Quality.HasDynsym) {
			return nil, nil, fmt.Errorf("check previously reported debuginfo quality: %w", ErrNoDebuginfo)
		}
	}

	key := dbginfo.BuildId
	countLocationsToSymbolize := s.countLocationsToSymbolize(key, locations)
	if countLocationsToSymbolize == 0 {
		pcRange := s.pcRanges[key]
		level.Debug(s.logger).Log("msg", "no locations to symbolize", "build_id", m.BuildId, "pc_range_start", fmt.Sprintf("0x%x", pcRange[0]), "pc_range_end", fmt.Sprintf("0x%x", pcRange[1]))
		return make([][]profile.LocationLine, len(locations)), nil, nil
	}

	liner, found := s.linerCache[key]
	if !found {
		switch dbginfo.Source {
		case debuginfopb.Debuginfo_SOURCE_UPLOAD:
			if dbginfo.Upload.State != debuginfopb.DebuginfoUpload_STATE_UPLOADED {
				return nil, nil, debuginfo.ErrNotUploadedYet
			}
		case debuginfopb.Debuginfo_SOURCE_DEBUGINFOD:
			// Nothing to do here, just covering all cases.
		default:
			return nil, nil, debuginfo.ErrUnknownDebuginfoSource
		}

		// Fetch the debug info for the build ID.
		rc, err := s.debuginfo.FetchDebuginfo(ctx, dbginfo)
		if err != nil {
			return nil, nil, fmt.Errorf("fetch debuginfo (BuildID: %q): %w", m.BuildId, err)
		}
		defer func() {
			if err := rc.Close(); err != nil {
				level.Error(s.logger).Log("msg", "failed to close debuginfo reader", "err", err)
			}
		}()

		f, err := os.CreateTemp(s.tmpDir, "parca-symbolizer-*")
		if err != nil {
			return nil, nil, fmt.Errorf("create temp file: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				level.Error(s.logger).Log("msg", "failed to close debuginfo file", "err", err)
			}
			if err := os.Remove(f.Name()); err != nil {
				level.Error(s.logger).Log("msg", "failed to remove debuginfo file", "err", err)
			}
		}()

		_, err = io.Copy(f, rc)
		if err != nil {
			return nil, nil, fmt.Errorf("copy debuginfo to temp file: %w", err)
		}

		e, err := elf.Open(f.Name())
		if err != nil {
			if merr := s.metadata.SetQuality(ctx, m.BuildId, &debuginfopb.DebuginfoQuality{
				NotValidElf: true,
			}); merr != nil {
				level.Error(s.logger).Log("msg", "failed to set metadata quality", "err", merr)
			}

			return nil, nil, fmt.Errorf("open temp file as ELF: %w", err)
		}
		defer func() {
			if err := e.Close(); err != nil {
				level.Error(s.logger).Log("msg", "failed to close debuginfo file", "err", err)
			}
		}()

		if dbginfo.Quality == nil {
			dbginfo.Quality = &debuginfopb.DebuginfoQuality{
				HasDwarf:     elfutils.HasDWARF(e),
				HasGoPclntab: elfutils.HasGoPclntab(e),
				HasSymtab:    elfutils.HasSymtab(e),
				HasDynsym:    elfutils.HasDynsym(e),
			}
			if err := s.metadata.SetQuality(ctx, m.BuildId, dbginfo.Quality); err != nil {
				return nil, nil, fmt.Errorf("set quality: %w", err)
			}
			if !dbginfo.Quality.HasDwarf && !dbginfo.Quality.HasGoPclntab && !(dbginfo.Quality.HasSymtab || dbginfo.Quality.HasDynsym) {
				return nil, nil, fmt.Errorf("check debuginfo quality: %w", ErrNoDebuginfo)
			}
		}
		liner, err = s.newLiner(f.Name(), e, dbginfo.Quality)
		if err != nil {
			return nil, nil, fmt.Errorf("new liner: %w", err)
		}
	}

	pcRange, found := s.pcRanges[key]
	if !found {
		pcRange, err = liner.PCRange()
		if err != nil {
			return nil, liner, fmt.Errorf("get pc range: %w", err)
		}
		s.pcRanges[key] = pcRange
	}

	countLocationsToSymbolize = s.countLocationsToSymbolize(key, locations)
	if countLocationsToSymbolize == 0 {
		level.Debug(s.logger).Log("msg", "no locations to symbolize", "build_id", m.BuildId, "pc_range_start", fmt.Sprintf("0x%x", pcRange[0]), "pc_range_end", fmt.Sprintf("0x%x", pcRange[1]))
		return make([][]profile.LocationLine, len(locations)), liner, nil
	}
	level.Debug(s.logger).Log("msg", "symbolizing locations", "build_id", m.BuildId, "count", countLocationsToSymbolize)

	locationsLines := make([][]profile.LocationLine, len(locations))
	for i, loc := range locations {
		// Check if we already attempt to symbolize this location and failed.
		// No need to try again.
		if _, failedBefore := s.symbolizationFailed[dbginfo.BuildId][loc.Address]; failedBefore {
			continue
		}
		if pcRange[0] <= loc.Address && loc.Address <= pcRange[1] {
			locationsLines[i] = s.pcToLines(liner, key, loc.Address)
		}
	}

	return locationsLines, liner, nil
}

func (s *Symbolizer) countLocationsToSymbolize(key string, locations []*pb.Location) int {
	locationsToSymbolize := 0
	for _, loc := range locations {
		if _, failedBefore := s.symbolizationFailed[key][loc.Address]; failedBefore {
			continue
		}
		pcRange, found := s.pcRanges[key]
		if !found {
			locationsToSymbolize++
			continue
		}
		if pcRange[0] <= loc.Address && loc.Address <= pcRange[1] {
			locationsToSymbolize++
		}
	}
	return locationsToSymbolize
}

// newLiner creates a new liner for the given mapping and object file path.
func (s *Symbolizer) newLiner(filepath string, f *elf.File, quality *debuginfopb.DebuginfoQuality) (liner, error) {
	switch {
	case quality.HasDwarf:
		lnr, err := addr2line.DWARF(s.logger, filepath, f, s.demangler)
		if err != nil {
			return nil, fmt.Errorf("failed to create DWARF liner: %w", err)
		}

		return lnr, nil
	case quality.HasGoPclntab:
		lnr, err := addr2line.Go(s.logger, filepath, f)
		if err != nil {
			return nil, fmt.Errorf("failed to create Go liner: %w", err)
		}

		return lnr, nil
		// TODO CHECK plt
	case quality.HasSymtab || quality.HasDynsym:
		lnr, err := addr2line.Symbols(s.logger, filepath, f, s.demangler)
		if err != nil {
			return nil, fmt.Errorf("failed to create Symtab liner: %w", err)
		}

		return lnr, nil
	default:
		return nil, ErrLinerFailed
	}
}

// pcToLines returns the line number of the given PC while keeping the track of symbolization attempts and failures.
func (s *Symbolizer) pcToLines(liner liner, key string, addr uint64) []profile.LocationLine {
	lines, err := liner.PCToLines(addr)
	level.Debug(s.logger).Log("msg", "symbolized location", "build_id", key, "address", addr, "lines_count", len(lines), "err", err, "liner_type", fmt.Sprintf("%T", liner))
	if err != nil {
		// Error bookkeeping.
		if prev, ok := s.symbolizationAttempts[key][addr]; ok {
			prev++
			if prev >= s.attemptThreshold {
				if _, ok := s.symbolizationFailed[key]; ok {
					s.symbolizationFailed[key][addr] = struct{}{}
				} else {
					s.symbolizationFailed[key] = map[uint64]struct{}{addr: {}}
				}
				delete(s.symbolizationAttempts[key], addr)
			} else {
				s.symbolizationAttempts[key][addr] = prev
			}
			return nil
		}
		// First failed attempt.
		s.symbolizationAttempts[key] = map[uint64]int{addr: 1}
		return nil
	}
	if len(lines) == 0 {
		if _, ok := s.symbolizationFailed[key]; ok {
			s.symbolizationFailed[key][addr] = struct{}{}
		} else {
			s.symbolizationFailed[key] = map[uint64]struct{}{addr: {}}
		}
		delete(s.symbolizationAttempts[key], addr)
	}
	return lines
}
