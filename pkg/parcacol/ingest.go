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

package parcacol

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/prometheus/model/labels"

	pprofproto "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	"github.com/parca-dev/parca/pkg/profile"
)

type Table interface {
	Schema() *dynparquet.Schema
	Insert(context.Context, []byte) (tx uint64, err error)
}

type Ingester struct {
	logger     log.Logger
	table      Table
	normalizer *Normalizer
	schema     *dynparquet.Schema
	bufferPool *sync.Pool
}

func NewIngester(
	logger log.Logger,
	normalizer *Normalizer,
	table Table,
	schema *dynparquet.Schema,
	bufferPool *sync.Pool,
) *Ingester {
	return &Ingester{
		logger:     logger,
		normalizer: normalizer,
		table:      table,
		schema:     schema,
		bufferPool: bufferPool,
	}
}

var ErrMissingNameLabel = errors.New("missing __name__ label")

func separateNameFromLabels(ls labels.Labels) (string, map[string]struct{}, labels.Labels, error) {
	names := make(map[string]struct{}, len(ls))
	out := make(labels.Labels, 0, len(ls))
	name := ""
	for _, l := range ls {
		if l.Name == "__name__" {
			name = l.Value
		} else {
			names[l.Name] = struct{}{}
			out = append(out, l)
		}
	}

	if name == "" {
		return "", nil, nil, ErrMissingNameLabel
	}

	return name, names, out, nil
}

func (ing Ingester) Ingest(ctx context.Context, ls labels.Labels, p *pprofproto.Profile, normalized bool) error {
	name, names, ls, err := separateNameFromLabels(ls)
	if err != nil {
		return fmt.Errorf("prepare labels: %w", err)
	}

	if err := validatePprofProfile(p); err != nil {
		return err
	}

	normalizedProfiles, err := ing.normalizer.NormalizePprof(ctx, name, names, p, normalized)
	if err != nil {
		return fmt.Errorf("normalize profile: %w", err)
	}

	for _, p := range normalizedProfiles {
		if len(p.Samples) == 0 {
			level.Debug(ing.logger).Log("msg", "no samples found in profile, dropping it", "name", p.Meta.Name, "sample_type", p.Meta.SampleType.Type, "sample_unit", p.Meta.SampleType.Unit, "labels", ls)
			continue
		}

		if err := ing.IngestProfile(ctx, ls, p); err != nil {
			return fmt.Errorf("ingest profile: %w", err)
		}
	}

	return nil
}

func (ing Ingester) IngestProfile(ctx context.Context, ls labels.Labels, p *profile.NormalizedProfile) error {
	buf := ing.bufferPool.Get().(*bytes.Buffer)
	buf.Reset()
	defer ing.bufferPool.Put(buf)

	err := NormalizedProfileToParquetBuffer(buf, ing.schema, ls, p)
	if err != nil {
		return fmt.Errorf("failed to convert samples to buffer: %w", err)
	}

	_, err = ing.table.Insert(ctx, buf.Bytes())
	if err != nil {
		return fmt.Errorf("insert buffer: %w", err)
	}

	return nil
}

func validatePprofProfile(p *pprofproto.Profile) error {
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
