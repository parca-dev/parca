package storage

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	"github.com/parca-dev/storage/chunk"
)

type Series struct {
	p *profile.Profile

	// Memoization tables for profile entities.
	stacktraceIDs map[[16]byte]*Stacktrace
	stacktraces   map[stacktraceKey]*Stacktrace
	locations     map[locationKey]*profile.Location
	functions     map[functionKey]*profile.Function
	mappings      map[mappingKey]*profile.Mapping

	chunk *chunk.Chunk
}

type Stacktrace struct {
	ID [16]byte

	Location []*profile.Location
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
}

type locationKey struct {
	addr, mappingID uint64
	lines           string
	isFolded        bool
}

type functionKey struct {
	startLine                  int64
	name, systemName, fileName string
}

type mapInfo struct {
	m      *profile.Mapping
	offset int64
}

type mappingKey struct {
	size, offset  uint64
	buildIDOrFile string
}

func (s *Series) Append(value *profile.Profile) error {
	if s.p == nil {
		s.p = &profile.Profile{
			PeriodType: value.PeriodType,
			SampleType: value.SampleType[:1],
		}
		s.stacktraces = make(map[stacktraceKey]*Stacktrace, len(value.Sample))
		s.stacktraceIDs = make(map[[16]byte]*Stacktrace, len(value.Sample))
		s.locations = make(map[locationKey]*profile.Location, len(value.Location))
		s.functions = make(map[functionKey]*profile.Function, len(value.Function))
		s.mappings = make(map[mappingKey]*profile.Mapping, len(value.Mapping))
	}

	if err := compatibleProfiles(s.p, value); err != nil {
		return err
	}

	pn := &profileNormalizer{
		p: s.p,

		locations:     s.locations,
		functions:     s.functions,
		mappings:      s.mappings,
		stacktraces:   s.stacktraces,
		stacktraceIDs: s.stacktraceIDs,

		// Profile-specific hash tables for each profile inserted.
		locationsByID: make(map[uint64]*profile.Location, len(value.Location)),
		functionsByID: make(map[uint64]*profile.Function, len(value.Function)),
		mappingsByID:  make(map[uint64]mapInfo, len(value.Mapping)),
	}

	if len(pn.mappings) == 0 && len(value.Mapping) > 0 {
		// The Mapping list has the property that the first mapping
		// represents the main binary. Take the first Mapping we see,
		// otherwise the operations below will add mappings in an
		// arbitrary order.
		pn.mapMapping(value.Mapping[0])
	}

	stacktraceSamples := make([]chunk.StacktraceSample, 0, len(value.Sample))
	for _, s := range value.Sample {
		if !isZeroSample(s) {
			stacktraceSamples = append(stacktraceSamples, pn.mapSample(s))
		}
	}
	sort.Slice(stacktraceSamples, func(i, j int) bool {
		return bytes.Compare(stacktraceSamples[i].StacktraceID[:], stacktraceSamples[j].StacktraceID[:]) == -1
	})

	return s.chunk.Append(chunk.Sample{
		Stacktraces: stacktraceSamples,
		Timestamp:   value.TimeNanos,
		Duration:    value.DurationNanos,
		Period:      value.Period,
	})
}

func (s *Series) Iterator() *SeriesIterator {
	return &SeriesIterator{
		series: s,
		data:   s.chunk.Data(),
		i:      0,
	}
}

type SeriesIterator struct {
	series *Series
	data   chunk.DecodedData
	i      int
	cur    *profile.Profile
	err    error
}

func (it *SeriesIterator) Next() bool {
	if it.i >= len(it.data.Timestamps) {
		return false
	}

	p := &profile.Profile{
		PeriodType:    it.series.p.PeriodType,
		SampleType:    it.series.p.SampleType,
		TimeNanos:     it.data.Timestamps[it.i],
		DurationNanos: it.data.Durations[it.i],
		Period:        it.data.Periods[it.i],
		Location:      it.series.p.Location,
		Function:      it.series.p.Function,
		Mapping:       it.series.p.Mapping,
	}

	for _, stacktrace := range it.data.Stacktraces {
		if stacktrace.Values[it.i] != 0 {
			st := it.series.stacktraceIDs[stacktrace.StacktraceID]
			p.Sample = append(p.Sample, &profile.Sample{
				Location: st.Location,
				Label:    st.Label,
				NumLabel: st.NumLabel,
				NumUnit:  st.NumUnit,
				Value:    []int64{stacktrace.Values[it.i]},
			})
		}
	}

	it.cur = p
	it.i++

	return true
}

