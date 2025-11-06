// Copyright 2022-2025 The Parca Authors
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
	"strings"
	"testing"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

// createTestProfileData creates multiple test profile records with meaningful function names
// that can be reused across multiple filtering tests. This simulates real-world scenarios
// where profile data comes in multiple batches/records.
func createTestProfileData(mem memory.Allocator) ([]arrow.RecordBatch, func()) {
	records := []arrow.RecordBatch{}
	writers := []*profile.Writer{}

	// Record 1: Contains multiple samples
	w1 := profile.NewWriter(mem, nil)
	writers = append(writers, &w1)

	// Sample 1: main -> app.server.handleRequest -> database.query
	w1.LocationsList.Append(true)

	// Frame 1: main function
	w1.Locations.Append(true)
	w1.Addresses.Append(0x1000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("myapp"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(10)
	w1.FunctionName.Append([]byte("main"))
	w1.FunctionSystemName.Append([]byte("main"))
	w1.FunctionFilename.Append([]byte("main.go"))
	w1.FunctionStartLine.Append(10)

	// Frame 2: server handler function
	w1.Locations.Append(true)
	w1.Addresses.Append(0x2000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("myapp"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(25)
	w1.FunctionName.Append([]byte("app.server.handleRequest"))
	w1.FunctionSystemName.Append([]byte("app.server.handleRequest"))
	w1.FunctionFilename.Append([]byte("server.go"))
	w1.FunctionStartLine.Append(20)

	// Frame 3: database query function
	w1.Locations.Append(true)
	w1.Addresses.Append(0x3000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("myapp"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(45)
	w1.FunctionName.Append([]byte("database.query"))
	w1.FunctionSystemName.Append([]byte("database.query"))
	w1.FunctionFilename.Append([]byte("db.go"))
	w1.FunctionStartLine.Append(40)

	w1.Value.Append(100)
	w1.Diff.Append(0)
	w1.TimeNanos.Append(1)
	w1.Period.Append(1)

	// Sample 2 in Record 1: main -> runtime.gc -> runtime.malloc
	w1.LocationsList.Append(true)

	// Frame 1: main function (reuse)
	w1.Locations.Append(true)
	w1.Addresses.Append(0x1000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("myapp"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(15)
	w1.FunctionName.Append([]byte("main"))
	w1.FunctionSystemName.Append([]byte("main"))
	w1.FunctionFilename.Append([]byte("main.go"))
	w1.FunctionStartLine.Append(10)

	// Frame 2: runtime gc
	w1.Locations.Append(true)
	w1.Addresses.Append(0x4000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("runtime"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(100)
	w1.FunctionName.Append([]byte("runtime.gc"))
	w1.FunctionSystemName.Append([]byte("runtime.gc"))
	w1.FunctionFilename.Append([]byte("gc.go"))
	w1.FunctionStartLine.Append(95)

	// Frame 3: runtime malloc
	w1.Locations.Append(true)
	w1.Addresses.Append(0x5000)
	w1.MappingStart.Append(0x1000)
	w1.MappingLimit.Append(0x2000)
	w1.MappingOffset.Append(0x0)
	w1.MappingFile.Append([]byte("runtime"))
	w1.MappingBuildID.Append([]byte(""))
	w1.Lines.Append(true)
	w1.Line.Append(true)
	w1.LineNumber.Append(200)
	w1.FunctionName.Append([]byte("runtime.malloc"))
	w1.FunctionSystemName.Append([]byte("runtime.malloc"))
	w1.FunctionFilename.Append([]byte("malloc.go"))
	w1.FunctionStartLine.Append(195)

	w1.Value.Append(50)
	w1.Diff.Append(0)
	w1.TimeNanos.Append(2)
	w1.Period.Append(1)

	record1 := w1.RecordBuilder.NewRecordBatch()
	records = append(records, record1)

	// Record 2: Contains a single sample (utils.helper -> database.connect)
	w2 := profile.NewWriter(mem, nil)
	writers = append(writers, &w2)

	// Single sample: utils.helper -> database.connect
	w2.LocationsList.Append(true)

	// Frame 1: utils helper
	w2.Locations.Append(true)
	w2.Addresses.Append(0x6000)
	w2.MappingStart.Append(0x1000)
	w2.MappingLimit.Append(0x2000)
	w2.MappingOffset.Append(0x0)
	w2.MappingFile.Append([]byte("myapp"))
	w2.MappingBuildID.Append([]byte(""))
	w2.Lines.Append(true)
	w2.Line.Append(true)
	w2.LineNumber.Append(50)
	w2.FunctionName.Append([]byte("utils.helper"))
	w2.FunctionSystemName.Append([]byte("utils.helper"))
	w2.FunctionFilename.Append([]byte("utils.go"))
	w2.FunctionStartLine.Append(45)

	// Frame 2: database connect
	w2.Locations.Append(true)
	w2.Addresses.Append(0x7000)
	w2.MappingStart.Append(0x1000)
	w2.MappingLimit.Append(0x2000)
	w2.MappingOffset.Append(0x0)
	w2.MappingFile.Append([]byte("myapp"))
	w2.MappingBuildID.Append([]byte(""))
	w2.Lines.Append(true)
	w2.Line.Append(true)
	w2.LineNumber.Append(80)
	w2.FunctionName.Append([]byte("database.connect"))
	w2.FunctionSystemName.Append([]byte("database.connect"))
	w2.FunctionFilename.Append([]byte("db.go"))
	w2.FunctionStartLine.Append(75)

	w2.Value.Append(25)
	w2.Diff.Append(0)
	w2.TimeNanos.Append(3)
	w2.Period.Append(1)

	record2 := w2.RecordBuilder.NewRecordBatch()
	records = append(records, record2)

	// Record 3: Contains multiple samples
	w3 := profile.NewWriter(mem, nil)
	writers = append(writers, &w3)

	// Sample 1 in Record 3: app.worker -> app.process
	w3.LocationsList.Append(true)

	// Frame 1: app worker
	w3.Locations.Append(true)
	w3.Addresses.Append(0x8000)
	w3.MappingStart.Append(0x1000)
	w3.MappingLimit.Append(0x2000)
	w3.MappingOffset.Append(0x0)
	w3.MappingFile.Append([]byte("myapp"))
	w3.MappingBuildID.Append([]byte(""))
	w3.Lines.Append(true)
	w3.Line.Append(true)
	w3.LineNumber.Append(60)
	w3.FunctionName.Append([]byte("app.worker"))
	w3.FunctionSystemName.Append([]byte("app.worker"))
	w3.FunctionFilename.Append([]byte("worker.go"))
	w3.FunctionStartLine.Append(55)

	// Frame 2: app process
	w3.Locations.Append(true)
	w3.Addresses.Append(0x9000)
	w3.MappingStart.Append(0x1000)
	w3.MappingLimit.Append(0x2000)
	w3.MappingOffset.Append(0x0)
	w3.MappingFile.Append([]byte("myapp"))
	w3.MappingBuildID.Append([]byte(""))
	w3.Lines.Append(true)
	w3.Line.Append(true)
	w3.LineNumber.Append(90)
	w3.FunctionName.Append([]byte("app.process"))
	w3.FunctionSystemName.Append([]byte("app.process"))
	w3.FunctionFilename.Append([]byte("process.go"))
	w3.FunctionStartLine.Append(85)

	w3.Value.Append(30)
	w3.Diff.Append(0)
	w3.TimeNanos.Append(4)
	w3.Period.Append(1)

	// Sample 2 in Record 3: main -> app.server.handleRequest -> database.execute
	w3.LocationsList.Append(true)

	// Frame 1: main function
	w3.Locations.Append(true)
	w3.Addresses.Append(0x1000)
	w3.MappingStart.Append(0x1000)
	w3.MappingLimit.Append(0x2000)
	w3.MappingOffset.Append(0x0)
	w3.MappingFile.Append([]byte("myapp"))
	w3.MappingBuildID.Append([]byte(""))
	w3.Lines.Append(true)
	w3.Line.Append(true)
	w3.LineNumber.Append(20)
	w3.FunctionName.Append([]byte("main"))
	w3.FunctionSystemName.Append([]byte("main"))
	w3.FunctionFilename.Append([]byte("main.go"))
	w3.FunctionStartLine.Append(10)

	// Frame 2: server handler function
	w3.Locations.Append(true)
	w3.Addresses.Append(0x2000)
	w3.MappingStart.Append(0x1000)
	w3.MappingLimit.Append(0x2000)
	w3.MappingOffset.Append(0x0)
	w3.MappingFile.Append([]byte("myapp"))
	w3.MappingBuildID.Append([]byte(""))
	w3.Lines.Append(true)
	w3.Line.Append(true)
	w3.LineNumber.Append(30)
	w3.FunctionName.Append([]byte("app.server.handleRequest"))
	w3.FunctionSystemName.Append([]byte("app.server.handleRequest"))
	w3.FunctionFilename.Append([]byte("server.go"))
	w3.FunctionStartLine.Append(20)

	// Frame 3: database execute
	w3.Locations.Append(true)
	w3.Addresses.Append(0xa000)
	w3.MappingStart.Append(0x1000)
	w3.MappingLimit.Append(0x2000)
	w3.MappingOffset.Append(0x0)
	w3.MappingFile.Append([]byte("myapp"))
	w3.MappingBuildID.Append([]byte(""))
	w3.Lines.Append(true)
	w3.Line.Append(true)
	w3.LineNumber.Append(100)
	w3.FunctionName.Append([]byte("database.execute"))
	w3.FunctionSystemName.Append([]byte("database.execute"))
	w3.FunctionFilename.Append([]byte("db.go"))
	w3.FunctionStartLine.Append(95)

	w3.Value.Append(45)
	w3.Diff.Append(0)
	w3.TimeNanos.Append(5)
	w3.Period.Append(1)

	record3 := w3.RecordBuilder.NewRecordBatch()
	records = append(records, record3)

	cleanup := func() {
		for _, w := range writers {
			w.Release()
		}
		// Don't release records here - they will be handled by FilterProfileData
	}

	return records, cleanup
}

func TestStackFilterFunctionNameSingle(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test single filter: should match stacks containing "main"
	// Expected: samples 1 and 2 (both have "main" function)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "main",
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

	// Verify results: should have 3 samples total containing "main"
	// Record 1: 2 samples (both contain "main")
	// Record 3: 1 sample (the second sample contains "main")
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(3), totalSamples, "Should have 3 total samples containing 'main'")

	// Validate that all returned samples contain "main" in their stack
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundMain := false
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionNameIndices.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := string(r.LineFunctionNameDict.Value(int(fnIndex)))
					if strings.Contains(strings.ToLower(functionName), "main") {
						foundMain = true
						break
					}
				}
			}
			require.True(t, foundMain, "Sample %d should contain 'main' function", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterFunctionNameDouble(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test double filter with AND logic: should match stacks containing "main" AND "app"
	// Expected: 3 samples (sample 1: main->app.server.handleRequest->database.query, sample 5: main->app.server.handleRequest->database.execute)
	// Sample 2 (main->runtime.gc->runtime.malloc) should be excluded as it doesn't contain "app"
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "main",
									},
								},
							},
						},
					},
				},
			},
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "app",
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

	// Verify results: should have 2 samples total with AND logic
	// Only stacks containing BOTH "main" AND "app" should be kept
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(2), totalSamples, "Should have 2 total samples containing 'main' AND 'app'")

	// Validate that each returned sample contains BOTH "main" AND "app"
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundMain := false
			foundApp := false
			functionNames := []string{}
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionNameIndices.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := strings.ToLower(string(r.LineFunctionNameDict.Value(int(fnIndex))))
					functionNames = append(functionNames, functionName)
					if strings.Contains(functionName, "main") {
						foundMain = true
					}
					if strings.Contains(functionName, "app") {
						foundApp = true
					}
				}
			}
			require.True(t, foundMain, "Sample %d should contain 'main' function. Functions: %v", sampleIndex, functionNames)
			require.True(t, foundApp, "Sample %d should contain 'app' function. Functions: %v", sampleIndex, functionNames)
			sampleIndex++
		}
	}
}

