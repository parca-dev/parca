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

package query

import (
	"bytes"
	"context"
	"fmt"
	"slices"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/apache/arrow-go/v18/arrow/array"
	"github.com/apache/arrow-go/v18/arrow/ipc"
	"github.com/apache/arrow-go/v18/arrow/math"
	"github.com/apache/arrow-go/v18/arrow/memory"
	"go.opentelemetry.io/otel/trace"

	pb "github.com/parca-dev/parca/gen/proto/go/parca/query/v1alpha1"
	"github.com/parca-dev/parca/pkg/profile"
)

func GenerateSourceReport(
	ctx context.Context,
	pool memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
	ref *pb.SourceReference,
	source string,
) (*pb.Source, int64, error) {
	record, cumulative, matchedFilenames, err := generateSourceReportRecord(
		ctx,
		pool,
		tracer,
		p,
		ref,
		source,
	)
	if err != nil {
		return nil, 0, err
	}
	defer record.Release()

	var buf bytes.Buffer
	w := ipc.NewWriter(&buf,
		ipc.WithSchema(record.Schema()),
		ipc.WithAllocator(pool),
	)
	defer w.Close()

	if err := w.Write(record); err != nil {
		return nil, 0, err
	}

	return &pb.Source{
		Record:           buf.Bytes(),
		Source:           source,
		Unit:             p.Meta.SampleType.Unit,
		MatchedFilenames: matchedFilenames,
	}, cumulative, nil
}

func generateSourceReportRecord(
	_ context.Context,
	pool memory.Allocator,
	_ trace.Tracer,
	p profile.Profile,
	ref *pb.SourceReference,
	_ string,
) (arrow.RecordBatch, int64, []string, error) {
	b := newSourceReportBuilder(pool, ref)
	for _, record := range p.Samples {
		if err := b.addRecord(record); err != nil {
			return nil, 0, nil, err
		}
	}

	rec, cumulative := b.finish()

	// Extract matched filenames from the map
	matchedFilenames := make([]string, 0, len(b.matchedFilenames))
	for filename := range b.matchedFilenames {
		matchedFilenames = append(matchedFilenames, filename)
	}
	slices.Sort(matchedFilenames)

	return rec, cumulative, matchedFilenames, nil
}

type lineMetrics struct {
	cumulative int64
	flat       int64
}

type sourceReportBuilder struct {
	pool memory.Allocator

	filename []byte
	buildID  []byte

	lineData         map[int64]*lineMetrics
	matchedFilenames map[string]struct{}
	cumulative       int64
}

// filenameMatches checks if profileFilename matches queryFilename using suffix matching.
// It returns true if:
// - They are exactly equal, OR
// - profileFilename ends with queryFilename AND is preceded by a '/' (path boundary).
func filenameMatches(profileFilename, queryFilename []byte) bool {
	if bytes.Equal(profileFilename, queryFilename) {
		return true
	}
	if len(queryFilename) > 0 && bytes.HasSuffix(profileFilename, queryFilename) {
		idx := len(profileFilename) - len(queryFilename) - 1
		if idx >= 0 && profileFilename[idx] == '/' {
			return true
		}
	}
	return false
}

func newSourceReportBuilder(
	pool memory.Allocator,
	ref *pb.SourceReference,
) *sourceReportBuilder {
	return &sourceReportBuilder{
		pool:             pool,
		filename:         []byte(ref.Filename),
		buildID:          []byte(ref.BuildId),
		lineData:         make(map[int64]*lineMetrics),
		matchedFilenames: make(map[string]struct{}),
	}
}

func (b *sourceReportBuilder) finish() (arrow.RecordBatch, int64) {
	lineNumbers := make([]int64, 0, len(b.lineData))
	for ln := range b.lineData {
		lineNumbers = append(lineNumbers, ln)
	}
	slices.Sort(lineNumbers)

	lineNumBuilder := array.NewInt64Builder(b.pool)
	defer lineNumBuilder.Release()
	cumuBuilder := array.NewInt64Builder(b.pool)
	defer cumuBuilder.Release()
	flatBuilder := array.NewInt64Builder(b.pool)
	defer flatBuilder.Release()

	lineNumBuilder.Reserve(len(lineNumbers))
	cumuBuilder.Reserve(len(lineNumbers))
	flatBuilder.Reserve(len(lineNumbers))

	for _, ln := range lineNumbers {
		metrics := b.lineData[ln]
		lineNumBuilder.Append(ln)
		cumuBuilder.Append(metrics.cumulative)
		flatBuilder.Append(metrics.flat)
	}

	lineNumArr := lineNumBuilder.NewInt64Array()
	defer lineNumArr.Release()
	cumuArr := cumuBuilder.NewInt64Array()
	defer cumuArr.Release()
	flatArr := flatBuilder.NewInt64Array()
	defer flatArr.Release()

	return array.NewRecordBatch(
		arrow.NewSchema(
			[]arrow.Field{
				{Name: "line_number", Type: arrow.PrimitiveTypes.Int64},
				{Name: "cumulative", Type: arrow.PrimitiveTypes.Int64},
				{Name: "flat", Type: arrow.PrimitiveTypes.Int64},
			},
			nil,
		),
		[]arrow.Array{
			lineNumArr,
			cumuArr,
			flatArr,
		},
		int64(len(lineNumbers)),
	), b.cumulative
}

func (b *sourceReportBuilder) addRecord(rec arrow.RecordBatch) error {
	r, err := profile.NewRecordReader(rec)
	if err != nil {
		return fmt.Errorf("failed to create record reader: %w", err)
	}
	b.cumulative += math.Int64.Sum(r.Value)

	for i := 0; i < int(rec.NumRows()); i++ {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			if !r.Locations.ListValues().IsValid(j) {
				continue // Skip null locations; they have been filtered out
			}
			buildIDMatches := len(b.buildID) == 0 || bytes.Equal(r.MappingBuildIDDict.Value(int(r.MappingBuildIDIndices.Value(j))), b.buildID)
			if r.MappingStart.IsValid(j) && buildIDMatches {
				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)

				for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
					if r.Line.IsValid(k) && r.LineFunctionNameIndices.IsValid(k) {
						profileFilename := r.LineFunctionFilenameDict.Value(int(r.LineFunctionFilenameIndices.Value(k)))
						if filenameMatches(profileFilename, b.filename) {
							b.matchedFilenames[string(profileFilename)] = struct{}{}

							lineNum := r.LineNumber.Value(k)
							metrics, exists := b.lineData[lineNum]
							if !exists {
								metrics = &lineMetrics{}
								b.lineData[lineNum] = metrics
							}

							metrics.cumulative += r.Value.Value(i)

							isLeaf := isFirstNonNil(i, j, r.Locations) && isFirstNonNil(j, k, r.Lines)
							if isLeaf {
								metrics.flat += r.Value.Value(i)
							}
						}
					}
				}
			}
		}
	}
	return nil
}
