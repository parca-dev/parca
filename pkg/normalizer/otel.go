// Copyright 2024-2025 The Parca Authors
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

package normalizer

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow"
	"github.com/polarsignals/frostdb/query/logicalplan"
	"github.com/prometheus/prometheus/util/strutil"
	otelgrpcprofilingpb "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	otelprofilingpb "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"golang.org/x/exp/maps"

	"github.com/parca-dev/parca/pkg/profile"
)

func OtlpRequestToArrowRecord(
	ctx context.Context,
	req *otelgrpcprofilingpb.ExportProfilesServiceRequest,
	schema *dynparquet.Schema,
	mem memory.Allocator,
) (arrow.Record, error) {
	if err := ValidateOtelExportProfilesServiceRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	w, err := newProfileWriter(
		mem,
		schema,
		getAllLabelNames(req),
	)
	if err != nil {
		return nil, err
	}

	if err := w.writeResourceProfiles(req); err != nil {
		return nil, err
	}

	return w.ArrowRecord(ctx)
}

type labelNames struct {
	labelNames map[string]struct{}
}

func newLabelNames() *labelNames {
	return &labelNames{
		labelNames: make(map[string]struct{}),
	}
}

func (n *labelNames) addLabel(name string) {
	n.labelNames[strutil.SanitizeLabelName(name)] = struct{}{}
}

func (n *labelNames) addOtelAttributes(attrs []*v1.KeyValue) {
	for _, kv := range attrs {
		if strings.TrimSpace(kv.Value.GetStringValue()) != "" {
			n.addLabel(kv.Key)
		}
	}
}

func (n *labelNames) addOtelAttributesFromTable(attrs []*v1.KeyValue, idxs []int32) {
	for _, idx := range idxs {
		attr := attrs[idx]
		if strings.TrimSpace(attr.Value.GetStringValue()) != "" {
			n.addLabel(attr.Key)
		}
	}
}

func (n *labelNames) sorted() []string {
	if len(n.labelNames) == 0 {
		return nil
	}

	out := maps.Keys(n.labelNames)
	sort.Strings(out)
	return out
}

type labelSet struct {
	labels map[string]string
}

func newLabelSet() *labelSet {
	return &labelSet{
		labels: make(map[string]string),
	}
}

func (s *labelSet) addLabel(name, value string) {
	if strings.TrimSpace(value) != "" {
		s.labels[strutil.SanitizeLabelName(name)] = strings.TrimSpace(value)
	}
}

func (s *labelSet) addOtelAttributes(attrs []*v1.KeyValue) {
	for _, attr := range attrs {
		s.addLabel(attr.Key, attr.Value.GetStringValue())
	}
}

func (s *labelSet) addOtelAttributesFromTable(attrs []*v1.KeyValue, idxs []int32) {
	for _, idx := range idxs {
		attr := attrs[idx]
		s.addLabel(attr.Key, attr.Value.GetStringValue())
	}
}

func getAllLabelNames(req *otelgrpcprofilingpb.ExportProfilesServiceRequest) []string {
	allLabelNames := newLabelNames()

	for _, rp := range req.ResourceProfiles {
		allLabelNames.addOtelAttributes(rp.Resource.Attributes)

		for _, sp := range rp.ScopeProfiles {
			allLabelNames.addOtelAttributes(sp.Scope.Attributes)

			for _, p := range sp.Profiles {
				allLabelNames.addOtelAttributesFromTable(sp.Scope.Attributes, p.AttributeIndices)

				for _, sample := range p.Sample {
					allLabelNames.addOtelAttributesFromTable(sp.Scope.Attributes, sample.AttributeIndices)
				}
			}
		}
	}

	return allLabelNames.sorted()
}

type profileWriter struct {
	mem memory.Allocator

	labelNames []string
	schema     *dynparquet.Schema
	buffer     *dynparquet.Buffer

	row parquet.Row
}

func newProfileWriter(
	mem memory.Allocator,
	schema *dynparquet.Schema,
	labelNames []string,
) (*profileWriter, error) {
	// Create a buffer with all possible labels, pprof labels and pprof num labels as dynamic columns.
	// We use NewBuffer instead of GetBuffer here since analysis showed a very
	// low hit rate, meaning buffers were GCed faster than they could be reused.
	// The downside of using a pool is that buffers are held around for longer.
	// Using NewBuffer means that we pay the price of reallocating a buffer,
	// but they get GCed a lot sooner.
	buffer, err := schema.NewBuffer(map[string][]string{
		profile.ColumnLabels: labelNames,
	})
	if err != nil {
		return nil, err
	}

	return &profileWriter{
		mem: mem,

		labelNames: labelNames,
		schema:     schema,
		buffer:     buffer,

		row: make(parquet.Row, 0, len(schema.ParquetSchema().Fields())),
	}, nil
}