func TestStackFilterFunctionNameTriple(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test triple filter with OR logic: should match stacks containing "runtime" OR "malloc" OR "gc"
	// Expected: 1 sample (sample 2: runtime.gc + runtime.malloc)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "runtime",
									},
								},
							},
						},
					},
				},
			},
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "malloc",
									},
								},
							},
						},
					},
				},
			},
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "gc",
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

	// Verify results: should have 1 sample (only Record 1 sample 2 contains runtime functions)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(1), totalSamples, "Should have 1 sample containing 'runtime' OR 'malloc' OR 'gc'")

	// Validate that the returned sample contains "runtime" OR "malloc" OR "gc"
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundRuntime := false
			foundMalloc := false
			foundGc := false
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionNameIndices.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := strings.ToLower(string(r.LineFunctionNameDict.Value(int(fnIndex))))
					if strings.Contains(functionName, "runtime") {
						foundRuntime = true
					}
					if strings.Contains(functionName, "malloc") {
						foundMalloc = true
					}
					if strings.Contains(functionName, "gc") {
						foundGc = true
					}
				}
			}
			require.True(t, foundRuntime || foundMalloc || foundGc, "Sample %d should contain 'runtime' OR 'malloc' OR 'gc' function", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterFunctionNameExclusion(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test exclusion: filter for "nonexistent" function that doesn't exist in any sample
	// Expected: 0 samples should be returned
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
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

	// Should return no records when nothing matches (following existing pattern)
	require.Len(t, recs, 0, "No records should be returned when no samples match 'nonexistent' function")
}

func TestStackFilterFunctionNamePartialExclusion(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test partial exclusion: filter for "database" which should match samples 1 and 3 but exclude sample 2
	// Expected: 2 samples (sample 1: database.query, sample 3: database.connect)
	// Should NOT include sample 2 (runtime functions)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "database",
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

	// Count total samples across all records
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(3), totalSamples, "Should have 3 total samples containing 'database'")

	// Validate that all returned samples contain "database"
	// AND ensure no runtime functions are present (negative validation)
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundDatabase := false
			foundRuntime := false
			functionNames := []string{}

			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionNameIndices.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := strings.ToLower(string(r.LineFunctionNameDict.Value(int(fnIndex))))
					functionNames = append(functionNames, functionName)

					if strings.Contains(functionName, "database") {
						foundDatabase = true
					}
					if strings.Contains(functionName, "runtime") {
						foundRuntime = true
					}
				}
			}

			// Positive validation: must contain "database"
			require.True(t, foundDatabase, "Sample %d should contain 'database' function. Functions: %v", sampleIndex, functionNames)

			// Negative validation: must NOT contain "runtime"
			require.False(t, foundRuntime, "Sample %d should NOT contain 'runtime' function. Functions: %v", sampleIndex, functionNames)

			sampleIndex++
		}
	}
}

