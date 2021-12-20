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
	"time"

	"github.com/google/pprof/profile"
)

type ValueType struct {
	Type string
	Unit string
}

type InstantProfileMeta struct {
	PeriodType ValueType
	SampleType ValueType
	Timestamp  int64
	Duration   int64
	Period     int64
}

func CopyInstantFlatProfile(p InstantProfile) *FlatProfile {
	return &FlatProfile{
		Meta:    p.ProfileMeta(),
		samples: p.Samples(),
	}
}

type InstantProfile interface {
	ProfileMeta() InstantProfileMeta
	Samples() map[string]*Sample
}

type ProfileSeriesIterator interface {
	Next() bool
	At() InstantProfile
	Err() error
}

type ProfileSeries interface {
	Iterator() ProfileSeriesIterator
}

type SliceProfileSeriesIterator struct {
	samples []InstantProfile
	i       int
	err     error
}

func (i *SliceProfileSeriesIterator) Next() bool {
	if i.err != nil {
		return false
	}

	i.i++
	return i.i < len(i.samples)
}

func (i *SliceProfileSeriesIterator) At() InstantProfile {
	return i.samples[i.i]
}

func (i *SliceProfileSeriesIterator) Err() error {
	return i.err
}

func ProfileMetaFromPprof(p *profile.Profile, sampleIndex int) InstantProfileMeta {
	return InstantProfileMeta{
		Timestamp:  p.TimeNanos / time.Millisecond.Nanoseconds(),
		Duration:   p.DurationNanos,
		Period:     p.Period,
		PeriodType: ValueType{Type: p.PeriodType.Type, Unit: p.PeriodType.Unit},
		SampleType: ValueType{Type: p.SampleType[sampleIndex].Type, Unit: p.SampleType[sampleIndex].Unit},
	}
}

type ScaledInstantProfile struct {
	p     InstantProfile
	ratio float64
}

func NewScaledInstantProfile(p InstantProfile, ratio float64) InstantProfile {
	return &ScaledInstantProfile{
		p:     p,
		ratio: ratio,
	}
}

func (p *ScaledInstantProfile) ProfileMeta() InstantProfileMeta {
	return p.p.ProfileMeta()
}

func (p *ScaledInstantProfile) Samples() map[string]*Sample {
	samples := p.p.Samples()
	for _, s := range samples {
		s.Value = int64(p.ratio * float64(s.Value))
	}
	return samples
}
