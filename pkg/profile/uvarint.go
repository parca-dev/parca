// Copyright 2024-2026 The Parca Authors
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

package profile

const maxUint64 = uint64(1<<64 - 1)

// MaxValN is the maximum varint-encoded integer that fits in N bytes.
const (
	MaxVal9 = maxUint64 >> (1 + iota*7)
	MaxVal8
	MaxVal7
	MaxVal6
	MaxVal5
	MaxVal4
	MaxVal3
	MaxVal2
	MaxVal1
)

// UvarintSize returns the number of bytes necessary to encode a given uint.
// Unfortunately the standard lib does not provide this function, so we create
// it here.
func UvarintSize(x uint64) int {
	if x <= MaxVal4 {
		if x <= MaxVal1 {
			return 1
		} else if x <= MaxVal2 {
			return 2
		} else if x <= MaxVal3 {
			return 3
		}
		return 4
	}
	if x <= MaxVal5 {
		return 5
	} else if x <= MaxVal6 {
		return 6
	} else if x <= MaxVal7 {
		return 7
	} else if x <= MaxVal8 {
		return 8
	} else if x <= MaxVal9 {
		return 9
	}
	return 10
}
