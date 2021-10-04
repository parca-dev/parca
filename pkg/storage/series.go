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
	"context"
	"errors"
	"fmt"
	"math"
	"sync"

	"github.com/parca-dev/parca/pkg/storage/chunkenc"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrOutOfOrderSample = errors.New("out of order sample")
)

type MemSeries struct {
	id   uint64
	lset labels.Labels

	periodType ValueType
	sampleType ValueType

	minTime, maxTime int64
	timestamps       timestampChunks
	durations        []chunkenc.Chunk
	periods          []chunkenc.Chunk

	// TODO: Might be worth combining behind some struct?
	// Or maybe not because it's easier to serialize?

	// mu locks the following maps for concurrent access.
	mu sync.RWMutex
	// Flat and cumulative values as well as labels by the node's ProfileTreeValueNodeKey.
	flatValues       map[ProfileTreeValueNodeKey][]chunkenc.Chunk
	cumulativeValues map[ProfileTreeValueNodeKey][]chunkenc.Chunk
	labels           map[ProfileTreeValueNodeKey]map[string][]string
	numLabels        map[ProfileTreeValueNodeKey]map[string][]int64
	numUnits         map[ProfileTreeValueNodeKey]map[string][]string

	seriesTree *MemSeriesTree
	numSamples uint16

	updateMaxTime func(int64)
	chunkPool     ChunkPool

	tracer          trace.Tracer
	samplesAppended prometheus.Counter
}

func NewMemSeries(id uint64, lset labels.Labels, updateMaxTime func(int64), chunkPool ChunkPool) *MemSeries {
	s := &MemSeries{
		id:   id,
		lset: lset,

		minTime: math.MaxInt64,
		maxTime: math.MinInt64,

		timestamps:       timestampChunks{},
		durations:        make([]chunkenc.Chunk, 0, 1),
		periods:          make([]chunkenc.Chunk, 0, 1),
		flatValues:       make(map[ProfileTreeValueNodeKey][]chunkenc.Chunk),
		cumulativeValues: make(map[ProfileTreeValueNodeKey][]chunkenc.Chunk),
		labels:           make(map[ProfileTreeValueNodeKey]map[string][]string),
		numLabels:        make(map[ProfileTreeValueNodeKey]map[string][]int64),
		numUnits:         make(map[ProfileTreeValueNodeKey]map[string][]string),

		updateMaxTime: updateMaxTime,
		tracer:        trace.NewNoopTracerProvider().Tracer(""),

		chunkPool: chunkPool,
	}
	s.seriesTree = &MemSeriesTree{s: s}

	return s
}

func (s *MemSeries) Labels() labels.Labels {
	return s.lset
}

func (s *MemSeries) storeMaxTime(t int64) {
	s.maxTime = t
	s.updateMaxTime(t)
}

func (s *MemSeries) Appender() (*MemSeriesAppender, error) {
	return &MemSeriesAppender{s: s}, nil
}

func (s *MemSeries) appendTree(profileTree *ProfileTree) error {
	if s.seriesTree == nil {
		s.seriesTree = &MemSeriesTree{s: s}
	}

	return s.seriesTree.Insert(s.numSamples%samplesPerChunk, profileTree)
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

	for _, chunks := range s.flatValues {
		for _, c := range chunks {
			flat = append(flat, MemSeriesValueStats{
				samples: c.NumSamples(),
				bytes:   len(c.Bytes()),
			})
		}
	}

	for _, chunks := range s.cumulativeValues {
		for _, c := range chunks {
			cumulative = append(cumulative, MemSeriesValueStats{
				samples: c.NumSamples(),
				bytes:   len(c.Bytes()),
			})
		}
	}

	return MemSeriesStats{
		samples:     s.numSamples,
		Cumulatives: cumulative,
		Flat:        flat,
	}
}

type MemSeriesAppender struct {
	s          *MemSeries
	timestamps chunkenc.Appender
	duration   chunkenc.Appender
	periods    chunkenc.Appender
}

const samplesPerChunk = 120

