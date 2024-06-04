// Copyright 2022-2024 The Parca Authors
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
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/apache/arrow/go/v16/arrow"
	"github.com/apache/arrow/go/v16/arrow/memory"
	"github.com/gogo/status"
	"github.com/parquet-go/parquet-go"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/prometheus/common/model"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pprofextended "github.com/parca-dev/parca/gen/proto/go/opentelemetry/proto/profiles/v1/alternatives/pprofextended"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

var ErrMissingNameLabel = errors.New("missing __name__ label")

type NormalizedProfile struct {
	Samples []*NormalizedSample
	Meta    profile.Meta
}

type NormalizedSample struct {
	Locations [][]byte
	Value     int64
	DiffValue int64
	Label     map[string]string
	NumLabel  map[string]int64
}

type Series struct {
	Labels  map[string]string
	Samples [][]*NormalizedProfile
}

type NormalizedWriteRawRequest struct {
	Series                []Series
	AllLabelNames         []string
	AllPprofLabelNames    []string
	AllPprofNumLabelNames []string
}

func MetaFromPprof(p *pprofpb.Profile, name string, sampleIndex int) profile.Meta {
	periodType := profile.ValueType{}
	if p.PeriodType != nil {
		periodType = profile.ValueType{Type: p.StringTable[p.PeriodType.Type], Unit: p.StringTable[p.PeriodType.Unit]}
	}

	sampleType := profile.ValueType{}
	if p.SampleType != nil {
		sampleType = profile.ValueType{Type: p.StringTable[p.SampleType[sampleIndex].Type], Unit: p.StringTable[p.SampleType[sampleIndex].Unit]}
	}

	return profile.Meta{
		Name:       name,
		Timestamp:  p.TimeNanos / time.Millisecond.Nanoseconds(),
		Duration:   p.DurationNanos,
		Period:     p.Period,
		PeriodType: periodType,
		SampleType: sampleType,
	}
}

func MetaFromOtelProfile(p *pprofextended.Profile, name string, sampleIndex int) profile.Meta {
	periodType := profile.ValueType{}
	if p.PeriodType != nil {
		periodType = profile.ValueType{Type: p.StringTable[p.PeriodType.Type], Unit: p.StringTable[p.PeriodType.Unit]}
	}

	sampleType := profile.ValueType{}
	if p.SampleType != nil {
		sampleType = profile.ValueType{Type: p.StringTable[p.SampleType[sampleIndex].Type], Unit: p.StringTable[p.SampleType[sampleIndex].Unit]}
	}

	return profile.Meta{
		Name:       name,
		Timestamp:  p.TimeNanos / time.Millisecond.Nanoseconds(),
		Duration:   p.DurationNanos,
		Period:     p.Period,
		PeriodType: periodType,
		SampleType: sampleType,
	}
}

