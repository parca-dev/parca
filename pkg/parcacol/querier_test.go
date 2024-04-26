// Copyright 2024 The Parca Authors
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

package parcacol

import (
	"context"
	"testing"
	"time"

	"github.com/apache/arrow/go/v15/arrow"
	"github.com/apache/arrow/go/v15/arrow/array"
	"github.com/apache/arrow/go/v15/arrow/memory"
	"github.com/go-kit/log"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/query"
	"github.com/polarsignals/frostdb/query/logicalplan"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"

	"github.com/parca-dev/parca/pkg/normalizer"
	"github.com/parca-dev/parca/pkg/profile"
)

func TestQueryDeltaRange(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	tracer := noop.NewTracerProvider().Tracer("")

	schema, err := dynparquet.SchemaFromDefinition(profile.SchemaDefinition())
	require.NoError(t, err)

	meta := profile.Meta{
		SampleType: profile.ValueType{
			Type: "samples",
			Unit: "count",
		},
		PeriodType: profile.ValueType{
			Type: "cpu",
			Unit: "nanoseconds",
		},
		Duration: (10 * time.Second).Nanoseconds(),
		Period:   (time.Second.Nanoseconds() / 19),
	}

	r1, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 10, // 19 samples per second for 10 seconds is 1 core
				}, {
					StacktraceID: "2",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (1 * time.Second).Milliseconds(),
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	r2, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (11 * time.Second).Milliseconds(), // 10 seconds later
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	r3, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (21 * time.Second).Milliseconds(), // 10 seconds later
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	q := NewQuerier(
		log.NewNopLogger(),
		tracer,
		query.NewEngine(
			mem,
			&query.FakeTableProvider{
				Tables: map[string]logicalplan.TableReader{
					"stacktraces": &query.FakeTableReader{
						FrostdbSchema: schema,
						Records:       []arrow.Record{r1, r2, r3},
					},
				},
			},
		),
		"stacktraces",
		nil,
		mem,
	)
	res, err := q.queryRangeDelta(
		context.Background(),
		nil,
		20*time.Second,
		meta,
	)
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Len(t, res[0].Samples, 2)
	require.Equal(t, int64(19999999640), res[0].Samples[0].Value)   // Rounding mistakes are expected
	require.Equal(t, 0.999999982, res[0].Samples[0].ValuePerSecond) // This uses the step for calculating per-second, because there are multiple samples in the step
	require.Equal(t, int64(4999999910), res[0].Samples[1].Value)    // Rounding mistakes are expected
	require.Equal(t, 0.499999991, res[0].Samples[1].ValuePerSecond) // This uses the duration for calculating per-second, because there is only one sample in the step
}

func TestQueryMerge(t *testing.T) {
	mem := memory.NewCheckedAllocator(memory.NewGoAllocator())
	tracer := noop.NewTracerProvider().Tracer("")

	schema, err := dynparquet.SchemaFromDefinition(profile.SchemaDefinition())
	require.NoError(t, err)

	meta := profile.Meta{
		Name: "cpu",
		SampleType: profile.ValueType{
			Type: "samples",
			Unit: "count",
		},
		PeriodType: profile.ValueType{
			Type: "cpu",
			Unit: "nanoseconds",
		},
		Duration: (10 * time.Second).Nanoseconds(),
		Period:   time.Second.Nanoseconds() / 19,
	}

	r1, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 10, // 19 samples per second for 10 seconds is 1 core
				}, {
					StacktraceID: "2",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					Name:       meta.Name,
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (1 * time.Second).Milliseconds(),
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	r2, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					Name:       meta.Name,
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (11 * time.Second).Milliseconds(), // 10 seconds later
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	r3, err := SeriesToArrowRecord(
		mem,
		schema,
		[]normalizer.Series{{
			Labels: map[string]string{"test": "test"},
			Samples: [][]*profile.NormalizedProfile{{{
				Samples: []*profile.NormalizedSample{{
					StacktraceID: "1",
					Value:        19 * 5, // 19 samples per second for 10 seconds is 0.5 cores
				}},
				Meta: profile.Meta{
					Name:       meta.Name,
					SampleType: meta.SampleType,
					PeriodType: meta.PeriodType,
					Duration:   meta.Duration,
					Period:     meta.Period,
					Timestamp:  (21 * time.Second).Milliseconds(), // 10 seconds later
				},
			}}},
		}},
		[]string{"test"}, nil, nil,
	)
	require.NoError(t, err)

	q := NewQuerier(
		log.NewNopLogger(),
		tracer,
		query.NewEngine(
			mem,
			&query.FakeTableProvider{
				Tables: map[string]logicalplan.TableReader{
					"stacktraces": &query.FakeTableReader{
						FrostdbSchema: schema,
						Records:       []arrow.Record{r1, r2, r3},
					},
				},
			},
		),
		"stacktraces",
		nil,
		mem,
	)
	records, _, _, err := q.selectMerge(
		context.Background(),
		`cpu:samples:count:cpu:nanoseconds:delta`,
		time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC),
		time.Date(3000, 0, 0, 0, 0, 0, 0, time.UTC),
		false,
	)
	require.NoError(t, err)
	require.Len(t, records, 1)

	require.Equal(t, int64(2), records[0].NumRows())
	require.Equal(t, int64(3), records[0].NumCols())

	stacktraces := records[0].Column(0).(*array.Binary)
	require.Equal(t, []string{"1", "2"}, append(
		[]string{},
		stacktraces.ValueString(0),
		stacktraces.ValueString(1),
	))

	require.Equal(t, []int64{
		19*5*meta.Period + 19*5*meta.Period + 19*10*meta.Period, // 3 samples
		19 * 5 * meta.Period, // 1 sample
	}, records[0].Column(1).(*array.Int64).Int64Values())

	require.Equal(t, []float64{
		// All 3 samples a seen over a total duration of 30s. Therefore, divide by 30s.
		float64(19*5*meta.Period+19*5*meta.Period+19*10*meta.Period) / float64((30 * time.Second).Nanoseconds()),
		// There is just one sample that was seen during a duration of 10s. Therefore, divide by 10s.
		float64(19*5*meta.Period) / float64(10*time.Second.Nanoseconds()),
	}, records[0].Column(2).(*array.Float64).Float64Values())
}
