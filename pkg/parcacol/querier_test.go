package parcacol

import (
	"context"
	"testing"
	"time"

	"github.com/apache/arrow/go/v14/arrow"
	"github.com/apache/arrow/go/v14/arrow/memory"
	"github.com/go-kit/log"
	"github.com/parca-dev/parca/pkg/normalizer"
	"github.com/parca-dev/parca/pkg/profile"
	"github.com/polarsignals/frostdb/dynparquet"
	"github.com/polarsignals/frostdb/query"
	"github.com/polarsignals/frostdb/query/logicalplan"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace/noop"
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
	require.Equal(t, int64(19999999640), res[0].Samples[0].Value)      // Rounding mistakes are expected
	require.Equal(t, 9.99999982e+08, res[0].Samples[0].ValuePerSecond) // This uses the step for calculating per-second, because there are multiple samples in the step
	require.Equal(t, int64(4999999910), res[0].Samples[1].Value)       // Rounding mistakes are expected
	require.Equal(t, 4.99999991e+08, res[0].Samples[1].ValuePerSecond) // This uses the duration for calculating per-second, because there is only one sample in the step
}
