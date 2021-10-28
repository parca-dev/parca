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
	"sort"
)

func sortSamples(samples []*Sample) {
	sort.Slice(samples, func(i, j int) bool {
		// TODO need to take labels into account
		stacktrace1 := samples[i].Location
		stacktrace2 := samples[j].Location

		stacktrace1Len := len(stacktrace1)
		stacktrace2Len := len(stacktrace2)

		k := 1
		for {
			switch {
			case k >= stacktrace1Len && k <= stacktrace2Len:
				// This means the stacktraces are identical up until this point, but stacktrace1 is ending, and shorter stactraces are "lower" than longer ones.
				return true
			case k <= stacktrace1Len && k >= stacktrace2Len:
				// This means the stacktraces are identical up until this point, but stacktrace2 is ending, and shorter stactraces are "lower" than longer ones.
				return false
			case uuidCompare(stacktrace1[stacktrace1Len-k].ID, stacktrace2[stacktrace2Len-k].ID) == -1:
				return true
			case uuidCompare(stacktrace1[stacktrace1Len-k].ID, stacktrace2[stacktrace2Len-k].ID) == 1:
				return false
			default:
				// This means the stack traces are identical up until this point. So advance to the next.
				k++
			}
		}
	})
}
