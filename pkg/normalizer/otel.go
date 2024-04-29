// Copyright 2024 The Parca Authors
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

	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/memory"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/pqarrow"
	"github.com/polarsignals/frostdb/query/logicalplan"

	otelgrpcprofilingpb "github.com/parca-dev/parca/gen/proto/go/opentelemetry/proto/collector/profiles/v1"
	pprofextended "github.com/parca-dev/parca/gen/proto/go/opentelemetry/proto/profiles/v1/alternatives/pprofextended"
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

	allLabelNames := make(map[string]struct{})

	for _, rp := range req.ResourceProfiles {
		for _, attr := range rp.Resource.Attributes {
			if attr.Value.String() != "" {
				allLabelNames[attr.Key] = struct{}{}
			}
		}

		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				for _, attr := range p.Attributes {
					if attr.Value.String() != "" {
						allLabelNames[attr.Key] = struct{}{}
					}
				}

				for _, attr := range rp.Resource.Attributes {
					if attr.Value.String() != "" {
						allLabelNames[attr.Key] = struct{}{}
					}
				}

				for _, sample := range p.Profile.Sample {
					for _, label := range sample.Label {
						allLabelNames[p.Profile.StringTable[label.Key]] = struct{}{}
					}
				}
			}
		}
	}

	// Create a buffer with all possible labels, pprof labels and pprof num labels as dynamic columns.
	// We use NewBuffer instead of GetBuffer here since analysis showed a very
	// low hit rate, meaning buffers were GCed faster than they could be reused.
	// The downside of using a pool is that buffers are held around for longer.
	// Using NewBuffer means that we pay the price of reallocating a buffer,
	// but they get GCed a lot sooner.
	allLabelNamesKeys := sortedKeys(allLabelNames)
	buffer, err := schema.NewBuffer(map[string][]string{
		profile.ColumnLabels: allLabelNamesKeys,
	})
	if err != nil {
		return nil, err
	}

	row := make(parquet.Row, 0, len(schema.ParquetSchema().Fields()))
	for _, rp := range req.ResourceProfiles {
		for _, attr := range rp.Resource.Attributes {
			if attr.Value.String() != "" {
				allLabelNames[attr.Key] = struct{}{}
			}
		}

		for _, sp := range rp.ScopeProfiles {
			for _, p := range sp.Profiles {
				metas := []profile.Meta{}
				// TODO: Validate that all sample values have the same length as sample type
				for i := 0; i < len(p.Profile.SampleType); i++ {
					metas = append(metas, MetaFromOtelProfile(p.Profile, string(p.ProfileId), i))
				}

				for _, sample := range p.Profile.Sample {
					for j, value := range sample.Value {
						if value == 0 {
							continue
						}

						ls := map[string]string{}

						for _, attr := range p.Attributes {
							if attr.Value.String() != "" {
								ls[attr.Key] = attr.Value.String()
							}
						}

						for _, attr := range rp.Resource.Attributes {
							if attr.Value.String() != "" {
								ls[attr.Key] = attr.Value.String()
							}
						}

						for _, label := range sample.Label {
							ls[p.Profile.StringTable[label.Key]] = p.Profile.StringTable[label.Str]
						}

						row := SampleToParquetRow(
							schema,
							row[:0],
							allLabelNamesKeys, nil, nil,
							ls,
							metas[j],
							&NormalizedSample{
								Locations: serializeOtelStacktrace(
									p.Profile,
									sample,
									p.Profile.Function,
									p.Profile.Mapping,
									p.Profile.StringTable,
									true,
								),
								Value: value,
							},
						)
						if _, err := buffer.WriteRows([]parquet.Row{row}); err != nil {
							return nil, fmt.Errorf("failed to write row to buffer: %w", err)
						}
					}
				}
			}
		}
	}

	if buffer.NumRows() == 0 {
		// If there are no rows in the buffer we simply return early
		return nil, nil
	}

	// We need to sort the buffer so the rows are inserted in sorted order later
	// on the storage nodes.
	buffer.Sort()

	// Convert the sorted buffer to an arrow record.
	converter := pqarrow.NewParquetConverter(mem, logicalplan.IterOptions{})
	defer converter.Close()

	if err := converter.Convert(ctx, buffer, schema); err != nil {
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

func ValidateOtelProfile(p *pprofextended.Profile) error {
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
	p *pprofextended.Profile,
	s *pprofextended.Sample,
	functions []*pprofextended.Function,
	mappings []*pprofextended.Mapping,
	stringTable []string,
	stabiliziedAddress bool,
) [][]byte {
	st := make([][]byte, 0, len(s.LocationIndex)+int(s.LocationsLength))

	// We handle the case where the location IDs are stored in the sample struct.
	for _, locationID := range s.LocationIndex {
		location := p.Location[locationID]
		var m *pprofextended.Mapping

		if location.MappingIndex != 0 && location.MappingIndex-1 < uint64(len(mappings)) {
			m = mappings[location.MappingIndex-1]
		}

		st = append(st, profile.EncodeOtelLocation(
			location,
			m,
			functions,
			stringTable,
			stabiliziedAddress,
		))
	}

	// And the case where the locations are stored in the location slice. And
	// the sample just points to the start and length.
	for _, location := range p.Location[s.LocationsStartIndex : s.LocationsStartIndex+s.LocationsLength] {
		var m *pprofextended.Mapping

		if location.MappingIndex != 0 && location.MappingIndex-1 < uint64(len(mappings)) {
			m = mappings[location.MappingIndex-1]
		}

		st = append(st, profile.EncodeOtelLocation(
			location,
			m,
			functions,
			stringTable,
			stabiliziedAddress,
		))
	}

	return st
}