func (w *profileWriter) writeResourceProfiles(
	req *otelgrpcprofilingpb.ExportProfilesServiceRequest,
) error {
	for _, rp := range req.ResourceProfiles {
		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				metas := []profile.Meta{}
				for i := range p.SampleType {
					duration := p.DurationNanos
					name := sp.Scope.Name
					if name == "" {
						name = "unknown"
					}
					metas = append(metas, MetaFromOtelProfile(req.Dictionary.StringTable, p, name, i, duration))
				}

				for _, sample := range p.Sample {
					ls := newLabelSet()
					ls.addOtelAttributesFromTable(req.Dictionary.AttributeTable, sample.AttributeIndices)
					ls.addOtelAttributesFromTable(req.Dictionary.AttributeTable, p.AttributeIndices)
					ls.addOtelAttributes(sp.Scope.Attributes)
					ls.addOtelAttributes(rp.Resource.Attributes)

					// It is unclear how to handle the case where there are
					// multiple sample types with timestamps. Where do the
					// timestamps start and end for each sample type?
					if len(sample.TimestampsUnixNano) > 0 {
						for i, ts := range sample.TimestampsUnixNano {
							row := SampleToParquetRow(
								w.schema,
								w.row[:0],
								w.labelNames, nil, nil,
								ls.labels,
								profile.Meta{
									Name:       metas[0].Name,
									PeriodType: metas[0].PeriodType,
									SampleType: metas[0].SampleType,
									Timestamp:  int64(ts) / time.Millisecond.Nanoseconds(),
									Duration:   metas[0].Duration,
									Period:     metas[0].Period,
								},
								&NormalizedSample{
									Locations: serializeOtelStacktrace(
										p,
										sample,
										req.Dictionary.FunctionTable,
										req.Dictionary.MappingTable,
										req.Dictionary.LocationTable,
										req.Dictionary.AttributeTable,
										req.Dictionary.StringTable,
									),
									Value: sample.Value[i],
								},
							)
							if _, err := w.buffer.WriteRows([]parquet.Row{row}); err != nil {
								return fmt.Errorf("failed to write row to buffer: %w", err)
							}
						}
					} else {
						for j, value := range sample.Value {
							if value == 0 {
								continue
							}

							row := SampleToParquetRow(
								w.schema,
								w.row[:0],
								w.labelNames, nil, nil,
								ls.labels,
								metas[j],
								&NormalizedSample{
									Locations: serializeOtelStacktrace(
										p,
										sample,
										req.Dictionary.FunctionTable,
										req.Dictionary.MappingTable,
										req.Dictionary.LocationTable,
										req.Dictionary.AttributeTable,
										req.Dictionary.StringTable,
									),
									Value: value,
								},
							)
							if _, err := w.buffer.WriteRows([]parquet.Row{row}); err != nil {
								return fmt.Errorf("failed to write row to buffer: %w", err)
							}
						}
					}
				}
			}
		}
	}

	return nil
}

func (w *profileWriter) ArrowRecord(ctx context.Context) (arrow.Record, error) {
	if w.buffer.NumRows() == 0 {
		// If there are no rows in the buffer we simply return early
		return nil, nil
	}

	// We need to sort the buffer so the rows are inserted in sorted order later
	// on the storage nodes.
	w.buffer.Sort()

	// Convert the sorted buffer to an arrow record.
	converter := pqarrow.NewParquetConverter(w.mem, logicalplan.IterOptions{})
	defer converter.Close()

	if err := converter.Convert(ctx, w.buffer, w.schema); err != nil {
		return nil, fmt.Errorf("failed to convert parquet to arrow: %w", err)
	}

	return converter.NewRecord(), nil
}