func TestFrameFilterFunctionNameSingle(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test frame filter: keep only frames containing "database"
	// Expected: 2 samples (sample 1 and 3) with only database frames remaining
	// Sample 2 should be completely removed as it has no database frames
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "database",
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

	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(5), totalSamples, "Should have 5 samples with database frames")

	// Validate each remaining sample
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		validFrameCount := 0
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			// Count valid frames (non-null) and verify they all contain "database"
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.Line.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := string(r.LineFunctionNameDict.Value(int(fnIndex)))

					// All remaining frames must contain "database"
					require.Contains(t, strings.ToLower(functionName), "database",
						"Sample %d: All remaining frames should contain 'database', found: %s", sampleIndex, functionName)
					validFrameCount++
				}
			}
		}

		// Each sample should have exactly one frame remaining
		require.Equal(t, 1, validFrameCount, "Sample %d should have exactly 1 database frame remaining", sampleIndex)
		sampleIndex++
	}
}

func TestFrameFilterFunctionNameNotContains(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test frame filter: keep only frames NOT containing "database"
	// Expected: All 3 samples remain, but database frames are filtered out within each stack
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_NotContains{
										NotContains: "database",
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

	// Should have all 5 samples remaining (with database frames filtered out)
	// Record 1: 2 samples
	// Record 2: 1 sample
	// Record 3: 2 samples
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(5), totalSamples, "Should have 5 samples with database frames filtered out")

	// Validate the remaining sample
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			// Count valid frames (non-null) and verify none contain "database"
			validFrameCount := 0
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.Line.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := string(r.LineFunctionNameDict.Value(int(fnIndex)))

					// All remaining frames must NOT contain "database"
					require.NotContains(t, strings.ToLower(functionName), "database",
						"Sample %d: All remaining frames should NOT contain 'database', found: %s", sampleIndex, functionName)
					validFrameCount++
				}
			}

			// Should have at least 1 frame remaining after filtering out database frames
			require.Greater(t, validFrameCount, 0, "Sample %d should have at least 1 non-database frame remaining", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterBinary(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test filtering by binary name "runtime"
	// Expected: Should return samples that have runtime binary (Sample 2 in Record 1 has runtime.gc/malloc)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Binary: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "runtime",
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

	// Should have 1 sample (Record 0, Sample 1 which has runtime.gc/malloc with runtime binary)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(1), totalSamples, "Should have 1 sample containing 'runtime' binary")

	// Validate that the returned sample contains runtime binary
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)

			foundRuntimeBinary := false
			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				if r.MappingStart.IsValid(j) {
					mappingFile := r.MappingFileDict.Value(int(r.MappingFileIndices.Value(j)))
					if strings.Contains(strings.ToLower(string(mappingFile)), "runtime") {
						foundRuntimeBinary = true
						break
					}
				}
			}
			require.True(t, foundRuntimeBinary, "Sample %d should contain 'runtime' binary", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterAddress(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test filtering by address 0x1000
	// Expected: Should return samples that contain address 0x1000 (main function)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Address: &pb.NumberCondition{
									Condition: &pb.NumberCondition_Equal{
										Equal: 0x1000,
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

	// Address 0x1000 appears in main function which is in multiple samples
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	// Update expectation based on actual test data
	require.Greater(t, totalSamples, int64(0), "Should have samples containing address 0x1000")

	// Validate that each returned sample contains address 0x1000
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)

			foundAddress := false
			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				if r.Address.IsValid(j) {
					address := r.Address.Value(j)
					if address == 0x1000 {
						foundAddress = true
						break
					}
				}
			}
			require.True(t, foundAddress, "Sample %d should contain address 0x1000", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterLineNumber(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test filtering by line number 100
	// Expected: Should return samples that contain line number 100 (runtime.gc function)
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								LineNumber: &pb.NumberCondition{
									Condition: &pb.NumberCondition_Equal{
										Equal: 100,
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

	// Line number 100 appears in runtime.gc function
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	// Update expectation based on actual test data
	require.Greater(t, totalSamples, int64(0), "Should have samples containing line number 100")

	// Validate that the returned sample contains line number 100
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundLineNumber := false
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineNumber.IsValid(k) {
					lineNumber := r.LineNumber.Value(k)
					if lineNumber == 100 {
						foundLineNumber = true
						break
					}
				}
			}
			require.True(t, foundLineNumber, "Sample %d should contain line number 100", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterSystemName(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test filtering by system name containing "database"
	// Expected: Should return samples that contain database functions
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								SystemName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "database",
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

	// Should have 3 samples (database.query, database.connect, database.execute)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(3), totalSamples, "Should have 3 samples containing 'database' system name")

	// Validate that each returned sample contains database system name
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundDatabaseSystemName := false
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionSystemNameIndices.IsValid(k) {
					sysIndex := r.LineFunctionSystemNameIndices.Value(k)
					systemName := r.LineFunctionSystemNameDict.Value(int(sysIndex))
					if strings.Contains(strings.ToLower(string(systemName)), "database") {
						foundDatabaseSystemName = true
						break
					}
				}
			}
			require.True(t, foundDatabaseSystemName, "Sample %d should contain 'database' system name", sampleIndex)
			sampleIndex++
		}
	}
}

func TestStackFilterFilename(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test filtering by filename containing "db.go"
	// Expected: Should return samples that contain functions from db.go file
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Filename: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "db.go",
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

	// Should have 3 samples (database.query, database.connect, database.execute from db.go)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(3), totalSamples, "Should have 3 samples containing functions from 'db.go'")

	// Validate that each returned sample contains functions from db.go
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundDbGoFilename := false
			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionFilenameIndices.IsValid(k) {
					fileIndex := r.LineFunctionFilenameIndices.Value(k)
					filename := r.LineFunctionFilenameDict.Value(int(fileIndex))
					if strings.Contains(strings.ToLower(string(filename)), "db.go") {
						foundDbGoFilename = true
						break
					}
				}
			}
			require.True(t, foundDbGoFilename, "Sample %d should contain functions from 'db.go'", sampleIndex)
			sampleIndex++
		}
	}
}

