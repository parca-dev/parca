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

type MemSeriesInstantFlatProfile struct {
	PeriodType ValueType
	SampleType ValueType

	timestampsIterator MemSeriesValuesIterator
	durationsIterator  MemSeriesValuesIterator
	periodsIterator    MemSeriesValuesIterator

	sampleIterators map[string]MemSeriesValuesIterator
}

func (m MemSeriesInstantFlatProfile) ProfileMeta() InstantProfileMeta {
	return InstantProfileMeta{
		PeriodType: m.PeriodType,
		SampleType: m.SampleType,
		Timestamp:  m.timestampsIterator.At(),
		Duration:   m.durationsIterator.At(),
		Period:     m.periodsIterator.At(),
	}
}

func (m MemSeriesInstantFlatProfile) Samples() map[string]*Sample {
	samples := make(map[string]*Sample, len(m.sampleIterators))
	for k, it := range m.sampleIterators {
		samples[k] = &Sample{
			Value: it.At(),
		}
	}
	return samples
}
