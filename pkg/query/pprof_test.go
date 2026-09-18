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

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/memory"
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

// Null metadata must remain absent even when another sample populates the
// dictionary. Reading its index can either panic or copy another frame's data.
func TestGeneratePprofNullMetadata(t *testing.T) {
	for _, field := range []string{"mapping_file", "mapping_build_id", "function_name", "function_system_name", "function_filename", "function"} {
		for _, populated := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/populated=%t", field, populated), func(t *testing.T) {
				mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
				defer mem.AssertSize(t, 0)
				w := profile.NewWriter(mem, []string{"service"})
				defer w.Release()
				appendSample := func(address uint64, missing string) {
					w.LocationsList.Append(true)
					w.Locations.Append(true)
					w.Addresses.Append(address)
					w.MappingStart.Append(0x1000)
					w.MappingLimit.Append(0x2000)
					w.MappingOffset.Append(0)
					w.Lines.Append(true)
					w.Line.Append(true)
					w.LineNumber.Append(42)
					w.ColumnNumber.Append(3)
					if missing == "function" {
						w.FunctionStartLine.AppendNull()
					} else {
						w.FunctionStartLine.Append(40)
					}
					for _, attr := range []struct {
						name, value string
						builder     *array.BinaryDictionaryBuilder
					}{
						{"mapping_file", "app", w.MappingFile},
						{"mapping_build_id", "build", w.MappingBuildID},
						{"function_name", "main.work", w.FunctionName},
						{"function_system_name", "main.work", w.FunctionSystemName},
						{"function_filename", "main.go", w.FunctionFilename},
					} {
						if missing == attr.name || (missing == "function" && (attr.name == "function_name" || attr.name == "function_system_name" || attr.name == "function_filename")) {
							attr.builder.AppendNull()
						} else {
							require.NoError(t, attr.builder.Append([]byte(attr.value)))
						}
					}
					require.NoError(t, w.LabelBuilders[0].Append([]byte("checkout")))
					w.Value.Append(13)
					w.Diff.Append(0)
					w.TimeNanos.Append(1)
					w.Period.Append(1)
				}
				if populated {
					appendSample(0x1010, "")
				}
				appendSample(0x1020, field)
				rec := w.RecordBuilder.NewRecordBatch()
				defer rec.Release()
				out, err := GenerateFlatPprof(context.Background(), false, profile.Profile{
					Meta:    profile.Meta{SampleType: profile.ValueType{Type: "cpu", Unit: "nanoseconds"}},
					Samples: []arrow.RecordBatch{rec},
				})
				require.NoError(t, err)
				raw, err := SerializePprof(out)
				require.NoError(t, err)
				parsed, err := pprofprofile.ParseData(raw)
				require.NoError(t, err)
				count := 1
				if populated {
					count++
				}
				require.Len(t, parsed.Sample, count)
				for _, sample := range parsed.Sample {
					require.Equal(t, []int64{13}, sample.Value)
					require.Equal(t, []string{"checkout"}, sample.Label["service"])
					require.Len(t, sample.Location, 1)
					loc := sample.Location[0]
					require.Contains(t, []uint64{0x1010, 0x1020}, loc.Address)
					require.NotNil(t, loc.Mapping)
					wantFile, wantBuild := "app", "build"
					if loc.Address == 0x1020 && field == "mapping_file" && !populated {
						wantFile = ""
					}
					if loc.Address == 0x1020 && field == "mapping_build_id" {
						wantBuild = ""
					}
					require.Equal(t, wantFile, loc.Mapping.File)
					require.Equal(t, wantBuild, loc.Mapping.BuildID)
					if loc.Address == 0x1020 && field == "function" {
						require.Empty(t, loc.Line)
						continue
					}
					require.Len(t, loc.Line, 1)
					require.Equal(t, int64(42), loc.Line[0].Line)
					require.Equal(t, int64(3), loc.Line[0].Column)
					fn := loc.Line[0].Function
					require.NotNil(t, fn)
					wantName, wantSystem, wantSource := "main.work", "main.work", "main.go"
					if loc.Address == 0x1020 {
						switch field {
						case "function_name":
							wantName = ""
						case "function_system_name":
							wantSystem = ""
						case "function_filename":
							wantSource = ""
						}
					}
					require.Equal(t, wantName, fn.Name)
					require.Equal(t, wantSystem, fn.SystemName)
					require.Equal(t, wantSource, fn.Filename)
					require.Equal(t, int64(40), fn.StartLine)
				}
			})
		}
	}
}