func WriteRawRequestToArrowRecord(
	ctx context.Context,
	mem memory.Allocator,
	req *profilestorepb.WriteRawRequest,
	schema *dynparquet.Schema,
) (arrow.Record, error) {
	nr, err := NormalizeWriteRawRequest(
		ctx,
		req,
	)
	if err != nil {
		return nil, err
	}

	r, err := ParquetBufToArrowRecord(
		ctx,
		mem,
		schema,
		nr,
	)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func NormalizePprof(
	ctx context.Context,
	name string,
	takenLabelNames map[string]string,
	p *pprofpb.Profile,
	normalizedAddress bool,
	executableInfo []*profilestorepb.ExecutableInfo,
) ([]*NormalizedProfile, error) {
	profiles := make([]*NormalizedProfile, 0, len(p.SampleType))
	for i := 0; i < len(p.SampleType); i++ {
		normalizedProfile := &NormalizedProfile{
			Meta:    MetaFromPprof(p, name, i),
			Samples: make([]*NormalizedSample, 0, len(p.Sample)),
		}
		profiles = append(profiles, normalizedProfile)
	}

	for _, sample := range p.Sample {
		labels, numLabels := LabelsFromSample(takenLabelNames, p.StringTable, sample.Label)
		for j, value := range sample.Value {
			if value == 0 {
				continue
			}

			profiles[j].Samples = append(profiles[j].Samples, &NormalizedSample{
				Locations: serializePprofStacktrace(
					sample.LocationId,
					p.Location,
					p.Function,
					p.Mapping,
					p.StringTable,
				),
				Value:    sample.Value[j],
				Label:    labels,
				NumLabel: numLabels,
			})
		}
	}

	return profiles, nil
}

// TODO: support num label units.
func LabelsFromSample(takenLabels map[string]string, stringTable []string, plabels []*pprofpb.Label) (map[string]string, map[string]int64) {
	labels := map[string][]string{}
	labelNames := []string{}
	for _, label := range plabels {
		// Only looking at string labels here.
		if label.Str == 0 {
			continue
		}

		key := stringTable[label.Key]
		if _, ok := labels[key]; !ok {
			labels[key] = []string{}
			labelNames = append(labelNames, key)
		}
		labels[key] = append(labels[key], stringTable[label.Str])
	}
	sort.Strings(labelNames)

	resLabels := map[string]string{}
	for _, labelName := range labelNames {
		resLabelName := labelName
		if _, ok := takenLabels[resLabelName]; ok {
			resLabelName = model.ExportedLabelPrefix + resLabelName
		}
		if _, ok := resLabels[resLabelName]; ok {
			resLabelName = model.ExportedLabelPrefix + resLabelName
		}
		resLabels[resLabelName] = labels[labelName][0]
	}

	numLabels := map[string]int64{}
	for _, label := range plabels {
		key := stringTable[label.Key]
		if label.Num != 0 {
			if _, ok := numLabels[key]; !ok {
				numLabels[key] = label.Num
			}
		}
	}

	return resLabels, numLabels
}

func serializePprofStacktrace(
	ids []uint64,
	locations []*pprofpb.Location,
	functions []*pprofpb.Function,
	mappings []*pprofpb.Mapping,
	stringTable []string,
) [][]byte {
	st := make([][]byte, 0, len(ids))

	for _, locationId := range ids {
		location := locations[locationId-1]
		var m *pprofpb.Mapping
		if location.MappingId != 0 {
			mappingIndex := location.MappingId - 1
			m = mappings[mappingIndex]
		}

		st = append(st, profile.EncodePprofLocation(location, m, functions, stringTable, false))
	}

	return st
}

func NormalizeWriteRawRequest(ctx context.Context, req *profilestorepb.WriteRawRequest) (NormalizedWriteRawRequest, error) {
	allLabelNames := make(map[string]struct{})
	allPprofLabelNames := make(map[string]struct{})
	allPprofNumLabelNames := make(map[string]struct{})

	series := make([]Series, 0, len(req.Series))
	for _, rawSeries := range req.Series {
		ls := make(map[string]string, len(rawSeries.Labels.Labels))
		name := ""
		for _, l := range rawSeries.Labels.Labels {
			if l.Name == model.MetricNameLabel {
				name = l.Value
				continue
			}

			if valid := model.LabelName(l.Name).IsValid(); !valid {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "invalid label name: %v", l.Name)
			}

			if _, ok := ls[l.Name]; ok {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "duplicate label name: %v", l.Name)
			}

			ls[l.Name] = l.Value
			allLabelNames[l.Name] = struct{}{}
		}

		if name == "" {
			return NormalizedWriteRawRequest{}, status.Error(codes.InvalidArgument, ErrMissingNameLabel.Error())
		}

		samples := make([][]*NormalizedProfile, 0, len(rawSeries.Samples))
		for _, sample := range rawSeries.Samples {
			if len(sample.RawProfile) >= 2 && sample.RawProfile[0] == 0x1f && sample.RawProfile[1] == 0x8b {
				gz, err := gzip.NewReader(bytes.NewBuffer(sample.RawProfile))
				if err == nil {
					sample.RawProfile, err = io.ReadAll(gz)
				}
				if err != nil {
					return NormalizedWriteRawRequest{}, fmt.Errorf("decompressing profile: %v", err)
				}

				if err := gz.Close(); err != nil {
					return NormalizedWriteRawRequest{}, fmt.Errorf("close gzip reader: %v", err)
				}
			}

			p := &pprofpb.Profile{}
			if err := p.UnmarshalVT(sample.RawProfile); err != nil {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "failed to parse profile: %v", err)
			}

			if err := ValidatePprofProfile(p, sample.ExecutableInfo); err != nil {
				return NormalizedWriteRawRequest{}, status.Errorf(codes.InvalidArgument, "invalid profile: %v", err)
			}

			LabelNamesFromSamples(
				ls,
				p.StringTable,
				p.Sample,
				allPprofLabelNames,
				allPprofNumLabelNames,
			)

			normalizedProfiles, err := NormalizePprof(ctx, name, ls, p, req.Normalized, sample.ExecutableInfo)
			if err != nil {
				return NormalizedWriteRawRequest{}, fmt.Errorf("normalize profile: %w", err)
			}

			samples = append(samples, normalizedProfiles)
		}

		series = append(series, Series{
			Labels:  ls,
			Samples: samples,
		})
	}

	return NormalizedWriteRawRequest{
		Series:                series,
		AllLabelNames:         sortedKeys(allLabelNames),
		AllPprofLabelNames:    sortedKeys(allPprofLabelNames),
		AllPprofNumLabelNames: sortedKeys(allPprofNumLabelNames),
	}, nil
}

