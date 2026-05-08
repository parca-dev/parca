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
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	pprofprofile "github.com/google/pprof/profile"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/kv"
	"github.com/parca-dev/parca/pkg/parcacol"
	"github.com/parca-dev/parca/pkg/profile"
)

type Testing interface {
	require.TestingT
	Helper()
}

func MustReadAllGzip(t Testing, filename string) []byte {
	t.Helper()

	f, err := os.Open(filename)
	require.NoError(t, err)
	defer f.Close()

	r, err := gzip.NewReader(f)
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}

func MustDecompressGzip(t Testing, b []byte) []byte {
	t.Helper()

	r, err := gzip.NewReader(bytes.NewReader(b))
	require.NoError(t, err)
	content, err := io.ReadAll(r)
	require.NoError(t, err)
	return content
}


func BenchmarkQuery(b *testing.B) {
	ctx := context.Background()
	tracer := noop.NewTracerProvider().Tracer("")

	fileContent, err := os.ReadFile("testdata/alloc_objects.pb.gz")
	require.NoError(b, err)

	p, err := pprofprofile.ParseData(fileContent)
	require.NoError(b, err)

	sp, err := PprofToSymbolizedProfile(profile.Meta{}, p, 0, []string{})
	require.NoError(b, err)

	b.ReportAllocs()
	b.ResetTimer()

	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(b, 0)
	for i := 0; i < b.N; i++ {
		_, _ = RenderReport(
			ctx,
			tracer,
			sp,
			pb.QueryRequest_REPORT_TYPE_FLAMEGRAPH_ARROW,
			0,
			0,
			[]string{FlamegraphFieldFunctionName},
			NewTableConverterPool(),
			mem,
			parcacol.NewArrowToProfileConverter(tracer, kv.NewKeyMaker()),
			nil,
			"",
			false,
		)
	}
}

func PprofToSymbolizedProfile(meta profile.Meta, prof *pprofprofile.Profile, index int, groupBy []string) (profile.Profile, error) {
	labelNameSet := make(map[string]struct{})
	for _, s := range prof.Sample {
		for k := range s.Label {
			labelNameSet[k] = struct{}{}
		}
	}
	labelNames := make([]string, 0, len(labelNameSet))
	for l := range labelNameSet {
		labelNames = append(labelNames, l)
	}

	groupBySet := make(map[string]struct{}, len(groupBy))
	for _, g := range groupBy {
		groupBySet[g] = struct{}{}
	}

	w := profile.NewWriter(memory.DefaultAllocator, labelNames)
	defer w.RecordBuilder.Release()
	for i := range prof.Sample {
		if len(prof.Sample[i].Value) <= index {
			return profile.Profile{}, status.Errorf(codes.InvalidArgument, "failed to find samples for profile type")
		}

		w.Value.Append(prof.Sample[i].Value[index])
		w.Diff.Append(0)
		w.TimeNanos.Append(prof.TimeNanos)
		w.Period.Append(prof.Period)

		for labelName, labelBuilder := range w.LabelBuildersMap {
			if prof.Sample[i].Label == nil {
				labelBuilder.AppendNull()
				continue
			}

			if labelValues, ok := prof.Sample[i].Label[labelName]; ok && len(labelValues) > 0 {
				labelBuilder.Append([]byte(labelValues[0]))
			} else {
				labelBuilder.AppendNull()
			}
		}

		w.LocationsList.Append(len(prof.Sample[i].Location) > 0)
		if len(prof.Sample[i].Location) > 0 {
			for _, loc := range prof.Sample[i].Location {
				w.Locations.Append(true)
				w.Addresses.Append(loc.Address)

				if loc.Mapping != nil {
					w.MappingStart.Append(loc.Mapping.Start)
					w.MappingLimit.Append(loc.Mapping.Limit)
					w.MappingOffset.Append(loc.Mapping.Offset)
					w.MappingFile.Append([]byte(loc.Mapping.File))
					w.MappingBuildID.Append([]byte(loc.Mapping.BuildID))
				} else {
					w.MappingStart.AppendNull()
					w.MappingLimit.AppendNull()
					w.MappingOffset.AppendNull()
					w.MappingFile.AppendNull()
					w.MappingBuildID.AppendNull()
				}

				w.Lines.Append(len(loc.Line) > 0)
				if len(loc.Line) > 0 {
					for _, line := range loc.Line {
						w.Line.Append(true)
						w.LineNumber.Append(line.Line)
						w.ColumnNumber.Append(uint64(line.Column))
						if line.Function != nil {
							w.FunctionName.Append([]byte(line.Function.Name))
							w.FunctionSystemName.Append([]byte(line.Function.SystemName))
							w.FunctionFilename.Append([]byte(line.Function.Filename))
							w.FunctionStartLine.Append(line.Function.StartLine)
						} else {
							w.FunctionName.AppendNull()
							w.FunctionSystemName.AppendNull()
							w.FunctionFilename.AppendNull()
							w.FunctionStartLine.AppendNull()
						}
					}
				}
			}
		}
	}

	return profile.Profile{
		Meta:    meta,
		Samples: []arrow.RecordBatch{w.RecordBuilder.NewRecordBatch()},
	}, nil
}

