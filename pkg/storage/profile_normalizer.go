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
	"strings"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type profileNormalizer struct {
	logger log.Logger

	// Memoization tables within a profile.
	locationsByID map[uint64]*metastore.Location
	functionsByID map[uint64]*metastore.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	samples   map[stacktraceKey]*Sample
	metaStore metastore.ProfileMetaStore
}

type Sample struct {
	Location []*metastore.Location
	Value    int64
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

// Returns the mapped sample and whether it is new or a known sample.
func (pn *profileNormalizer) mapSample(ctx context.Context, src *profile.Sample, sampleIndex int) (*Sample, bool, error) {
	var err error

	s := &Sample{
		Location: make([]*metastore.Location, len(src.Location)),
		Label:    make(map[string][]string, len(src.Label)),
		NumLabel: make(map[string][]int64, len(src.NumLabel)),
		NumUnit:  make(map[string][]string, len(src.NumLabel)),
	}
	for i, l := range src.Location {
		s.Location[i], err = pn.mapLocation(ctx, l)
		if err != nil {
			return nil, false, err
		}
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
		sa.Value += src.Value[sampleIndex]
		return sa, false, nil
	}

	s.Value += src.Value[sampleIndex]
	pn.samples[k] = s
	return s, true, nil
}

func (pn *profileNormalizer) mapLocation(ctx context.Context, src *profile.Location) (*metastore.Location, error) {
	var err error

	if src == nil {
		return nil, nil
	}

	if l, ok := pn.locationsByID[src.ID]; ok {
		return l, nil
	}

	mi, err := pn.mapMapping(ctx, src.Mapping)
	if err != nil {
		return nil, err
	}
	l := &metastore.Location{
		Mapping:  mi.m,
		Address:  uint64(int64(src.Address) + mi.offset),
		Lines:    make([]metastore.LocationLine, len(src.Line)),
		IsFolded: src.IsFolded,
	}
	for i, ln := range src.Line {
		l.Lines[i], err = pn.mapLine(ctx, ln)
		if err != nil {
			return nil, err
		}
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping ID.
	k := metastore.MakeLocationKey(l)
	loc, err := pn.metaStore.GetLocationByKey(ctx, k)
	if err != nil && err != metastore.ErrLocationNotFound {
		return nil, err
	}
	if loc != nil {
		pn.locationsByID[src.ID] = loc
		return loc, nil
	}
	pn.locationsByID[src.ID] = l

	id, err := pn.metaStore.CreateLocation(ctx, l)
	if err != nil {
		return nil, err
	}

	l.ID = id
	return l, nil
}

type mapInfo struct {
	m      *metastore.Mapping
	offset int64
}

func (pn *profileNormalizer) mapMapping(ctx context.Context, src *profile.Mapping) (mapInfo, error) {
	if src == nil {
		return mapInfo{}, nil
	}

	if mi, ok := pn.mappingsByID[src.ID]; ok {
		return mi, nil
	}

	// Check memoization tables.
	mk := metastore.MakeMappingKey(&metastore.Mapping{
		Start:   src.Start,
		Limit:   src.Limit,
		Offset:  src.Offset,
		File:    src.File,
		BuildID: src.BuildID,
	})
	m, err := pn.metaStore.GetMappingByKey(ctx, mk)
	if err != nil && err != metastore.ErrMappingNotFound {
		return mapInfo{}, err
	}
	if m != nil {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi, nil
	}
	m = &metastore.Mapping{
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
		return mapInfo{}, err
	}
	m.ID = id
	mi := mapInfo{m, 0}
	pn.mappingsByID[src.ID] = mi
	return mi, nil
}

func (pn *profileNormalizer) mapLine(ctx context.Context, src profile.Line) (metastore.LocationLine, error) {
	f, err := pn.mapFunction(ctx, src.Function)
	if err != nil {
		return metastore.LocationLine{}, err
	}

	return metastore.LocationLine{
		Function: f,
		Line:     src.Line,
	}, nil
}

func (pn *profileNormalizer) mapFunction(ctx context.Context, src *profile.Function) (*metastore.Function, error) {
	if src == nil {
		return nil, nil
	}
	if f, ok := pn.functionsByID[src.ID]; ok {
		return f, nil
	}
	k := metastore.MakeFunctionKey(&metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name:       src.Name,
			SystemName: src.SystemName,
			Filename:   src.Filename,
			StartLine:  src.StartLine,
		},
	})
	f, err := pn.metaStore.GetFunctionByKey(ctx, k)
	if err != nil && err != metastore.ErrFunctionNotFound {
		return nil, err
	}
	if f != nil {
		pn.functionsByID[src.ID] = f
		return f, nil
	}
	f = &metastore.Function{
		FunctionKey: metastore.FunctionKey{
			Name:       src.Name,
			SystemName: src.SystemName,
			Filename:   src.Filename,
			StartLine:  src.StartLine,
		},
	}

	id, err := pn.metaStore.CreateFunction(ctx, f)
	if err != nil {
		return nil, err
	}
	f.ID = id

	pn.functionsByID[src.ID] = f
	return f, nil
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
}

// key generates stacktraceKey to be used as a key for maps.
func makeStacktraceKey(sample *Sample) stacktraceKey {
	ids := make([]string, len(sample.Location))
	for i, l := range sample.Location {
		ids[i] = l.ID.String()
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
