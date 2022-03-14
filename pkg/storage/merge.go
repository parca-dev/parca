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
	"errors"
	"fmt"
	"runtime"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/atomic"
	"golang.org/x/sync/errgroup"

	"github.com/parca-dev/parca/pkg/profile"
)

var (
	ErrPeriodTypeMismatch = errors.New("cannot merge profiles of different period type")
	ErrSampleTypeMismatch = errors.New("cannot merge profiles of different sample type")
)

type MergeProfile struct {
	a profile.InstantProfile
	b profile.InstantProfile

	meta profile.InstantProfileMeta
}

func MergeProfiles(profiles ...profile.InstantProfile) (profile.InstantProfile, error) {
	profileCh := make(chan profile.InstantProfile)

	return MergeProfilesConcurrent(
		context.Background(),
		trace.NewNoopTracerProvider().Tracer(""),
		profileCh,
		runtime.NumCPU(),
		func() error {
			for _, p := range profiles {
				profileCh <- p
			}
			close(profileCh)
			return nil
		},
	)
}

func MergeSeriesSetProfiles(ctx context.Context, tracer trace.Tracer, set SeriesSet) (profile.InstantProfile, error) {
	profileCh := make(chan profile.InstantProfile)

	return MergeProfilesConcurrent(
		ctx,
		tracer,
		profileCh,
		runtime.NumCPU(),
		func() error {
			_, seriesSpan := tracer.Start(ctx, "seriesIterate")
			defer seriesSpan.End()
			defer close(profileCh)
			for {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
				}

				if !set.Next() {
					return nil
				}
				series := set.At()

				i := 0
				_, profileSpan := tracer.Start(ctx, "profileIterate")
				it := series.Iterator()
				for it.Next() {
					// Have to copy as profile pointer is not stable for more than the
					// current iteration.
					profileCh <- profile.CopyInstantProfile(it.At())
					i++
				}
				profileSpan.End()
				if err := it.Err(); err != nil {
					profileSpan.RecordError(err)
					return err
				}
			}
		},
	)
}

func MergeProfilesConcurrent(
	ctx context.Context,
	tracer trace.Tracer,
	profileCh chan profile.InstantProfile,
	concurrency int,
	producerFunc func() error,
) (profile.InstantProfile, error) {
	ctx, span := tracer.Start(ctx, "MergeProfilesConcurrent")
	span.SetAttributes(attribute.Int("concurrency", concurrency))
	defer span.End()

	var res profile.InstantProfile

	resCh := make(chan profile.InstantProfile, concurrency)
	pairCh := make(chan [2]profile.InstantProfile)

	var mergesPerformed atomic.Uint32
	var profilesRead atomic.Uint32

	g := &errgroup.Group{}

	g.Go(producerFunc)

	g.Go(func() error {
		var first profile.InstantProfile
		select {
		case first = <-profileCh:
			if first == nil {
				close(pairCh)
				return nil
			}
			profilesRead.Inc()
		case <-ctx.Done():
			return ctx.Err()
		}

		var second profile.InstantProfile
		select {
		case second = <-profileCh:
			if second == nil {
				res = first
				close(pairCh)
				return nil
			}
			profilesRead.Inc()
		case <-ctx.Done():
			return ctx.Err()
		}

		pairCh <- [2]profile.InstantProfile{first, second}

		for {
			first = nil
			second = nil
			select {
			case first = <-resCh:
				mergesPerformed.Inc()
			case <-ctx.Done():
				return ctx.Err()
			}

			select {
			case second = <-profileCh:
				if second != nil {
					profilesRead.Inc()
				}
			case <-ctx.Done():
				return ctx.Err()
			}

			if second == nil {
				read := profilesRead.Load()
				merged := mergesPerformed.Load()
				// For any N inputs we need exactly N-1 merge operations. So we
				// know we are done when we have done that many operations.
				if read == merged+1 {
					res = first
					close(pairCh)
					return nil
				}
				select {
				case second = <-resCh:
					if second != nil {
						mergesPerformed.Inc()
					}
				case <-ctx.Done():
					return ctx.Err()
				}
			}

			pairCh <- [2]profile.InstantProfile{first, second}
		}
	})

	for i := 0; i < concurrency; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case pair := <-pairCh:
					if pair == [2]profile.InstantProfile{nil, nil} {
						return nil
					}

					m, err := NewMergeProfile(pair[0], pair[1])
					if err != nil {
						return err
					}

					resCh <- profile.CopyInstantProfile(m)
				}
			}
		})
	}

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if res == nil {
		return nil, fmt.Errorf("no profiles to merge")
	}

	return res, nil
}

func NewMergeProfile(a, b profile.InstantProfile) (profile.InstantProfile, error) {
	if a != nil && b == nil {
		return a, nil
	}
	if a == nil && b != nil {
		return b, nil
	}

	metaA := a.ProfileMeta()
	metaB := b.ProfileMeta()

	if !equalValueType(metaA.PeriodType, metaB.PeriodType) {
		return nil, ErrPeriodTypeMismatch
	}

	if !equalValueType(metaA.SampleType, metaB.SampleType) {
		return nil, ErrSampleTypeMismatch
	}

	timestamp := metaA.Timestamp
	if metaA.Timestamp > metaB.Timestamp {
		timestamp = metaB.Timestamp
	}

	period := metaA.Period
	if metaA.Period > metaB.Period {
		period = metaB.Period
	}

	return &MergeProfile{
		a: a,
		b: b,
		meta: profile.InstantProfileMeta{
			PeriodType: metaA.PeriodType,
			SampleType: metaA.SampleType,
			Timestamp:  timestamp,
			Duration:   metaA.Duration + metaB.Duration,
			Period:     period,
		},
	}, nil
}

func equalValueType(a, b profile.ValueType) bool {
	return a.Type == b.Type && a.Unit == b.Unit
}

func (m *MergeProfile) ProfileMeta() profile.InstantProfileMeta {
	return m.meta
}

func (m *MergeProfile) Samples() map[string]*profile.Sample {
	as := m.a.Samples()
	bs := m.b.Samples()

	samples := make(map[string]*profile.Sample, len(as)+len(bs)) // TODO: Don't allocate a new map, and especially not worst case

	// Merge intersection for A to B
	for k, s := range as {
		samples[k] = &profile.Sample{
			Value:    s.Value,
			Location: s.Location,
			Label:    s.Label,
			NumLabel: s.NumLabel,
			NumUnit:  s.NumUnit,
		}
		if b, found := bs[k]; found {
			// Sum the actual values if k is found in bs
			samples[k].Value += b.Value
		}
	}
	for k, s := range bs {
		if _, found := samples[k]; found {
			// skip samples that exist in the final map, they've been merged already
			continue
		}
		samples[k] = &profile.Sample{
			Value:    s.Value,
			Location: s.Location,
			Label:    s.Label,
			NumLabel: s.NumLabel,
			NumUnit:  s.NumUnit,
		}
	}

	return samples
}
