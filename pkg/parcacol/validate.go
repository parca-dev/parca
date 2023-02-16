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

package parcacol

import (
	"fmt"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func ValidatePprofProfile(p *pprofpb.Profile) error {
	stringTableLen := int64(len(p.StringTable))

	if stringTableLen > 0 && p.StringTable[0] != "" {
		return fmt.Errorf("first item in string table is expected to be empty string, but it is %q", p.StringTable[0])
	}

	// Check that all mappings/locations/functions are in the tables
	// Check that there are no duplicate ids
	mappingsNum := uint64(len(p.Mapping))
	for i, m := range p.Mapping {
		if m == nil {
			return fmt.Errorf("profile has nil mapping")
		}
		if m.Id != uint64(i+1) {
			return fmt.Errorf("mapping id is not sequential")
		}
		if m.Filename != 0 && m.Filename > stringTableLen {
			return fmt.Errorf("mapping (id: %d) has invalid filename index %d", m.Id, m.Filename)
		}
		if m.BuildId != 0 && m.BuildId > stringTableLen {
			return fmt.Errorf("mapping (id: %d) has invalid buildid index %d", m.Id, m.Filename)
		}
	}

	functionsNum := uint64(len(p.Function))
	for i, f := range p.Function {
		if f == nil {
			return fmt.Errorf("profile has nil function")
		}
		if f.Id != uint64(i+1) {
			return fmt.Errorf("function id is not sequential")
		}
		if f.Name != 0 && f.Name > stringTableLen {
			return fmt.Errorf("function (id: %d) has invalid name index %d", f.Id, f.Name)
		}
		if f.SystemName != 0 && f.SystemName > stringTableLen {
			return fmt.Errorf("function (id: %d) has invalid systemname index %d", f.Id, f.SystemName)
		}
		if f.Filename != 0 && f.Filename > stringTableLen {
			return fmt.Errorf("function (id: %d) has invalid filename index %d", f.Id, f.Filename)
		}
	}

	locationsNum := uint64(len(p.Location))
	for i, l := range p.Location {
		if l == nil {
			return fmt.Errorf("profile has nil location")
		}
		if l.Id != uint64(i+1) {
			return fmt.Errorf("location id is not sequential")
		}
		if l.MappingId != 0 && l.MappingId > mappingsNum {
			return fmt.Errorf("location has invalid mapping id: %d", l.MappingId)
		}
		for _, ln := range l.Line {
			if ln.FunctionId != 0 && ln.FunctionId > functionsNum {
				return fmt.Errorf("location %d has invalid function id: %d", l.Id, ln.FunctionId)
			}
		}
	}

	// Check that sample values are consistent
	sampleLen := len(p.SampleType)
	if sampleLen == 0 && len(p.Sample) != 0 {
		return fmt.Errorf("missing sample type information")
	}

	for i, s := range p.Sample {
		if s == nil {
			return fmt.Errorf("profile has nil sample")
		}
		if len(s.Value) != sampleLen {
			return fmt.Errorf("mismatch: sample has %d values vs. %d types", len(s.Value), len(p.SampleType))
		}
		for j, l := range s.LocationId {
			if l == 0 {
				return fmt.Errorf("location ids of stacktraces must be non-zero")
			}
			if l > locationsNum {
				return fmt.Errorf("sample %d location number %d (%d) is out of range", i, j, l)
			}
		}
		for j, label := range s.Label {
			if label.Key == 0 {
				return fmt.Errorf("sample %d label %d has no key", i, j)
			}
			if label.Key != 0 && label.Key > stringTableLen {
				return fmt.Errorf("sample %d label %d has invalid key index %d", i, j, label.Key)
			}
			if label.Str != 0 && label.Str > stringTableLen {
				return fmt.Errorf("sample %d label %d has invalid str index %d", i, j, label.Str)
			}
		}
	}

	return nil
}
