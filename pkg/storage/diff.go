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
	"errors"
)

var (
	ErrDiffPeriodTypeMismatch = errors.New("cannot diff profiles of different period type")
	ErrDiffSampleTypeMismatch = errors.New("cannot diff profiles of different sample type")
)

type DiffProfile struct {
	base    InstantProfile
	compare InstantProfile

	meta InstantProfileMeta
}

func NewDiffProfile(base, compare InstantProfile) (*DiffProfile, error) {
	baseMeta := base.ProfileMeta()
	compareMeta := compare.ProfileMeta()

	if !equalValueType(baseMeta.PeriodType, compareMeta.PeriodType) {
		return nil, ErrDiffPeriodTypeMismatch
	}

	if !equalValueType(baseMeta.SampleType, compareMeta.SampleType) {
		return nil, ErrDiffSampleTypeMismatch
	}

	return &DiffProfile{
		base:    base,
		compare: compare,
		meta: InstantProfileMeta{
			PeriodType: baseMeta.PeriodType,
			SampleType: baseMeta.SampleType,
		},
	}, nil
}

func (d *DiffProfile) ProfileMeta() InstantProfileMeta {
	return d.meta
}

func (d *DiffProfile) Samples() map[string]*Sample {
	bs := d.base.Samples()
	cs := d.compare.Samples()

	ss := make(map[string]*Sample, len(bs))

	for k, s := range cs {
		if sb, ok := bs[k]; ok {
			s.DiffValue = s.Value - sb.Value
			ss[k] = s
		} else {
			ss[k] = s
		}
	}

	return ss
}

type DiffProfileTree struct {
	d *DiffProfile
}

func (d *DiffProfileTree) RootCumulativeValue() int64 {
	return 0
}
