// Copyright 2022-2025 The Parca Authors
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

package query

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	parcaprofile "github.com/parca-dev/parca/pkg/profile"
)

func SerializePprof(p *pprofpb.Profile) ([]byte, error) {
	data, err := p.MarshalVT()
	if err != nil {
		return nil, fmt.Errorf("marshal profile: %w", err)
	}

	gzipped, err := Gzip(data)
	if err != nil {
		return nil, fmt.Errorf("gzip profile: %w", err)
	}

	return gzipped, nil
}

func Gzip(data []byte) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	w := gzip.NewWriter(buf)
	if _, err := w.Write(data); err != nil {
		return nil, fmt.Errorf("write data to gzip writer: %w", err)
	}

	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("close gzip writer: %w", err)
	}

	return buf.Bytes(), nil
}

func GenerateFlatPprof(
	ctx context.Context,
	isDiff bool,
	p parcaprofile.Profile,
) (*pprofpb.Profile, error) {
	w := NewPprofWriter(p.Meta, isDiff)

	for _, rec := range p.Samples {
		w.WriteRecord(rec)
	}

	return w.res, nil
}

type PprofWriter struct {
	isDiff bool

	res              *pprofpb.Profile
	mappingByKey     map[mappingKey]uint64
	functionByKey    map[functionKey]uint64
	locationByKey    map[string]uint64
	sampleByKey      map[string]int
	stringTableIndex map[string]int64

	buf []byte
}

func (w *PprofWriter) string(s string) int64 {
	if idx, ok := w.stringTableIndex[s]; ok {
		return idx
	}

	idx := int64(len(w.res.StringTable))
	w.res.StringTable = append(w.res.StringTable, s)
	w.stringTableIndex[s] = idx
	return idx
}

func (w *PprofWriter) byteString(s []byte) int64 {
	if idx, ok := w.stringTableIndex[unsafeString(s)]; ok {
		return idx
	}

	idx := int64(len(w.res.StringTable))
	str := string(s)
	w.res.StringTable = append(w.res.StringTable, str)
	w.stringTableIndex[str] = idx
	return idx
}

func NewPprofWriter(
	meta parcaprofile.Meta,
	isDiff bool,
) *PprofWriter {
	w := &PprofWriter{
		isDiff: isDiff,

		res: &pprofpb.Profile{
			TimeNanos:     meta.Timestamp * 1000000, // We store timestamps in millisecond not nanoseconds.
			DurationNanos: meta.Duration,
			Period:        meta.Period,
			StringTable:   []string{""}, // The first string must be empty, pprof specifies this.
		},
		mappingByKey:     map[mappingKey]uint64{},
		functionByKey:    map[functionKey]uint64{},
		locationByKey:    map[string]uint64{},
		sampleByKey:      map[string]int{},
		stringTableIndex: map[string]int64{"": 0},

		buf: make([]byte, 4096),
	}

	w.res.PeriodType = &pprofpb.ValueType{Type: w.string(meta.PeriodType.Type), Unit: w.string(meta.PeriodType.Unit)}
	w.res.SampleType = []*pprofpb.ValueType{{Type: w.string(meta.SampleType.Type), Unit: w.string(meta.SampleType.Unit)}}

	return w
}

func (w *PprofWriter) getBuf(capacity int) []byte {
	if len(w.buf) < capacity {
		w.buf = make([]byte, capacity)
	}
	return w.buf[:capacity]
}

func (w *PprofWriter) WriteRecord(rec arrow.RecordBatch) {
	r := parcaprofile.NewRecordReader(rec)
	t := w.transpose(r)

	for i := 0; i < int(rec.NumRows()); i++ {
		w.sample(r, t, i)
	}
}

func (w *PprofWriter) sample(
	r *parcaprofile.RecordReader,
	t *pprofTranspositions,
	i int,
) {
	locStart, locEnd := r.Locations.ValueOffsets(i)
	if locStart != locEnd {
		s := &pprofpb.Sample{
			LocationId: make([]uint64, 0, locEnd-locStart),
		}

		for j := int(locStart); j < int(locEnd); j++ {
			if !r.Locations.ListValues().IsValid(j) {
				continue
			}
			l := w.location(r, t, j)
			if l != 0 {
				s.LocationId = append(s.LocationId, l)
			}
		}

		// There must be at least one location per sample.
		if len(s.LocationId) > 0 {
			key, labelNum := w.sampleKey(r, t, i, s)

			if idx, ok := w.sampleByKey[unsafeString(key)]; ok {
				if w.isDiff {
					w.res.Sample[idx].Value[0] += r.Diff.Value(i)
				} else {
					w.res.Sample[idx].Value[0] += r.Value.Value(i)
				}
				return
			}

			s.Label = make([]*pprofpb.Label, 0, labelNum)
			j := 0
			for k, label := range r.LabelColumns {
				if label.Col.IsValid(i) {
					s.Label = append(s.Label, &pprofpb.Label{
						Key: t.labelNameIndices[k],
						Str: t.labelValueTranspositions[k][label.Col.Value(i)],
					})
					j++
				}
			}

			if w.isDiff {
				s.Value = []int64{r.Diff.Value(i)}
			} else {
				s.Value = []int64{r.Value.Value(i)}
			}
			w.res.Sample = append(w.res.Sample, s)
		}
	}
}

