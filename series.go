package storage

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/parca-dev/storage/chunk"
)

type SeriesTreeValueNode struct {
	Values   chunk.Chunk
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type SeriesTreeNode struct {
	LocationID       uint64
	FlatValues       []*SeriesTreeValueNode
	CumulativeValues []*SeriesTreeValueNode
	Children         []*SeriesTreeNode
}

type SeriesTree struct {
	Roots *SeriesTreeNode
}

func (t *SeriesTree) Insert(i int, profileTree *ProfileTree) {
	if t.Roots == nil {
		t.Roots = &SeriesTreeNode{}
	}

	t.insert(i, t.Roots, profileTree.Roots)
}

type ProfileTreeStackEntry struct {
	node  *ProfileTreeNode
	child int
}

type SeriesTreeStackEntry struct {
	node  *SeriesTreeNode
	child int
}

func (t *SeriesTree) insert(i int, seriesRoot *SeriesTreeNode, profileRoot *ProfileTreeNode) {
	// Put a fake node round the roots so the cumulative values of the roots are also appended.
	seriesTreeStack := []SeriesTreeStackEntry{{
		node:  &SeriesTreeNode{Children: []*SeriesTreeNode{seriesRoot}},
		child: 0,
	}}
	profileTreeStack := []ProfileTreeStackEntry{{
		node:  &ProfileTreeNode{Children: []*ProfileTreeNode{profileRoot}},
		child: 0,
	}}

	curSeriesStackItem := seriesTreeStack[0]
	curProfileStackItem := profileTreeStack[0]

	for len(profileTreeStack) > 0 {
		if (len(curSeriesStackItem.node.Children) <= curSeriesStackItem.child &&
			len(curProfileStackItem.node.Children) <= curProfileStackItem.child) ||
			len(curProfileStackItem.node.Children) <= curProfileStackItem.child {

			// We're at the end of the children array of both nodes, so we go
			// one node up the stack to see what else needs to be done there.

			curSeriesStackItem = seriesTreeStack[len(seriesTreeStack)-1]
			curSeriesStackItem.child++
			seriesTreeStack = seriesTreeStack[:len(seriesTreeStack)-1]

			curProfileStackItem = profileTreeStack[len(profileTreeStack)-1]
			curProfileStackItem.child++
			profileTreeStack = profileTreeStack[:len(profileTreeStack)-1]
			continue
		}

		if len(curSeriesStackItem.node.Children) == curSeriesStackItem.child &&
			len(curProfileStackItem.node.Children) > curProfileStackItem.child {

			// This means the series-tree-node is at the end of its known children, but the profile-tree still has children so we append the next one.

			profileNodeChild := curProfileStackItem.node.Children[curProfileStackItem.child]
			newSeriesNode := &SeriesTreeNode{
				LocationID: profileNodeChild.LocationID,
			}
			if len(profileNodeChild.FlatValues) > 0 {
				newSeriesNode.FlatValues = []*SeriesTreeValueNode{{
					Values: chunk.NewFakeChunk(),
				}}
				newSeriesNode.FlatValues[0].Values.AppendAt(
					i,
					profileNodeChild.FlatValues[0].Value,
				)
			}

			newSeriesNode.CumulativeValues = []*SeriesTreeValueNode{{
				Values: chunk.NewFakeChunk(),
			}}
			newSeriesNode.CumulativeValues[0].Values.AppendAt(
				i,
				profileNodeChild.CumulativeValues[0].Value,
			)
			curSeriesStackItem.node.Children = append(curSeriesStackItem.node.Children, newSeriesNode)

			curSeriesStackItem = SeriesTreeStackEntry{
				node:  newSeriesNode,
				child: 0,
			}
			seriesTreeStack = append(seriesTreeStack, curSeriesStackItem)
			curProfileStackItem = ProfileTreeStackEntry{
				node:  profileNodeChild,
				child: 0,
			}
			profileTreeStack = append(profileTreeStack, curProfileStackItem)
			continue
		}

		loc1 := curSeriesStackItem.node.Children[curSeriesStackItem.child].LocationID
		loc2 := curProfileStackItem.node.Children[curProfileStackItem.child].LocationID

		if loc1 < loc2 {
			// Nothing to insert as current location is smaller than the one to insert. And larger ones may still come.
			curSeriesStackItem.child++
			continue
		}

		if loc1 > loc2 {
			// The node with the location id in the profile-tree is smaller, this means this node is not present yet in the series-tree, so it has to be added at the current child position.
			newChildren := make([]*SeriesTreeNode, len(curSeriesStackItem.node.Children)+1)
			copy(newChildren, curSeriesStackItem.node.Children[:curSeriesStackItem.child])
			child := &SeriesTreeNode{
				LocationID: curProfileStackItem.node.Children[curProfileStackItem.child].LocationID,
			}
			newChildren[curSeriesStackItem.child] = child
			copy(newChildren[curSeriesStackItem.child+1:], curSeriesStackItem.node.Children[curSeriesStackItem.child:])
			curSeriesStackItem.node.Children = newChildren
			continue
		}

		if loc1 == loc2 {
			// Locations are identical this means we need to add values and go one step deeper into the trees.

			if len(curProfileStackItem.node.Children[curProfileStackItem.child].FlatValues) > 0 {
				// It's possible that a node has no flat value, and only a cumulative value.
				if len(curSeriesStackItem.node.Children[curSeriesStackItem.child].FlatValues) == 0 {
					curSeriesStackItem.node.Children[curSeriesStackItem.child].FlatValues = []*SeriesTreeValueNode{{
						Values: chunk.NewFakeChunk(),
					}}
				}
				curSeriesStackItem.node.Children[curSeriesStackItem.child].FlatValues[0].Values.AppendAt(
					i,
					curProfileStackItem.node.Children[curProfileStackItem.child].FlatValues[0].Value,
				)
			}

			if len(curSeriesStackItem.node.Children[curSeriesStackItem.child].CumulativeValues) == 0 {
				curSeriesStackItem.node.Children[curSeriesStackItem.child].CumulativeValues = []*SeriesTreeValueNode{{
					Values: chunk.NewFakeChunk(),
				}}
			}
			curSeriesStackItem.node.Children[curSeriesStackItem.child].CumulativeValues[0].Values.AppendAt(
				i,
				curProfileStackItem.node.Children[curProfileStackItem.child].CumulativeValues[0].Value,
			)

			// Node from profile-tree was already present in the series-tree.
			// So we go one step further down.

			curSeriesStackItem = SeriesTreeStackEntry{
				node:  curSeriesStackItem.node.Children[curSeriesStackItem.child],
				child: 0,
			}
			seriesTreeStack = append(seriesTreeStack, curSeriesStackItem)

			curProfileStackItem = ProfileTreeStackEntry{
				node:  curProfileStackItem.node.Children[curProfileStackItem.child],
				child: 0,
			}
			profileTreeStack = append(profileTreeStack, curProfileStackItem)
		}
	}
}

type ProfileTreeValueNode struct {
	Value    int64
	Label    map[string][]string
	NumLabel map[string][]int64
	NumUnit  map[string][]string
}

type ProfileTreeNode struct {
	LocationID       uint64
	FlatValues       []*ProfileTreeValueNode
	CumulativeValues []*ProfileTreeValueNode
	Children         []*ProfileTreeNode
}

func (n *ProfileTreeNode) ChildWithID(id uint64) *ProfileTreeNode {
	for _, child := range n.Children {
		if child.LocationID == id {
			return child
		}
	}
	return nil
}

type ProfileTree struct {
	Roots *ProfileTreeNode
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
		i := sort.Search(len(cur.Children), func(i int) bool { return cur.Children[i].LocationID >= nextId })
		if i < len(cur.Children) && cur.Children[i].LocationID == nextId {
			// Child with this ID already exists.
			child = cur.Children[i]
		} else {
			// No child with ID exists, but it should be inserted at `i`.
			newChildren := make([]*ProfileTreeNode, len(cur.Children)+1)
			copy(newChildren, cur.Children[:i])
			child = &ProfileTreeNode{
				LocationID: nextId,
			}
			newChildren[i] = child
			copy(newChildren[i+1:], cur.Children[i:])
			cur.Children = newChildren
		}

		if cur.CumulativeValues == nil {
			cur.CumulativeValues = []*ProfileTreeValueNode{{}}
		}
		cur.CumulativeValues[0].Value += sample.Value[0]

		cur = child
	}

	if cur.CumulativeValues == nil {
		cur.CumulativeValues = []*ProfileTreeValueNode{{}}
	}
	cur.CumulativeValues[0].Value += sample.Value[0]

	if cur.FlatValues == nil {
		cur.FlatValues = []*ProfileTreeValueNode{{}}
	}
	cur.FlatValues[0].Value += sample.Value[0]
}

type Series struct {
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

	seriesTree *SeriesTree
	i          int
}

func NewSeries() *Series {
	return &Series{
		timestamps: chunk.NewFakeChunk(),
		durations:  chunk.NewFakeChunk(),
		periods:    chunk.NewFakeChunk(),
		seriesTree: &SeriesTree{},
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

func (s *Series) Append(value *profile.Profile) error {
	profileTree, err := s.prepareSamplesForInsert(value)
	if err != nil {
		return err
	}

	if s.seriesTree == nil {
		s.seriesTree = &SeriesTree{}
	}

	s.seriesTree.Insert(s.i, profileTree)
	s.i++

	return nil
}

func (s *Series) prepareSamplesForInsert(value *profile.Profile) (*ProfileTree, error) {
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

//func (s *Series) Iterator() *SeriesIterator {
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
