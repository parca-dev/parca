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

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func TestEncodeDecode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		location    *pprofpb.Location
		mapping     *pprofpb.Mapping
		functions   []*pprofpb.Function
		stringTable []string
	}{{
		name: "one line",
		location: &pprofpb.Location{
			Line: []*pprofpb.Line{{
				Line:       1,
				FunctionId: 1,
			}},
		},
		functions: []*pprofpb.Function{{
			StartLine:  2,
			Name:       1,
			SystemName: 1,
			Filename:   2,
		}},
		stringTable: []string{"", "foo", "bar"},
	}, {
		name:     "mapping no line",
		location: &pprofpb.Location{},
		mapping:  &pprofpb.Mapping{},
	}, {
		name:     "no line",
		location: &pprofpb.Location{},
	}}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			buf := EncodePprofLocation(
				tc.location,
				tc.mapping,
				tc.functions,
				tc.stringTable,
			)
			lw := NewLocationsWriter(memory.DefaultAllocator)
			_, err := DecodeInto(lw, buf, nil)
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
