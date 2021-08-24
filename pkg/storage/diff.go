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

	for locBase < locCompare && !i.baseStack.Peek().done {
		nextBase := i.base.NextChild()
		if nextBase {
			atBase = i.base.At()
			locBase = atBase.LocationID()
			continue
		}
		i.baseStack.Peek().done = true
	}

	if locBase > locCompare || i.baseStack.Peek().done {
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

	if locBase > locCompare {
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

	cumulativeValues := []*ProfileTreeValueNode{{}}
	cumulativeA := base.CumulativeValues()
	if len(cumulativeA) > 0 {
		cumulativeValues[0].Value -= cumulativeA[0].Value
	}

	cumulativeB := compare.CumulativeValues()
	if len(cumulativeB) > 0 {
		cumulativeValues[0].Value += cumulativeB[0].Value
	}

	return &ProfileTreeNode{
		locationID:           compare.LocationID(),
		flatValues:           compare.FlatValues(),
		cumulativeValues:     compare.CumulativeValues(),
		flatDiffValues:       flatValues,
		cumulativeDiffValues: cumulativeValues,
	}
}
