package storage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunk"
)

type MemSeriesTreeValueNode struct {
	Values   chunk.Chunk
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

func (t *MemSeriesTree) Insert(i int, profileTree *ProfileTree) {
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
								Values: chunk.NewFakeChunk(),
							}}
						}
						seriesTreeChild.FlatValues[0].Values.AppendAt(i, profileTreeChild.FlatValues()[0].Value)
					}
					if seriesTreeChild.CumulativeValues == nil {
						seriesTreeChild.CumulativeValues = []*MemSeriesTreeValueNode{{
							Values: chunk.NewFakeChunk(),
						}}
					}
					seriesTreeChild.CumulativeValues[0].Values.AppendAt(i, profileTreeChild.CumulativeValues()[0].Value)
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

			if pId == sId {
				if len(profileTreeChild.FlatValues()) > 0 {
					if seriesTreeChild.FlatValues == nil {
						seriesTreeChild.FlatValues = []*MemSeriesTreeValueNode{{
							Values: chunk.NewFakeChunk(),
						}}
					}
					seriesTreeChild.FlatValues[0].Values.AppendAt(i, profileTreeChild.FlatValues()[0].Value)
				}
				if seriesTreeChild.CumulativeValues == nil {
					seriesTreeChild.CumulativeValues = []*MemSeriesTreeValueNode{{
						Values: chunk.NewFakeChunk(),
					}}
				}
				seriesTreeChild.CumulativeValues[0].Values.AppendAt(i, profileTreeChild.CumulativeValues()[0].Value)
				pit.StepInto()
				sit.StepInto()
				continue
			}

			if pId < sId {
				// The node with the location id in the profile-tree is smaller, this means this node is not present yet in the series-tree, so it has to be added at the current child position.
				node := sit.Node()
				childIndex := sit.ChildIndex()
				newChildren := make([]*MemSeriesTreeNode, len(node.Children)+1)
				copy(newChildren, node.Children[:childIndex])
				newChild := &MemSeriesTreeNode{
					LocationID: pId,
				}
				if len(profileTreeChild.FlatValues()) > 0 {
					newChild.FlatValues = []*MemSeriesTreeValueNode{{
						Values: chunk.NewFakeChunk(),
					}}
					newChild.FlatValues[0].Values.AppendAt(i, profileTreeChild.FlatValues()[0].Value)
				}
				newChild.CumulativeValues = []*MemSeriesTreeValueNode{{
					Values: chunk.NewFakeChunk(),
				}}
				newChild.CumulativeValues[0].Values.AppendAt(i, profileTreeChild.CumulativeValues()[0].Value)
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

func (t *ProfileTree) Iterator() InstantProfileTreeIterator {
	return NewProfileTreeIterator(t)
}

func (t *ProfileTree) Insert(sample *profile.Sample) {
	if t.Roots == nil {
		t.Roots = &ProfileTreeNode{}
	}

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
	p *profile.Profile

	// Memoization tables for profile entities.
	stacktraceIDs map[[16]byte]*Stacktrace
	stacktraces   map[stacktraceKey]*Stacktrace
	locations     map[locationKey]*profile.Location
	functions     map[functionKey]*profile.Function
	mappings      map[mappingKey]*profile.Mapping

	sampleNumber uint64

	timestamps chunk.Chunk
	durations  chunk.Chunk
	periods    chunk.Chunk

	seriesTree *MemSeriesTree
	i          int
}

func NewMemSeries() *MemSeries {
	return &MemSeries{
		timestamps: chunk.NewFakeChunk(),
		durations:  chunk.NewFakeChunk(),
		periods:    chunk.NewFakeChunk(),
		seriesTree: &MemSeriesTree{},
	}
}

type Stacktrace struct {
	ID [16]byte

	Location []*profile.Location
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type stacktraceKey struct {
	locations string
	labels    string
	numlabels string
}

type locationKey struct {
	addr, mappingID uint64
	lines           string
	isFolded        bool
}

type functionKey struct {
	startLine                  int64
	name, systemName, fileName string
}

type mapInfo struct {
	m      *profile.Mapping
	offset int64
}

type mappingKey struct {
	size, offset  uint64
	buildIDOrFile string
}

func (s *MemSeries) Append(value *profile.Profile) error {
	profileTree, err := s.prepareSamplesForInsert(value)
	if err != nil {
		return err
	}

	s.append(profileTree)

	return nil
}

func (s *MemSeries) append(profileTree *ProfileTree) {
	if s.seriesTree == nil {
		s.seriesTree = &MemSeriesTree{}
	}

	s.seriesTree.Insert(s.i, profileTree)
	s.i++
}

func (s *MemSeries) prepareSamplesForInsert(value *profile.Profile) (*ProfileTree, error) {
	if s.p == nil {
		s.p = &profile.Profile{
			PeriodType: value.PeriodType,
			SampleType: value.SampleType[:1],
		}
		s.locations = make(map[locationKey]*profile.Location, len(value.Location))
		s.functions = make(map[functionKey]*profile.Function, len(value.Function))
		s.mappings = make(map[mappingKey]*profile.Mapping, len(value.Mapping))
	}

	if err := compatibleProfiles(s.p, value); err != nil {
		return nil, err
	}

	pn := &profileNormalizer{
		p: s.p,

		locations: s.locations,
		functions: s.functions,
		mappings:  s.mappings,

		samples: make(map[stacktraceKey]*profile.Sample, len(value.Sample)),

		// Profile-specific hash tables for each profile inserted.
		locationsByID: make(map[uint64]*profile.Location, len(value.Location)),
		functionsByID: make(map[uint64]*profile.Function, len(value.Function)),
		mappingsByID:  make(map[uint64]mapInfo, len(value.Mapping)),
	}

	if len(pn.mappings) == 0 && len(value.Mapping) > 0 {
		// The Mapping list has the property that the first mapping
		// represents the main binary. Take the first Mapping we see,
		// otherwise the operations below will add mappings in an
		// arbitrary order.
		pn.mapMapping(value.Mapping[0])
	}

	samples := make([]*profile.Sample, 0, len(value.Sample))
	for _, s := range value.Sample {
		if !isZeroSample(s) {
			sa, isNew := pn.mapSample(s)
			if isNew {
				samples = append(samples, sa)
			}
		}
	}
	sortSamples(samples)

	profileTree := &ProfileTree{}
	for _, s := range samples {
		profileTree.Insert(s)
	}

	return profileTree, nil
}

type MemSeriesIteratorTree struct {
	Roots *MemSeriesIteratorTreeNode
}

type MemSeriesIteratorTreeValueNode struct {
	Values   chunk.ChunkIterator
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
		res += v.Values.At()
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
	tree *MemSeriesIteratorTree
}

func (s *MemSeries) Iterator() *MemSeriesIterator {
	root := &MemSeriesIteratorTreeNode{}

	for _, v := range s.seriesTree.Roots.CumulativeValues {
		root.cumulativeValues = append(root.cumulativeValues, &MemSeriesIteratorTreeValueNode{
			Values:   v.Values.Iterator(),
			Label:    v.Label,
			NumLabel: v.NumLabel,
			NumUnit:  v.NumUnit,
		})
	}

	res := &MemSeriesIterator{
		tree: &MemSeriesIteratorTree{
			Roots: root,
		},
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
					Values:   v.Values.Iterator(),
					Label:    v.Label,
					NumLabel: v.NumLabel,
					NumUnit:  v.NumUnit,
				})
			}

			for _, v := range child.CumulativeValues {
				n.cumulativeValues = append(n.cumulativeValues, &MemSeriesIteratorTreeValueNode{
					Values:   v.Values.Iterator(),
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
	iit := NewMemSeriesIteratorTreeIterator(it.tree)

	for iit.HasMore() {
		if iit.NextChild() {
			child := iit.At().(*MemSeriesIteratorTreeNode)

			for _, v := range child.flatValues {
				if !v.Values.Next() {
					return false
				}
			}

			for _, v := range child.cumulativeValues {
				if !v.Values.Next() {
					return false
				}
			}

			iit.StepInto()
			continue
		}
		iit.StepUp()
	}

	return true
}

type MemSeriesInstantProfile struct {
	itt *MemSeriesIteratorTree
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
	return InstantProfileMeta{}
}

func (it *MemSeriesIterator) At() InstantProfile {
	return &MemSeriesInstantProfile{
		itt: it.tree,
	}
}

func (it *MemSeriesIterator) Err() error {
	return nil
}

//func (s *MemSeries) Iterator() *SeriesIterator {
//	return &SeriesIterator{
//		series: s,
//		data:   s.chunk.Data(),
//		i:      0,
//	}
//}
//
//type SeriesIterator struct {
//	series *Series
//	data   chunk.DecodedData
//	i      int
//	cur    *profile.Profile
//	err    error
//}
//
//func (it *SeriesIterator) Next() bool {
//	if it.i >= len(it.data.Timestamps) {
//		return false
//	}
//
//	p := &profile.Profile{
//		PeriodType:    it.series.p.PeriodType,
//		SampleType:    it.series.p.SampleType,
//		TimeNanos:     it.data.Timestamps[it.i],
//		DurationNanos: it.data.Durations[it.i],
//		Period:        it.data.Periods[it.i],
//		Location:      it.series.p.Location,
//		Function:      it.series.p.Function,
//		Mapping:       it.series.p.Mapping,
//	}
//
//	for _, stacktrace := range it.data.Stacktraces {
//		if stacktrace.Values[it.i] != 0 {
//			st := it.series.stacktraceIDs[stacktrace.StacktraceID]
//			p.Sample = append(p.Sample, &profile.Sample{
//				Location: st.Location,
//				Label:    st.Label,
//				NumLabel: st.NumLabel,
//				NumUnit:  st.NumUnit,
//				Value:    []int64{stacktrace.Values[it.i]},
//			})
//		}
//	}
//
//	it.cur = p
//	it.i++
//
//	return true
//}
//
//func (it *SeriesIterator) At() *profile.Profile {
//	return it.cur
//}
//
//func (it *SeriesIterator) Err() error {
//	return it.err
//}

type profileNormalizer struct {
	p *profile.Profile

	// Memoization tables within a profile.
	locationsByID map[uint64]*profile.Location
	functionsByID map[uint64]*profile.Function
	mappingsByID  map[uint64]mapInfo

	// Memoization tables for profile entities.
	samples   map[stacktraceKey]*profile.Sample
	locations map[locationKey]*profile.Location
	functions map[functionKey]*profile.Function
	mappings  map[mappingKey]*profile.Mapping

	// A slice of samples for each unique stack trace.
	c *chunk.Chunk
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
		ID:       uint64(len(pn.p.Location) + 1),
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
	k := makeLocationKey(l)
	if ll, ok := pn.locations[k]; ok {
		pn.locationsByID[src.ID] = ll
		return ll
	}
	pn.locationsByID[src.ID] = l
	pn.locations[k] = l
	pn.p.Location = append(pn.p.Location, l)
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
	mk := makeMappingKey(src)
	if m, ok := pn.mappings[mk]; ok {
		mi := mapInfo{m, int64(m.Start) - int64(src.Start)}
		pn.mappingsByID[src.ID] = mi
		return mi
	}
	m := &profile.Mapping{
		ID:              uint64(len(pn.p.Mapping) + 1),
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
	pn.p.Mapping = append(pn.p.Mapping, m)

	// Update memoization tables.
	pn.mappings[mk] = m
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
	k := makeFunctionKey(src)
	if f, ok := pn.functions[k]; ok {
		pn.functionsByID[src.ID] = f
		return f
	}
	f := &profile.Function{
		ID:         uint64(len(pn.p.Function) + 1),
		Name:       src.Name,
		SystemName: src.SystemName,
		Filename:   src.Filename,
		StartLine:  src.StartLine,
	}
	pn.functions[k] = f
	pn.functionsByID[src.ID] = f
	pn.p.Function = append(pn.p.Function, f)
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

func makeLocationKey(l *profile.Location) locationKey {
	key := locationKey{
		addr:     l.Address,
		isFolded: l.IsFolded,
	}
	if l.Mapping != nil {
		// Normalizes address to handle address space randomization.
		key.addr -= l.Mapping.Start
		key.mappingID = l.Mapping.ID
	}
	lines := make([]string, len(l.Line)*2)
	for i, line := range l.Line {
		if line.Function != nil {
			lines[i*2] = strconv.FormatUint(line.Function.ID, 16)
		}
		lines[i*2+1] = strconv.FormatInt(line.Line, 16)
	}
	key.lines = strings.Join(lines, "|")
	return key
}

func makeFunctionKey(f *profile.Function) functionKey {
	return functionKey{
		f.StartLine,
		f.Name,
		f.SystemName,
		f.Filename,
	}
}

func makeMappingKey(m *profile.Mapping) mappingKey {
	// Normalize addresses to handle address space randomization.
	// Round up to next 4K boundary to avoid minor discrepancies.
	const mapsizeRounding = 0x1000

	size := m.Limit - m.Start
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)
	key := mappingKey{
		size:   size,
		offset: m.Offset,
	}

	switch {
	case m.BuildID != "":
		key.buildIDOrFile = m.BuildID
	case m.File != "":
		key.buildIDOrFile = m.File
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}
	return key
}

// compatible determines if two profiles can be compared/merged.
// returns nil if the profiles are compatible; otherwise an error with
// details on the incompatibility.
func compatibleProfiles(p *profile.Profile, pb *profile.Profile) error {
	if !equalValueType(p.PeriodType, pb.PeriodType) {
		return fmt.Errorf("incompatible period types %v and %v", p.PeriodType, pb.PeriodType)
	}

	if !equalValueType(p.SampleType[0], pb.SampleType[0]) {
		return fmt.Errorf("incompatible sample types %v and %v", p.SampleType, pb.SampleType)
	}
	return nil
}

// equalValueType returns true if the two value types are semantically
// equal. It ignores the internal fields used during encode/decode.
func equalValueType(st1, st2 *profile.ValueType) bool {
	return st1.Type == st2.Type && st1.Unit == st2.Unit
}

func isZeroSample(s *profile.Sample) bool {
	for _, v := range s.Value {
		if v != 0 {
			return false
		}
	}
	return true
}
