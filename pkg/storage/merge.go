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
)

var (
	ErrPeriodTypeMismatch = errors.New("cannot merge profiles of different period type")
	ErrSampleTypeMismatch = errors.New("cannot merge profiles of different sample type")
)

type MergeProfile struct {
	a InstantProfile
	b InstantProfile

	meta InstantProfileMeta
}

func MergeProfiles(profiles ...InstantProfile) (InstantProfile, error) {
	profileCh := make(chan InstantProfile)

	return MergeProfilesConcurrent(
		trace.NewNoopTracerProvider().Tracer(""),
		context.Background(),
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

func MergeSeriesSetProfiles(tracer trace.Tracer, ctx context.Context, set SeriesSet) (InstantProfile, error) {
	profileCh := make(chan InstantProfile)

	return MergeProfilesConcurrent(
		tracer,
		ctx,
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
					profileCh <- CopyInstantProfile(it.At())
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
	tracer trace.Tracer,
	ctx context.Context,
	profileCh chan InstantProfile,
	concurrency int,
	producerFunc func() error,
) (InstantProfile, error) {
	ctx, span := tracer.Start(ctx, "MergeProfilesConcurrent")
	span.SetAttributes(attribute.Int("concurrency", concurrency))
	defer span.End()

	var res InstantProfile

	resCh := make(chan InstantProfile, concurrency)
	pairCh := make(chan [2]InstantProfile)

	var mergesPerformed atomic.Uint32
	var profilesRead atomic.Uint32

	g := &errgroup.Group{}

	g.Go(producerFunc)

	g.Go(func() error {
		var first InstantProfile
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

		var second InstantProfile
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

		pairCh <- [2]InstantProfile{first, second}

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

			pairCh <- [2]InstantProfile{first, second}
		}
	})

	for i := 0; i < concurrency; i++ {
		g.Go(func() error {
			for {
				select {
				case <-ctx.Done():
					return nil
				case pair := <-pairCh:
					if pair == [2]InstantProfile{nil, nil} {
						return nil
					}

					m, err := NewMergeProfile(pair[0], pair[1])
					if err != nil {
						return err
					}

					p := CopyInstantProfile(m)

					resCh <- p
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

func NewMergeProfile(a, b InstantProfile) (InstantProfile, error) {
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
		meta: InstantProfileMeta{
			PeriodType: metaA.PeriodType,
			SampleType: metaA.SampleType,
			Timestamp:  timestamp,
			Duration:   metaA.Duration + metaB.Duration,
			Period:     period,
		},
	}, nil
}

func equalValueType(a ValueType, b ValueType) bool {
	return a.Type == b.Type && a.Unit == b.Unit
}

func (m *MergeProfile) ProfileMeta() InstantProfileMeta {
	return m.meta
}

type MergeProfileTree struct {
	m *MergeProfile
}

func (m *MergeProfile) ProfileTree() InstantProfileTree {
	return &MergeProfileTree{
		m: m,
	}
}

func (m *MergeProfileTree) RootCumulativeValue() int64 {
	return 0
}

func (m *MergeProfileTree) Iterator() InstantProfileTreeIterator {
	return &MergeProfileTreeIterator{
		a:      m.m.a.ProfileTree().Iterator(),
		b:      m.m.b.ProfileTree().Iterator(),
		stackA: InstantProfileTreeStack{{}},
		stackB: InstantProfileTreeStack{{}},
	}
}

type InstantProfileTreeStackItem struct {
	node    InstantProfileTreeNode
	started bool
	done    bool
}

type InstantProfileTreeStack []*InstantProfileTreeStackItem

func (s *InstantProfileTreeStack) Push(e *InstantProfileTreeStackItem) {
	*s = append(*s, e)
}

func (s *InstantProfileTreeStack) Pop() (*InstantProfileTreeStackItem, bool) {
	if s.IsEmpty() {
		return nil, false
	} else {
		index := len(*s) - 1   // Get the index of the top most element.
		element := (*s)[index] // Index into the slice and obtain the element.
		*s = (*s)[:index]      // Remove it from the stack by slicing it off.
		return element, true
	}
}

func (s *InstantProfileTreeStack) Peek() *InstantProfileTreeStackItem {
	return (*s)[len(*s)-1]
}

func (s *InstantProfileTreeStack) IsEmpty() bool {
	return len(*s) == 0
}

func (s *InstantProfileTreeStack) Size() int {
	return len(*s)
}

type MergeProfileTreeIterator struct {
	a InstantProfileTreeIterator
	b InstantProfileTreeIterator

	stackA InstantProfileTreeStack
	stackB InstantProfileTreeStack
}

func (i *MergeProfileTreeIterator) HasMore() bool {
	return !i.stackA.IsEmpty() || !i.stackB.IsEmpty()
}

func (i *MergeProfileTreeIterator) NextChild() bool {
	sizeA := i.stackA.Size()
	sizeB := i.stackB.Size()
	if sizeA > sizeB {
		nextA := i.a.NextChild()
		if nextA {
			return true
		}
		i.stackA.Peek().done = true
		return false
	}
	if sizeA < sizeB {
		nextB := i.b.NextChild()
		if nextB {
			return true
		}
		i.stackB.Peek().done = true
		return false
	}

	peekA := i.stackA.Peek()
	peekB := i.stackB.Peek()

	if !peekA.started || !peekB.started {
		if !peekA.started {
			nextA := i.a.NextChild()
			if !nextA {
				peekA.done = true
			}
			peekA.started = true
		}
		if !peekB.started {
			nextB := i.b.NextChild()
			if !nextB {
				peekB.done = true
			}
			peekB.started = true
		}
		return !(peekA.done && peekB.done)
	}

	aDone := peekA.done
	bDone := peekB.done

	if aDone && bDone {
		return false
	}

	if !aDone && bDone {
		nextA := i.a.NextChild()
		if nextA {
			return true
		}
		i.stackA.Peek().done = true
		return false
	}

	if aDone && !bDone {
		nextB := i.b.NextChild()
		if nextB {
			return true
		}
		i.stackB.Peek().done = true
		return false
	}

	// both are not done

	atA := i.a.At()
	atB := i.b.At()
	locA := atA.LocationID()
	locB := atB.LocationID()

	if uuidCompare(locA, locB) == -1 {
		nextA := i.a.NextChild()
		if nextA {
			return true
		}
		i.stackA.Peek().done = true
		// Must return true to let the curB finish.
		return true
	}

	if uuidCompare(locA, locB) == 1 {
		nextB := i.b.NextChild()
		if nextB {
			return true
		}
		i.stackB.Peek().done = true
		// Must return true to let the curA finish.
		return true
	}

	aHasNext := i.a.NextChild()
	if !aHasNext {
		i.stackA.Peek().done = true
	}
	bHasNext := i.b.NextChild()
	if !bHasNext {
		i.stackB.Peek().done = true
	}

	return aHasNext || bHasNext
}

func (i *MergeProfileTreeIterator) At() InstantProfileTreeNode {
	sizeA := i.stackA.Size()
	sizeB := i.stackB.Size()

	if sizeA > sizeB {
		return i.a.At()
	}
	if sizeA < sizeB {
		return i.b.At()
	}

	peekA := i.stackA.Peek()
	peekB := i.stackB.Peek()

	aDone := peekA.done
	bDone := peekB.done

	if !aDone && bDone {
		return i.a.At()
	}

	if aDone && !bDone {
		return i.b.At()
	}

	atA := i.a.At()
	atB := i.b.At()
	locA := atA.LocationID()
	locB := atB.LocationID()

	if uuidCompare(locA, locB) == -1 {
		return atA
	}

	if uuidCompare(locA, locB) == 1 {
		return atB
	}

	return MergeInstantProfileTreeNodes(atA, atB)
}

func (i *MergeProfileTreeIterator) StepInto() bool {
	sizeA := i.stackA.Size()
	sizeB := i.stackB.Size()

	if sizeA > sizeB {
		atA := i.a.At()
		steppedInto := i.a.StepInto()
		if steppedInto {
			i.stackA.Push(&InstantProfileTreeStackItem{node: atA})
		}

		return steppedInto
	}
	if sizeA < sizeB {
		atB := i.b.At()
		steppedInto := i.b.StepInto()
		if steppedInto {
			i.stackB.Push(&InstantProfileTreeStackItem{node: atB})
		}

		return steppedInto
	}

	peekA := i.stackA.Peek()
	peekB := i.stackB.Peek()

	aDone := peekA.done
	bDone := peekB.done

	if !aDone && bDone {
		atA := i.a.At()
		steppedInto := i.a.StepInto()
		if steppedInto {
			i.stackA.Push(&InstantProfileTreeStackItem{node: atA})
		}

		return steppedInto
	}

	if aDone && !bDone {
		atB := i.b.At()
		steppedInto := i.b.StepInto()
		if steppedInto {
			i.stackB.Push(&InstantProfileTreeStackItem{node: atB})
		}

		return steppedInto
	}

	atA := i.a.At()
	atB := i.b.At()
	locA := atA.LocationID()
	locB := atB.LocationID()

	if uuidCompare(locA, locB) == -1 {
		steppedInto := i.a.StepInto()
		if steppedInto {
			i.stackA.Push(&InstantProfileTreeStackItem{node: atA})
		}

		return steppedInto
	}

	if uuidCompare(locA, locB) == 1 {
		steppedInto := i.b.StepInto()
		if steppedInto {
			i.stackB.Push(&InstantProfileTreeStackItem{node: atB})
		}

		return steppedInto
	}

	steppedIntoA := i.a.StepInto()
	if steppedIntoA {
		i.stackA.Push(&InstantProfileTreeStackItem{node: atA})
	}
	steppedIntoB := i.b.StepInto()
	if steppedIntoB {
		i.stackB.Push(&InstantProfileTreeStackItem{node: atB})
	}

	return steppedIntoA || steppedIntoB
}

func (i *MergeProfileTreeIterator) StepUp() {
	sizeA := i.stackA.Size()
	sizeB := i.stackB.Size()

	// Using greater than or equal to and less than or equal to in order to pop
	// both stacks if they are equal.

	if sizeA >= sizeB {
		i.stackA.Pop()
		i.a.StepUp()
	}
	if sizeA <= sizeB {
		i.stackB.Pop()
		i.b.StepUp()
	}
}

func MergeInstantProfileTreeNodes(a, b InstantProfileTreeNode) InstantProfileTreeNode {
	var flatValues []*ProfileTreeValueNode
	flatA := a.FlatValues()
	if len(flatA) > 0 {
		flatValues = append(flatValues, &ProfileTreeValueNode{Value: flatA[0].Value})
	}

	flatB := b.FlatValues()
	if len(flatB) > 0 {
		if len(flatValues) > 0 {
			flatValues[0].Value += flatB[0].Value
		} else {
			flatValues = append(flatValues, &ProfileTreeValueNode{Value: flatB[0].Value})
		}
	}

	return &ProfileTreeNode{
		locationID: a.LocationID(),
		flatValues: flatValues,
	}
}