func (it *SeriesIterator) At() *profile.Profile {
	return it.cur
}

func (it *SeriesIterator) Err() error {
	return it.err
}

type profileNormalizer struct {
	p *profile.Profile

	// Memoization tables within a profile.
	locationsByID map[uint64]*profile.Location
	functionsByID map[uint64]*profile.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	stacktraceIDs map[[16]byte]*Stacktrace
	stacktraces   map[stacktraceKey]*Stacktrace
	locations     map[locationKey]*profile.Location
	functions     map[functionKey]*profile.Function
	mappings      map[mappingKey]*profile.Mapping

	// A slice of samples for each unique stack trace.
	c *chunk.Chunk
}

func (pn *profileNormalizer) mapSample(src *profile.Sample) chunk.StacktraceSample {
	s := &profile.Sample{
		Location: make([]*profile.Location, len(src.Location)),
		Label:    make(map[string][]string, len(src.Label)),
		NumLabel: make(map[string][]int64, len(src.NumLabel)),
		NumUnit:  make(map[string][]string, len(src.NumLabel)),
	}
	for i, l := range src.Location {
		s.Location[i] = pn.mapLocation(l)
	}
	for k, v := range src.Label {
		vv := make([]string, len(v))
		copy(vv, v)
		s.Label[k] = vv
	}
	for k, v := range src.NumLabel {
		u := src.NumUnit[k]
		vv := make([]int64, len(v))
		uu := make([]string, len(u))
		copy(vv, v)
		copy(uu, u)
		s.NumLabel[k] = vv
		s.NumUnit[k] = uu
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping. Add current values to the
	// existing sample.
	k := makeStacktraceKey(s)
	st, found := pn.stacktraces[k]
	if !found {
		st = &Stacktrace{
			ID:       [16]byte(uuid.New()),
			Location: s.Location,
			Label:    s.Label,
			NumLabel: s.NumLabel,
			NumUnit:  s.NumUnit,
		}
		pn.stacktraces[k] = st
		pn.stacktraceIDs[st.ID] = st
	}

	return chunk.StacktraceSample{
		StacktraceID: st.ID,
		Value:        src.Value[0],
	}
}

func (pn *profileNormalizer) mapLocation(src *profile.Location) *profile.Location {
	if src == nil {
		return nil
	}

	if l, ok := pn.locationsByID[src.ID]; ok {
		return l
	}

	mi := pn.mapMapping(src.Mapping)
	l := &profile.Location{
		ID:       uint64(len(pn.p.Location) + 1),
		Mapping:  mi.m,
		Address:  uint64(int64(src.Address) + mi.offset),
		Line:     make([]profile.Line, len(src.Line)),
		IsFolded: src.IsFolded,
	}
	for i, ln := range src.Line {
		l.Line[i] = pn.mapLine(ln)
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping ID.
	k := makeLocationKey(l)
	if ll, ok := pn.locations[k]; ok {
		pn.locationsByID[src.ID] = ll
		return ll
	}
	pn.locationsByID[src.ID] = l
	pn.locations[k] = l
	pn.p.Location = append(pn.p.Location, l)
	return l
}

func (pn *profileNormalizer) mapMapping(src *profile.Mapping) mapInfo {
	if src == nil {
		return mapInfo{}
	}

	if mi, ok := pn.mappingsByID[src.ID]; ok {
		return mi
	}

	// Check memoization tables.
	mk := makeMappingKey(src)
	if m, ok := pn.mappings[mk]; ok {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi
	}
	m := &profile.Mapping{
		ID:              uint64(len(pn.p.Mapping) + 1),
		Start:           src.Start,
		Limit:           src.Limit,
		Offset:          src.Offset,
		File:            src.File,
		BuildID:         src.BuildID,
		HasFunctions:    src.HasFunctions,
		HasFilenames:    src.HasFilenames,
		HasLineNumbers:  src.HasLineNumbers,
		HasInlineFrames: src.HasInlineFrames,
	}
	pn.p.Mapping = append(pn.p.Mapping, m)

	// Update memoization tables.
	pn.mappings[mk] = m
	mi := mapInfo{m, 0}
	pn.mappingsByID[src.ID] = mi
	return mi
}

func (pn *profileNormalizer) mapLine(src profile.Line) profile.Line {
	ln := profile.Line{
		Function: pn.mapFunction(src.Function),
		Line:     src.Line,
	}
	return ln
}

func (pn *profileNormalizer) mapFunction(src *profile.Function) *profile.Function {
	if src == nil {
		return nil
	}
	if f, ok := pn.functionsByID[src.ID]; ok {
		return f
	}
	k := makeFunctionKey(src)
	if f, ok := pn.functions[k]; ok {
		pn.functionsByID[src.ID] = f
		return f
	}
	f := &profile.Function{
		ID:         uint64(len(pn.p.Function) + 1),
		Name:       src.Name,
		SystemName: src.SystemName,
		Filename:   src.Filename,
		StartLine:  src.StartLine,
	}
	pn.functions[k] = f
	pn.functionsByID[src.ID] = f
	pn.p.Function = append(pn.p.Function, f)
	return f
}

// key generates stacktraceKey to be used as a key for maps.
func makeStacktraceKey(sample *profile.Sample) stacktraceKey {
	ids := make([]string, len(sample.Location))
	for i, l := range sample.Location {
		ids[i] = strconv.FormatUint(l.ID, 16)
	}

	labels := make([]string, 0, len(sample.Label))
	for k, v := range sample.Label {
		labels = append(labels, fmt.Sprintf("%q%q", k, v))
	}
	sort.Strings(labels)

	numlabels := make([]string, 0, len(sample.NumLabel))
	for k, v := range sample.NumLabel {
		numlabels = append(numlabels, fmt.Sprintf("%q%x%x", k, v, sample.NumUnit[k]))
	}
	sort.Strings(numlabels)

	return stacktraceKey{
		strings.Join(ids, "|"),
		strings.Join(labels, ""),
		strings.Join(numlabels, ""),
	}
}

func makeLocationKey(l *profile.Location) locationKey {
	key := locationKey{
		addr:     l.Address,
		isFolded: l.IsFolded,
	}
	if l.Mapping != nil {
		// Normalizes address to handle address space randomization.
		key.addr -= l.Mapping.Start
		key.mappingID = l.Mapping.ID
	}
	lines := make([]string, len(l.Line)*2)
	for i, line := range l.Line {
		if line.Function != nil {
			lines[i*2] = strconv.FormatUint(line.Function.ID, 16)
		}
		lines[i*2+1] = strconv.FormatInt(line.Line, 16)
	}
	key.lines = strings.Join(lines, "|")
	return key
}

func makeFunctionKey(f *profile.Function) functionKey {
	return functionKey{
		f.StartLine,
		f.Name,
		f.SystemName,
		f.Filename,
	}
}

func makeMappingKey(m *profile.Mapping) mappingKey {
	// Normalize addresses to handle address space randomization.
	// Round up to next 4K boundary to avoid minor discrepancies.
	const mapsizeRounding = 0x1000

	size := m.Limit - m.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)
	key := mappingKey{
		size:   size,
		offset: m.Offset,
	}

	switch {
	case m.BuildID != "":
		key.buildIDOrFile = m.BuildID
	case m.File != "":
		key.buildIDOrFile = m.File
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}
	return key
}

// compatible determines if two profiles can be compared/merged.
// returns nil if the profiles are compatible; otherwise an error with
// details on the incompatibility.
func compatibleProfiles(p *profile.Profile, pb *profile.Profile) error {
	if !equalValueType(p.PeriodType, pb.PeriodType) {
		return fmt.Errorf("incompatible period types %v and %v", p.PeriodType, pb.PeriodType)
	}

	if !equalValueType(p.SampleType[0], pb.SampleType[0]) {
		return fmt.Errorf("incompatible sample types %v and %v", p.SampleType, pb.SampleType)
	}
	return nil
}

// equalValueType returns true if the two value types are semantically
// equal. It ignores the internal fields used during encode/decode.
func equalValueType(st1, st2 *profile.ValueType) bool {
	return st1.Type == st2.Type && st1.Unit == st2.Unit
}

func isZeroSample(s *profile.Sample) bool {
	for _, v := range s.Value {
		if v != 0 {
			return false
		}
	}
	return true
}
