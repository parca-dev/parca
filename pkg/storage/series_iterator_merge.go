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
	"github.com/prometheus/prometheus/pkg/labels"
)

// MemMergeSeries is an iterator that sums up all values while iterating that are within the range.
// In the end it returns a slice iterator with only the merge profile in it.
type MemMergeSeries struct {
	s    *MemSeries
	mint int64
	maxt int64
}

func (ms *MemMergeSeries) Labels() labels.Labels {
	return ms.s.Labels()
}

func iteratorRangeMax(it MemSeriesValuesIterator, start, end uint64) (int64, error) {
	max := int64(0)
	i := uint64(0)
	for it.Next() {
		if i >= end {
			break
		}
		cur := it.At()
		if i >= start && cur > max {
			max = cur
		}
		i++
	}
	return max, it.Err()
}

func iteratorRangeSum(it MemSeriesValuesIterator, start, end uint64) (int64, error) {
	sum := int64(0)
	i := uint64(0)
	for it.Next() {
		if i >= end {
			break
		}
		if i >= start {
			sum += it.At()
		}
		i++
	}
	return sum, it.Err()
}
