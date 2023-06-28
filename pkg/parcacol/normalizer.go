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
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/prometheus/common/model"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	UnsymolizableLocationAddress = 0x0
)

type MetastoreNormalizer struct {
	metastore pb.MetastoreServiceClient
	// isAddrNormEnabled indicates whether the metastore normalizer has to
	// normalize sampled addresses for PIC/PIE (position independent code/executable).
	isAddrNormEnabled bool
}

func NewNormalizer(metastore pb.MetastoreServiceClient, enableAddressNormalization bool) *MetastoreNormalizer {
	return &MetastoreNormalizer{
		metastore:         metastore,
		isAddrNormEnabled: enableAddressNormalization,
	}
}

func (n *MetastoreNormalizer) NormalizePprof(ctx context.Context, name string, takenLabelNames map[string]string, p *pprofpb.Profile, normalizedAddress bool) ([]*profile.NormalizedProfile, error) {
	mappings, err := n.NormalizeMappings(ctx, p.Mapping, p.StringTable)
	if err != nil {
		return nil, fmt.Errorf("normalize mappings: %w", err)
	}

	functions, err := n.NormalizeFunctions(ctx, p.Function, p.StringTable)
	if err != nil {
		return nil, fmt.Errorf("normalize functions: %w", err)
	}

	locations, err := n.NormalizeLocations(
		ctx,
		p.Location,
		mappings,
		functions,
		normalizedAddress,
		p.StringTable,
	)
	if err != nil {
		return nil, fmt.Errorf("normalize locations: %w", err)
	}

	stacktraces, err := n.NormalizeStacktraces(ctx, p.Sample, locations)
	if err != nil {
		return nil, fmt.Errorf("normalize stacktraces: %w", err)
	}

	sampleIndex := map[int]map[string]int{}
	profiles := make([]*profile.NormalizedProfile, 0, len(p.SampleType))
	for i := 0; i < len(p.SampleType); i++ {
		normalizedProfile := &profile.NormalizedProfile{
			Meta:    profile.MetaFromPprof(p, name, i),
			Samples: make([]*profile.NormalizedSample, 0, len(p.Sample)),
		}
		profiles = append(profiles, normalizedProfile)
		sampleIndex[i] = map[string]int{}
	}

	for i, sample := range p.Sample {
		labels, numLabels := LabelsFromSample(takenLabelNames, p.StringTable, sample.Label)
		key := sampleKey(stacktraces[i].Id, labels, numLabels)
		for j, value := range sample.Value {
			if value == 0 {
				continue
			}

			ns := &profile.NormalizedSample{
				StacktraceID: stacktraces[i].Id,
				Value:        sample.Value[j],
				Label:        labels,
				NumLabel:     numLabels,
			}

			index, ok := sampleIndex[j][key]
			if !ok {
				profiles[j].Samples = append(profiles[j].Samples, ns)
				sampleIndex[j][key] = len(profiles[j].Samples) - 1
			} else {
				profiles[j].Samples[index].Value += ns.Value
			}
		}
	}

	return profiles, nil
}

