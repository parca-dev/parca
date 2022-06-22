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
	"context"
	"fmt"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

type Normalizer struct {
	metastore pb.MetastoreServiceClient
}

func NewNormalizer(metastore pb.MetastoreServiceClient) *Normalizer {
	return &Normalizer{
		metastore: metastore,
	}
}

func (n *Normalizer) NormalizePprof(ctx context.Context, name string, p *pprofpb.Profile, normalizedAddress bool) ([]*profile.NormalizedProfile, error) {
	// TODO(brancz): validate incoming pprof profile

	mappings, err := n.NormalizeMappings(ctx, p.Mapping, p.StringTable)
	if err != nil {
		return nil, err
	}

	functions, err := n.NormalizeFunctions(ctx, p.Function, p.StringTable)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	stacktraces, err := n.NormalizeStacktraces(ctx, p.Sample, locations)
	if err != nil {
		return nil, err
	}

	profiles := make([]*profile.NormalizedProfile, 0, len(p.SampleType))
	for i := 0; i < len(p.SampleType); i++ {
		normalizedProfile := &profile.NormalizedProfile{
			Meta:    profile.MetaFromPprof(p, name, i),
			Samples: make([]*profile.NormalizedSample, 0, len(p.Sample)),
		}
		profiles = append(profiles, normalizedProfile)
	}

	for i, sample := range p.Sample {
		labels, numLabels := labelsFromSample(p.StringTable, sample.Label)
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

			profiles[j].Samples = append(profiles[j].Samples, ns)
		}
	}

	return profiles, nil
}

func labelsFromSample(stringTable []string, plabels []*pprofpb.Label) (map[string]string, map[string]int64) {
	// TODO: support num label units.
	labels := map[string]string{}
	numLabels := map[string]int64{}

	for _, label := range plabels {
		key := stringTable[label.Key]
		if label.Str != 0 {
			if _, ok := labels[key]; !ok {
				labels[key] = stringTable[label.Str]
			}
			continue
		}
		if label.Num != 0 {
			if _, ok := numLabels[key]; !ok {
				numLabels[key] = label.Num
			}
		}
	}

	return labels, numLabels
}

type mappingNormalizationInfo struct {
	id     string
	offset int64
}

func (n *Normalizer) NormalizeMappings(ctx context.Context, mappings []*pprofpb.Mapping, stringTable []string) ([]mappingNormalizationInfo, error) {
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

func (n *Normalizer) NormalizeFunctions(ctx context.Context, functions []*pprofpb.Function, stringTable []string) ([]*pb.Function, error) {
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

func (n *Normalizer) NormalizeLocations(
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
		addr := location.Address
		mappingId := ""
		if location.MappingId != 0 {
			mappingIndex := location.MappingId - 1
			mappingNormalizationInfo := mappings[mappingIndex]

			if !normalizedAddress {
				addr = uint64(int64(addr) + mappingNormalizationInfo.offset)
			}
			mappingId = mappingNormalizationInfo.id
		}

		lines := &pb.LocationLines{Entries: make([]*pb.Line, 0, len(location.Line))}
		for _, line := range location.Line {
			functionId := ""
			if line.FunctionId != 0 {
				functionIndex := line.FunctionId - 1
				functionId = functions[functionIndex].Id
			}
			lines.Entries = append(lines.Entries, &pb.Line{
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

func (n *Normalizer) NormalizeStacktraces(ctx context.Context, samples []*pprofpb.Sample, locations []*pb.Location) ([]*pb.Stacktrace, error) {
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
