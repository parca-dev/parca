// Copyright 2022-2026 The Parca Authors
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

	"github.com/stretchr/testify/require"
)

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name        string
		query       string
		wantName    string
		wantDelta   bool
		wantErr     bool
		wantMatches int
	}{
		{
			name:        "simple profile type",
			query:       "process_cpu:samples:count:cpu:nanoseconds{}",
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 0,
		},
		{
			name:        "delta profile type",
			query:       "process_cpu:samples:count:cpu:nanoseconds:delta{}",
			wantName:    "process_cpu",
			wantDelta:   true,
			wantErr:     false,
			wantMatches: 0,
		},
		{
			name:        "profile type with label matcher",
			query:       `process_cpu:samples:count:cpu:nanoseconds{node="test"}`,
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 1,
		},
		{
			name:        "profile type with multiple label matchers",
			query:       `process_cpu:samples:count:cpu:nanoseconds{node="test",container="app"}`,
			wantName:    "process_cpu",
			wantDelta:   false,
			wantErr:     false,
			wantMatches: 2,
		},
		{
			name:    "invalid query",
			query:   "invalid",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qp, err := ParseQuery(tt.query)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.wantName, qp.Meta.Name)
			require.Equal(t, tt.wantDelta, qp.Delta)
			require.Len(t, qp.Matchers, tt.wantMatches)
		})
	}
}