// sampleKey combines stack trace ID and all key-value label pairs
// with a semicolon delimeter.
func sampleKey(stacktraceID string, labels map[string]string, numLabels map[string]int64) string {
	var key strings.Builder
	key.WriteString(stacktraceID)
	key.WriteRune(';')

	for k, v := range labels {
		key.WriteString(k)
		key.WriteRune('=')
		key.WriteString(v)
		key.WriteRune(';')
	}
	key.WriteRune(';')

	for k, v := range numLabels {
		key.WriteString(k)
		key.WriteRune('=')
		key.WriteString(strconv.FormatInt(v, 10))
		key.WriteRune(';')
	}

	return key.String()
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

type mappingNormalizationInfo struct {
	id     string
	offset int64
}

func (n *MetastoreNormalizer) NormalizeMappings(ctx context.Context, mappings []*pprofpb.Mapping, stringTable []string) ([]mappingNormalizationInfo, error) {
	req := &pb.GetOrCreateMappingsRequest{
		Mappings: make([]*pb.Mapping, 0, len(mappings)),
	}

	for _, mapping := range mappings {
		req.Mappings = append(req.Mappings, &pb.Mapping{
			Start:           mapping.MemoryStart,
			Limit:           mapping.MemoryLimit,
			Offset:          mapping.FileOffset,
			File:            stringTable[mapping.Filename],
			BuildId:         stringTable[mapping.BuildId],
			HasFunctions:    mapping.HasFunctions,
			HasFilenames:    mapping.HasFilenames,
			HasLineNumbers:  mapping.HasLineNumbers,
			HasInlineFrames: mapping.HasInlineFrames,
		})
	}

	res, err := n.metastore.GetOrCreateMappings(ctx, req)
	if err != nil {
		return nil, err
	}

	mapInfos := make([]mappingNormalizationInfo, 0, len(res.Mappings))
	for i, mapping := range res.Mappings {
		mapInfos = append(mapInfos, mappingNormalizationInfo{
			id:     mapping.Id,
			offset: int64(mappings[i].MemoryStart) - int64(mapping.Start),
		})
	}

	return mapInfos, nil
}

func (n *MetastoreNormalizer) NormalizeFunctions(ctx context.Context, functions []*pprofpb.Function, stringTable []string) ([]*pb.Function, error) {
	req := &pb.GetOrCreateFunctionsRequest{
		Functions: make([]*pb.Function, 0, len(functions)),
	}

	for _, function := range functions {
		req.Functions = append(req.Functions, &pb.Function{
			StartLine:  function.StartLine,
			Name:       stringTable[function.Name],
			SystemName: stringTable[function.SystemName],
			Filename:   stringTable[function.Filename],
		})
	}

	res, err := n.metastore.GetOrCreateFunctions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("get or create functions: %w", err)
	}

	return res.Functions, nil
}

func (n *MetastoreNormalizer) NormalizeLocations(
	ctx context.Context,
	locations []*pprofpb.Location,
	mappings []mappingNormalizationInfo,
	functions []*pb.Function,
	normalizedAddress bool,
	stringTable []string,
) ([]*pb.Location, error) {
	req := &pb.GetOrCreateLocationsRequest{
		Locations: make([]*pb.Location, 0, len(locations)),
	}

	for _, location := range locations {
		if location.MappingId == 0 && len(location.Line) == 0 {
			req.Locations = append(req.Locations, &pb.Location{
				// Locations that have no lines and no mapping are never going
				// to be possible to be symbolized, so might as well at least
				// make them the same and therefore deduplicate them.
				Address: UnsymolizableLocationAddress,
			})
			continue
		}

		addr := location.Address
		mappingId := ""
		if location.MappingId != 0 {
			mappingIndex := location.MappingId - 1
			mappingNormalizationInfo := mappings[mappingIndex]

			if n.isAddrNormEnabled && !normalizedAddress {
				addr = uint64(int64(addr) + mappingNormalizationInfo.offset)
			}
			mappingId = mappingNormalizationInfo.id
		}

		lines := make([]*pb.Line, 0, len(location.Line))
		for _, line := range location.Line {
			functionId := ""
			if line.FunctionId != 0 {
				functionIndex := line.FunctionId - 1
				functionId = functions[functionIndex].Id
			}
			lines = append(lines, &pb.Line{
				FunctionId: functionId,
				Line:       line.Line,
			})
		}

		req.Locations = append(req.Locations, &pb.Location{
			Address:   addr,
			IsFolded:  location.IsFolded,
			MappingId: mappingId,
			Lines:     lines,
		})
	}

	res, err := n.metastore.GetOrCreateLocations(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Locations, nil
}

func (n *MetastoreNormalizer) NormalizeStacktraces(ctx context.Context, samples []*pprofpb.Sample, locations []*pb.Location) ([]*pb.Stacktrace, error) {
	req := &pb.GetOrCreateStacktracesRequest{
		Stacktraces: make([]*pb.Stacktrace, 0, len(samples)),
	}

	for _, sample := range samples {
		locationIds := make([]string, 0, len(sample.LocationId))

		for _, locationId := range sample.LocationId {
			locationIds = append(locationIds, locations[locationId-1].Id)
		}

		req.Stacktraces = append(req.Stacktraces, &pb.Stacktrace{
			LocationIds: locationIds,
		})
	}

	res, err := n.metastore.GetOrCreateStacktraces(ctx, req)
	if err != nil {
		return nil, err
	}

	return res.Stacktraces, nil
}