func TestFilterData(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test"))
	w.FunctionStartLine.Append(1)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "test",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 2, valid)
	require.Equal(t, "test", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(0)))))
	require.Equal(t, "test1", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(1)))))
}

func TestFilterUnsymbolized(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(false)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "test",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, recs, 1)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 1, valid)
}

func TestFilterDataWithPath(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libc.so.6"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("__libc_start_main"))
	w.FunctionSystemName.Append([]byte("__libc_start_main"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("/usr/lib/libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test1"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(0)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("interpreter"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test.py"))
	w.FunctionStartLine.Append(0)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "libpython3.11.so.1.0",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	validIndexes := []uint32{}
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			start, end := r.Lines.ValueOffsets(i)
			for j := int(start); j < int(end); j++ {
				if r.Line.IsValid(j) {
					validIndexes = append(validIndexes, r.LineFunctionNameIndices.Value(j))
				}
			}
		}
	}
	require.Equal(t, 1, len(validIndexes))
	require.Equal(t, "test1", string(r.LineFunctionNameDict.Value(int(validIndexes[0]))))
}

func TestFilterDataFrameFilter(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("libc.so.6"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("__libc_start_main"))
	w.FunctionSystemName.Append([]byte("__libc_start_main"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("/usr/lib/libpython3.11.so.1.0"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test1"))
	w.FunctionSystemName.Append([]byte("test1"))
	w.FunctionFilename.Append([]byte(""))
	w.FunctionStartLine.Append(0)

	w.Locations.Append(true)
	w.Addresses.Append(0x1234)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("interpreter"))
	w.MappingBuildID.Append([]byte(""))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(0)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("test"))
	w.FunctionSystemName.Append([]byte("test"))
	w.FunctionFilename.Append([]byte("test.py"))
	w.FunctionStartLine.Append(0)
	w.Value.Append(1)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		[]arrow.RecordBatch{originalRecord},
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "interpreter",
									},
								},
							},
						},
					},
				},
			},
		},
	)
	require.NoError(t, err)
	defer func() {
		for _, r := range recs {
			r.Release()
		}
	}()
	r, err := profile.NewRecordReader(recs[0])
	require.NoError(t, err)
	valid := 0
	for i := 0; i < r.Location.Len(); i++ {
		if r.Location.IsValid(i) {
			valid++
		}
	}
	require.Equal(t, 1, valid)
	require.Equal(t, "test", string(r.LineFunctionNameDict.Value(int(r.LineFunctionNameIndices.Value(2)))))
}

