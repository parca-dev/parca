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

type DiffProfileTree struct {
	d *DiffProfile
}

func (d *DiffProfile) ProfileTree() InstantProfileTree {
	return &DiffProfileTree{
		d: d,
	}
}

func (d *DiffProfileTree) RootCumulativeValue() int64 {
	return 0
}

func (d *DiffProfileTree) Iterator() InstantProfileTreeIterator {
	return &DiffProfileTreeIterator{
		base:         d.d.base.ProfileTree().Iterator(),
		compare:      d.d.compare.ProfileTree().Iterator(),
		baseStack:    InstantProfileTreeStack{{}},
		compareStack: InstantProfileTreeStack{{}},
	}
}

type DiffProfileTreeIterator struct {
	base    InstantProfileTreeIterator
	compare InstantProfileTreeIterator

	baseStack    InstantProfileTreeStack
	compareStack InstantProfileTreeStack
}

func (i *DiffProfileTreeIterator) HasMore() bool {
	return !i.compareStack.IsEmpty()
}

func (i *DiffProfileTreeIterator) NextChild() bool {
	baseStackSize := i.baseStack.Size()
	compareStackSize := i.compareStack.Size()
	if baseStackSize < compareStackSize {
		nextCompare := i.compare.NextChild()
		if nextCompare {
			return true
		}
		i.compareStack.Peek().done = true
		return false
	}

	peekBase := i.baseStack.Peek()
	peekCompare := i.compareStack.Peek()

	if !peekBase.started || !peekCompare.started {
		if !peekBase.started {
			nextBase := i.base.NextChild()
			if !nextBase {
				peekBase.done = true
			}
			peekBase.started = true
		}
		if !peekCompare.started {
			nextCompare := i.compare.NextChild()
			if !nextCompare {
				peekCompare.done = true
			}
			peekCompare.started = true
		}
		return !peekCompare.done
	}

	baseDone := peekBase.done
	compareDone := peekCompare.done

	if compareDone {
		return false
	}

	if baseDone {
		nextCompare := i.compare.NextChild()
		if nextCompare {
			return true
		}
		i.compareStack.Peek().done = true
		return false
	}

	// both are not done

	atBase := i.base.At()
	atCompare := i.compare.At()
	locBase := atBase.LocationID()
	locCompare := atCompare.LocationID()

	for uuidCompare(locBase, locCompare) == -1 && !i.baseStack.Peek().done {
		nextBase := i.base.NextChild()
		if nextBase {
			atBase = i.base.At()
			locBase = atBase.LocationID()
			continue
		}
		i.baseStack.Peek().done = true
	}

	if uuidCompare(locBase, locCompare) == 1 || i.baseStack.Peek().done {
		nextCompare := i.compare.NextChild()
		if nextCompare {
			return true
		}
		i.compareStack.Peek().done = true
		// Must return true to let the curA finish.
		return false
	}

	baseHasNext := i.base.NextChild()
	if !baseHasNext {
		i.baseStack.Peek().done = true
	}
	compareHasNext := i.compare.NextChild()
	if !compareHasNext {
		i.compareStack.Peek().done = true
	}

	return compareHasNext
}

func (i *DiffProfileTreeIterator) At() InstantProfileTreeNode {
	sizeBase := i.baseStack.Size()
	sizeCompare := i.compareStack.Size()

	if sizeBase < sizeCompare {
		return DiffInstantProfileTreeNodes(&ProfileTreeNode{}, i.compare.At())
	}

	peekBase := i.baseStack.Peek()
	peekCompare := i.compareStack.Peek()

	baseDone := peekBase.done
	compareDone := peekCompare.done

	if baseDone && !compareDone {
		return DiffInstantProfileTreeNodes(&ProfileTreeNode{}, i.compare.At())
	}

	atBase := i.base.At()
	atCompare := i.compare.At()
	locBase := atBase.LocationID()
	locCompare := atCompare.LocationID()

	if locBase == locCompare {
		return DiffInstantProfileTreeNodes(atBase, atCompare)
	}

	return DiffInstantProfileTreeNodes(&ProfileTreeNode{}, atCompare)
}

func (i *DiffProfileTreeIterator) StepInto() bool {
	sizeBase := i.baseStack.Size()
	sizeCompare := i.compareStack.Size()

	if sizeCompare > sizeBase {
		atCompare := i.compare.At()
		steppedInto := i.compare.StepInto()
		if steppedInto {
			i.compareStack.Push(&InstantProfileTreeStackItem{node: atCompare})
		}

		return steppedInto
	}

	peekBase := i.baseStack.Peek()
	peekCompare := i.compareStack.Peek()

	baseDone := peekBase.done
	compareDone := peekCompare.done

	if baseDone && !compareDone {
		atCompare := i.compare.At()
		steppedInto := i.compare.StepInto()
		if steppedInto {
			i.compareStack.Push(&InstantProfileTreeStackItem{node: atCompare})
		}

		return steppedInto
	}

	atBase := i.base.At()
	atCompare := i.compare.At()
	locBase := atBase.LocationID()
	locCompare := atCompare.LocationID()

	if uuidCompare(locBase, locCompare) == 1 {
		atCompare := i.compare.At()
		steppedInto := i.compare.StepInto()
		if steppedInto {
			i.compareStack.Push(&InstantProfileTreeStackItem{node: atCompare})
		}

		return steppedInto
	}

	steppedIntoCompare := i.compare.StepInto()
	if steppedIntoCompare {
		i.compareStack.Push(&InstantProfileTreeStackItem{node: atCompare})

		steppedIntoBase := i.base.StepInto()
		if steppedIntoBase {
			i.baseStack.Push(&InstantProfileTreeStackItem{node: atBase})
		}
	}

	return steppedIntoCompare
}

func (i *DiffProfileTreeIterator) StepUp() {
	sizeBase := i.baseStack.Size()
	sizeCompare := i.compareStack.Size()

	// Using greater than or equal to and less than or equal to in order to pop
	// both stacks if they are equal.

	if sizeBase >= sizeCompare {
		i.baseStack.Pop()
		i.base.StepUp()
	}
	if sizeBase <= sizeCompare {
		i.compareStack.Pop()
		i.compare.StepUp()
	}
}

func DiffInstantProfileTreeNodes(base, compare InstantProfileTreeNode) InstantProfileTreeNode {
	var flatValues []*ProfileTreeValueNode
	flatA := base.FlatValues()
	if len(flatA) > 0 {
		if flatValues == nil {
			flatValues = []*ProfileTreeValueNode{{}}
		}
		flatValues[0].Value -= flatA[0].Value
	}

	flatB := compare.FlatValues()
	if len(flatB) > 0 {
		if flatValues == nil {
			flatValues = []*ProfileTreeValueNode{{}}
		}
		flatValues[0].Value += flatB[0].Value
	}

	return &ProfileTreeNode{
		locationID:     compare.LocationID(),
		flatValues:     compare.FlatValues(),
		flatDiffValues: flatValues,
	}
}
