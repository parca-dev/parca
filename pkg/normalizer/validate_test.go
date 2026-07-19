// Copyright 2026 The Parca Authors
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

package normalizer

import (
	"testing"

	"github.com/stretchr/testify/require"

	pprofpb "github.com/parca-dev/parca/gen/proto/go/google/pprof"
)

func TestValidatePprofProfile_StringTableBounds(t *testing.T) {
	// String table has 3 entries, so the only valid indices are 0, 1 and 2.
	// An index equal to len(StringTable) is one past the end.
	stringTable := []string{"", "cpu", "nanoseconds"}

	for _, tc := range []struct {
		name    string
		profile *pprofpb.Profile
		wantErr bool
	}{
		{
			name: "valid",
			profile: &pprofpb.Profile{
				StringTable: stringTable,
				SampleType:  []*pprofpb.ValueType{{Type: 1, Unit: 2}},
				PeriodType:  &pprofpb.ValueType{Type: 1, Unit: 2},
			},
			wantErr: false,
		},
		{
			// period_type is optional; omitting it must validate cleanly,
			// not panic inside the validator.
			name: "nil period type",
			profile: &pprofpb.Profile{
				StringTable: stringTable,
				SampleType:  []*pprofpb.ValueType{{Type: 1, Unit: 2}},
				PeriodType:  nil,
			},
			wantErr: false,
		},
		{
			name: "sample type index equal to string table length",
			profile: &pprofpb.Profile{
				StringTable: stringTable,
				SampleType:  []*pprofpb.ValueType{{Type: int64(len(stringTable))}},
				PeriodType:  &pprofpb.ValueType{Type: 1, Unit: 2},
			},
			wantErr: true,
		},
		{
			name: "period type index equal to string table length",
			profile: &pprofpb.Profile{
				StringTable: stringTable,
				SampleType:  []*pprofpb.ValueType{{Type: 1, Unit: 2}},
				PeriodType:  &pprofpb.ValueType{Type: int64(len(stringTable))},
			},
			wantErr: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// require.NotPanics keeps a regression from crashing the whole
			// test binary and reports it as a normal failure instead.
			require.NotPanics(t, func() {
				err := ValidatePprofProfile(tc.profile, nil)
				if tc.wantErr {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
			})
		})
	}
}