func ValidateOtelExportProfilesServiceRequest(req *otelgrpcprofilingpb.ExportProfilesServiceRequest) error {
	if req == nil {
		return fmt.Errorf("request is nil")
	}

	if len(req.ResourceProfiles) == 0 {
		return fmt.Errorf("resource profiles are empty")
	}

	if err := ValidateOtelDictionary(req.Dictionary); err != nil {
		return fmt.Errorf("invalid dictionary: %w", err)
	}

	for _, rp := range req.ResourceProfiles {
		if rp.Resource != nil {
			seenKeys := make(map[string]struct{})
			for j, attr := range rp.Resource.Attributes {
				if attr.Key == "" {
					return fmt.Errorf("attribute key at index %d in resource attributes is empty", j)
				}

				if _, exists := seenKeys[attr.Key]; exists {
					return fmt.Errorf("duplicate attribute key %q in resource attributes", attr.Key)
				}
				seenKeys[attr.Key] = struct{}{}
				if attr.Value == nil {
					return fmt.Errorf("attribute value for key %q is nil in resource attributes", attr.Key)
				}

				if attr.Value.Value == nil {
					return fmt.Errorf("attribute value for key %q is nil in resource attributes", attr.Key)
				}
			}
		}

		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				if err := ValidateOtelProfile(req.Dictionary, p); err != nil {
					return fmt.Errorf("invalid profile: %w", err)
				}
			}
		}
	}

	return nil
}

func isEmptyMapping(m *otelprofilingpb.Mapping) bool {
	if m == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return m.MemoryStart == 0 &&
		m.MemoryLimit == 0 &&
		m.FileOffset == 0 &&
		m.FilenameStrindex == 0 &&
		len(m.AttributeIndices) == 0 &&
		!m.HasFunctions &&
		!m.HasFilenames &&
		!m.HasLineNumbers &&
		!m.HasInlineFrames
}

func isEmptyFunction(f *otelprofilingpb.Function) bool {
	if f == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return f.NameStrindex == 0 &&
		f.SystemNameStrindex == 0 &&
		f.FilenameStrindex == 0 &&
		f.StartLine == 0
}

func isEmptyAttribute(a *v1.KeyValue) bool {
	if a == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return a.Key == "" &&
		a.Value == nil ||
		a.Value.Value == nil
}

func isEmptyLocation(l *otelprofilingpb.Location) bool {
	if l == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return (l.MappingIndex == nil || *l.MappingIndex == 0) &&
		l.Address == 0 &&
		len(l.Line) == 0 &&
		!l.IsFolded &&
		len(l.AttributeIndices) == 0
}

func isEmptyLink(l *otelprofilingpb.Link) bool {
	if l == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return len(l.TraceId) == 0 &&
		len(l.SpanId) == 0
}

func isEmptyAttributeUnit(a *otelprofilingpb.AttributeUnit) bool {
	if a == nil {
		return true
	}

	// Check if all fields are zero values or nil.
	return a.AttributeKeyStrindex == 0 &&
		a.UnitStrindex == 0
}

