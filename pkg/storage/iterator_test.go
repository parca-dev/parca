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
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProfileTreeIterator(t *testing.T) {
	pt := NewProfileTree()
	pt.Insert(makeSample(2, []uint64{2, 1}))
	pt.Insert(makeSample(1, []uint64{5, 3, 2, 1}))
	pt.Insert(makeSample(3, []uint64{4, 3, 2, 1}))
	pt.Insert(makeSample(1, []uint64{3, 3, 1}))

	it := pt.Iterator()

	res := []uint64{}
	for {
		if !it.HasMore() {
			break
		}

		if it.NextChild() {
			res = append(res, it.At().LocationID())
			it.StepInto()
			continue
		}
		it.StepUp()
	}

	require.Equal(t, []uint64{0, 1, 2, 3, 4, 5, 3, 3}, res)
}
