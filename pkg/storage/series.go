package storage

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
)

var (
	ErrOutOfOrderSample = errors.New("out of order sample")
)

type MemSeriesTreeNode struct {
	keys []ProfileTreeValueNodeKey

	LocationID uint64
	Children   []*MemSeriesTreeNode
}

func (n *MemSeriesTreeNode) addKey(key ProfileTreeValueNodeKey) {
	for _, k := range n.keys {
		if k.Equals(key) {
			return
		}
	}
	n.keys = append(n.keys, key)
}

type MemSeriesTree struct {
	s     *MemSeries
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

					for _, n := range profileTreeChild.FlatValues() {
						if n.key == nil {
							n.Key(profileTreeChild.LocationID())
						}

						t.s.mu.Lock()
						if t.s.flatValues[*n.key] == nil {
							t.s.flatValues[*n.key] = chunkenc.NewXORChunk()
						}

						app, err := t.s.flatValues[*n.key].Appender()
						if err != nil {
							t.s.mu.Unlock()
							return err
						}
						app.AppendAt(index, n.Value)
						t.s.mu.Unlock()

						// We need to keep track of the node keys.
						seriesTreeChild.addKey(*n.key)

						if len(n.Label) > 0 {
							t.s.mu.Lock()
							if t.s.labels[*n.key] == nil {
								t.s.labels[*n.key] = n.Label
							}

							if t.s.numLabels[*n.key] == nil {
								t.s.numLabels[*n.key] = n.NumLabel
							}

							if t.s.numUnits[*n.key] == nil {
								t.s.numUnits[*n.key] = n.NumUnit
							}
							t.s.mu.Unlock()
						}
					}

					for _, n := range profileTreeChild.CumulativeValues() {
						if n.key == nil {
							n.Key(profileTreeChild.LocationID())
						}

						t.s.mu.Lock()
						if t.s.cumulativeValues[*n.key] == nil {
							t.s.cumulativeValues[*n.key] = chunkenc.NewXORChunk()
						}
						app, err := t.s.cumulativeValues[*n.key].Appender()
						if err != nil {
							t.s.mu.Unlock()
							return err
						}
						app.AppendAt(index, n.Value)
						t.s.mu.Unlock()

						// We need to keep track of the node keys.
						seriesTreeChild.addKey(*n.key)
					}

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

			// The node with the location id in the profile-tree is the same (except Location ID 0 - the root),
			// this means this node present in the series-tree, so we need add the new values to the existing node.
			if pId == sId {
				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					// Even if the location exists.
					// labels can be different and then the key is different, so we need check.
					t.s.mu.Lock()
					if t.s.flatValues[*n.key] == nil {
						t.s.flatValues[*n.key] = chunkenc.NewXORChunk()
					}
					app, err := t.s.flatValues[*n.key].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node IDs.
					seriesTreeChild.addKey(*n.key)
				}

				for _, n := range profileTreeChild.CumulativeValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if t.s.cumulativeValues[*n.key] == nil {
						t.s.cumulativeValues[*n.key] = chunkenc.NewXORChunk()
					}
					app, err := t.s.cumulativeValues[*n.key].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open cumulative appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					seriesTreeChild.addKey(*n.key)
				}

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

				for _, n := range profileTreeChild.FlatValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if t.s.flatValues[*n.key] == nil {
						t.s.flatValues[*n.key] = chunkenc.NewXORChunk()
					}
					app, err := t.s.flatValues[*n.key].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open flat appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					newChild.addKey(*n.key)
				}

				for _, n := range profileTreeChild.CumulativeValues() {
					if n.key == nil {
						n.Key(profileTreeChild.LocationID())
					}

					t.s.mu.Lock()
					if t.s.cumulativeValues[*n.key] == nil {
						t.s.cumulativeValues[*n.key] = chunkenc.NewXORChunk()
					}
					app, err := t.s.cumulativeValues[*n.key].Appender()
					if err != nil {
						t.s.mu.Unlock()
						return fmt.Errorf("failed to open cumulative appender: %w", err)
					}
					app.AppendAt(index, n.Value)
					t.s.mu.Unlock()

					// We need to keep track of the node keys.
					newChild.addKey(*n.key)
				}

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
	locationID           uint64
	flatValues           []*ProfileTreeValueNode
	flatDiffValues       []*ProfileTreeValueNode
	cumulativeValues     []*ProfileTreeValueNode
	cumulativeDiffValues []*ProfileTreeValueNode
	Children             []*ProfileTreeNode
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

func (n *ProfileTreeNode) CumulativeDiffValue() int64 {
	res := int64(0)
	for _, cv := range n.cumulativeDiffValues {
		res += cv.Value
	}

	return res
}

func (n *ProfileTreeNode) CumulativeDiffValues() []*ProfileTreeValueNode {
	return n.cumulativeDiffValues
}

