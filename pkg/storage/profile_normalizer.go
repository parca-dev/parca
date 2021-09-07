// Copyright 2021 The Parca Authors
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

package storage

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type profileNormalizer struct {
	logger log.Logger

	// Memoization tables within a profile.
	locationsByID map[uint64]*profile.Location
	functionsByID map[uint64]*profile.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	samples   map[stacktraceKey]*profile.Sample
	metaStore metastore.ProfileMetaStore
}

// Returns the mapped sample and whether it is new or a known sample.
func (pn *profileNormalizer) mapSample(ctx context.Context, src *profile.Sample, sampleIndex int) (*profile.Sample, bool) {
	s := &profile.Sample{
		Location: make([]*profile.Location, len(src.Location)),
		Label:    make(map[string][]string, len(src.Label)),
		NumLabel: make(map[string][]int64, len(src.NumLabel)),
		NumUnit:  make(map[string][]string, len(src.NumLabel)),
	}
	for i, l := range src.Location {
		s.Location[i] = pn.mapLocation(ctx, l)
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
	sa, found := pn.samples[k]
	if found {
		sa.Value[0] += src.Value[sampleIndex]
		return sa, false
	}

	s.Value = []int64{src.Value[sampleIndex]}
	pn.samples[k] = s
	return s, true
}

func (pn *profileNormalizer) mapLocation(ctx context.Context, src *profile.Location) *profile.Location {
	if src == nil {
		return nil
	}

	if l, ok := pn.locationsByID[src.ID]; ok {
		return l
	}

	mi := pn.mapMapping(ctx, src.Mapping)
	l := &profile.Location{
		Mapping:  mi.m,
		Address:  uint64(int64(src.Address) + mi.offset),
		Line:     make([]profile.Line, len(src.Line)),
		IsFolded: src.IsFolded,
	}
	for i, ln := range src.Line {
		l.Line[i] = pn.mapLine(ctx, ln)
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping ID.
	k := metastore.MakeLocationKey(l)
	loc, err := pn.metaStore.GetLocationByKey(ctx, k)
	if err != nil {
		level.Debug(pn.logger).Log("msg", "location not found", "key", k, "err", err)
	}
	if loc != nil {
		pn.locationsByID[src.ID] = loc
		return loc
	}
	pn.locationsByID[src.ID] = l

	id, err := pn.metaStore.CreateLocation(ctx, l)
	if err != nil {
		level.Warn(pn.logger).Log("msg", "failed to create location", "err", err)
	} else {
		l.ID = id
	}
	return l
}

type mapInfo struct {
	m      *profile.Mapping
	offset int64
}

func (pn *profileNormalizer) mapMapping(ctx context.Context, src *profile.Mapping) mapInfo {
	if src == nil {
		return mapInfo{}
	}

	if mi, ok := pn.mappingsByID[src.ID]; ok {
		return mi
	}

	// Check memoization tables.
	mk := metastore.MakeMappingKey(src)
	m, err := pn.metaStore.GetMappingByKey(ctx, mk)
	if err != nil {
		level.Debug(pn.logger).Log("msg", "mapping not found", "key", mk, "err", err)
	}
	if m != nil {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi
	}
	m = &profile.Mapping{
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

	// Update memoization tables.
	id, err := pn.metaStore.CreateMapping(ctx, m)
	if err != nil {
		level.Warn(pn.logger).Log("msg", "failed to create mapping", "err", err)
	} else {
		m.ID = id
	}
	mi := mapInfo{m, 0}
	pn.mappingsByID[src.ID] = mi
	return mi
}

func (pn *profileNormalizer) mapLine(ctx context.Context, src profile.Line) profile.Line {
	ln := profile.Line{
		Function: pn.mapFunction(ctx, src.Function),
		Line:     src.Line,
	}
	return ln
}

func (pn *profileNormalizer) mapFunction(ctx context.Context, src *profile.Function) *profile.Function {
	if src == nil {
		return nil
	}
	if f, ok := pn.functionsByID[src.ID]; ok {
		return f
	}
	k := metastore.MakeFunctionKey(src)
	f, err := pn.metaStore.GetFunctionByKey(ctx, k)
	if err != nil {
		level.Debug(pn.logger).Log("msg", "function not found", "key", k, "err", err)
	}
	if f != nil {
		pn.functionsByID[src.ID] = f
		return f
	}
	f = &profile.Function{
		Name:       src.Name,
		SystemName: src.SystemName,
		Filename:   src.Filename,
		StartLine:  src.StartLine,
	}

	id, err := pn.metaStore.CreateFunction(ctx, f)
	if err != nil {
		level.Warn(pn.logger).Log("msg", "failed to create function", "err", err)
	} else {
		f.ID = id
	}

	pn.functionsByID[src.ID] = f
	return f
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
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
		locations: strings.Join(ids, "|"),
		labels:    strings.Join(labels, ""),
		numlabels: strings.Join(numlabels, ""),
	}
}
