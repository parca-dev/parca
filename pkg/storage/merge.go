package storage

import (
	"errors"
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
	h := len(profiles) / 2

	var (
		firstHalfMerge  InstantProfile
		secondHalfMerge InstantProfile
		err             error
	)

	firstHalf := profiles[:h]
	secondHalf := profiles[h:]

	if len(firstHalf) == 1 {
		firstHalfMerge = firstHalf[0]
	} else if len(firstHalf) == 0 {
		// intentionally do nothing
	} else {
		firstHalfMerge, err = MergeProfiles(firstHalf...)
		if err != nil {
			return nil, err
		}
	}

	if len(secondHalf) == 1 {
		secondHalfMerge = secondHalf[0]
	} else if len(secondHalf) == 0 {
		// intentionally do nothing
	} else {
		secondHalfMerge, err = MergeProfiles(secondHalf...)
		if err != nil {
			return nil, err
		}
	}

	if firstHalfMerge == nil {
		return secondHalfMerge, nil
	}

	if secondHalfMerge == nil {
		return firstHalfMerge, nil
	}

	return NewMergeProfile(
		firstHalfMerge,
		secondHalfMerge,
	)
}

func NewMergeProfile(a, b InstantProfile) (*MergeProfile, error) {
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

	if locA < locB {
		nextA := i.a.NextChild()
		if nextA {
			return true
		}
		i.stackA.Peek().done = true
		// Must return true to let the curB finish.
		return true
	}

	if locA > locB {
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

	if locA < locB {
		return atA
	}

	if locA > locB {
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

	if locA < locB {
		steppedInto := i.a.StepInto()
		if steppedInto {
			i.stackA.Push(&InstantProfileTreeStackItem{node: atA})
		}

		return steppedInto
	}

	if locA > locB {
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
	flatValues := []*ProfileTreeValueNode{{}}
	flatA := a.FlatValues()
	if len(flatA) > 0 {
		flatValues[0].Value += flatA[0].Value
	}

	flatB := b.FlatValues()
	if len(flatB) > 0 {
		flatValues[0].Value += flatB[0].Value
	}

	cumulativeValues := []*ProfileTreeValueNode{{}}
	cumulativeA := a.CumulativeValues()
	if len(cumulativeA) > 0 {
		cumulativeValues[0].Value += cumulativeA[0].Value
	}

	cumulativeB := b.CumulativeValues()
	if len(cumulativeB) > 0 {
		cumulativeValues[0].Value += cumulativeB[0].Value
	}

	return &ProfileTreeNode{
		locationID:       a.LocationID(),
		flatValues:       flatValues,
		cumulativeValues: cumulativeValues,
	}
}
