// Copyright 2023 The Parca Authors
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
	"strings"

	"github.com/apache/arrow/go/v13/arrow"
	"github.com/apache/arrow/go/v13/arrow/array"
	"github.com/apache/arrow/go/v13/arrow/ipc"
	"github.com/apache/arrow/go/v13/arrow/math"
	"github.com/apache/arrow/go/v13/arrow/memory"
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
	record, cumulative := generateSourceReportRecord(
		ctx,
		pool,
		tracer,
		p,
		ref,
		source,
	)
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
		Record: buf.Bytes(),
		Source: source,
	}, cumulative, nil
}

func generateSourceReportRecord(
	_ context.Context,
	pool memory.Allocator,
	tracer trace.Tracer,
	p profile.Profile,
	ref *pb.SourceReference,
	source string,
) (arrow.Record, int64) {
	b := newSourceReportBuilder(pool, ref, int64(strings.Count(source, "\n")))
	for _, record := range p.Samples {
		b.addRecord(record)
	}

	return b.finish()
}

type sourceReportBuilder struct {
	pool memory.Allocator

	filename []byte
	buildID  []byte
	numLines int64

	flatValues       []int64
	cumulativeValues []int64

	cumulative int64
}

func newSourceReportBuilder(
	pool memory.Allocator,
	ref *pb.SourceReference,
	numLines int64,
) *sourceReportBuilder {
	return &sourceReportBuilder{
		pool: pool,

		filename: []byte(ref.Filename),
		buildID:  []byte(ref.BuildId),

		flatValues:       make([]int64, numLines),
		cumulativeValues: make([]int64, numLines),

		numLines: numLines,
	}
}

func (b *sourceReportBuilder) finish() (arrow.Record, int64) {
	flat := array.NewInt64Builder(b.pool)
	defer flat.Release()
	cumu := array.NewInt64Builder(b.pool)
	defer cumu.Release()

	flat.AppendValues(b.flatValues, nil)
	cumu.AppendValues(b.cumulativeValues, nil)

	return array.NewRecord(
		arrow.NewSchema(
			[]arrow.Field{
				{Name: "cumulative", Type: arrow.PrimitiveTypes.Int64},
				{Name: "flat", Type: arrow.PrimitiveTypes.Int64},
			},
			nil,
		),
		[]arrow.Array{
			cumu.NewInt64Array(),
			flat.NewInt64Array(),
		},
		int64(len(b.flatValues)),
	), b.cumulative
}

func (b *sourceReportBuilder) addRecord(rec arrow.Record) {
	r := profile.NewRecordReader(rec)
	b.cumulative += math.Int64.Sum(r.Value)

	for i := 0; i < int(rec.NumRows()); i++ {
		lOffsetStart, lOffsetEnd := r.Locations.ValueOffsets(i)
		for j := int(lOffsetStart); j < int(lOffsetEnd); j++ {
			if r.Mapping.IsValid(j) && bytes.Equal(r.MappingBuildIDDict.Value(r.MappingBuildID.GetValueIndex(j)), b.buildID) {
				llOffsetStart, llOffsetEnd := r.Lines.ValueOffsets(j)

				for k := int(llOffsetStart); k < int(llOffsetEnd); k++ {
					if r.Line.IsValid(k) && r.LineNumber.Value(k) <= b.numLines &&
						r.LineFunction.IsValid(k) && bytes.Equal(r.LineFunctionFilenameDict.Value(r.LineFunctionFilename.GetValueIndex(k)), b.filename) {
						b.cumulativeValues[r.LineNumber.Value(k)-1] += r.Value.Value(i)

						isLeaf := j == int(lOffsetStart) && k == int(llOffsetStart)
						if isLeaf {
							b.flatValues[r.LineNumber.Value(k)-1] += r.Value.Value(i)
						}
					}
				}
			}
		}
	}
}
