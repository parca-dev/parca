// Copyright 2022-2023 The Parca Authors
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
	"time"

	"github.com/apache/arrow/go/v14/arrow"

	pprofproto "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

const (
	ColumnLabelsPrefix      = ColumnLabels + "."
	ColumnPprofLabelsPrefix = ColumnPprofLabels + "."
)

var LocationsField = arrow.Field{
	Name: "locations",
	Type: arrow.ListOf(arrow.StructOf([]arrow.Field{{
		Name: "address",
		Type: arrow.PrimitiveTypes.Uint64,
	}, {
		Name: "mapping",
		Type: arrow.StructOf([]arrow.Field{{
			Name: "start",
			Type: arrow.PrimitiveTypes.Uint64,
		}, {
			Name: "limit",
			Type: arrow.PrimitiveTypes.Uint64,
		}, {
			Name: "offset",
			Type: arrow.PrimitiveTypes.Uint64,
		}, {
			Name: "file",
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: "build_id",
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}}...),
	}, {
		Name: "lines",
		Type: arrow.ListOf(arrow.StructOf([]arrow.Field{{
			Name: "line",
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: "function",
			Type: arrow.StructOf([]arrow.Field{{
				Name: "name",
				Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
			}, {
				Name: "system_name",
				Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
			}, {
				Name: "filename",
				Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
			}, {
				Name: "start_line",
				Type: arrow.PrimitiveTypes.Int64,
			}}...),
		}}...)),
	}}...)),
}

func LocationsArrowSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{LocationsField}, nil)
}

func ArrowSamplesField(profileLabelFields []arrow.Field) []arrow.Field {
	numFields := len(profileLabelFields) + 3 // +3 for stacktraces, value and diff
	fields := make([]arrow.Field, numFields)
	copy(fields, profileLabelFields)
	fields[numFields-3] = LocationsField
	fields[numFields-2] = arrow.Field{
		Name: "value",
		Type: arrow.PrimitiveTypes.Int64,
	}
	fields[numFields-1] = arrow.Field{
		Name: "diff",
		Type: arrow.PrimitiveTypes.Int64,
	}

	return fields
}

func ArrowSchema(profileLabelFields []arrow.Field) *arrow.Schema {
	return arrow.NewSchema(ArrowSamplesField(profileLabelFields), nil)
}

type LocationLine struct {
	Line     int64
	Function *pb.Function
}

type Location struct {
	ID       string
	Address  uint64
	IsFolded bool
	Mapping  *pb.Mapping
	Lines    []LocationLine
}

type Label struct {
	Name  string
	Value string
}

type NumLabel struct {
	Name  string
	Value int64
}

type SymbolizedSample struct {
	Locations []*Location
	Value     int64
	DiffValue int64
	Label     map[string]string
	NumLabel  map[string]int64
}

type NormalizedSample struct {
	StacktraceID string
	Value        int64
	DiffValue    int64
	Label        map[string]string
	NumLabel     map[string]int64
}

type Profile struct {
	Samples []arrow.Record
	Meta    Meta
}

type OldProfile struct {
	Meta    Meta
	Samples []*SymbolizedSample
}

type NormalizedProfile struct {
	Samples []*NormalizedSample
	Meta    Meta
}

type ValueType struct {
	Type string
	Unit string
}

type Meta struct {
	Name       string
	PeriodType ValueType
	SampleType ValueType
	Timestamp  int64
	Duration   int64
	Period     int64
}

func MetaFromPprof(p *pprofproto.Profile, name string, sampleIndex int) Meta {
	periodType := ValueType{}
	if p.PeriodType != nil {
		periodType = ValueType{Type: p.StringTable[p.PeriodType.Type], Unit: p.StringTable[p.PeriodType.Unit]}
	}

	sampleType := ValueType{}
	if p.SampleType != nil {
		sampleType = ValueType{Type: p.StringTable[p.SampleType[sampleIndex].Type], Unit: p.StringTable[p.SampleType[sampleIndex].Unit]}
	}

	return Meta{
		Name:       name,
		Timestamp:  p.TimeNanos / time.Millisecond.Nanoseconds(),
		Duration:   p.DurationNanos,
		Period:     p.Period,
		PeriodType: periodType,
		SampleType: sampleType,
	}
}