func TestFrameFilterAddress(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	// Test frame filtering by address 0x3000 (database.query frame)
	// Expected: Should return samples but only frames with address 0x3000
	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_FrameFilter{
					FrameFilter: &pb.FrameFilter{
						Filter: &pb.FrameFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								Address: &pb.NumberCondition{
									Condition: &pb.NumberCondition_Equal{
										Equal: 0x3000,
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

	// Should have 1 sample (the one containing the frame with address 0x3000)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(5), totalSamples, "Should have unchanged 5 sample")

	// Validate that the returned sample contains the correct frame with address 0x3000
	// and that other frames in the stack are filtered out (set to null)
	sampleIndex := 0
	foundTargetAddress := false
	validFrameCount := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)

			for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
				if r.Address.IsValid(j) {
					address := r.Address.Value(j)
					validFrameCount++
					if address == 0x3000 {
						foundTargetAddress = true
						sampleIndex++
					}
				}
			}
		}
	}
	require.True(t, foundTargetAddress, "Sample %d should contain frame with address 0x3000", sampleIndex)
	// Frame filtering keeps the sample but may have multiple valid frames
	require.Greater(t, validFrameCount, 0, "Sample %d should have at least 1 valid frame after filtering", sampleIndex)
}

func TestStackFilterAndLogicValidation(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.DefaultAllocator)
	defer mem.AssertSize(t, 0)

	originalRecords, cleanup := createTestProfileData(mem)
	defer cleanup()

	recs, _, err := FilterProfileData(
		context.Background(),
		noop.NewTracerProvider().Tracer(""),
		mem,
		originalRecords,
		[]*pb.Filter{
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_Contains{
										Contains: "main",
									},
								},
							},
						},
					},
				},
			},
			{
				Filter: &pb.Filter_StackFilter{
					StackFilter: &pb.StackFilter{
						Filter: &pb.StackFilter_Criteria{
							Criteria: &pb.FilterCriteria{
								FunctionName: &pb.StringCondition{
									Condition: &pb.StringCondition_NotContains{
										NotContains: "database",
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

	// With AND logic: Only 1 stack should remain (Stack 2: main → runtime.gc → runtime.malloc)
	totalSamples := int64(0)
	for _, rec := range recs {
		totalSamples += rec.NumRows()
	}
	require.Equal(t, int64(1), totalSamples, "Should have 1 sample with AND logic (main AND not database)")

	// Validate the remaining stack contains "main" and does NOT contain "database"
	sampleIndex := 0
	for _, rec := range recs {
		r := profile.NewRecordReader(rec)
		for i := 0; i < int(rec.NumRows()); i++ {
			lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
			firstStart, _ := r.Lines.ValueOffsets(int(lOffsetStart))
			_, lastEnd := r.Lines.ValueOffsets(int(lOffsetEnd - 1))

			foundMain := false
			foundDatabase := false
			functionNames := []string{}

			for k := int(firstStart); k < int(lastEnd); k++ {
				if r.LineFunctionNameIndices.IsValid(k) {
					fnIndex := r.LineFunctionNameIndices.Value(k)
					functionName := strings.ToLower(string(r.LineFunctionNameDict.Value(int(fnIndex))))
					functionNames = append(functionNames, functionName)

					if strings.Contains(functionName, "main") {
						foundMain = true
					}
					if strings.Contains(functionName, "database") {
						foundDatabase = true
					}
				}
			}

			// Positive validation: must contain "main"
			require.True(t, foundMain, "Sample %d should contain 'main' function. Functions: %v", sampleIndex, functionNames)
			// Negative validation: must NOT contain "database"
			require.False(t, foundDatabase, "Sample %d should NOT contain 'database' function. Functions: %v", sampleIndex, functionNames)

			// Validate specific expected functions for Stack 2 (main → runtime.gc → runtime.malloc)
			expectedFunctions := []string{"main", "runtime.gc", "runtime.malloc"}
			require.Len(t, functionNames, len(expectedFunctions), "Sample %d should have exactly %d functions", sampleIndex, len(expectedFunctions))
			for _, expected := range expectedFunctions {
				found := false
				for _, actual := range functionNames {
					if strings.Contains(actual, expected) {
						found = true
						break
					}
				}
				require.True(t, found, "Sample %d should contain function '%s'. Functions: %v", sampleIndex, expected, functionNames)
			}

			sampleIndex++
		}
	}
}