func (w *PprofWriter) sampleKey(
	r *parcaprofile.RecordReader,
	t *pprofTranspositions,
	i int,
	s *pprofpb.Sample,
) ([]byte, int) {
	labelNum := 0
	for _, label := range r.LabelColumns {
		if label.Col.IsValid(i) {
			labelNum++
		}
	}

	buf := w.getBuf(16*labelNum + 8*len(s.LocationId))
	j := 0
	for k, label := range r.LabelColumns {
		if label.Col.IsValid(i) {
			binary.BigEndian.PutUint64(buf[j*16:], uint64(t.labelNameIndices[k]))
			binary.BigEndian.PutUint64(buf[j*16+8:], uint64(t.labelValueTranspositions[k][label.Col.Value(i)]))
			j++
		}
	}

	offset := labelNum * 16
	for k, l := range s.LocationId {
		binary.BigEndian.PutUint64(buf[offset+k*8:], l)
	}

	return buf, labelNum
}

func (w *PprofWriter) mapping(
	r *parcaprofile.RecordReader,
	t *pprofTranspositions,
	j int,
) uint64 {
	if r.MappingStart.IsNull(j) {
		return 0
	}

	m := &pprofpb.Mapping{
		MemoryStart:  r.MappingStart.Value(j),
		MemoryLimit:  r.MappingLimit.Value(j),
		FileOffset:   r.MappingOffset.Value(j),
		Filename:     t.mappingFile(r.MappingFileIndices.Value(j)),
		BuildId:      t.mappingBuildID(r.MappingBuildIDIndices.Value(j)),
		HasFunctions: true,
	}

	key := makeMappingKey(m)

	if idx, ok := w.mappingByKey[key]; ok {
		return idx
	}

	m.Id = uint64(len(w.res.Mapping)) + 1
	w.res.Mapping = append(w.res.Mapping, m)
	w.mappingByKey[key] = m.Id

	return m.Id
}

func (w *PprofWriter) location(
	r *parcaprofile.RecordReader,
	t *pprofTranspositions,
	j int,
) uint64 {
	loc := &pprofpb.Location{
		MappingId: w.mapping(r, t, j),
		Address:   r.Address.Value(j),
	}

	lineStart, lineEnd := r.Lines.ValueOffsets(j)
	if lineStart != lineEnd {
		loc.Line = make([]*pprofpb.Line, 0, lineEnd-lineStart)

		for k := int(lineStart); k < int(lineEnd); k++ {
			if r.Line.IsValid(k) {
				functionId := w.function(r, t, k)
				loc.Line = append(loc.Line, &pprofpb.Line{
					FunctionId: functionId,
					Line:       r.LineNumber.Value(k),
				})
			}
		}
	}

	key := w.makeLocationKey(loc)
	if idx, ok := w.locationByKey[unsafeString(key)]; ok {
		return idx
	}

	loc.Id = uint64(len(w.res.Location)) + 1
	w.res.Location = append(w.res.Location, loc)
	w.locationByKey[string(key)] = loc.Id

	return loc.Id
}

func (w *PprofWriter) function(
	r *parcaprofile.RecordReader,
	t *pprofTranspositions,
	k int,
) uint64 {
	if r.LineFunctionNameIndices.IsNull(k) {
		return 0
	}

	f := &pprofpb.Function{
		Name:       t.functionName(r.LineFunctionNameIndices.Value(k)),
		SystemName: t.functionSystemName(r.LineFunctionSystemNameIndices.Value(k)),
		Filename:   t.functionFilename(r.LineFunctionFilenameIndices.Value(k)),
		StartLine:  r.LineFunctionStartLine.Value(k),
	}

	key := makeFunctionKey(f)
	if idx, ok := w.functionByKey[key]; ok {
		return idx
	}

	f.Id = uint64(len(w.res.Function)) + 1
	w.res.Function = append(w.res.Function, f)
	w.functionByKey[key] = f.Id

	return f.Id
}