func (n *ProfileTreeNode) FlatDiffValues() []*ProfileTreeValueNode {
	return n.flatDiffValues
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

	locationIDs := make([]uint64, 0, len(sample.Location)+1)
	for _, l := range sample.Location {
		locationIDs = append(locationIDs, l.ID)
	}
	locationIDs = append(locationIDs, 0) // add the root

	for i := len(locations) - 1; i >= 0; i-- {
		nextId := locations[i].ID

		var child *ProfileTreeNode

		// Binary search for child in list. If it exists continue to use the existing one.
		index := sort.Search(len(cur.Children), func(i int) bool { return cur.Children[i].LocationID() >= nextId })
		if index < len(cur.Children) && cur.Children[index].LocationID() == nextId {
			// Child with this ID already exists.
			child = cur.Children[index]
		} else {
			// No child with ID exists, but it should be inserted at `index`.
			newChildren := make([]*ProfileTreeNode, len(cur.Children)+1)
			copy(newChildren, cur.Children[:index])
			child = &ProfileTreeNode{
				locationID: nextId,
			}
			newChildren[index] = child
			copy(newChildren[index+1:], cur.Children[index:])
			cur.Children = newChildren
		}

		// Nodes that might only have cumulativeValues
		if cur.cumulativeValues == nil {
			cur.cumulativeValues = []*ProfileTreeValueNode{{}}
		}
		cur.cumulativeValues[0].Value += sample.Value[0]

		for _, cv := range cur.cumulativeValues {
			// Populate the keys with the current subset of locations.
			// i+1 because we additionally have the root in locationIDs.
			cv.Key(locationIDs[i+1:]...)
		}

		cur = child
	}

	if cur.cumulativeValues == nil {
		cur.cumulativeValues = []*ProfileTreeValueNode{{}}
	}

	cur.cumulativeValues[0].Value += sample.Value[0]
	// TODO: We probably need to merge labels, numLabels and numUnits
	cur.cumulativeValues[0].Label = sample.Label
	cur.cumulativeValues[0].NumLabel = sample.NumLabel
	cur.cumulativeValues[0].NumUnit = sample.NumUnit

	for _, cv := range cur.cumulativeValues {
		cv.Key(locationIDs...) // populate the keys
	}

	if cur.flatValues == nil {
		cur.flatValues = []*ProfileTreeValueNode{{}}
	}
	cur.flatValues[0].Value += sample.Value[0]
	// TODO: We probably need to merge labels, numLabels and numUnits
	cur.flatValues[0].Label = sample.Label
	cur.flatValues[0].NumLabel = sample.NumLabel
	cur.flatValues[0].NumUnit = sample.NumUnit

	for _, fv := range cur.flatValues {
		fv.Key(locationIDs...) //populate the keys
	}
}

type MemSeries struct {
	lset labels.Labels
	id   uint64

	periodType ValueType
	sampleType ValueType

	minTime, maxTime int64
	timestamps       chunkenc.Chunk
	durations        chunkenc.Chunk
	periods          chunkenc.Chunk

	// TODO: Might be worth combining behind some struct?
	// Or maybe not because it's easier to serialize?

	// mu locks the following maps for concurrent access.
	mu sync.RWMutex
	// Flat and cumulative values as well as labels by the node's ProfileTreeValueNodeKey.
	flatValues       map[ProfileTreeValueNodeKey]chunkenc.Chunk
	cumulativeValues map[ProfileTreeValueNodeKey]chunkenc.Chunk
	labels           map[ProfileTreeValueNodeKey]map[string][]string
	numLabels        map[ProfileTreeValueNodeKey]map[string][]int64
	numUnits         map[ProfileTreeValueNodeKey]map[string][]string

	seriesTree *MemSeriesTree
	numSamples uint16

	samplesAppended prometheus.Counter
}

