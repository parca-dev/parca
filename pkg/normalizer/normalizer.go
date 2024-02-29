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
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/gogo/status"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc/codes"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	profilestorepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

const (
	UnsymolizableLocationAddress = 0x0
)

var ErrMissingNameLabel = errors.New("missing __name__ label")

type Series struct {
	Labels  map[string]string
	Samples [][]*profile.NormalizedProfile
}

type NormalizedWriteRawRequest struct {
	Series                []Series
	AllLabelNames         []string
	AllPprofLabelNames    []string
	AllPprofNumLabelNames []string
}

type Normalizer interface {
	NormalizePprof(
		ctx context.Context,
		name string,
		takenLabelNames map[string]string,
		p *pprofpb.Profile,
		normalizedAddress bool,
		executableInfo []*profilestorepb.ExecutableInfo,
	) ([]*profile.NormalizedProfile, error)
	NormalizeWriteRawRequest(
		ctx context.Context,
		req *profilestorepb.WriteRawRequest,
	) (NormalizedWriteRawRequest, error)
}

type MetastoreNormalizer struct {
	metastore pb.MetastoreServiceClient
	// isAddrNormEnabled indicates whether the metastore normalizer has to
	// normalize sampled addresses for PIC/PIE (position independent code/executable).
	isAddrNormEnabled bool

	addressNormalizationFailed prometheus.Counter
}

func NewNormalizer(
	metastore pb.MetastoreServiceClient,
	enableAddressNormalization bool,
	addressNormalizationFailed prometheus.Counter,
) *MetastoreNormalizer {
	return &MetastoreNormalizer{
		metastore:                  metastore,
		isAddrNormEnabled:          enableAddressNormalization,
		addressNormalizationFailed: addressNormalizationFailed,
	}
}

func (n *MetastoreNormalizer) NormalizePprof(
	ctx context.Context,
	name string,
	takenLabelNames map[string]string,
	p *pprofpb.Profile,
	normalizedAddress bool,
	executableInfo []*profilestorepb.ExecutableInfo,
) ([]*profile.NormalizedProfile, error) {
	// Normalize Location addresses before processing them further. We do this
	// here because otherwise it's very easy to accidentally normalize
	// addresses "multiple" times.
	n.normalizeLocationAddresses(p.Location, p.Mapping, normalizedAddress, executableInfo)

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
		p.Mapping,
		functions,
		len(executableInfo) == len(p.Mapping),
		p.StringTable,
		executableInfo,
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
	mappingsInfo []mappingNormalizationInfo,
	mappings []*pprofpb.Mapping,
	functions []*pb.Function,
	normalizedAddress bool,
	stringTable []string,
	executableInfo []*profilestorepb.ExecutableInfo,
) ([]*pb.Location, error) {
	var err error

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
			mappingNormalizationInfo := mappingsInfo[mappingIndex]

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

func (n *MetastoreNormalizer) normalizeLocationAddresses(
	locations []*pprofpb.Location,
	mappings []*pprofpb.Mapping,
	normalizedAddress bool,
	executableInfo []*profilestorepb.ExecutableInfo,
) {
	var err error
	for _, location := range locations {
		var m *pprofpb.Mapping
		if location.MappingId != 0 {
			mappingIndex := location.MappingId - 1
			m = mappings[mappingIndex]

			if !normalizedAddress {
				if uint64(len(executableInfo)) > mappingIndex && executableInfo[mappingIndex] != nil {
					ei := executableInfo[mappingIndex]
					location.Address, err = NormalizeAddress(location.Address, ei, m.MemoryStart, m.MemoryLimit, m.FileOffset)
					if err != nil {
						// This should never happen, since we already checked that
						// in the agent, but other clients might not. If debugging
						// this is a problem in the futute we should attach this to
						// the distributed trace.
						n.addressNormalizationFailed.Inc()
					}
				} else {
					// This is just best effort and will only work for main executables.
					location.Address = location.Address - m.MemoryStart + m.FileOffset
				}
			}
		}
	}
}

func NormalizeAddress(addr uint64, ei *profilestorepb.ExecutableInfo, start, limit, offset uint64) (uint64, error) {
	base, err := CalculateBase(ei, start, limit, offset)
	if err != nil {
		return addr, fmt.Errorf("calculate base: %w", err)
	}

	return addr - base, nil
}

// Base determines the base address to subtract from virtual
// address to get symbol table address. For an executable, the base
// is 0. Otherwise, it's a shared library, and the base is the
// address where the mapping starts. The kernel needs special handling.
func CalculateBase(ei *profilestorepb.ExecutableInfo, start, limit, offset uint64) (uint64, error) {
	if ei == nil {
		return 0, nil
	}

	if start == 0 && offset == 0 && (limit == ^uint64(0) || limit == 0) {
		// Some tools may introduce a fake mapping that spans the entire
		// address space. Assume that the address has already been
		// adjusted, so no additional base adjustment is necessary.
		return 0, nil
	}

	//nolint:exhaustive
	switch elf.Type(ei.ElfType) {
	case elf.ET_EXEC:
		if ei.LoadSegment == nil {
			// Assume fixed-address executable and so no adjustment.
			return 0, nil
		}
		return start - offset + ei.LoadSegment.Offset - ei.LoadSegment.Vaddr, nil
	case elf.ET_REL:
		if offset != 0 {
			return 0, fmt.Errorf("don't know how to handle mapping.Offset")
		}
		return start, nil
	case elf.ET_DYN:
		// The process mapping information, start = start of virtual address range,
		// and offset = offset in the executable file of the start address, tells us
		// that a runtime virtual address x maps to a file offset
		// fx = x - start + offset.
		if ei.LoadSegment == nil {
			return start - offset, nil
		}

		// The program header, if not nil, indicates the offset in the file where
		// the executable segment is located (loadSegment.Off), and the base virtual
		// address where the first byte of the segment is loaded
		// (loadSegment.Vaddr). A file offset fx maps to a virtual (symbol) address
		// sx = fx - loadSegment.Off + loadSegment.Vaddr.
		//
		// Thus, a runtime virtual address x maps to a symbol address
		// sx = x - start + offset - loadSegment.Off + loadSegment.Vaddr.
		return start - offset + ei.LoadSegment.Offset - ei.LoadSegment.Vaddr, nil
	}

	return 0, fmt.Errorf("don't know how to handle FileHeader.Type %v", elf.Type(ei.ElfType))
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

// NormalizeWriteRawRequest normalizes the profiles
// (mappings, functions, locations, stack traces) to prepare for ingestion.
// It also validates label names of profiles' series,
// decompresses the samples, unmarshals and validates them.
func (n *MetastoreNormalizer) NormalizeWriteRawRequest(ctx context.Context, req *profilestorepb.WriteRawRequest) (NormalizedWriteRawRequest, error) {
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

		samples := make([][]*profile.NormalizedProfile, 0, len(rawSeries.Samples))
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

			normalizedProfiles, err := n.NormalizePprof(ctx, name, ls, p, req.Normalized, sample.ExecutableInfo)
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

func sortedKeys(m map[string]struct{}) []string {
	if len(m) == 0 {
		return nil
	}

	out := maps.Keys(m)
	sort.Strings(out)
	return out
}
