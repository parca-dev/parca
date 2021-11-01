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
	"time"

	"github.com/go-kit/log"
	"github.com/google/pprof/profile"
	"github.com/google/uuid"
	"github.com/parca-dev/parca/pkg/storage/metastore"
)

type InstantProfileTreeNode interface {
	LocationID() uuid.UUID
	FlatValues() []*ProfileTreeValueNode
	FlatDiffValues() []*ProfileTreeValueNode
}

type InstantProfileTreeIterator interface {
	HasMore() bool
	NextChild() bool
	At() InstantProfileTreeNode
	StepInto() bool
	StepUp()
}

type InstantProfileTree interface {
	RootCumulativeValue() int64
	Iterator() InstantProfileTreeIterator
}

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

func WalkProfileTree(pt InstantProfileTree, f func(n InstantProfileTreeNode) error) error {
	it := pt.Iterator()

	for it.HasMore() {
		if it.NextChild() {
			if err := f(it.At()); err != nil {
				return err
			}
			it.StepInto()
			continue
		}
		it.StepUp()
	}

	return nil
}

func CopyInstantProfile(p InstantProfile) *Profile {
	return &Profile{
		Meta: p.ProfileMeta(),
		Tree: CopyInstantProfileTree(p.ProfileTree()),
	}
}

func CopyInstantProfileTree(pt InstantProfileTree) *ProfileTree {
	it := pt.Iterator()
	if !it.HasMore() || !it.NextChild() {
		return nil
	}

	node := it.At()
	cur := &ProfileTreeNode{
		locationID:     node.LocationID(),
		flatDiffValues: node.FlatDiffValues(),
		flatValues:     node.FlatValues(),
	}
	tree := &ProfileTree{Roots: &ProfileTreeRootNode{
		CumulativeValue: pt.RootCumulativeValue(),
		ProfileTreeNode: cur,
	}}
	stack := ProfileTreeStack{{node: cur}}

	steppedInto := it.StepInto()
	if !steppedInto {
		return tree
	}

	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			cur := &ProfileTreeNode{
				locationID:     node.LocationID(),
				flatDiffValues: node.FlatDiffValues(),
				flatValues:     node.FlatValues(),
			}

			stack.Peek().node.Children = append(stack.Peek().node.Children, cur)

			steppedInto := it.StepInto()
			if steppedInto {
				stack.Push(&ProfileTreeStackEntry{
					node: cur,
				})
			}
			continue
		}
		it.StepUp()
		stack.Pop()
	}

	return tree
}

type InstantProfile interface {
	ProfileTree() InstantProfileTree
	ProfileMeta() InstantProfileMeta
}

type ProfileSeriesIterator interface {
	Next() bool
	At() InstantProfile
	Err() error
}

type ProfileSeries interface {
	Iterator() ProfileSeriesIterator
}

type Profile struct {
	Tree *ProfileTree
	Meta InstantProfileMeta
}

func (p *Profile) ProfileTree() InstantProfileTree {
	return p.Tree
}

func (p *Profile) ProfileMeta() InstantProfileMeta {
	return p.Meta
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

// ProfilesFromPprof extracts a Profile from each sample index included in the
// pprof profile.
func ProfilesFromPprof(ctx context.Context, l log.Logger, s metastore.ProfileMetaStore, p *profile.Profile) ([]*Profile, error) {
	ps := make([]*Profile, 0, len(p.SampleType))

	for i := range p.SampleType {
		tree, err := ProfileTreeFromPprof(ctx, l, s, p, i)
		if err != nil {
			return nil, err
		}

		ps = append(ps, &Profile{
			Tree: tree,
			Meta: ProfileMetaFromPprof(p, i),
		})
	}

	return ps, nil
}

func ProfileFromPprof(ctx context.Context, l log.Logger, s metastore.ProfileMetaStore, p *profile.Profile, sampleIndex int) (*Profile, error) {
	tree, err := ProfileTreeFromPprof(ctx, l, s, p, sampleIndex)
	if err != nil {
		return nil, err
	}

	return &Profile{
		Tree: tree,
		Meta: ProfileMetaFromPprof(p, sampleIndex),
	}, nil
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

func (p *ScaledInstantProfile) ProfileTree() InstantProfileTree {
	return &ScaledInstantProfileTree{
		tree:  p.p.ProfileTree(),
		ratio: p.ratio,
	}
}

type ScaledInstantProfileTree struct {
	tree  InstantProfileTree
	ratio float64
}

func (t *ScaledInstantProfileTree) RootCumulativeValue() int64 {
	return 0
}

func (t *ScaledInstantProfileTree) Iterator() InstantProfileTreeIterator {
	return &ScaledInstantProfileTreeIterator{
		it:    t.tree.Iterator(),
		ratio: t.ratio,
	}
}

type ScaledInstantProfileTreeIterator struct {
	it    InstantProfileTreeIterator
	ratio float64
}

func (i *ScaledInstantProfileTreeIterator) HasMore() bool {
	return i.it.HasMore()
}

func (i *ScaledInstantProfileTreeIterator) NextChild() bool {
	return i.it.NextChild()
}

func (i *ScaledInstantProfileTreeIterator) At() InstantProfileTreeNode {
	n := i.it.At()

	flatValues := n.FlatValues()
	for _, v := range flatValues {
		v.Value = int64(i.ratio * float64(v.Value))
	}

	return &ProfileTreeNode{
		locationID: n.LocationID(),
		flatValues: flatValues,
	}
}

func (i *ScaledInstantProfileTreeIterator) StepInto() bool {
	return i.it.StepInto()
}

func (i *ScaledInstantProfileTreeIterator) StepUp() {
	i.it.StepUp()
}
