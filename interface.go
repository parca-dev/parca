package storage

type InstantProfileTreeNode interface {
	LocationID() uint64
	CumulativeValue() int64
	CumulativeValues() []*ProfileTreeValueNode
	FlatValues() []*ProfileTreeValueNode
}

type ProfileTreeValueNode struct {
	Value    int64
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type InstantProfileTreeIterator interface {
	HasMore() bool
	NextChild() bool
	At() InstantProfileTreeNode
	StepInto() bool
	StepUp()
}

type InstantProfileTree interface {
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

func WalkProfileTree(pt InstantProfileTree, f func(n InstantProfileTreeNode)) {
	it := pt.Iterator()

	for it.HasMore() {
		if it.NextChild() {
			f(it.At())
			it.StepInto()
			continue
		}
		it.StepUp()
	}
}

func CopyInstantProfileTree(pt InstantProfileTree) *ProfileTree {
	it := pt.Iterator()
	if !it.HasMore() || !it.NextChild() {
		return nil
	}

	node := it.At()
	cur := &ProfileTreeNode{
		locationID:       node.LocationID(),
		flatValues:       node.FlatValues(),
		cumulativeValues: node.CumulativeValues(),
	}
	tree := &ProfileTree{Roots: cur}
	stack := ProfileTreeStack{{node: cur}}

	steppedInto := it.StepInto()
	if !steppedInto {
		return tree
	}

	for it.HasMore() {
		if it.NextChild() {
			node := it.At()
			cur := &ProfileTreeNode{
				locationID:       node.LocationID(),
				flatValues:       node.FlatValues(),
				cumulativeValues: node.CumulativeValues(),
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
	tree *ProfileTree
	meta InstantProfileMeta
}

func (p *Profile) ProfileTree() InstantProfileTree {
	return p.tree
}

func (p *Profile) ProfileMeta() InstantProfileMeta {
	return p.meta
}
