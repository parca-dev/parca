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

package query

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	pprofprofile "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/profile"
)

func TestGenerateFlatPprof(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	fileContent := MustReadAllGzip(t, "testdata/alloc_objects.pb.gz")
	pp, err := pprofprofile.ParseData(fileContent)
	require.NoError(t, err)

	p, err := PprofToSymbolizedProfile(
		profile.Meta{
			Name: "memory",
			SampleType: profile.ValueType{
				Type: "alloc_objects",
				Unit: "count",
			},
			PeriodType: profile.ValueType{
				Type: "space",
				Unit: "bytes",
			},
			Timestamp: time.Date(2020, 12, 17, 10, 8, 38, 549000000, time.UTC).UnixMilli(),
			Period:    524288,
		},
		pp,
		0,
		[]string{},
	)
	require.NoError(t, err)

	resProfile, err := GenerateFlatPprof(ctx, false, p)
	require.NoError(t, err)

	data, err := resProfile.MarshalVT()
	require.NoError(t, err)

	res, err := pprofprofile.ParseData(data)
	require.NoError(t, err)

	require.Equal(t, &pprofprofile.ValueType{Type: "space", Unit: "bytes"}, res.PeriodType)
	require.Equal(t, []*pprofprofile.ValueType{{Type: "alloc_objects", Unit: "count"}}, res.SampleType)
	require.Equal(t, time.Date(2020, 12, 17, 10, 8, 38, 549000000, time.UTC).UnixNano(), res.TimeNanos)
	require.Equal(t, int64(0), res.DurationNanos)
	require.Equal(t, int64(524288), res.Period)

	require.Equal(t, []*pprofprofile.Mapping{{
		ID:              1,
		Start:           4194304,
		Limit:           23252992,
		Offset:          0,
		File:            "/bin/operator",
		BuildID:         "",
		HasFunctions:    true,
		HasFilenames:    false,
		HasLineNumbers:  false,
		HasInlineFrames: false,
	}}, res.Mapping)

	require.Equal(t, 974, len(res.Function))
	require.Equal(t, 1886, len(res.Location))
	require.Equal(t, 4661, len(res.Sample))

	tmpfile, err := os.CreateTemp("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	data, err = SerializePprof(resProfile)
	require.NoError(t, err)
	_, err = tmpfile.Write(data)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	f, err := os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := pprofprofile.Parse(f)

	for _, s := range resProf.Sample {
		if s.Location == nil {
			fmt.Println("locations nil")
		}
	}

	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}

func TestGeneratePprofNilMapping(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	var err error

	functions := []*pprofprofile.Function{{
		ID:   1,
		Name: "1",
	}, {
		ID:   2,
		Name: "2",
	}}

	locations := []*pprofprofile.Location{{
		ID:      1,
		Mapping: nil,
		Line:    []pprofprofile.Line{{Function: functions[0]}},
	}, {
		ID:      2,
		Mapping: nil,
		Line:    []pprofprofile.Line{{Function: functions[1]}},
	}}

	p, err := PprofToSymbolizedProfile(
		profile.Meta{},
		&pprofprofile.Profile{
			Function: functions,
			Location: locations,
			Sample: []*pprofprofile.Sample{{
				Location: []*pprofprofile.Location{locations[1], locations[0]},
				Value:    []int64{1},
			}},
		},
		0,
		[]string{},
	)
	require.NoError(t, err)

	res, err := GenerateFlatPprof(ctx, false, p)
	require.NoError(t, err)

	tmpfile, err := os.CreateTemp("", "pprof")
	defer os.Remove(tmpfile.Name())
	require.NoError(t, err)
	data, err := SerializePprof(res)
	require.NoError(t, err)
	_, err = tmpfile.Write(data)
	require.NoError(t, err)
	require.NoError(t, tmpfile.Close())

	f, err := os.Open(tmpfile.Name())
	require.NoError(t, err)
	resProf, err := pprofprofile.Parse(f)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, resProf.CheckValid())
}
