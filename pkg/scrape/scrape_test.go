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

package scrape

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	profilepb "github.com/parca-dev/parca/gen/proto/go/parca/profilestore/v1alpha1"
)

func TestParseExecutableInfo(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected *profilepb.ExecutableInfo
		err      error
	}{
		{
			name:  "valid",
			input: "executableInfo=0x1;0x1000;0x1000",
			expected: &profilepb.ExecutableInfo{
				ElfType: 0x1,
				LoadSegment: &profilepb.LoadSegment{
					Offset: 0x1000,
					Vaddr:  0x1000,
				},
			},
			err: nil,
		},
		{
			name:  "only_elf_type",
			input: "executableInfo=0x1",
			expected: &profilepb.ExecutableInfo{
				ElfType: 0x1,
			},
		},
		{
			name:  "invalid_elf_type",
			input: "test",
			err:   errors.New("parse elfType: strconv.ParseUint: parsing \"test\": invalid syntax"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := parseExecutableInfo(tc.input)
			if tc.err != nil && (err == nil || err.Error() != tc.err.Error()) {
				t.Fatalf("unexpected error: %v", err)
				return
			}
			if tc.err != nil && err != nil && err.Error() == tc.err.Error() {
				return
			}
			require.Equal(t, tc.expected.ElfType, info.ElfType)
			if tc.expected.LoadSegment != nil {
				require.Equal(t, tc.expected.LoadSegment.Offset, info.LoadSegment.Offset)
				require.Equal(t, tc.expected.LoadSegment.Vaddr, info.LoadSegment.Vaddr)
			}
		})
	}
}