func ValidateOtelDictionary(d *otelprofilingpb.ProfilesDictionary) error {
	if d == nil {
		return fmt.Errorf("dictionary is nil")
	}

	if len(d.StringTable) == 0 {
		return fmt.Errorf("string table is empty")
	}

	if d.StringTable[0] != "" {
		return fmt.Errorf("first string table entry must be empty, got %q", d.StringTable[0])
	}

	if len(d.MappingTable) == 0 {
		return fmt.Errorf("mapping table is empty")
	}

	if !isEmptyMapping(d.MappingTable[0]) {
		return fmt.Errorf("first mapping table entry must be nil, got %v", d.MappingTable[0])
	}

	for i, m := range d.MappingTable[1:] { // Skip the first entry which is nil
		if m == nil {
			return fmt.Errorf("mapping at index %d is nil", i)
		}

		if !existsInStringTable(m.FilenameStrindex, d.StringTable) {
			return fmt.Errorf("mapping file index %d out of bounds", m.FilenameStrindex)
		}

		for _, i := range m.AttributeIndices {
			if i < 0 || i >= int32(len(d.AttributeTable)) {
				return fmt.Errorf("mapping attribute index %d out of bounds", i)
			}
		}
	}

	if len(d.FunctionTable) == 0 {
		return fmt.Errorf("function table is empty")
	}

	if !isEmptyFunction(d.FunctionTable[0]) {
		return fmt.Errorf("first function table entry must be nil, got %v", d.FunctionTable[0])
	}

	for i, f := range d.FunctionTable[1:] { // Skip the first entry which is nil
		if f == nil {
			return fmt.Errorf("function at index %d is nil", i)
		}

		if !existsInStringTable(f.NameStrindex, d.StringTable) {
			return fmt.Errorf("function name index %d out of bounds", f.NameStrindex)
		}

		if !existsInStringTable(f.SystemNameStrindex, d.StringTable) {
			return fmt.Errorf("function system name index %d out of bounds", f.SystemNameStrindex)
		}

		if !existsInStringTable(f.FilenameStrindex, d.StringTable) {
			return fmt.Errorf("function filename index %d out of bounds", f.FilenameStrindex)
		}
	}

	if len(d.AttributeTable) == 0 {
		return fmt.Errorf("attribute table is empty")
	}

	if !isEmptyAttribute(d.AttributeTable[0]) {
		return fmt.Errorf("first attribute table entry must be nil, got %v", d.AttributeTable[0])
	}

	for i, a := range d.AttributeTable[1:] { // Skip the first entry which is nil
		if a == nil {
			return fmt.Errorf("attribute at index %d is nil", i)
		}

		if a.Key == "" {
			return fmt.Errorf("attribute key at index %d is empty", i)
		}

		if a.Value == nil {
			return fmt.Errorf("attribute value at index %d is nil", i)
		}

		if a.Value.Value == nil {
			return fmt.Errorf("attribute value at index %d is nil", i)
		}
	}

	if len(d.LocationTable) == 0 {
		return fmt.Errorf("location table is empty")
	}

	if !isEmptyLocation(d.LocationTable[0]) {
		return fmt.Errorf("first location table entry must be nil, got %v", d.LocationTable[0])
	}

	for i, l := range d.LocationTable[1:] { // Skip the first entry which is nil
		if l == nil {
			return fmt.Errorf("location at index %d is nil", i)
		}

		if l.MappingIndex != nil && (*l.MappingIndex < 0 || *l.MappingIndex >= int32(len(d.MappingTable))) {
			return fmt.Errorf("location mapping index %d out of bounds", *l.MappingIndex)
		}

		for j, line := range l.Line {
			if j < 0 || line.FunctionIndex >= int32(len(d.FunctionTable)) {
				return fmt.Errorf("location line function id %d out of bounds at line %d", line.FunctionIndex, j)
			}
		}

		for _, j := range l.AttributeIndices {
			if j < 0 || j >= int32(len(d.AttributeTable)) {
				return fmt.Errorf("location attribute index %d out of bounds", j)
			}
		}
	}

	if len(d.LinkTable) == 0 {
		return fmt.Errorf("link table is empty")
	}

	if !isEmptyLink(d.LinkTable[0]) {
		return fmt.Errorf("first link table entry must be nil, got %v", d.LinkTable[0])
	}

	for i, l := range d.LinkTable[1:] { // Skip the first entry which is nil
		if l == nil {
			return fmt.Errorf("link at index %d is nil", i)
		}

		if len(l.TraceId) != 16 {
			return fmt.Errorf("link trace ID at index %d must be 16 bytes long, got %d bytes", i, len(l.TraceId))
		}

		if len(l.SpanId) != 8 {
			return fmt.Errorf("link span ID at index %d must be 8 bytes long, got %d bytes", i, len(l.SpanId))
		}
	}

	if len(d.AttributeUnits) == 0 {
		return fmt.Errorf("attribute units table is empty")
	}

	if !isEmptyAttributeUnit(d.AttributeUnits[0]) {
		return fmt.Errorf("first attribute unit entry must be nil, got %v", d.AttributeUnits[0])
	}

	for i, a := range d.AttributeUnits[1:] { // Skip the first entry which is nil
		if a == nil {
			return fmt.Errorf("attribute unit at index %d is nil", i)
		}

		if !existsInStringTable(a.AttributeKeyStrindex, d.StringTable) {
			return fmt.Errorf("attribute unit key index %d out of bounds", a.AttributeKeyStrindex)
		}

		if !existsInStringTable(a.UnitStrindex, d.StringTable) {
			return fmt.Errorf("attribute unit string index %d out of bounds", a.UnitStrindex)
		}
	}

	return nil
}