type pprofTranspositions struct {
	mappingFileTransposition        []int64
	mappingBuildIdTransposition     []int64
	functionNameTransposition       []int64
	functionSystemNameTransposition []int64
	functionFilenameTransposition   []int64

	labelNameIndices         []int64
	labelValueTranspositions [][]int64
}

func (t *pprofTranspositions) mappingFile(prevIdx uint32) int64 {
	return t.mappingFileTransposition[prevIdx]
}

func (t *pprofTranspositions) mappingBuildID(prevIdx uint32) int64 {
	return t.mappingBuildIdTransposition[prevIdx]
}

func (t *pprofTranspositions) functionName(prevIdx uint32) int64 {
	return t.functionNameTransposition[prevIdx]
}

func (t *pprofTranspositions) functionSystemName(prevIdx uint32) int64 {
	return t.functionSystemNameTransposition[prevIdx]
}

func (t *pprofTranspositions) functionFilename(prevIdx uint32) int64 {
	return t.functionFilenameTransposition[prevIdx]
}

func (w *PprofWriter) transpose(r *parcaprofile.RecordReader) *pprofTranspositions {
	t := &pprofTranspositions{
		mappingFileTransposition:        w.transposeBinaryArray(r.MappingFileDict),
		mappingBuildIdTransposition:     w.transposeBinaryArray(r.MappingBuildIDDict),
		functionNameTransposition:       w.transposeBinaryArray(r.LineFunctionNameDict),
		functionSystemNameTransposition: w.transposeBinaryArray(r.LineFunctionSystemNameDict),
		functionFilenameTransposition:   w.transposeBinaryArray(r.LineFunctionFilenameDict),

		labelNameIndices:         make([]int64, 0, len(r.LabelFields)),
		labelValueTranspositions: make([][]int64, 0, len(r.LabelFields)),
	}

	for i, f := range r.LabelFields {
		t.labelNameIndices = append(t.labelNameIndices, w.string(strings.TrimPrefix(f.Name, parcaprofile.ColumnLabelsPrefix)))
		t.labelValueTranspositions = append(t.labelValueTranspositions, w.transposeBinaryArray(r.LabelColumns[i].Dict))
	}

	return t
}

func (w *PprofWriter) transposeBinaryArray(arr *array.Binary) []int64 {
	res := make([]int64, 0, arr.Len())

	for i := 0; i < arr.Len(); i++ {
		res = append(res, w.byteString(arr.Value(i)))
	}

	return res
}

type mappingKey struct {
	size, offset  uint64
	buildIDOrFile int64
}

func makeMappingKey(m *pprofpb.Mapping) mappingKey {
	// Normalize addresses to handle address space randomization.
	// Round up to next 4K boundary to avoid minor discrepancies.
	const mapsizeRounding = 0x1000

	size := m.MemoryLimit - m.MemoryStart
	size = size + mapsizeRounding - 1
	size = size - (size % mapsizeRounding)
	key := mappingKey{
		size:   size,
		offset: m.FileOffset,
	}

	switch {
	case m.BuildId != 0:
		key.buildIDOrFile = m.BuildId
	case m.Filename != 0:
		key.buildIDOrFile = m.Filename
	default:
		// A mapping containing neither build ID nor file name is a fake mapping. A
		// key with empty buildIDOrFile is used for fake mappings so that they are
		// treated as the same mapping during merging.
	}
	return key
}

type functionKey struct {
	startLine                  int64
	name, systemName, fileName int64
}

func makeFunctionKey(f *pprofpb.Function) functionKey {
	return functionKey{
		startLine:  f.StartLine,
		name:       f.Name,
		systemName: f.SystemName,
		fileName:   f.Filename,
	}
}

func (w *PprofWriter) makeLocationKey(l *pprofpb.Location) []byte {
	if l.MappingId != 0 && l.Address != 0 {
		// Normalizes address to handle address space randomization.
		m := w.res.Mapping[l.MappingId-1]
		addr := l.Address - m.MemoryStart

		key := w.getBuf(16)
		binary.BigEndian.PutUint64(key, l.MappingId)
		binary.BigEndian.PutUint64(key[8:], addr)
		return key
	}

	key := w.getBuf(16 * len(l.Line))
	for i, line := range l.Line {
		binary.BigEndian.PutUint64(key[i*16:], line.FunctionId)
		binary.BigEndian.PutUint64(key[i*16+8:], uint64(line.Line))
	}
	return key
}