func BenchmarkFilterData(t *testing.B) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	for i := 0; i < 10000; i++ {
		w.LocationsList.Append(true)
		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("test"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.ColumnNumber.Append(0)
		w.FunctionName.Append([]byte("test"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)

		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("libpython3.11.so.1.0"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.ColumnNumber.Append(0)
		w.FunctionName.Append([]byte("test1"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)

		w.Locations.Append(true)
		w.Addresses.Append(0x1234)
		w.MappingStart.Append(0x1000)
		w.MappingLimit.Append(0x2000)
		w.MappingOffset.Append(0x0)
		w.MappingFile.Append([]byte("test"))
		w.MappingBuildID.Append([]byte("test"))
		w.Lines.Append(true)
		w.Line.Append(true)
		w.LineNumber.Append(1)
		w.ColumnNumber.Append(0)
		w.FunctionName.Append([]byte("test1"))
		w.FunctionSystemName.Append([]byte("test"))
		w.FunctionFilename.Append([]byte("test"))
		w.FunctionStartLine.Append(1)
		w.Value.Append(1)
		w.Diff.Append(0)
		w.TimeNanos.Append(1)
		w.Period.Append(1)
	}

	originalRecord := w.RecordBuilder.NewRecordBatch()
	defer originalRecord.Release()
	for i := 0; i < t.N; i++ {
		originalRecord.Retain() // retain each time since FilterProfileData will release it
		recs, _, err := FilterProfileData(
			context.Background(),
			noop.NewTracerProvider().Tracer(""),
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_FrameFilter{
						FrameFilter: &pb.FrameFilter{
							Filter: &pb.FrameFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									Binary: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "test",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		for _, r := range recs {
			r.Release()
		}
	}
}

func TestFilterDataExclude(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	tracer := noop.NewTracerProvider().Tracer("")
	ctx := context.Background()

	// Create a profile with 3 samples:
	// Sample 1: function "foo" -> "bar" -> "baz"
	// Sample 2: function "main" -> "process" -> "handle"
	// Sample 3: function "foo" -> "qux"
	w := profile.NewWriter(mem, nil)
	defer w.Release()

	// Sample 1: has "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x1000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(1)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("foo"))
	w.FunctionSystemName.Append([]byte("foo"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x1100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(2)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("bar"))
	w.FunctionSystemName.Append([]byte("bar"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(10)

	w.Locations.Append(true)
	w.Addresses.Append(0x1200)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x2000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(3)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("baz"))
	w.FunctionSystemName.Append([]byte("baz"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(20)
	w.Value.Append(100)
	w.Diff.Append(0)
	w.TimeNanos.Append(1)
	w.Period.Append(1)

	// Sample 2: no "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x2000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(4)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("main"))
	w.FunctionSystemName.Append([]byte("main"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x2100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(5)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("process"))
	w.FunctionSystemName.Append([]byte("process"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(10)

	w.Locations.Append(true)
	w.Addresses.Append(0x2200)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x3000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(6)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("handle"))
	w.FunctionSystemName.Append([]byte("handle"))
	w.FunctionFilename.Append([]byte("main.go"))
	w.FunctionStartLine.Append(20)
	w.Value.Append(200)
	w.Diff.Append(0)
	w.TimeNanos.Append(2)
	w.Period.Append(1)

	// Sample 3: has "foo"
	w.LocationsList.Append(true)
	w.Locations.Append(true)
	w.Addresses.Append(0x3000)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x4000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(7)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("foo"))
	w.FunctionSystemName.Append([]byte("foo"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(1)

	w.Locations.Append(true)
	w.Addresses.Append(0x3100)
	w.MappingStart.Append(0x1000)
	w.MappingLimit.Append(0x4000)
	w.MappingOffset.Append(0x0)
	w.MappingFile.Append([]byte("test"))
	w.MappingBuildID.Append([]byte("test"))
	w.Lines.Append(true)
	w.Line.Append(true)
	w.LineNumber.Append(8)
	w.ColumnNumber.Append(0)
	w.FunctionName.Append([]byte("qux"))
	w.FunctionSystemName.Append([]byte("qux"))
	w.FunctionFilename.Append([]byte("test.go"))
	w.FunctionStartLine.Append(30)
	w.Value.Append(300)
	w.Diff.Append(0)
	w.TimeNanos.Append(3)
	w.Period.Append(1)

	originalRecord := w.RecordBuilder.NewRecordBatch()
	defer originalRecord.Release()

	t.Run("exclude=false filters to only samples with foo", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "foo",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should have 2 samples (sample 1 and 3 which have "foo")
		// The filtered value is the sum of values that were REMOVED, not kept
		totalRows := int64(0)
		totalValue := int64(0)
		for _, rec := range recs {
			totalRows += rec.NumRows()
			r, err := profile.NewRecordReader(rec)
			require.NoError(t, err)
			totalValue += math.Int64.Sum(r.Value)
		}
		require.Equal(t, int64(2), totalRows)
		require.Equal(t, int64(400), totalValue) // kept: 100 + 300
		require.Equal(t, int64(200), filtered)   // removed: 200 (sample 2)
	})

	t.Run("exclude=true filters out samples with foo", func(t *testing.T) {
		originalRecord.Retain()
		// Note: The new API doesn't support exclude functionality directly
		// This test now tests include behavior for non-foo functions
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_NotContains{
											NotContains: "foo",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should have 1 sample (sample 2 which doesn't have "foo")
		require.Len(t, recs, 1)
		require.Equal(t, int64(1), recs[0].NumRows())
		// The filtered value is the sum of values that were REMOVED
		require.Equal(t, int64(400), filtered) // removed: 100 + 300 (samples with foo)
	})

	t.Run("empty filter with exclude=true returns all samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{}, // no filters
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return all samples
		require.Greater(t, len(recs), 0, "Expected at least one record")
		if len(recs) > 0 {
			require.Equal(t, int64(3), recs[0].NumRows())
			// The filtered value is the sum of values that were REMOVED
			require.Equal(t, int64(0), filtered) // nothing removed with empty filter
		}
	})

	t.Run("function not found with exclude=true returns all samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, filtered, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{}, // no filters
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return all samples (nothing to exclude)
		totalRows := int64(0)
		for _, rec := range recs {
			totalRows += rec.NumRows()
		}
		require.Equal(t, int64(3), totalRows)
		// The filtered value is the sum of values that were REMOVED
		require.Equal(t, int64(0), filtered) // nothing removed
	})

	t.Run("function not found with exclude=false returns no samples", func(t *testing.T) {
		originalRecord.Retain()
		recs, _, err := FilterProfileData(
			ctx,
			tracer,
			mem,
			[]arrow.RecordBatch{originalRecord},
			[]*pb.Filter{
				{
					Filter: &pb.Filter_StackFilter{
						StackFilter: &pb.StackFilter{
							Filter: &pb.StackFilter_Criteria{
								Criteria: &pb.FilterCriteria{
									FunctionName: &pb.StringCondition{
										Condition: &pb.StringCondition_Contains{
											Contains: "nonexistent",
										},
									},
								},
							},
						},
					},
				},
			},
		)
		require.NoError(t, err)
		defer func() {
			for _, r := range recs {
				r.Release()
			}
		}()

		// Should return no samples (nothing matches)
		require.Len(t, recs, 0)
	})
}

func TestKwayMerge(t *testing.T) {
	arr1 := []string{"a", "c", "e"}
	arr2 := []string{"f", "i", "m", "o", "r"}

	merged := MergeTwoSortedSlices(arr1, arr2)

	require.Equal(t, []string{"a", "c", "e", "f", "i", "m", "o", "r"}, merged)
}

func TestSetArrayElementToNull(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	tests := []struct {
		name         string
		buildArray   func() arrow.Array
		indexToNull  int
		expectedNull int
	}{
		{
			name: "array with existing null bitmap",
			buildArray: func() arrow.Array {
				b := array.NewInt64Builder(mem)
				defer b.Release()
				b.AppendValues([]int64{1, 2, 3}, []bool{true, false, true})
				return b.NewArray()
			},
			indexToNull:  0,
			expectedNull: 2,
		},
		{
			name: "array with no null bitmap",
			buildArray: func() arrow.Array {
				b := array.NewInt64Builder(mem)
				defer b.Release()
				b.AppendValues([]int64{1, 2, 3}, nil)
				return b.NewArray()
			},
			indexToNull:  1,
			expectedNull: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			arr := tc.buildArray()
			defer arr.Release()

			setArrayElementToNull(arr, tc.indexToNull, mem)

			require.True(t, arr.IsNull(tc.indexToNull))
			require.Equal(t, tc.expectedNull, arr.NullN())
		})
	}
}