func LabelNamesFromSamples(
	takenLabels map[string]string,
	stringTable []string,
	samples []*pprofpb.Sample,
	allLabels map[string]struct{},
	allNumLabels map[string]struct{},
) {
	labels := map[string]struct{}{}
	for _, sample := range samples {
		for _, label := range sample.Label {
			// Only looking at string labels here.
			if label.Str == 0 {
				continue
			}

			key := stringTable[label.Key]
			if _, ok := labels[key]; !ok {
				labels[key] = struct{}{}
			}
		}
	}

	resLabels := map[string]struct{}{}
	for labelName := range labels {
		resLabelName := labelName
		if _, ok := takenLabels[labelName]; ok {
			resLabelName = model.ExportedLabelPrefix + resLabelName
		}
		if _, ok := resLabels[resLabelName]; ok {
			resLabelName = model.ExportedLabelPrefix + resLabelName
		}
		resLabels[resLabelName] = struct{}{}
	}

	for labelName := range resLabels {
		allLabels[labelName] = struct{}{}
	}

	for _, sample := range samples {
		for _, label := range sample.Label {
			key := stringTable[label.Key]
			if label.Num != 0 {
				if _, ok := allNumLabels[key]; !ok {
					allNumLabels[key] = struct{}{}
				}
			}
		}
	}
}

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	out := maps.Keys(m)
	sort.Strings(out)
	return out
}

// SampleToParquetRow converts a sample to a Parquet row. The passed labels
// must be sorted.
func SampleToParquetRow(
	schema *dynparquet.Schema,
	row parquet.Row,
	labelNames, profileLabelNames, profileNumLabelNames []string,
	lset map[string]string,
	meta profile.Meta,
	s *NormalizedSample,
) parquet.Row {
	// schema.Columns() returns a sorted list of all columns.
	// We match on the column's name to insert the correct values.
	// We track the columnIndex to insert each column at the correct index.
	columnIndex := 0
	for _, column := range schema.Columns() {
		switch column.Name {
		case profile.ColumnDuration:
			row = append(row, parquet.ValueOf(meta.Duration).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnName:
			row = append(row, parquet.ValueOf(meta.Name).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriod:
			row = append(row, parquet.ValueOf(meta.Period).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriodType:
			row = append(row, parquet.ValueOf(meta.PeriodType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnPeriodUnit:
			row = append(row, parquet.ValueOf(meta.PeriodType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnSampleType:
			row = append(row, parquet.ValueOf(meta.SampleType.Type).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnSampleUnit:
			row = append(row, parquet.ValueOf(meta.SampleType.Unit).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnStacktrace:
			if len(s.Locations) == 0 {
				row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
			}
			for i, s := range s.Locations {
				switch i {
				case 0:
					row = append(row, parquet.ValueOf(s).Level(0, 1, columnIndex))
				default:
					row = append(row, parquet.ValueOf(s).Level(1, 1, columnIndex))
				}
			}
			columnIndex++
		case profile.ColumnTimestamp:
			row = append(row, parquet.ValueOf(meta.Timestamp).Level(0, 0, columnIndex))
			columnIndex++
		case profile.ColumnValue:
			row = append(row, parquet.ValueOf(s.Value).Level(0, 0, columnIndex))
			columnIndex++

		// All remaining cases take care of dynamic columns
		case profile.ColumnLabels:
			for _, name := range labelNames {
				if value, ok := lset[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case profile.ColumnPprofLabels:
			for _, name := range profileLabelNames {
				if value, ok := s.Label[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		case profile.ColumnPprofNumLabels:
			for _, name := range profileNumLabelNames {
				if value, ok := s.NumLabel[name]; ok {
					row = append(row, parquet.ValueOf(value).Level(0, 1, columnIndex))
				} else {
					row = append(row, parquet.ValueOf(nil).Level(0, 0, columnIndex))
				}
				columnIndex++
			}
		default:
			panic(fmt.Errorf("conversion not implement for column: %s", column.Name))
		}
	}

	return row
}
