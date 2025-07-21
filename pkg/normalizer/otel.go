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
	otelgrpcprofilingpb "go.opentelemetry.io/proto/otlp/collector/profiles/v1experimental"
	v1 "go.opentelemetry.io/proto/otlp/common/v1"
	otelprofilingpb "go.opentelemetry.io/proto/otlp/profiles/v1experimental"
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

	if err := w.writeResourceProfiles(req.ResourceProfiles); err != nil {
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

func (n *labelNames) addOtelAttributesFromTable(attrs []*v1.KeyValue, idxs []uint64) {
	for _, idx := range idxs {
		attr := attrs[idx]
		if strings.TrimSpace(attr.Value.GetStringValue()) != "" {
			n.addLabel(attr.Key)
		}
	}
}

func (n *labelNames) addOtelPprofExtendedLabels(stringTable []string, labels []*otelprofilingpb.Label) {
	for _, label := range labels {
		if label.Str != 0 && strings.TrimSpace(stringTable[label.Str]) != "" {
			n.addLabel(stringTable[label.Key])
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

func (s *labelSet) addOtelAttributesFromTable(attrs []*v1.KeyValue, idxs []uint64) {
	for _, idx := range idxs {
		attr := attrs[idx]
		s.addLabel(attr.Key, attr.Value.GetStringValue())
	}
}

func (s *labelSet) addOtelPprofExtendedLabels(stringTable []string, labels []*otelprofilingpb.Label) {
	for _, label := range labels {
		s.addLabel(stringTable[label.Key], stringTable[label.Str])
	}
}

func getAllLabelNames(req *otelgrpcprofilingpb.ExportProfilesServiceRequest) []string {
	allLabelNames := newLabelNames()

	for _, rp := range req.ResourceProfiles {
		allLabelNames.addOtelAttributes(rp.Resource.Attributes)

		for _, sp := range rp.ScopeProfiles {
			allLabelNames.addOtelAttributes(sp.Scope.Attributes)

			for _, p := range sp.Profiles {
				allLabelNames.addOtelAttributes(p.Attributes)

				for _, sample := range p.Profile.Sample {
					allLabelNames.addOtelPprofExtendedLabels(p.Profile.StringTable, sample.Label)
					allLabelNames.addOtelAttributesFromTable(p.Profile.AttributeTable, sample.Attributes)
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
	rp []*otelprofilingpb.ResourceProfiles,
) error {
	for _, rp := range rp {
		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				metas := []profile.Meta{}
				for i, st := range p.Profile.SampleType {
					duration := p.Profile.DurationNanos
					if duration == 0 {
						duration = int64(p.EndTimeUnixNano - p.StartTimeUnixNano)
					}
					if duration == 0 && st.AggregationTemporality == otelprofilingpb.AggregationTemporality_AGGREGATION_TEMPORALITY_DELTA {
						duration = time.Second.Nanoseconds()
					}

					name := sp.Scope.Name
					if name == "" {
						name = "unknown"
					}
					metas = append(metas, MetaFromOtelProfile(p.Profile, name, i, duration))
				}

				for _, sample := range p.Profile.Sample {
					ls := newLabelSet()
					ls.addOtelPprofExtendedLabels(p.Profile.StringTable, sample.Label)
					ls.addOtelAttributesFromTable(p.Profile.AttributeTable, sample.Attributes)
					ls.addOtelAttributes(p.Attributes)
					ls.addOtelAttributes(sp.Scope.Attributes)
					ls.addOtelAttributes(rp.Resource.Attributes)

					// It is unclear how to handle the case where there are
					// multiple sample types with timestamps. Where do the
					// timestamps start and end for each sample type?
					if len(sample.TimestampsUnixNano) > 0 && len(p.Profile.SampleType) == 1 {
						for _, ts := range sample.TimestampsUnixNano {
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
										p.Profile,
										sample,
										p.Profile.Function,
										p.Profile.Mapping,
										p.Profile.StringTable,
									),
									Value: 1,
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
										p.Profile,
										sample,
										p.Profile.Function,
										p.Profile.Mapping,
										p.Profile.StringTable,
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
	for _, rp := range req.ResourceProfiles {
		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				if err := ValidateOtelProfile(p.Profile); err != nil {
					return fmt.Errorf("invalid profile: %w", err)
				}
			}
		}
	}

	return nil
}

func ValidateOtelProfile(p *otelprofilingpb.Profile) error {
	for _, f := range p.Function {
		if !existsInStringTable(f.Name, p.StringTable) {
			return fmt.Errorf("function name index %d out of bounds", f.Name)
		}

		if !existsInStringTable(f.SystemName, p.StringTable) {
			return fmt.Errorf("function system name index %d out of bounds", f.SystemName)
		}

		if !existsInStringTable(f.Filename, p.StringTable) {
			return fmt.Errorf("function filename index %d out of bounds", f.Filename)
		}
	}

	for _, m := range p.Mapping {
		if !existsInStringTable(m.Filename, p.StringTable) {
			return fmt.Errorf("mapping file index %d out of bounds", m.Filename)
		}

		if !existsInStringTable(m.BuildId, p.StringTable) {
			return fmt.Errorf("mapping build id index %d out of bounds", m.BuildId)
		}
	}

	for _, l := range p.Location {
		if l.MappingIndex > uint64(len(p.Mapping)) {
			return fmt.Errorf("location mapping index %d out of bounds", l.MappingIndex)
		}

		for _, l := range l.Line {
			if l.FunctionIndex > uint64(len(p.Function)) {
				return fmt.Errorf("location line function id %d out of bounds", l.FunctionIndex)
			}
		}
	}

	for _, s := range p.Sample {
		for _, id := range s.LocationIndex {
			if id > uint64(len(p.Location)) {
				return fmt.Errorf("sample location index %d out of bounds", id)
			}
		}

		if s.LocationsStartIndex > uint64(len(p.Location)) {
			return fmt.Errorf("sample locations start index %d out of bounds", s.LocationsStartIndex)
		}

		if s.LocationsStartIndex+s.LocationsLength > uint64(len(p.Location)) {
			return fmt.Errorf("sample locations end index %d out of bounds", s.LocationsStartIndex+s.LocationsLength)
		}

		if s.LocationsStartIndex > s.LocationsStartIndex+s.LocationsLength {
			return fmt.Errorf("sample locations start index %d is greater than end index %d", s.LocationsStartIndex, s.LocationsStartIndex+s.LocationsLength)
		}

		for _, l := range s.Label {
			if !existsInStringTable(l.Key, p.StringTable) {
				return fmt.Errorf("sample label key index %d out of bounds", l.Key)
			}

			if !existsInStringTable(l.Str, p.StringTable) {
				return fmt.Errorf("sample label string index %d out of bounds", l.Str)
			}
		}

		if len(s.Value) != len(p.SampleType) {
			return fmt.Errorf("sample value length %d does not match sample type length %d", len(s.Value), len(p.SampleType))
		}
	}

	return nil
}

func existsInStringTable(i int64, stringTable []string) bool {
	return i < int64(len(stringTable)) && i >= 0
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
	stringTable []string,
) [][]byte {
	st := make([][]byte, 0, len(s.LocationIndex)+int(s.LocationsLength))

	// We handle the case where the location IDs are stored in the sample struct.
	for _, locationID := range s.LocationIndex {
		location := p.Location[locationID]
		var m *otelprofilingpb.Mapping

		if location.MappingIndex != 0 && location.MappingIndex-1 < uint64(len(mappings)) {
			m = mappings[location.MappingIndex-1]
		}

		st = append(st, profile.EncodeOtelLocation(
			location,
			m,
			functions,
			stringTable,
		))
	}

	// And the case where the locations are stored in the location slice. And
	// the sample just points to the start and length.
	for _, location := range p.Location[s.LocationsStartIndex : s.LocationsStartIndex+s.LocationsLength] {
		var m *otelprofilingpb.Mapping

		if location.MappingIndex != 0 && location.MappingIndex-1 < uint64(len(mappings)) {
			m = mappings[location.MappingIndex-1]
		}

		st = append(st, profile.EncodeOtelLocation(
			location,
			m,
			functions,
			stringTable,
		))
	}

	return st
}