func ValidateOtelProfile(d *otelprofilingpb.ProfilesDictionary, p *otelprofilingpb.Profile) error {
	if p == nil {
		return fmt.Errorf("profile is nil")
	}

	// There is already an API change that hasn't landed in the generated code
	// yet. Where SampleType is no longer a repeated field, but a single SampleType.
	if len(p.SampleType) != 1 {
		return fmt.Errorf("sample type is empty")
	}

	for i, st := range p.SampleType {
		if st == nil {
			return fmt.Errorf("sample type at index %d is nil", i)
		}
		if !existsInStringTable(st.TypeStrindex, d.StringTable) {
			return fmt.Errorf("sample type index %d out of bounds", st.TypeStrindex)
		}

		if !existsInStringTable(st.UnitStrindex, d.StringTable) {
			return fmt.Errorf("sample unit index %d out of bounds", st.UnitStrindex)
		}
	}

	if len(p.Sample) == 0 {
		return fmt.Errorf("sample is empty")
	}

	if p.DurationNanos < 0 {
		return fmt.Errorf("duration nanos %d must be non-negative", p.DurationNanos)
	}

	start := uint64(p.TimeNanos)
	end := start + uint64(p.DurationNanos)
	for _, s := range p.Sample {
		// Location start index must not be negative or the 0 element as that's the nil element.
		if s.LocationsStartIndex <= 0 || s.LocationsStartIndex >= int32(len(p.LocationIndices)) {
			return fmt.Errorf("sample locations start index %d out of bounds", s.LocationsStartIndex)
		}

		if s.LocationsLength < 0 || s.LocationsStartIndex+s.LocationsLength > int32(len(p.LocationIndices)) {
			return fmt.Errorf("sample locations length %d out of bounds with start index %d", s.LocationsLength, s.LocationsStartIndex)
		}

		if len(s.Value) == 0 && len(s.TimestampsUnixNano) == 0 {
			return fmt.Errorf("sample value and timestamps cannot both be empty")
		}

		if len(s.Value) > 0 && len(s.TimestampsUnixNano) > 0 && len(s.Value) != len(s.TimestampsUnixNano) {
			return fmt.Errorf("sample value length %d does not match sample timestamps length %d", len(s.Value), len(s.TimestampsUnixNano))
		}

		for _, a := range s.AttributeIndices {
			if a < 0 || a >= int32(len(d.AttributeTable)) {
				return fmt.Errorf("sample attribute index %d out of bounds", a)
			}
		}

		if s.LinkIndex != nil && (*s.LinkIndex < 0 || *s.LinkIndex >= int32(len(d.LinkTable))) {
			return fmt.Errorf("sample link index %d out of bounds", *s.LinkIndex)
		}

		for _, ts := range s.TimestampsUnixNano {
			if ts < start || ts > end {
				return fmt.Errorf("sample timestamp %d out of bounds, must be between %d and %d", ts, start, end)
			}
		}
	}

	for _, i := range p.LocationIndices {
		if i < 0 || i >= int32(len(d.LocationTable)) {
			return fmt.Errorf("location indices location index %d out of bounds", i)
		}
	}

	if p.PeriodType == nil {
		return fmt.Errorf("period type is nil")
	}

	if p.Period < 0 {
		return fmt.Errorf("period %d must be non-negative", p.Period)
	}

	for _, i := range p.CommentStrindices {
		if i < 0 || i >= int32(len(d.StringTable)) {
			return fmt.Errorf("comment string index %d out of bounds", i)
		}
	}

	if p.DefaultSampleTypeIndex < 0 || p.DefaultSampleTypeIndex >= int32(len(p.SampleType)) {
		return fmt.Errorf("default sample type index %d out of bounds", p.DefaultSampleTypeIndex)
	}

	if len(p.ProfileId) > 0 {
		if len(p.ProfileId) != 16 {
			return fmt.Errorf("profile ID must be 16 bytes long, got %d bytes", len(p.ProfileId))
		}

		// A profile ID that is all zeros is considered invalid.
		if isAllZeroBytes(p.ProfileId) {
			return fmt.Errorf("profile ID must not be all zeros")
		}
	}

	for _, i := range p.AttributeIndices {
		if i < 0 || i >= int32(len(d.AttributeTable)) {
			return fmt.Errorf("attribute index %d out of bounds", i)
		}
	}

	return nil
}

func isAllZeroBytes(id []byte) bool {
	for _, b := range id {
		if b != 0 {
			return false
		}
	}
	return true
}

func existsInStringTable(i int32, stringTable []string) bool {
	return i < int32(len(stringTable)) && i >= 0
}

// serializeOtelStacktrace serializes the stacktrace of an OTLP profile. It
// handles both cases where the location IDs are stored in the sample struct
// and where the locations are stored in the location slice. These are
// technically mutually exclusive.
func serializeOtelStacktrace(
	p *otelprofilingpb.Profile,
	s *otelprofilingpb.Sample,
	functions []*otelprofilingpb.Function,
	mappings []*otelprofilingpb.Mapping,
	locations []*otelprofilingpb.Location,
	attributes []*v1.KeyValue,
	stringTable []string,
) [][]byte {
	st := make([][]byte, 0, s.LocationsLength)

	for i := s.LocationsStartIndex; i < s.LocationsStartIndex+s.LocationsLength; i++ {
		location := locations[p.LocationIndices[i]]
		var m *otelprofilingpb.Mapping

		if location.MappingIndex != nil && *location.MappingIndex != 0 && *location.MappingIndex < int32(len(mappings)) {
			m = mappings[*location.MappingIndex]
		}

		st = append(st, profile.EncodeOtelLocation(
			attributes,
			location,
			m,
			functions,
			stringTable,
		))
	}

	return st
}
