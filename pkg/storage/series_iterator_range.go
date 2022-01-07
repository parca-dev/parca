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
	"fmt"

	"github.com/parca-dev/parca/pkg/profile"
	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/prometheus/model/labels"
)

// MemRangeSeries is an iterator that only queries certain chunks within the range and
// then only the samples within the range.
type MemRangeSeries struct {
	s    *MemSeries
	mint int64
	maxt int64
}

func (rs *MemRangeSeries) Labels() labels.Labels {
	return rs.s.Labels()
}

func (rs *MemRangeSeries) Iterator() ProfileSeriesIterator {
	rs.s.mu.RLock()
	defer rs.s.mu.RUnlock()

	var numSamples uint64

	chunkStart, chunkEnd := rs.s.timestamps.indexRange(rs.mint, rs.maxt)
	timestamps := make([]chunkenc.Chunk, 0, chunkEnd-chunkStart)
	for _, t := range rs.s.timestamps[chunkStart:chunkEnd] {
		numSamples += uint64(t.chunk.NumSamples())
		timestamps = append(timestamps, t.chunk)
	}

	timestampIt := NewMultiChunkIterator(timestamps)
	start, end, err := getIndexRange(timestampIt, numSamples, rs.mint, rs.maxt)
	if err != nil {
		return &MemRangeSeriesIterator{err: err}
	}

	rootIt := NewMultiChunkIterator(rs.s.root[chunkStart:chunkEnd])
	if start != 0 {
		rootIt.Seek(start)
	}

	sampleIterators := make(map[string]*MultiChunksIterator, len(rs.s.samples))
	for key, chunks := range rs.s.samples {
		sampleIterators[key] = NewMultiChunkIterator(chunks)
	}

	timestampIterator := NewMultiChunkIterator(timestamps)
	durationsIterator := NewMultiChunkIterator(rs.s.durations[chunkStart:chunkEnd])
	periodsIterator := NewMultiChunkIterator(rs.s.periods[chunkStart:chunkEnd])

	if start != 0 {
		timestampIterator.Seek(start)
		durationsIterator.Seek(start)
		periodsIterator.Seek(start)
		for _, sampleIterator := range sampleIterators {
			sampleIterator.Seek(start)
		}
	}

	if end-start < numSamples {
		numSamples = end - start - 1
	}

	return &MemRangeSeriesIterator{
		s:    rs.s,
		mint: rs.mint,
		maxt: rs.maxt,

		numSamples:         numSamples,
		timestampsIterator: timestampIterator,
		durationsIterator:  durationsIterator,
		periodsIterator:    periodsIterator,

		sampleIterators: sampleIterators,
	}
}

func getIndexRange(it MemSeriesValuesIterator, numSamples uint64, mint, maxt int64) (uint64, uint64, error) {
	// figure out the index of the first sample > mint and the last sample < maxt
	start := uint64(0)
	end := uint64(0)
	i := uint64(0)
	for it.Next() {
		if i == numSamples {
			end++
			break
		}
		t := it.At()
		// MultiChunkIterator might return sparse values - shouldn't usually happen though.
		if t == 0 {
			break
		}
		if t < mint {
			start++
		}
		if t <= maxt {
			end++
		} else {
			break
		}
		i++
	}

	return start, end, it.Err()
}

// MemSeriesValuesIterator is an abstraction on iterator over values from possible multiple chunks.
// It most likely is an abstraction like the MultiChunksIterator over []chunkenc.Chunk.
type MemSeriesValuesIterator interface {
	// Next iterates to the next value and returns true if there's more.
	Next() bool
	// At returns the current value.
	At() int64
	// Err returns the underlying errors. Next will return false when encountering errors.
	Err() error
	// Read returns how many iterations the iterator has read at any given moment.
	Read() uint64
}

type MemRangeSeriesIterator struct {
	s    *MemSeries
	mint int64
	maxt int64

	timestampsIterator MemSeriesValuesIterator
	durationsIterator  MemSeriesValuesIterator
	periodsIterator    MemSeriesValuesIterator

	sampleIterators map[string]*MultiChunksIterator

	numSamples uint64 // uint16 might not be enough for many chunks (~500+)
	err        error
}

func (it *MemRangeSeriesIterator) Next() bool {
	if it.err != nil || it.numSamples == 0 {
		return false
	}

	it.s.mu.RLock()
	defer it.s.mu.RUnlock()

	if !it.timestampsIterator.Next() {
		it.err = errors.New("unexpected end of timestamps iterator")
		return false
	}

	if it.timestampsIterator.Err() != nil {
		it.err = fmt.Errorf("next timestamp: %w", it.timestampsIterator.Err())
		return false
	}

	if !it.durationsIterator.Next() {
		it.err = errors.New("unexpected end of durations iterator")
		return false
	}

	if it.durationsIterator.Err() != nil {
		it.err = fmt.Errorf("next duration: %w", it.durationsIterator.Err())
		return false
	}

	if !it.periodsIterator.Next() {
		it.err = errors.New("unexpected end of periods iterator")
		return false
	}

	if it.periodsIterator.Err() != nil {
		it.err = fmt.Errorf("next period: %w", it.periodsIterator.Err())
		return false
	}

	read := it.timestampsIterator.Read()

	if dread := it.durationsIterator.Read(); dread != read {
		it.err = fmt.Errorf("duration iterator in wrong iteration, expected %d got %d", read, dread)
		return false
	}
	if pread := it.periodsIterator.Read(); pread != read {
		it.err = fmt.Errorf("period iterator in wrong iteration, expected %d got %d", read, pread)
		return false
	}

	for _, sit := range it.sampleIterators {
		if !sit.Next() {
			it.err = errors.New("unexpected end of numSamples iterator")
			return false
		}
		if sread := sit.Read(); sread != read {
			it.err = fmt.Errorf("sample iterator in wrong iteration, expected %d got %d", read, sread)
			return false
		}
	}

	it.numSamples--
	return true
}

func (it *MemRangeSeriesIterator) At() profile.InstantProfile {
	return &MemSeriesInstantFlatProfile{
		PeriodType: it.s.periodType,
		SampleType: it.s.sampleType,

		timestampsIterator: it.timestampsIterator,
		durationsIterator:  it.durationsIterator,
		periodsIterator:    it.periodsIterator,
		sampleIterators:    it.sampleIterators,
	}
}

func (it *MemRangeSeriesIterator) Err() error {
	return it.err
}

type MemSeriesInstantFlatProfile struct {
	PeriodType profile.ValueType
	SampleType profile.ValueType

	timestampsIterator MemSeriesValuesIterator
	durationsIterator  MemSeriesValuesIterator
	periodsIterator    MemSeriesValuesIterator

	sampleIterators map[string]*MultiChunksIterator
}

func (m MemSeriesInstantFlatProfile) ProfileMeta() profile.InstantProfileMeta {
	return profile.InstantProfileMeta{
		PeriodType: m.PeriodType,
		SampleType: m.SampleType,
		Timestamp:  m.timestampsIterator.At(),
		Duration:   m.durationsIterator.At(),
		Period:     m.periodsIterator.At(),
	}
}

func (m MemSeriesInstantFlatProfile) Samples() map[string]*profile.Sample {
	samples := make(map[string]*profile.Sample, len(m.sampleIterators))
	for k, it := range m.sampleIterators {
		samples[k] = &profile.Sample{
			Value: it.At(),
		}
	}
	return samples
}
