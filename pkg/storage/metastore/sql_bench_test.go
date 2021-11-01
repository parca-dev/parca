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

package metastore

import (
	"fmt"
	"math"
	"testing"

	"github.com/google/uuid"
)

var result string

func BenchmarkBuildLinesByLocationIDsQuery(b *testing.B) {
	for k := 0.; k <= 6; k++ {
		n := uint64(math.Pow(10, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.StopTimer()
			input := make([]uuid.UUID, 0, n)
			for i := uint64(0); i < n; i++ {
				input = append(input, uuid.New())
			}
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				result = buildLinesByLocationIDsQuery(input)
			}
		})
	}
}

func BenchmarkBuildLocationsByIDsQuery(b *testing.B) {
	for k := 0.; k <= 5; k++ {
		n := uint64(math.Pow(10, k))
		b.Run(fmt.Sprintf("%d", n), func(b *testing.B) {
			b.StopTimer()
			input := make([]uuid.UUID, 0, n)
			for i := uint64(0); i < n; i++ {
				input = append(input, uuid.New())
			}
			b.StartTimer()

			for i := 0; i < b.N; i++ {
				result = buildLocationsByIDsQuery(input)
			}
		})
	}
}