func (a *MemSeriesAppender) Append(ctx context.Context, p *Profile) error {
	ctx, span := a.s.tracer.Start(ctx, "Append")
	defer span.End()

	a.s.mu.Lock()
	defer a.s.mu.Unlock()

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

	timestamp := p.Meta.Timestamp

	if timestamp <= a.s.maxTime {
		return ErrOutOfOrderSample
	}

	newChunks := false
	if len(a.s.timestamps) == 0 {
		newChunks = true
	} else if a.s.timestamps[len(a.s.timestamps)-1].chunk.NumSamples() == 0 {
		newChunks = true
	} else if a.s.timestamps[len(a.s.timestamps)-1].chunk.NumSamples() >= samplesPerChunk {
		newChunks = true
	}

	if newChunks {
		_, newChunksSpan := a.s.tracer.Start(ctx, "newChunks")
		defer newChunksSpan.End()

		tc := a.s.chunkPool.GetTimestamp()
		tc.minTime = timestamp
		tc.maxTime = timestamp
		a.s.timestamps = append(a.s.timestamps, tc)
		timeApp, err := a.s.timestamps[len(a.s.timestamps)-1].chunk.Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next timestamp chunk: %w", err)
		}
		a.timestamps = timeApp

		a.s.durations = append(a.s.durations, a.s.chunkPool.GetRLE())
		durationApp, err := a.s.durations[len(a.s.durations)-1].Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next durations chunk: %w", err)
		}
		a.duration = durationApp

		a.s.periods = append(a.s.periods, a.s.chunkPool.GetRLE())
		periodsApp, err := a.s.periods[len(a.s.periods)-1].Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next periods chunk: %w", err)
		}
		a.periods = periodsApp

		for k := range a.s.cumulativeValues {
			for len(a.s.cumulativeValues[k]) < len(a.s.timestamps) {
				a.s.cumulativeValues[k] = append(a.s.cumulativeValues[k], a.s.chunkPool.GetXOR())
			}
		}
		for k := range a.s.flatValues {
			for len(a.s.flatValues[k]) < len(a.s.timestamps) {
				a.s.flatValues[k] = append(a.s.flatValues[k], a.s.chunkPool.GetXOR())
			}
		}

		newChunksSpan.End()
	}

	if a.timestamps == nil {
		app, err := a.s.timestamps[len(a.s.timestamps)-1].chunk.Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next timestamp chunk: %w", err)
		}
		a.timestamps = app
	}
	if a.duration == nil {
		app, err := a.s.durations[len(a.s.durations)-1].Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next duration chunk: %w", err)
		}
		a.duration = app
	}
	if a.periods == nil {
		app, err := a.s.periods[len(a.s.periods)-1].Appender()
		if err != nil {
			return fmt.Errorf("failed to add the next periods chunk: %w", err)
		}
		a.periods = app
	}

	a.timestamps.AppendAt(a.s.numSamples%samplesPerChunk, timestamp)
	a.duration.AppendAt(a.s.numSamples%samplesPerChunk, p.Meta.Duration)
	a.periods.AppendAt(a.s.numSamples%samplesPerChunk, p.Meta.Period)

	if a.s.timestamps[len(a.s.timestamps)-1].minTime > timestamp {
		a.s.timestamps[len(a.s.timestamps)-1].minTime = timestamp
	}
	if a.s.timestamps[len(a.s.timestamps)-1].maxTime < timestamp {
		a.s.timestamps[len(a.s.timestamps)-1].maxTime = timestamp
	}

	// Set the timestamp as minTime if timestamp != 0
	if a.s.minTime == math.MaxInt64 && timestamp != 0 {
		a.s.minTime = timestamp
	}

	_, appendTreeSpan := a.s.tracer.Start(ctx, "appendTree")
	// appendTree locks the maps itself.
	if err := a.s.appendTree(p.Tree); err != nil {
		appendTreeSpan.End()
		return err
	}
	appendTreeSpan.End()

	a.s.storeMaxTime(timestamp)

	a.s.numSamples++

	if a.s.samplesAppended != nil {
		a.s.samplesAppended.Inc()
	}
	return nil
}

func (s *MemSeries) truncateChunksBefore(mint int64) (removed int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.timestamps[0].maxTime > mint {
		// We don't have anything to do and can exist early.
		return 0
	}

	// Quickly check if we can get rid of all chunks.
	if s.timestamps[len(s.timestamps)-1].maxTime < mint {
		length := len(s.timestamps)
		// delete all chunks but keep the slices allocated.
		// TODO: We might want to delete the entire series here.

		for _, c := range s.timestamps {
			_ = s.chunkPool.Put(c)
		}
		for _, c := range s.durations {
			_ = s.chunkPool.Put(c)
		}
		for _, c := range s.periods {
			_ = s.chunkPool.Put(c)
		}

		s.timestamps = s.timestamps[:0]
		s.durations = s.durations[:0]
		s.periods = s.periods[:0]

		for key, chunks := range s.cumulativeValues {
			for _, c := range chunks {
				_ = s.chunkPool.Put(c)
			}
			s.cumulativeValues[key] = chunks[:0]
		}
		for key, chunks := range s.flatValues {
			for _, c := range chunks {
				_ = s.chunkPool.Put(c)
			}
			s.flatValues[key] = chunks[:0]
		}

		s.minTime = math.MaxInt64
		s.maxTime = math.MinInt64

		return length
	}

	start := 0
	for i, t := range s.timestamps {
		if t.minTime > mint {
			break
		}
		start = i
	}

	for i := 0; i < start; i++ {
		_ = s.chunkPool.Put(s.timestamps[i])
		_ = s.chunkPool.Put(s.durations[i])
		_ = s.chunkPool.Put(s.periods[i])
	}

	// Truncate the beginning of the slices.
	s.timestamps = s.timestamps[start:]
	s.durations = s.durations[start:]
	s.periods = s.periods[start:]

	for key, chunks := range s.cumulativeValues {
		s.cumulativeValues[key] = chunks[start:]
	}
	for key, chunks := range s.flatValues {
		s.flatValues[key] = chunks[start:]
	}

	s.minTime = s.timestamps[0].minTime

	// TODO: Truncate seriesTree and labels...
	// We could somehow a list of the keys for empty chunks while iterating through them above.
	// With that list we could at least somewhat more quickly figure out which nodes in the tree
	// and also which labels to get rid of.

	return start
}
