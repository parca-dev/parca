package storage

import (
	"context"
	"fmt"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/stretchr/testify/require"
)

func TestHeadIndexReader_Postings(t *testing.T) {
	ir := headIndexReader{head: NewHead(prometheus.NewRegistry())}
	ir.head.postings.Add(1, labels.Labels{{"foo", "bar"}, {"container", "test1"}})
	ir.head.postings.Add(2, labels.Labels{{"foo", "bar"}, {"container", "test2"}})
	ir.head.postings.Add(3, labels.Labels{{"foo", "baz"}, {"container", "test3"}})

	bm, err := ir.Postings("foo", "bar")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2}, bm.ToArray())

	bm, err = ir.Postings("foo", "bar", "baz")
	require.NoError(t, err)
	require.Equal(t, []uint64{1, 2, 3}, bm.ToArray())
}

func TestHeadIndexReader_LabelValues(t *testing.T) {
	h := NewHead(prometheus.NewRegistry())

	for i := 0; i < 100; i++ {
		app, err := h.Appender(context.Background(), labels.Labels{
			{Name: "unique", Value: fmt.Sprintf("value%d", i)},
			{Name: "tens", Value: fmt.Sprintf("value%d", i/10)},
		})
		require.NoError(t, err)
		err = app.Append(&Profile{
			Tree: NewProfileTree(),
			Meta: InstantProfileMeta{
				Timestamp: int64(100 + i),
			},
		})
		require.NoError(t, err)
	}

	ir, err := h.Index()
	require.NoError(t, err)

	testcases := []struct {
		name      string
		labelName string
		matchers  []*labels.Matcher
		exp       []string
	}{{
		name:      "get tens based on unique id",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchEqual, "unique", "value35")},
		exp:       []string{"value3"},
	}, {
		name:      "get unique ids based on a ten",
		labelName: "unique",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchEqual, "tens", "value1")},
		exp:       []string{"value10", "value11", "value12", "value13", "value14", "value15", "value16", "value17", "value18", "value19"},
	}, {
		name:      "get tens by pattern matching on unique id",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchRegexp, "unique", "value[5-7]5")},
		exp:       []string{"value5", "value6", "value7"},
	}, {
		name:      "get tens by matching for absence of unique label",
		labelName: "tens",
		matchers:  []*labels.Matcher{labels.MustNewMatcher(labels.MatchNotEqual, "unique", "")},
		exp:       []string{"value0", "value1", "value2", "value3", "value4", "value5", "value6", "value7", "value8", "value9"},
	}}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := ir.LabelValues(tc.labelName, tc.matchers...)
			require.NoError(t, err)
			require.ElementsMatch(t, tc.exp, values)
		})
	}
}
