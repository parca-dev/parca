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
	"strings"
	"time"

	"github.com/apache/arrow-go/v18/arrow"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/promql/parser"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pprofproto "github.com/parca-dev/parca/gen/proto/go/google/pprof"
	pb "github.com/parca-dev/parca/gen/proto/go/parca/metastore/v1alpha1"
)

const (
	ColumnLabelsPrefix = ColumnLabels + "."
)

var LocationsField = arrow.Field{
	Name: "locations",
	Type: arrow.ListOf(arrow.StructOf([]arrow.Field{{
		Name: "address",
		Type: arrow.PrimitiveTypes.Uint64,
	}, {
		Name: "mapping_start",
		Type: arrow.PrimitiveTypes.Uint64,
	}, {
		Name: "mapping_limit",
		Type: arrow.PrimitiveTypes.Uint64,
	}, {
		Name: "mapping_offset",
		Type: arrow.PrimitiveTypes.Uint64,
	}, {
		Name: "mapping_file",
		Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
	}, {
		Name: "mapping_build_id",
		Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
	}, {
		Name: "lines",
		Type: arrow.ListOf(arrow.StructOf([]arrow.Field{{
			Name: "line",
			Type: arrow.PrimitiveTypes.Int64,
		}, {
			Name: "function_name",
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: "function_system_name",
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: "function_filename",
			Type: &arrow.DictionaryType{IndexType: arrow.PrimitiveTypes.Uint32, ValueType: arrow.BinaryTypes.Binary},
		}, {
			Name: "function_start_line",
			Type: arrow.PrimitiveTypes.Int64,
		}}...)),
	}}...)),
}

func LocationsArrowSchema() *arrow.Schema {
	return arrow.NewSchema([]arrow.Field{LocationsField}, nil)
}

func ArrowSamplesField(profileLabelFields []arrow.Field) []arrow.Field {
	numFields := len(profileLabelFields) + 5 // +5 for stacktraces, value, diff, timestamp and period
	fields := make([]arrow.Field, numFields)
	copy(fields, profileLabelFields)
	fields[numFields-5] = LocationsField
	fields[numFields-4] = arrow.Field{
		Name: "value",
		Type: arrow.PrimitiveTypes.Int64,
	}
	fields[numFields-3] = arrow.Field{
		Name: "diff",
		Type: arrow.PrimitiveTypes.Int64,
	}
	fields[numFields-2] = arrow.Field{
		Name: ColumnTimestamp,
		Type: arrow.PrimitiveTypes.Int64,
	}
	fields[numFields-1] = arrow.Field{
		Name: ColumnPeriod,
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
	Samples []arrow.RecordBatch
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
	TimeNanos  int64
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
		TimeNanos:  p.TimeNanos,
		Duration:   p.DurationNanos,
		Period:     p.Period,
		PeriodType: periodType,
		SampleType: sampleType,
	}
}

// QueryParts contains parsed query components.
type QueryParts struct {
	Meta     Meta
	Delta    bool
	Matchers []*labels.Matcher
}

// ParseQuery parses a Parca query string into its components.
// The query format is: <name>:<sample-type>:<sample-unit>:<period-type>:<period-unit>[:delta]{label=value,...}.
func ParseQuery(query string) (QueryParts, error) {
	parsedSelector, err := parser.ParseMetricSelector(query)
	if err != nil {
		return QueryParts{}, status.Error(codes.InvalidArgument, "failed to parse query")
	}

	sel := make([]*labels.Matcher, 0, len(parsedSelector))
	var nameLabel *labels.Matcher
	for _, matcher := range parsedSelector {
		if matcher.Name == labels.MetricName {
			nameLabel = matcher
		} else {
			sel = append(sel, matcher)
		}
	}
	if nameLabel == nil {
		return QueryParts{}, status.Error(codes.InvalidArgument, "query must contain a profile-type selection")
	}

	parts := strings.Split(nameLabel.Value, ":")
	if len(parts) != 5 && len(parts) != 6 {
		return QueryParts{}, status.Errorf(codes.InvalidArgument, "profile-type selection must be of the form <name>:<sample-type>:<sample-unit>:<period-type>:<period-unit>(:delta), got(%d): %q", len(parts), nameLabel.Value)
	}
	delta := len(parts) == 6 && parts[5] == "delta"

	return QueryParts{
		Meta: Meta{
			Name: parts[0],
			SampleType: ValueType{
				Type: parts[1],
				Unit: parts[2],
			},
			PeriodType: ValueType{
				Type: parts[3],
				Unit: parts[4],
			},
		},
		Delta:    delta,
		Matchers: sel,
	}, nil
}
