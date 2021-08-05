package storage

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/storage/chunkenc"
	"github.com/prometheus/prometheus/pkg/labels"
)

var (
	ErrOutOfOrderSample = errors.New("out of order sample")
)

type MemSeriesTreeValueNode struct {
	Values   chunkenc.Chunk
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type MemSeriesTreeNode struct {
	LocationID       uint64
	FlatValues       []*MemSeriesTreeValueNode
	CumulativeValues []*MemSeriesTreeValueNode
	Children         []*MemSeriesTreeNode
}

type MemSeriesTree struct {
	Roots *MemSeriesTreeNode
}

func (t *MemSeriesTree) Iterator() *MemSeriesTreeIterator {
	return NewMemSeriesTreeIterator(t)
}

func (t *MemSeriesTree) Insert(index uint16, profileTree *ProfileTree) error {
	if t.Roots == nil {
		t.Roots = &MemSeriesTreeNode{}
	}

	pit := profileTree.Iterator()
	sit := t.Iterator()

	for pit.HasMore() {
		if pit.NextChild() {
			profileTreeChild := pit.At()
			pId := profileTreeChild.LocationID()

			done := false
			for {
				if !sit.NextChild() {
					node := sit.Node()
					seriesTreeChild := &MemSeriesTreeNode{
						LocationID: pId,
					}
					if len(profileTreeChild.FlatValues()) > 0 {
						if seriesTreeChild.FlatValues == nil {
							seriesTreeChild.FlatValues = []*MemSeriesTreeValueNode{{
								Values: chunkenc.NewXORChunk(),
							}}
						}
						app, err := seriesTreeChild.FlatValues[0].Values.Appender()
						if err != nil {
							return err
						}
						app.AppendAt(index, profileTreeChild.FlatValues()[0].Value)
					}
					if seriesTreeChild.CumulativeValues == nil {
						seriesTreeChild.CumulativeValues = []*MemSeriesTreeValueNode{{
							Values: chunkenc.NewXORChunk(),
						}}
					}
					app, err := seriesTreeChild.CumulativeValues[0].Values.Appender()
					if err != nil {
						return err
					}
					app.AppendAt(index, profileTreeChild.CumulativeValues()[0].Value)

					node.Children = append(node.Children, seriesTreeChild)

					pit.StepInto()
					sit.StepInto()
					done = true
					break
				}
				sId := sit.At().LocationID
				if pId == sId || pId < sId {
					break
				}
			}
			if done {
				continue
			}

			seriesTreeChild := sit.At()
			sId := seriesTreeChild.LocationID

			// The node with the location id in the profile-tree is the same,
			// this means this node present in the series-tree, so we need add the new values to the existing node.
			if pId == sId {
				if len(profileTreeChild.FlatValues()) > 0 {
					// REVIEW: I don't think this can ever be nil, can it? It'll have existing values if pId == sId?!
					if seriesTreeChild.FlatValues == nil {
						seriesTreeChild.FlatValues = []*MemSeriesTreeValueNode{{
							Values: chunkenc.NewXORChunk(),
						}}
					}
					app, err := seriesTreeChild.FlatValues[0].Values.Appender()
					if err != nil {
						return err
					}
					app.AppendAt(index, profileTreeChild.FlatValues()[0].Value)
				}
				if seriesTreeChild.CumulativeValues == nil {
					seriesTreeChild.CumulativeValues = []*MemSeriesTreeValueNode{{
						Values: chunkenc.NewXORChunk(),
					}}
				}
				app, err := seriesTreeChild.CumulativeValues[0].Values.Appender()
				if err != nil {
					return err
				}
				app.AppendAt(index, profileTreeChild.CumulativeValues()[0].Value)

				pit.StepInto()
				sit.StepInto()
				continue
			}

			// The node with the location id in the profile-tree is smaller,
			// this means this node is not present yet in the series-tree, so it has to be added at the current child position.
			if pId < sId {
				node := sit.Node()
				childIndex := sit.ChildIndex()
				newChildren := make([]*MemSeriesTreeNode, len(node.Children)+1)
				copy(newChildren, node.Children[:childIndex])
				newChild := &MemSeriesTreeNode{
					LocationID: pId,
				}
				if len(profileTreeChild.FlatValues()) > 0 {
					newChild.FlatValues = []*MemSeriesTreeValueNode{{
						Values: chunkenc.NewXORChunk(),
					}}
					app, err := newChild.FlatValues[0].Values.Appender()
					if err != nil {
						return err
					}
					app.AppendAt(index, profileTreeChild.FlatValues()[0].Value)
				}
				newChild.CumulativeValues = []*MemSeriesTreeValueNode{{
					Values: chunkenc.NewXORChunk(),
				}}
				app, err := newChild.CumulativeValues[0].Values.Appender()
				if err != nil {
					return err
				}
				app.AppendAt(index, profileTreeChild.CumulativeValues()[0].Value)

				newChildren[childIndex] = newChild
				copy(newChildren[childIndex+1:], node.Children[childIndex:])
				node.Children = newChildren

				pit.StepInto()
				sit.StepInto()
				continue
			}
		}
		pit.StepUp()
		sit.StepUp()
	}

	return nil
}

type ProfileTreeNode struct {
	locationID       uint64
	flatValues       []*ProfileTreeValueNode
	cumulativeValues []*ProfileTreeValueNode
	Children         []*ProfileTreeNode
}

func (n *ProfileTreeNode) LocationID() uint64 {
	return n.locationID
}

func (n *ProfileTreeNode) CumulativeValue() int64 {
	res := int64(0)
	for _, cv := range n.cumulativeValues {
		res += cv.Value
	}

	return res
}

func (n *ProfileTreeNode) CumulativeValues() []*ProfileTreeValueNode {
	return n.cumulativeValues
}

func (n *ProfileTreeNode) FlatValues() []*ProfileTreeValueNode {
	return n.flatValues
}

type ProfileTree struct {
	Roots *ProfileTreeNode
}

func NewProfileTree() *ProfileTree {
	return &ProfileTree{
		Roots: &ProfileTreeNode{
			cumulativeValues: []*ProfileTreeValueNode{{
				Value: 0,
			}},
		},
	}
}

func (t *ProfileTree) Iterator() InstantProfileTreeIterator {
	return NewProfileTreeIterator(t)
}

func (t *ProfileTree) Insert(sample *profile.Sample) {

	cur := t.Roots
	locations := sample.Location

	for i := len(locations) - 1; i >= 0; i-- {
		nextId := locations[i].ID

		var child *ProfileTreeNode

		// Binary search for child in list. If it exists continue to use the existing one.
		i := sort.Search(len(cur.Children), func(i int) bool { return cur.Children[i].LocationID() >= nextId })
		if i < len(cur.Children) && cur.Children[i].LocationID() == nextId {
			// Child with this ID already exists.
			child = cur.Children[i]
		} else {
			// No child with ID exists, but it should be inserted at `i`.
			newChildren := make([]*ProfileTreeNode, len(cur.Children)+1)
			copy(newChildren, cur.Children[:i])
			child = &ProfileTreeNode{
				locationID: nextId,
			}
			newChildren[i] = child
			copy(newChildren[i+1:], cur.Children[i:])
			cur.Children = newChildren
		}

		if cur.cumulativeValues == nil {
			cur.cumulativeValues = []*ProfileTreeValueNode{{}}
		}
		cur.cumulativeValues[0].Value += sample.Value[0]

		cur = child
	}

	if cur.cumulativeValues == nil {
		cur.cumulativeValues = []*ProfileTreeValueNode{{}}
	}
	cur.cumulativeValues[0].Value += sample.Value[0]

	if cur.flatValues == nil {
		cur.flatValues = []*ProfileTreeValueNode{{}}
	}
	cur.flatValues[0].Value += sample.Value[0]
}

type MemSeries struct {
	lset labels.Labels
	id   uint64

	periodType ValueType
	sampleType ValueType

	minTime, maxTime int64
	timestamps       chunkenc.Chunk
	timestampsApp    chunkenc.Appender
	durations        chunkenc.Chunk
	durationsApp     chunkenc.Appender
	periods          chunkenc.Chunk
	periodsApp       chunkenc.Appender

	seriesTree *MemSeriesTree
	numSamples uint16
}

func NewMemSeries(lset labels.Labels, id uint64) (*MemSeries, error) {
	timestamps := chunkenc.NewDeltaChunk()
	durations := chunkenc.NewRLEChunk()
	periods := chunkenc.NewRLEChunk()

	timestampsApp, err := timestamps.Appender()
	if err != nil {
		return nil, err
	}
	durationsApp, err := durations.Appender()
	if err != nil {
		return nil, err
	}
	periodsApp, err := periods.Appender()
	if err != nil {
		return nil, err
	}

	return &MemSeries{
		lset:          lset,
		id:            id,
		timestamps:    timestamps,
		timestampsApp: timestampsApp,
		durations:     durations,
		durationsApp:  durationsApp,
		periods:       periods,
		periodsApp:    periodsApp,
		seriesTree:    &MemSeriesTree{},
	}, nil
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
}

type mapInfo struct {
	m      *profile.Mapping
	offset int64
}

func (s *MemSeries) Append(p *Profile) error {
	if s.numSamples == 0 {
		s.periodType = p.Meta.PeriodType
		s.sampleType = p.Meta.SampleType
	}

	if !equalValueType(s.periodType, p.Meta.PeriodType) {
		return ErrPeriodTypeMismatch
	}

	if !equalValueType(s.sampleType, p.Meta.SampleType) {
		return ErrSampleTypeMismatch
	}

	s.append(p.Tree)

	if s.timestamps == nil {
		s.timestamps = chunkenc.NewDeltaChunk()
		s.timestampsApp, _ = s.timestamps.Appender()
	}

	timestamp := p.Meta.Timestamp

	if timestamp <= s.maxTime {
		return ErrOutOfOrderSample
	}

	s.timestampsApp.AppendAt(s.numSamples, timestamp)

	if s.durations == nil {
		s.durations = chunkenc.NewRLEChunk()
		s.durationsApp, _ = s.durations.Appender() // TODO: Handle err
	}
	s.durationsApp.AppendAt(s.numSamples, p.Meta.Duration)

	if s.periods == nil {
		s.periods = chunkenc.NewRLEChunk()
		s.periodsApp, _ = s.periods.Appender() // TODO: Handle err
	}
	s.periodsApp.AppendAt(s.numSamples, p.Meta.Period)

	s.maxTime = timestamp

	return nil
}

func (s *MemSeries) append(profileTree *ProfileTree) {
	if s.seriesTree == nil {
		s.seriesTree = &MemSeriesTree{}
	}

	s.seriesTree.Insert(s.numSamples, profileTree)
	s.numSamples++
}

func (s *MemSeries) Labels() labels.Labels {
	return s.lset
}

type MemSeriesIteratorTree struct {
	Roots *MemSeriesIteratorTreeNode
}

type MemSeriesIteratorTreeValueNode struct {
	Values   chunkenc.Iterator
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type MemSeriesIteratorTreeNode struct {
	locationID       uint64
	flatValues       []*MemSeriesIteratorTreeValueNode
	cumulativeValues []*MemSeriesIteratorTreeValueNode
	Children         []*MemSeriesIteratorTreeNode
}

func (n *MemSeriesIteratorTreeNode) LocationID() uint64 {
	return n.locationID
}

func (n *MemSeriesIteratorTreeNode) CumulativeValue() int64 {
	res := int64(0)
	for _, v := range n.cumulativeValues {
		if v.Values != nil {
			res += v.Values.At()
		}
	}
	return res
}

func (n *MemSeriesIteratorTreeNode) CumulativeValues() []*ProfileTreeValueNode {
	res := make([]*ProfileTreeValueNode, 0, len(n.cumulativeValues))

	for _, v := range n.cumulativeValues {
		res = append(res, &ProfileTreeValueNode{
			Value:    v.Values.At(),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	return res
}

func (n *MemSeriesIteratorTreeNode) FlatValues() []*ProfileTreeValueNode {
	res := make([]*ProfileTreeValueNode, 0, len(n.flatValues))

	for _, v := range n.flatValues {
		res = append(res, &ProfileTreeValueNode{
			Value:    v.Values.At(),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	return res
}

type MemSeriesIterator struct {
	tree               *MemSeriesIteratorTree
	timestampsIterator chunkenc.Iterator
	durationsIterator  chunkenc.Iterator
	periodsIterator    chunkenc.Iterator

	series     *MemSeries
	numSamples uint16
}

func (s *MemSeries) Iterator() ProfileSeriesIterator {
	root := &MemSeriesIteratorTreeNode{}

	for _, v := range s.seriesTree.Roots.CumulativeValues {
		root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
			Values:   v.Values.Iterator(nil),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	res := &MemSeriesIterator{
		tree: &MemSeriesIteratorTree{
			Roots: root,
		},
		timestampsIterator: s.timestamps.Iterator(nil),
		durationsIterator:  s.durations.Iterator(nil),
		periodsIterator:    s.periods.Iterator(nil),
		series:             s,
		numSamples:         s.numSamples,
	}

	memItStack := MemSeriesIteratorTreeStack{{
		node:  root,
		child: 0,
	}}

	it := s.seriesTree.Iterator()

	for it.HasMore() {
		if it.NextChild() {
			child := it.At()

			n := &MemSeriesIteratorTreeNode{
				locationID:       child.LocationID,
				flatValues:       make([]*MemSeriesIteratorTreeValueNode, 0, len(child.FlatValues)),
				cumulativeValues: make([]*MemSeriesIteratorTreeValueNode, 0, len(child.CumulativeValues)),
				Children:         make([]*MemSeriesIteratorTreeNode, 0, len(child.Children)),
			}

			for _, v := range child.FlatValues {
				n.flatValues = append(n.flatValues, &MemSeriesIteratorTreeValueNode{
					Values:   v.Values.Iterator(nil),
					Label:    v.Label,
					NumLabel: v.NumLabel,
					NumUnit:  v.NumUnit,
				})
			}

			for _, v := range child.CumulativeValues {
				n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
					Values:   v.Values.Iterator(nil),
					Label:    v.Label,
					NumLabel: v.NumLabel,
					NumUnit:  v.NumUnit,
				})
			}

			cur := memItStack.Peek()
			cur.node.Children = append(cur.node.Children, n)

			memItStack.Push(&MemSeriesIteratorTreeStackEntry{
				node:  n,
				child: 0,
			})
			it.StepInto()
			continue
		}
		it.StepUp()
		memItStack.Pop()
	}

	return res
}

func (it *MemSeriesIterator) Next() bool {
	if it.numSamples == 0 {
		return false
	}

	if !it.timestampsIterator.Next() {
		return false
	}

	if !it.durationsIterator.Next() {
		return false
	}

	if !it.periodsIterator.Next() {
		return false
	}

	iit := NewMemSeriesIteratorTreeIterator(it.tree)
	for iit.HasMore() {
		if iit.NextChild() {
			child := iit.at()

			for _, v := range child.flatValues {
				v.Values.Next()
			}

			for _, v := range child.cumulativeValues {
				v.Values.Next()
			}

			iit.StepInto()
			continue
		}
		iit.StepUp()
	}

	it.numSamples--
	return true
}

type MemSeriesInstantProfile struct {
	itt *MemSeriesIteratorTree
	it  *MemSeriesIterator
}

type MemSeriesInstantProfileTree struct {
	itt *MemSeriesIteratorTree
}

func (t *MemSeriesInstantProfileTree) Iterator() InstantProfileTreeIterator {
	return NewMemSeriesIteratorTreeIterator(t.itt)
}

func (p *MemSeriesInstantProfile) ProfileTree() InstantProfileTree {
	return &MemSeriesInstantProfileTree{
		itt: p.itt,
	}
}

func (p *MemSeriesInstantProfile) ProfileMeta() InstantProfileMeta {
	return InstantProfileMeta{
		PeriodType: p.it.series.periodType,
		SampleType: p.it.series.sampleType,
		Timestamp:  p.it.timestampsIterator.At(),
		Duration:   p.it.durationsIterator.At(),
		Period:     p.it.periodsIterator.At(),
	}
}

func (it *MemSeriesIterator) At() InstantProfile {
	return &MemSeriesInstantProfile{
		itt: it.tree,
		it:  it,
	}
}

func (it *MemSeriesIterator) Err() error {
	return nil
}

type profileNormalizer struct {
	// Memoization tables within a profile.
	locationsByID map[uint64]*profile.Location
	functionsByID map[uint64]*profile.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	samples   map[stacktraceKey]*profile.Sample
	metaStore ProfileMetaStore
}

// Returns the mapped sample and whether it is new or a known sample.
func (pn *profileNormalizer) mapSample(src *profile.Sample) (*profile.Sample, bool) {
	s := &profile.Sample{
		Location: make([]*profile.Location, len(src.Location)),
		Label:    make(map[string][]string, len(src.Label)),
		NumLabel: make(map[string][]int64, len(src.NumLabel)),
		NumUnit:  make(map[string][]string, len(src.NumLabel)),
	}
	for i, l := range src.Location {
		s.Location[i] = pn.mapLocation(l)
	}
	for k, v := range src.Label {
		vv := make([]string, len(v))
		copy(vv, v)
		s.Label[k] = vv
	}
	for k, v := range src.NumLabel {
		u := src.NumUnit[k]
		vv := make([]int64, len(v))
		uu := make([]string, len(u))
		copy(vv, v)
		copy(uu, u)
		s.NumLabel[k] = vv
		s.NumUnit[k] = uu
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping. Add current values to the
	// existing sample.
	k := makeStacktraceKey(s)
	sa, found := pn.samples[k]
	if found {
		sa.Value[0] += src.Value[0]
		return sa, false
	}

	s.Value = []int64{src.Value[0]}
	pn.samples[k] = s
	return s, true
}

func (pn *profileNormalizer) mapLocation(src *profile.Location) *profile.Location {
	if src == nil {
		return nil
	}

	if l, ok := pn.locationsByID[src.ID]; ok {
		return l
	}

	mi := pn.mapMapping(src.Mapping)
	l := &profile.Location{
		Mapping:  mi.m,
		Address:  uint64(int64(src.Address) + mi.offset),
		Line:     make([]profile.Line, len(src.Line)),
		IsFolded: src.IsFolded,
	}
	for i, ln := range src.Line {
		l.Line[i] = pn.mapLine(ln)
	}
	// Check memoization table. Must be done on the remapped location to
	// account for the remapped mapping ID.
	k := MakeLocationKey(l)
	ll, err := pn.metaStore.GetLocationByKey(k)
	if err != ErrLocationNotFound {
		pn.locationsByID[src.ID] = ll
		return ll
	}
	pn.locationsByID[src.ID] = l
	pn.metaStore.CreateLocation(l)
	return l
}

func (pn *profileNormalizer) mapMapping(src *profile.Mapping) mapInfo {
	if src == nil {
		return mapInfo{}
	}

	if mi, ok := pn.mappingsByID[src.ID]; ok {
		return mi
	}

	// Check memoization tables.
	mk := MakeMappingKey(src)
	m, err := pn.metaStore.GetMappingByKey(mk)
	if err != ErrMappingNotFound {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi
	}
	m = &profile.Mapping{
		Start:           src.Start,
		Limit:           src.Limit,
		Offset:          src.Offset,
		File:            src.File,
		BuildID:         src.BuildID,
		HasFunctions:    src.HasFunctions,
		HasFilenames:    src.HasFilenames,
		HasLineNumbers:  src.HasLineNumbers,
		HasInlineFrames: src.HasInlineFrames,
	}

	// Update memoization tables.
	pn.metaStore.CreateMapping(m)
	mi := mapInfo{m, 0}
	pn.mappingsByID[src.ID] = mi
	return mi
}

func (pn *profileNormalizer) mapLine(src profile.Line) profile.Line {
	ln := profile.Line{
		Function: pn.mapFunction(src.Function),
		Line:     src.Line,
	}
	return ln
}

func (pn *profileNormalizer) mapFunction(src *profile.Function) *profile.Function {
	if src == nil {
		return nil
	}
	if f, ok := pn.functionsByID[src.ID]; ok {
		return f
	}
	k := MakeFunctionKey(src)
	f, err := pn.metaStore.GetFunctionByKey(k)
	if err != ErrFunctionNotFound {
		pn.functionsByID[src.ID] = f
		return f
	}
	f = &profile.Function{
		Name:       src.Name,
		SystemName: src.SystemName,
		Filename:   src.Filename,
		StartLine:  src.StartLine,
	}
	pn.metaStore.CreateFunction(f)
	pn.functionsByID[src.ID] = f
	return f
}

// key generates stacktraceKey to be used as a key for maps.
func makeStacktraceKey(sample *profile.Sample) stacktraceKey {
	ids := make([]string, len(sample.Location))
	for i, l := range sample.Location {
		ids[i] = strconv.FormatUint(l.ID, 16)
	}

	labels := make([]string, 0, len(sample.Label))
	for k, v := range sample.Label {
		labels = append(labels, fmt.Sprintf("%q%q", k, v))
	}
	sort.Strings(labels)

	numlabels := make([]string, 0, len(sample.NumLabel))
	for k, v := range sample.NumLabel {
		numlabels = append(numlabels, fmt.Sprintf("%q%x%x", k, v, sample.NumUnit[k]))
	}
	sort.Strings(numlabels)

	return stacktraceKey{
		strings.Join(ids, "|"),
		strings.Join(labels, ""),
		strings.Join(numlabels, ""),
	}
}

func isZeroSample(s *profile.Sample) bool {
	for _, v := range s.Value {
		if v != 0 {
			return false
		}
	}
	return true
}