func NewMemSeries(lset labels.Labels, id uint64) *MemSeries {
	s := &MemSeries{
		lset:       lset,
		id:         id,
		timestamps: chunkenc.NewDeltaChunk(),
		durations:  chunkenc.NewRLEChunk(),
		periods:    chunkenc.NewRLEChunk(),

		flatValues:       make(map[ProfileTreeValueNodeKey]chunkenc.Chunk),
		cumulativeValues: make(map[ProfileTreeValueNodeKey]chunkenc.Chunk),
		labels:           make(map[ProfileTreeValueNodeKey]map[string][]string),
		numLabels:        make(map[ProfileTreeValueNodeKey]map[string][]int64),
		numUnits:         make(map[ProfileTreeValueNodeKey]map[string][]string),
	}
	s.seriesTree = &MemSeriesTree{s: s}

	return s
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

func (s *MemSeries) Appender() (*MemSeriesAppender, error) {
	timestamps, err := s.timestamps.Appender()
	if err != nil {
		return nil, err
	}
	durations, err := s.durations.Appender()
	if err != nil {
		return nil, err
	}
	periods, err := s.periods.Appender()
	if err != nil {
		return nil, err
	}

	return &MemSeriesAppender{
		s:          s,
		timestamps: timestamps,
		duration:   durations,
		periods:    periods,
	}, nil
}

type MemSeriesAppender struct {
	s          *MemSeries
	timestamps chunkenc.Appender
	duration   chunkenc.Appender
	periods    chunkenc.Appender
}

func (a *MemSeriesAppender) Append(p *Profile) error {
	if a.s.numSamples == 0 {
		a.s.periodType = p.Meta.PeriodType
		a.s.sampleType = p.Meta.SampleType
	}

	if !equalValueType(a.s.periodType, p.Meta.PeriodType) {
		return ErrPeriodTypeMismatch
	}

	if !equalValueType(a.s.sampleType, p.Meta.SampleType) {
		return ErrSampleTypeMismatch
	}

	if err := a.s.appendTree(p.Tree); err != nil {
		return err
	}

	timestamp := p.Meta.Timestamp

	if timestamp <= a.s.maxTime {
		return ErrOutOfOrderSample
	}

	a.s.mu.Lock()
	a.timestamps.AppendAt(a.s.numSamples, timestamp)
	a.duration.AppendAt(a.s.numSamples, p.Meta.Duration)
	a.periods.AppendAt(a.s.numSamples, p.Meta.Period)

	// Set the timestamp as minTime if timestamp != 0
	if a.s.minTime == 0 && timestamp != 0 {
		a.s.minTime = timestamp
	}

	a.s.maxTime = timestamp
	a.s.numSamples++
	a.s.mu.Unlock()

	a.s.samplesAppended.Inc()
	return nil
}

func (s *MemSeries) appendTree(profileTree *ProfileTree) error {
	if s.seriesTree == nil {
		s.seriesTree = &MemSeriesTree{s: s}
	}

	return s.seriesTree.Insert(s.numSamples, profileTree)
}

func (s *MemSeries) Labels() labels.Labels {
	return s.lset
}

type MemSeriesStats struct {
	samples     uint16
	Cumulatives []MemSeriesValueStats
	Flat        []MemSeriesValueStats
}

type MemSeriesValueStats struct {
	samples int
	bytes   int
}

func (s *MemSeries) stats() MemSeriesStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	flat := make([]MemSeriesValueStats, 0, len(s.flatValues))
	cumulative := make([]MemSeriesValueStats, 0, len(s.cumulativeValues))

	for _, c := range s.flatValues {
		flat = append(flat, MemSeriesValueStats{
			samples: c.NumSamples(),
			bytes:   len(c.Bytes()),
		})
	}

	for _, c := range s.cumulativeValues {
		cumulative = append(cumulative, MemSeriesValueStats{
			samples: c.NumSamples(),
			bytes:   len(c.Bytes()),
		})
	}

	return MemSeriesStats{
		samples:     s.numSamples,
		Cumulatives: cumulative,
		Flat:        flat,
	}
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

func (n *MemSeriesIteratorTreeNode) CumulativeDiffValue() int64                    { return 0 }
func (n *MemSeriesIteratorTreeNode) CumulativeDiffValues() []*ProfileTreeValueNode { return nil }

func (n *MemSeriesIteratorTreeNode) CumulativeValues() []*ProfileTreeValueNode {
	if len(n.cumulativeValues) == 0 { // For consistency with other iterators
		return nil
	}

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

func (n *MemSeriesIteratorTreeNode) FlatDiffValues() []*ProfileTreeValueNode { return nil }

func (n *MemSeriesIteratorTreeNode) FlatValues() []*ProfileTreeValueNode {
	if len(n.flatValues) == 0 { // For consistency with other iterators
		return nil
	}

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
	s.mu.RLock()
	defer s.mu.RUnlock()

	root := &MemSeriesIteratorTreeNode{}

	// TODO: this might be still wrong in case there are multiple roots with different labels?
	// We might be never reading roots with labels...
	rootKey := ProfileTreeValueNodeKey{location: "0"}
	root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
		Values:   s.cumulativeValues[rootKey].Iterator(nil),
		Label:    s.labels[rootKey],
		NumLabel: s.numLabels[rootKey],
		NumUnit:  s.numUnits[rootKey],
	})

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
				locationID: child.LocationID,
				Children:   make([]*MemSeriesIteratorTreeNode, 0, len(child.Children)),
			}

			for _, key := range child.keys {
				if chunk, ok := s.flatValues[key]; ok {
					n.flatValues = append(n.flatValues, &MemSeriesIteratorTreeValueNode{
						Values:   chunk.Iterator(nil),
						Label:    s.labels[key],
						NumLabel: s.numLabels[key],
						NumUnit:  s.numUnits[key],
					})
				}
				if chunk, ok := s.cumulativeValues[key]; ok {
					n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
						Values:   chunk.Iterator(nil),
						Label:    s.labels[key],
						NumLabel: s.numLabels[key],
						NumUnit:  s.numUnits[key],
					})
				}
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

	it.series.mu.RLock()
	defer it.series.mu.RUnlock()

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
func (pn *profileNormalizer) mapSample(src *profile.Sample, sampleIndex int) (*profile.Sample, bool) {
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
		sa.Value[0] += src.Value[sampleIndex]
		return sa, false
	}

	s.Value = []int64{src.Value[sampleIndex]}
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
		locations: strings.Join(ids, "|"),
		labels:    strings.Join(labels, ""),
		numlabels: strings.Join(numlabels, ""),
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
