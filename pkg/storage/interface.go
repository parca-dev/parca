package storage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

type InstantProfileTreeNode interface {
	LocationID() uint64
	CumulativeValue() int64
	CumulativeValues() []*ProfileTreeValueNode
	FlatValues() []*ProfileTreeValueNode
}

type ProfileTreeValueNode struct {
	key *ProfileTreeValueNodeKey

	Value    int64
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type ProfileTreeValueNodeKey struct {
	location  string
	labels    string
	numlabels string
}

func (k *ProfileTreeValueNodeKey) Equals(o ProfileTreeValueNodeKey) bool {
	if k.location != o.location {
		return false
	}
	if k.labels != o.labels {
		return false
	}
	if k.numlabels != o.numlabels {
		return false
	}
	return true
}

func (n *ProfileTreeValueNode) Key(locationID uint64) {
	if n.key != nil {
		return
	}

	labels := make([]string, 0, len(n.Label))
	for k, v := range n.Label {
		labels = append(labels, fmt.Sprintf("%q%q", k, v))
	}
	sort.Strings(labels)

	numlabels := make([]string, 0, len(n.NumLabel))
	for k, v := range n.NumLabel {
		numlabels = append(numlabels, fmt.Sprintf("%q%x%x", k, v, n.NumUnit[k]))
	}
	sort.Strings(numlabels)

	n.key = &ProfileTreeValueNodeKey{
		location:  strconv.FormatUint(locationID, 10),
		labels:    strings.Join(labels, ""),
		numlabels: strings.Join(numlabels, ""),
	}
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
	Tree *ProfileTree
	Meta InstantProfileMeta
}

func (p *Profile) ProfileTree() InstantProfileTree {
	return p.Tree
}

func (p *Profile) ProfileMeta() InstantProfileMeta {
	return p.Meta
}

// ProfilesFromPprof extracts a Profile from each sample index included in the
// pprof profile.
func ProfilesFromPprof(s ProfileMetaStore, p *profile.Profile) []*Profile {
	ps := make([]*Profile, 0, len(p.SampleType))

	for i := range p.SampleType {
		ps = append(ps, &Profile{
			Tree: ProfileTreeFromPprof(s, p, i),
			Meta: ProfileMetaFromPprof(p, i),
		})
	}

	return ps
}

func ProfileFromPprof(s ProfileMetaStore, p *profile.Profile, sampleIndex int) *Profile {
	return &Profile{
		Tree: ProfileTreeFromPprof(s, p, sampleIndex),
		Meta: ProfileMetaFromPprof(p, sampleIndex),
	}
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

func ProfileTreeFromPprof(s ProfileMetaStore, p *profile.Profile, sampleIndex int) *ProfileTree {
	pn := &profileNormalizer{
		metaStore: s,

		samples: make(map[stacktraceKey]*profile.Sample, len(p.Sample)),

		// Profile-specific hash tables for each profile inserted.
		locationsByID: make(map[uint64]*profile.Location, len(p.Location)),
		functionsByID: make(map[uint64]*profile.Function, len(p.Function)),
		mappingsByID:  make(map[uint64]mapInfo, len(p.Mapping)),
	}

	samples := make([]*profile.Sample, 0, len(p.Sample))
	for _, s := range p.Sample {
		if !isZeroSample(s) {
			sa, isNew := pn.mapSample(s, sampleIndex)
			if isNew {
				samples = append(samples, sa)
			}
		}
	}
	sortSamples(samples)

	profileTree := NewProfileTree()
	for _, s := range samples {
		profileTree.Insert(s)
	}

	return profileTree
}
