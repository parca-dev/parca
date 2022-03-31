package parcacol

import (
	"testing"

	"github.com/google/uuid"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/stretchr/testify/require"

	"github.com/parca-dev/parca/pkg/metastore"
)

func TestFlatProfileToBuffer(t *testing.T) {
	// Create a test sample without pprof labels.
	s := Samples{{
		SampleType:     "alloc_objects",
		SampleUnit:     "count",
		PeriodType:     "space",
		PeriodUnit:     "bytes",
		PprofLabels:    nil,
		PprofNumLabels: nil,
		Labels: labels.Labels{
			{Name: "__name__", Value: "alloc_objects_count"},
		},
		Stacktrace: extractLocationIDs([]*metastore.Location{{ID: uuid.New()}}),
		Timestamp:  1608199718549,
		Duration:   0,
		Period:     524288,
		Value:      14044,
	}}

	buf, err := s.ToBuffer(Schema())
	require.NoError(t, err)
	require.Equal(t, map[string][]string{
		columnLabels:         {"__name__"},
		columnPprofLabels:    {},
		columnPprofNumLabels: {},
	}, buf.DynamicColumns())

	// Add pprof labels to the test sample.
	s[0].PprofNumLabels = map[string]int64{"bytes": 32}

	buf, err = s.ToBuffer(Schema())
	require.NoError(t, err)
	require.Equal(t,
		map[string][]string{
			columnLabels:         {"__name__"},
			columnPprofLabels:    {},
			columnPprofNumLabels: {"bytes"},
		},
		buf.DynamicColumns(),
	)
}
