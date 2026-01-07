// Copyright 2023-2026 The Parca Authors
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

package parcacol

import (
	"testing"

	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestBuildArrowLocations(t *testing.T) {
	stacktraces := []*pb.Stacktrace{{
		LocationIds: []string{"1"},
	}, {
		LocationIds: []string{"2"},
	}}
	locations := []*profile.Location{{
		ID:      "1",
		Address: 0x1,
		Mapping: &pb.Mapping{
			Id:      "1",
			BuildId: "1",
		},
		Lines: []profile.LocationLine{{
			Line: 1,
			Function: &pb.Function{
				Id:   "1",
				Name: "main",
			},
		}},
	}, {
		ID:      "2",
		Address: 0x1,
		Mapping: &pb.Mapping{
			Id:      "2",
			BuildId: "2",
		},
	}}
	locationIndex := map[string]int{"1": 0, "2": 1}

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	r, err := BuildArrowLocations(mem, stacktraces, locations, locationIndex)
	require.NoError(t, err)
	defer r.Release()
}
